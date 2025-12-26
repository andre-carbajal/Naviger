package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"naviger/internal/config"
	"naviger/internal/domain"
	"naviger/internal/updater"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var BaseURL string

func printHelp() {
	prog := filepath.Base(os.Args[0])
	fmt.Printf("Usage: %s <resource> <action> [flags]\n\n", prog)
	fmt.Println("Resources and actions:")
	fmt.Printf("  %-60s %s\n", "server create --name <name> --version <version> --loader <loader> --ram <MB>", "Create new server")
	fmt.Printf("  %-60s %s\n", "server list", "List servers")
	fmt.Printf("  %-60s %s\n", "server start <id>", "Start server")
	fmt.Printf("  %-60s %s\n", "server stop <id>", "Stop server")
	fmt.Printf("  %-60s %s\n", "server delete <id>", "Delete server")
	fmt.Printf("  %-60s %s\n", "server logs <id>", "View server console and send commands")
	fmt.Println()
	fmt.Printf("  %-60s %s\n", "backup create <id> [name]", "Create server backup")
	fmt.Printf("  %-60s %s\n", "backup list [id]", "List backups (all or by server)")
	fmt.Printf("  %-60s %s\n", "backup delete <name>", "Delete backup")
	fmt.Printf("  %-60s %s\n", "backup restore <name> --target <id>", "Restore backup to existing server")
	fmt.Printf("  %-60s %s\n", "backup restore <name> --new --name <name> --version <ver> --loader <loader> --ram <MB>", "Restore backup to new server")
	fmt.Println()
	fmt.Printf("  %-60s %s\n", "ports get", "Show port range")
	fmt.Printf("  %-60s %s\n", "ports set --start <n> --end <m>", "Set port range")
	fmt.Println()
	fmt.Printf("  %-60s %s\n", "loaders", "Show available server loaders")
	fmt.Printf("  %-60s %s\n", "update", "Check for updates")
	fmt.Printf("  %-60s %s\n", "help", "Show this help message")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Printf("  %s server create --name \"My Server\" --version \"1.20.1\" --loader \"vanilla\" --ram 2048\n", prog)
}

func parseFlags(fs *flag.FlagSet, args []string, ctx string) {
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Error parsing flags for %s: %v", ctx, err)
	}
}

func main() {
	flag.Usage = printHelp
	flag.Parse()

	port := config.GetPort()
	BaseURL = fmt.Sprintf("http://localhost:%d", port)

	args := flag.Args()
	if len(args) < 1 {
		printHelp()
		os.Exit(1)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error getting user config directory: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "naviger")

	_, err = config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	serverCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	serverListCmd := flag.NewFlagSet("list", flag.ExitOnError)
	serverStartCmd := flag.NewFlagSet("start", flag.ExitOnError)
	serverStopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	serverDeleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)

	serverCreateName := serverCreateCmd.String("name", "", "Server name")
	serverCreateVer := serverCreateCmd.String("version", "", "Minecraft version")
	serverCreateLoader := serverCreateCmd.String("loader", "", "Loader (vanilla, paper, etc.)")
	serverCreateRam := serverCreateCmd.Int("ram", 0, "RAM in MB")

	backupCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	backupListCmd := flag.NewFlagSet("list", flag.ExitOnError)
	backupDeleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	backupRestoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)

	backupRestoreTarget := backupRestoreCmd.String("target", "", "Target server ID (to restore to existing)")
	backupRestoreNew := backupRestoreCmd.Bool("new", false, "Create new server from backup")
	backupRestoreName := backupRestoreCmd.String("name", "", "New server name")
	backupRestoreVer := backupRestoreCmd.String("version", "1.20.1", "New server version")
	backupRestoreLoader := backupRestoreCmd.String("loader", "vanilla", "New server loader")
	backupRestoreRam := backupRestoreCmd.Int("ram", 2048, "New server RAM")

	portsGetCmd := flag.NewFlagSet("get", flag.ExitOnError)
	portsSetCmd := flag.NewFlagSet("set", flag.ExitOnError)
	portsSetStart := portsSetCmd.Int("start", 0, "Start port")
	portsSetEnd := portsSetCmd.Int("end", 0, "End port")

	loadersCmd := flag.NewFlagSet("loaders", flag.ExitOnError)
	logsCmd := flag.NewFlagSet("logs", flag.ExitOnError)
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "server":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: naviger-cli server <subcommand>")
			fmt.Println("Subcommands: create, list, start, stop, delete, logs")
			os.Exit(1)
		}
		sub := cmdArgs[0]
		subArgs := cmdArgs[1:]

		switch sub {
		case "create":
			parseFlags(serverCreateCmd, subArgs, "server create")
			handleCreate(*serverCreateName, *serverCreateVer, *serverCreateLoader, *serverCreateRam)

		case "list":
			parseFlags(serverListCmd, subArgs, "server list")
			handleList()

		case "start":
			parseFlags(serverStartCmd, subArgs, "server start")
			if serverStartCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the server ID. Ex: naviger-cli server start <UUID>")
			}
			handleStart(serverStartCmd.Arg(0))

		case "stop":
			parseFlags(serverStopCmd, subArgs, "server stop")
			if serverStopCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the server ID.")
			}
			handleStop(serverStopCmd.Arg(0))

		case "delete":
			parseFlags(serverDeleteCmd, subArgs, "server delete")
			if serverDeleteCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the server ID.")
			}
			handleDelete(serverDeleteCmd.Arg(0))

		case "logs":
			parseFlags(logsCmd, subArgs, "server logs")
			if logsCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the server ID. Ex: naviger-cli server logs <UUID>")
			}
			handleLogs(logsCmd.Arg(0))

		default:
			fmt.Println("Unknown subcommand for 'server':", sub)
			os.Exit(1)
		}

	case "backup":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: naviger-cli backup <subcommand>")
			fmt.Println("Subcommands: create, list, delete, restore")
			os.Exit(1)
		}
		sub := cmdArgs[0]
		subArgs := cmdArgs[1:]

		switch sub {
		case "create":
			parseFlags(backupCreateCmd, subArgs, "backup create")
			if backupCreateCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the server ID. Ex: naviger-cli backup create <UUID> [optional-name]")
			}
			serverID := backupCreateCmd.Arg(0)
			backupName := ""
			if backupCreateCmd.NArg() > 1 {
				backupName = backupCreateCmd.Arg(1)
			}
			handleBackup(serverID, backupName)

		case "list":
			parseFlags(backupListCmd, subArgs, "backup list")
			if backupListCmd.NArg() > 0 {
				handleListBackups(backupListCmd.Arg(0))
			} else {
				handleListAllBackups()
			}

		case "delete":
			parseFlags(backupDeleteCmd, subArgs, "backup delete")
			if backupDeleteCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the backup name.")
			}
			handleDeleteBackup(backupDeleteCmd.Arg(0))

		case "restore":
			parseFlags(backupRestoreCmd, subArgs, "backup restore")
			if backupRestoreCmd.NArg() < 1 {
				log.Fatal("Error: You must specify the backup name.")
			}
			backupName := backupRestoreCmd.Arg(0)
			handleRestoreBackup(backupName, *backupRestoreTarget, *backupRestoreNew, *backupRestoreName, *backupRestoreVer, *backupRestoreLoader, *backupRestoreRam)

		default:
			fmt.Println("Unknown subcommand for 'backup':", sub)
			fmt.Println("Usage: naviger-cli backup <subcommand>")
			fmt.Println("Subcommands: create, list, delete, restore")
			os.Exit(1)
		}

	case "ports":
		if len(cmdArgs) < 1 {
			fmt.Println("Usage: naviger-cli ports <subcommand>")
			fmt.Println("Subcommands: get, set")
			os.Exit(1)
		}
		sub := cmdArgs[0]
		subArgs := cmdArgs[1:]

		switch sub {
		case "get":
			parseFlags(portsGetCmd, subArgs, "ports get")
			handleGetPortRange()

		case "set":
			parseFlags(portsSetCmd, subArgs, "ports set")
			if *portsSetStart == 0 || *portsSetEnd == 0 {
				log.Fatal("Error: You must specify both --start and --end flags to update the port range")
			}
			handleSetPortRange(*portsSetStart, *portsSetEnd)

		default:
			fmt.Println("Unknown subcommand for 'ports':", sub)
			fmt.Println("Usage: naviger-cli ports <subcommand>")
			fmt.Println("Subcommands: get, set")
			os.Exit(1)
		}

	case "loaders":
		parseFlags(loadersCmd, cmdArgs, "loaders")
		handleListLoaders()

	case "update":
		parseFlags(updateCmd, cmdArgs, "update")
		handleCheckUpdates()

	case "help":
		parseFlags(helpCmd, cmdArgs, "help")
		printHelp()

	default:
		fmt.Println("Unknown command:", command)
		printHelp()
		os.Exit(1)
	}
}

func handleCheckUpdates() {
	resp, err := http.Get(BaseURL + "/updates")
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Server error: %s", resp.Status)
	}

	var updateInfo updater.UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&updateInfo); err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Println("\n--- UPDATE CHECK ---")
	fmt.Printf("Current version: %s\n", updateInfo.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", updateInfo.LatestVersion)

	if updateInfo.UpdateAvailable {
		fmt.Println("\nUpdate available!")
		fmt.Printf("Download it here: %s\n", updateInfo.ReleaseURL)
	} else {
		fmt.Println("\nYou are up to date.")
	}
}

func handleListLoaders() {
	resp, err := http.Get(BaseURL + "/loaders")
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v\n(Is the server running in another terminal?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Server error: %s", resp.Status)
	}

	var loaders []string
	if err := json.NewDecoder(resp.Body).Decode(&loaders); err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Println("\n--- AVAILABLE LOADERS ---")
	for _, l := range loaders {
		fmt.Printf("- %s\n", l)
	}
}

func handleDelete(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s", BaseURL, id)
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error deleting server: %s", string(body))
	}

	fmt.Println("Server deleted successfully.")
}

func handleDeleteBackup(name string) {
	reqURL := fmt.Sprintf("%s/backups/%s", BaseURL, name)
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error deleting backup: %s", string(body))
	}

	fmt.Println("Backup deleted successfully.")
}

func handleRestoreBackup(backupName, targetID string, isNew bool, newName, newVer, newLoader string, newRam int) {
	reqURL := fmt.Sprintf("%s/backups/%s/restore", BaseURL, backupName)

	payload := map[string]interface{}{}

	if isNew {
		if newName == "" {
			log.Fatal("Error: You must specify --name for the new server")
		}
		payload["newServerName"] = newName
		payload["newServerVersion"] = newVer
		payload["newServerLoader"] = newLoader
		payload["newServerRam"] = newRam
	} else {
		if targetID == "" {
			log.Fatal("Error: You must specify --target <ID> or use --new")
		}
		payload["targetServerId"] = targetID
	}

	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(reqURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error restoring backup: %s", string(body))
	}

	fmt.Println("Backup restored successfully.")
}

func handleListAllBackups() {
	reqURL := fmt.Sprintf("%s/backups", BaseURL)
	resp, err := http.Get(reqURL)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Server error: %s", string(body))
	}

	var backups []domain.BackupInfo
	if err := json.NewDecoder(resp.Body).Decode(&backups); err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Println("\n--- ALL BACKUPS ---")
	for _, b := range backups {
		fmt.Printf("- %s (%.2f MB)\n", b.Name, float64(b.Size)/1024/1024)
	}
}

func handleListBackups(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s/backups", BaseURL, id)
	resp, err := http.Get(reqURL)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Server error: %s", string(body))
	}

	var backups []domain.BackupInfo
	if err := json.NewDecoder(resp.Body).Decode(&backups); err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Printf("\n--- BACKUPS FOR SERVER %s ---\n", id)
	for _, b := range backups {
		fmt.Printf("- %s (%.2f MB)\n", b.Name, float64(b.Size)/1024/1024)
	}
}

func handleLogs(id string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(BaseURL)
	if err != nil {
		log.Fatal("Error parsing base URL:", err)
	}
	u.Scheme = "ws"
	wsURL := fmt.Sprintf("%s/ws/servers/%s/console", u.String(), id)

	fmt.Printf("Connecting to console of %s...\n", id)
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Error connecting to WebSocket. Is the server running? Error: %v", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("Unexpected error reading message: %v", err)
				}
				fmt.Println("\nDisconnected from console.")
				return
			}
			fmt.Println(string(message))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			err := c.WriteMessage(websocket.TextMessage, []byte(input+"\n"))
			if err != nil {
				return
			}
		}
	}()

	fmt.Println("Connected. Type commands and press Enter. Press Ctrl+C to exit.")

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error sending close message:", err)
			}
			return
		}
	}
}

func handleList() {
	resp, err := http.Get(BaseURL + "/servers")
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v\n(Is the server running in another terminal?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Server error: %s", resp.Status)
	}

	var servers []domain.Server
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Println("\n--- REMOTE SERVERS ---")
	for _, s := range servers {
		statusIcon := "🔴"
		if s.Status == "RUNNING" {
			statusIcon = "🟢"
		} else if s.Status == "STARTING" {
			statusIcon = "🟡"
		}

		fmt.Printf("%s [%s] %s (v%s)\n", statusIcon, s.ID, s.Name, s.Version)
		fmt.Printf("      Port: %d | RAM: %dMB | Loader: %s\n", s.Port, s.RAM, s.Loader)
	}
}

func handleCreate(name, version, loader string, ram int) {
	if name == "" || version == "" || loader == "" || ram == 0 {
		log.Println("Error: Missing arguments to create server.")
		fmt.Println("\nCorrect usage:")
		fmt.Println("  naviger-cli server create --name \"My Server\" --version \"1.20.1\" --loader \"vanilla\" --ram 2048")
		os.Exit(1)
	}

	requestID := uuid.New().String()

	payload := map[string]interface{}{
		"name":      name,
		"version":   version,
		"loader":    loader,
		"ram":       ram,
		"requestId": requestID,
	}
	jsonData, _ := json.Marshal(payload)

	u, err := url.Parse(BaseURL)
	if err != nil {
		log.Fatal("Error parsing base URL:", err)
	}
	u.Scheme = "ws"
	wsURL := fmt.Sprintf("%s/ws/progress/%s", u.String(), requestID)

	done := make(chan struct{})

	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("Warning: Could not connect to progress WebSocket: %v", err)
		close(done)
	} else {
		defer c.Close()
		go func() {
			defer close(done)
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					return
				}
				var event domain.ProgressEvent
				if err := json.Unmarshal(message, &event); err == nil {
					fmt.Printf("\r[Progress] %s", event.Message)
					if event.Progress == 100 {
						fmt.Println()
						return
					}
				}
			}
		}()
	}

	resp, err := http.Post(BaseURL+"/servers", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error creating server: %s", string(body))
	}

	fmt.Println("\nCreation request received. Waiting for completion...")

	if c != nil {
		<-done
	}
}

func handleStart(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s/start", BaseURL, id)
	resp, err := http.Post(reqURL, "application/json", nil)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Start failed: %s", string(body))
	}

	fmt.Println("Start command sent. The server will start in the background.")
}

func handleStop(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s/stop", BaseURL, id)
	resp, err := http.Post(reqURL, "application/json", nil)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Stop failed: %s", string(body))
	}

	fmt.Println("Stop command sent.")
}

func handleBackup(id, name string) {
	reqURL := fmt.Sprintf("%s/servers/%s/backup", BaseURL, id)

	payload := map[string]string{
		"name": name,
	}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(reqURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Backup creation failed: %s", string(body))
	}

	var backupResponse struct {
		Message string `json:"message"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&backupResponse); err != nil {
		log.Fatalf("Error reading backup response: %v", err)
	}

	fmt.Println(backupResponse.Message)
	fmt.Printf("Location: %s\n", backupResponse.Path)
}

func handleGetPortRange() {
	resp, err := http.Get(BaseURL + "/settings/port-range")
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error getting configuration: %s", string(body))
	}

	var portRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&portRange); err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Println("\n--- PORT CONFIGURATION ---")
	fmt.Printf("Start port: %d\n", portRange.Start)
	fmt.Printf("End port:   %d\n", portRange.End)
	fmt.Printf("Range:      %d ports available\n", portRange.End-portRange.Start+1)
}

func handleSetPortRange(start, end int) {
	if start == 0 || end == 0 {
		log.Fatal("Error: You must specify both ports (--start and --end)")
	}

	if start > end {
		log.Fatal("Error: Start port must be less than or equal to end port")
	}

	payload := map[string]int{
		"start": start,
		"end":   end,
	}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, BaseURL+"/settings/port-range", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error updating configuration: %s", string(body))
	}

	fmt.Println("Port configuration updated successfully!")
	fmt.Printf("New range: %d - %d\n", start, end)
}
