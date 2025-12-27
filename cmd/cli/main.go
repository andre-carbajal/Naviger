package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	"github.com/spf13/cobra"
)

var BaseURL string

func main() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error getting user config directory: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "naviger")

	_, err = config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	port := config.GetPort()
	BaseURL = fmt.Sprintf("http://localhost:%d", port)

	var rootCmd = &cobra.Command{
		Use:   "naviger-cli",
		Short: "CLI for Naviger Server Manager",
	}

	// Server Commands
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Manage servers",
	}

	var createName, createVer, createLoader string
	var createRam int
	var serverCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new server",
		Run: func(cmd *cobra.Command, args []string) {
			handleCreate(createName, createVer, createLoader, createRam)
		},
	}
	serverCreateCmd.Flags().StringVar(&createName, "name", "", "Server name")
	serverCreateCmd.Flags().StringVar(&createVer, "version", "", "Minecraft version")
	serverCreateCmd.Flags().StringVar(&createLoader, "loader", "", "Loader (vanilla, paper, etc.)")
	serverCreateCmd.Flags().IntVar(&createRam, "ram", 0, "RAM in MB")
	serverCreateCmd.MarkFlagRequired("name")
	serverCreateCmd.MarkFlagRequired("version")
	serverCreateCmd.MarkFlagRequired("loader")
	serverCreateCmd.MarkFlagRequired("ram")

	var serverListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all servers",
		Run: func(cmd *cobra.Command, args []string) {
			handleList()
		},
	}

	var serverStartCmd = &cobra.Command{
		Use:   "start [id]",
		Short: "Start a server",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handleStart(args[0])
		},
	}

	var serverStopCmd = &cobra.Command{
		Use:   "stop [id]",
		Short: "Stop a server",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handleStop(args[0])
		},
	}

	var serverDeleteCmd = &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a server",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handleDelete(args[0])
		},
	}

	var serverLogsCmd = &cobra.Command{
		Use:   "logs [id]",
		Short: "View server logs and console",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handleLogs(args[0])
		},
	}

	serverCmd.AddCommand(serverCreateCmd, serverListCmd, serverStartCmd, serverStopCmd, serverDeleteCmd, serverLogsCmd)

	// Backup Commands
	var backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "Manage backups",
	}

	var backupCreateCmd = &cobra.Command{
		Use:   "create [serverId] [name]",
		Short: "Create a backup",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := ""
			if len(args) > 1 {
				name = args[1]
			}
			handleBackup(args[0], name)
		},
	}

	var backupListCmd = &cobra.Command{
		Use:   "list [serverId]",
		Short: "List backups",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				handleListBackups(args[0])
			} else {
				handleListAllBackups()
			}
		},
	}

	var backupDeleteCmd = &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a backup",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handleDeleteBackup(args[0])
		},
	}

	var restoreTarget, restoreName, restoreVer, restoreLoader string
	var restoreRam int
	var restoreNew bool
	var backupRestoreCmd = &cobra.Command{
		Use:   "restore [name]",
		Short: "Restore a backup",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handleRestoreBackup(args[0], restoreTarget, restoreNew, restoreName, restoreVer, restoreLoader, restoreRam)
		},
	}
	backupRestoreCmd.Flags().StringVar(&restoreTarget, "target", "", "Target server ID (to restore to existing)")
	backupRestoreCmd.Flags().BoolVar(&restoreNew, "new", false, "Create new server from backup")
	backupRestoreCmd.Flags().StringVar(&restoreName, "name", "", "New server name")
	backupRestoreCmd.Flags().StringVar(&restoreVer, "version", "1.20.1", "New server version")
	backupRestoreCmd.Flags().StringVar(&restoreLoader, "loader", "vanilla", "New server loader")
	backupRestoreCmd.Flags().IntVar(&restoreRam, "ram", 2048, "New server RAM")

	backupCmd.AddCommand(backupCreateCmd, backupListCmd, backupDeleteCmd, backupRestoreCmd)

	// Ports Commands
	var portsCmd = &cobra.Command{
		Use:   "ports",
		Short: "Manage port range",
	}

	var portsGetCmd = &cobra.Command{
		Use:   "get",
		Short: "Get port range",
		Run: func(cmd *cobra.Command, args []string) {
			handleGetPortRange()
		},
	}

	var portsStart, portsEnd int
	var portsSetCmd = &cobra.Command{
		Use:   "set",
		Short: "Set port range",
		Run: func(cmd *cobra.Command, args []string) {
			if portsStart == 0 || portsEnd == 0 {
				log.Fatal("Error: You must specify both --start and --end flags to update the port range")
			}
			handleSetPortRange(portsStart, portsEnd)
		},
	}
	portsSetCmd.Flags().IntVar(&portsStart, "start", 0, "Start port")
	portsSetCmd.Flags().IntVar(&portsEnd, "end", 0, "End port")

	portsCmd.AddCommand(portsGetCmd, portsSetCmd)

	// Loaders Command
	var loadersCmd = &cobra.Command{
		Use:   "loaders",
		Short: "List available loaders",
		Run: func(cmd *cobra.Command, args []string) {
			handleListLoaders()
		},
	}

	// Update Command
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Check for updates",
		Run: func(cmd *cobra.Command, args []string) {
			handleCheckUpdates()
		},
	}

	rootCmd.AddCommand(serverCmd, backupCmd, portsCmd, loadersCmd, updateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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
