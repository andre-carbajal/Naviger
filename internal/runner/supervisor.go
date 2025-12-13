package runner

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"mc-manager/internal/jvm"
	"mc-manager/internal/storage"
	"mc-manager/internal/ws"
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
		return fmt.Errorf("el servidor ya está corriendo")
	}

	srv, err := s.Store.GetServerByID(serverID)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("servidor no encontrado")
	}

	serverDir := filepath.Join(s.ServersPath, srv.ID)
	absServerDir, err := filepath.Abs(serverDir)
	if err != nil {
		return fmt.Errorf("error obteniendo ruta absoluta del server: %w", err)
	}

	configFile := filepath.Join(absServerDir, "server.properties")
	if err := ensurePortInProperties(configFile, srv.Port); err != nil {
		fmt.Printf("⚠Advertencia: No se pudo actualizar server.properties: %v\n", err)
	}

	requiredJava := GetJavaVersionForMC(srv.Version)
	javaPath, err := s.JVM.EnsureJava(requiredJava)
	if err != nil {
		return fmt.Errorf("error preparando Java: %w", err)
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
			return fmt.Errorf("directorio libraries no encontrado en %s (necesario para Forge/NeoForge)", librariesDir)
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
				return fmt.Errorf("no se encontró el archivo de argumentos %s en libraries", targetFile)
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
				return fmt.Errorf("server jar no encontrado en %s", jarFull)
			}
			return fmt.Errorf("error accediendo a %s: %w", jarFull, err)
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
		return fmt.Errorf("falló el arranque: %w", err)
	}

	if err := s.Store.UpdateStatus(serverID, "RUNNING"); err != nil {
		fmt.Printf("advertencia: no se pudo actualizar el estado a RUNNING: %v\n", err)
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
				fmt.Printf("advertencia: no se pudo actualizar el estado a STOPPED: %v\n", uerr)
			}
			return
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			_ = exitErr.ExitCode()
			if uerr := s.Store.UpdateStatus(id, "ERROR"); uerr != nil {
				fmt.Printf("advertencia: no se pudo actualizar el estado a ERROR: %v\n", uerr)
			}
		} else {
			if uerr := s.Store.UpdateStatus(id, "ERROR"); uerr != nil {
				fmt.Printf("advertencia: no se pudo actualizar el estado a ERROR: %v\n", uerr)
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
		return fmt.Errorf("servidor no está corriendo")
	}

	if err := s.Store.UpdateStatus(serverID, "STOPPING"); err != nil {
		fmt.Printf("advertencia: no se pudo actualizar el estado a STOPPING: %v\n", err)
	}
	_, err := io.WriteString(proc.Stdin, "stop\n")
	return err
}

func (s *Supervisor) SendCommand(serverID string, cmd string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("servidor no está corriendo")
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
