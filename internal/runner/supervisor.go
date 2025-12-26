package runner

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"naviger/internal/jvm"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Supervisor struct {
	Store       *storage.GormStore
	JVM         *jvm.Manager
	HubManager  *ws.HubManager
	ServersPath string
	processes   map[string]*ActiveProcess
	mu          sync.Mutex
}

type ActiveProcess struct {
	Cmd   *exec.Cmd
	Stdin io.WriteCloser
}

func NewSupervisor(store *storage.GormStore, jvm *jvm.Manager, hubManager *ws.HubManager, serversPath string) *Supervisor {
	return &Supervisor{
		Store:       store,
		JVM:         jvm,
		HubManager:  hubManager,
		ServersPath: serversPath,
		processes:   make(map[string]*ActiveProcess),
	}
}

func (s *Supervisor) StartServer(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.processes[serverID]; exists {
		return fmt.Errorf("server is already running")
	}

	srv, err := s.Store.GetServerByID(serverID)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("server not found")
	}

	serverDir := filepath.Join(s.ServersPath, srv.ID)
	absServerDir, err := filepath.Abs(serverDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path for server: %w", err)
	}

	if err := checkPortAvailable(srv.Port); err != nil {
		fmt.Printf("Port %d is busy, attempting to allocate a new one...\n", srv.Port)
		newPort, err := server.AllocatePort(s.Store)
		if err != nil {
			return fmt.Errorf("failed to allocate new port: %w", err)
		}

		if err := s.Store.UpdateServerPort(srv.ID, newPort); err != nil {
			return fmt.Errorf("failed to update server port in database: %w", err)
		}
		srv.Port = newPort
		fmt.Printf("Reassigned server %s to port %d\n", srv.Name, newPort)
	}

	configFile := filepath.Join(absServerDir, "server.properties")
	if err := ensurePortInProperties(configFile, srv.Port); err != nil {
		fmt.Printf("Warning: Could not update server.properties: %v\n", err)
	}

	requiredJava := GetJavaVersionForMC(srv.Version)
	javaPath, err := s.JVM.EnsureJava(requiredJava)
	if err != nil {
		return fmt.Errorf("error preparing Java: %w", err)
	}

	var cmd *exec.Cmd
	var args []string

	if srv.Loader == "forge" || srv.Loader == "neoforge" {
		librariesDir := filepath.Join(absServerDir, "libraries")
		var argsFile string
		targetFile := "unix_args.txt"
		if runtime.GOOS == "windows" {
			targetFile = "win_args.txt"
		}

		if _, err := os.Stat(librariesDir); os.IsNotExist(err) {
			return fmt.Errorf("libraries directory not found in %s (required for Forge/NeoForge)", librariesDir)
		}

		err := filepath.WalkDir(librariesDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == targetFile {
				argsFile = path
				return io.EOF
			}
			return nil
		})

		if argsFile == "" {
			if err != io.EOF {
				return fmt.Errorf("args file %s not found in libraries", targetFile)
			}
		}

		args = []string{
			fmt.Sprintf("-Xmx%dM", srv.RAM),
			"-Xms512M",
		}

		userJvmArgs := filepath.Join(absServerDir, "user_jvm_args.txt")
		if _, err := os.Stat(userJvmArgs); err == nil {
			args = append(args, fmt.Sprintf("@%s", userJvmArgs))
		}

		args = append(args, fmt.Sprintf("@%s", argsFile))

		args = append(args, "nogui")

	} else {
		jarPath := "server.jar"
		jarFull := filepath.Join(absServerDir, jarPath)
		if _, err := os.Stat(jarFull); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("server jar not found at %s", jarFull)
			}
			return fmt.Errorf("error accessing %s: %w", jarFull, err)
		}

		args = []string{
			fmt.Sprintf("-Xmx%dM", srv.RAM),
			"-Xms512M",
			"-jar", jarPath,
			"nogui",
		}
	}

	cmd = exec.Command(javaPath, args...)
	cmd.Dir = absServerDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	hub := s.HubManager.GetHub(serverID)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			text := scanner.Text()
			hub.Broadcast([]byte(text))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			hub.Broadcast([]byte(text))
		}
	}()

	go func() {
		for command := range hub.Commands {
			_, err := io.WriteString(stdin, string(command))
			if err != nil {
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	if err := s.Store.UpdateStatus(serverID, "RUNNING"); err != nil {
		fmt.Printf("warning: could not update status to RUNNING: %v\n", err)
	}

	s.processes[serverID] = &ActiveProcess{
		Cmd:   cmd,
		Stdin: stdin,
	}

	go func(id string, c *exec.Cmd) {
		err := c.Wait()

		s.mu.Lock()
		delete(s.processes, id)
		s.mu.Unlock()

		s.HubManager.RemoveHub(id)

		if err == nil {
			if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
				fmt.Printf("warning: could not update status to STOPPED: %v\n", uerr)
			}
			return
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			_ = exitErr.ExitCode()
			if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
				fmt.Printf("warning: could not update status to STOPPED: %v\n", uerr)
			}
		} else {
			if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
				fmt.Printf("warning: could not update status to STOPPED: %v\n", uerr)
			}
		}

	}(serverID, cmd)

	return nil
}

func (s *Supervisor) StopServer(serverID string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("server is not running")
	}

	if err := s.Store.UpdateStatus(serverID, "STOPPING"); err != nil {
		fmt.Printf("warning: could not update status to STOPPING: %v\n", err)
	}
	_, err := io.WriteString(proc.Stdin, "stop\n")
	return err
}

func (s *Supervisor) SendCommand(serverID string, cmd string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("server is not running")
	}

	_, err := io.WriteString(proc.Stdin, cmd+"\n")
	return err
}

func ensurePortInProperties(path string, port int) error {
	props := make(map[string]string)
	var lines []string

	if file, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)

			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				props[key] = val
			}
		}
		file.Close()
	}

	portStr := fmt.Sprintf("%d", port)
	if currentVal, ok := props["server-port"]; ok && currentVal == portStr {
		return nil
	}

	var newContent []string
	portUpdated := false

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "server-port=") || strings.HasPrefix(strings.TrimSpace(line), "server-port =") {
			newContent = append(newContent, fmt.Sprintf("server-port=%s", portStr))
			portUpdated = true
		} else {
			newContent = append(newContent, line)
		}
	}

	if !portUpdated {
		newContent = append(newContent, fmt.Sprintf("server-port=%s", portStr))
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range newContent {
		writer.WriteString(line + "\n")
	}
	return writer.Flush()
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port %d is not available: %w", port, err)
	}
	_ = ln.Close()
	return nil
}
