package main

import (
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

	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	dashboardLoop := func() {
		for {
			serverID := handleDashboard()
			if serverID == "" {
				break
			}
			back := handleLogs(serverID)
			if !back {
				break
			}
		}
	}

	var rootCmd = &cobra.Command{
		Use:   "naviger-cli",
		Short: "CLI for Naviger Server Manager",
		Run: func(cmd *cobra.Command, args []string) {
			dashboardLoop()
		},
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

	// Dashboard Command
	var dashboardCmd = &cobra.Command{
		Use:   "dashboard",
		Short: "View live server dashboard",
		Run: func(cmd *cobra.Command, args []string) {
			dashboardLoop()
		},
	}

	rootCmd.AddCommand(serverCmd, backupCmd, portsCmd, loadersCmd, updateCmd, dashboardCmd)

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

type logModel struct {
	sub       chan string
	conn      *websocket.Conn
	viewport  viewport.Model
	textInput textinput.Model
	err       error
	ready     bool
	serverID  string
	server    *domain.Server
	content   string
	quitting  bool
	back      bool
}

func initialLogModel(id string, conn *websocket.Conn, sub chan string) logModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return logModel{
		sub:       sub,
		conn:      conn,
		textInput: ti,
		serverID:  id,
	}
}

func (m logModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForLog(m.sub),
		getServerDetails(m.serverID),
	)
}

type logMsg string
type errMsg2 error
type serverDetailsMsg *domain.Server

func waitForLog(sub chan string) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return logMsg(msg)
	}
}

func getServerDetails(id string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(BaseURL + "/servers/" + id)
		if err != nil {
			return errMsg2(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return errMsg2(fmt.Errorf("server error: %s", resp.Status))
		}

		var srv domain.Server
		if err := json.NewDecoder(resp.Body).Decode(&srv); err != nil {
			return errMsg2(err)
		}
		return serverDetailsMsg(&srv)
	}
}

func (m logModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			m.back = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" {
				cmd := m.textInput.Value()
				m.textInput.SetValue("")
				if m.conn != nil {
					_ = m.conn.WriteMessage(websocket.TextMessage, []byte(cmd+"\n"))
				}
			}
		}

	case tea.WindowSizeMsg:
		// Server Info:   1 line content + 2 border = 3
		// Footer:        2 lines content + 2 border = 4
		// Console Border: 2 lines
		// Total Chrome: 3 + 4 + 2 = 9 lines
		headerHeight := 3
		footerHeight := 4
		verticalMarginHeight := headerHeight + footerHeight + 2

		// Width Calculation:
		// Screen Width
		// - 2 (Margin Left)
		// - 2 (Margin Right / Space for symmetry)
		// - 2 (Border Left/Right)
		// = Width - 6 for CONTENT
		contentWidth := msg.Width - 6

		if !m.ready {
			m.viewport = viewport.New(contentWidth, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case logMsg:
		m.content += string(msg) + "\n"
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()
		return m, waitForLog(m.sub)

	case serverDetailsMsg:
		m.server = msg

	case errMsg2:
		m.err = msg
		return m, tea.Quit
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m logModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// 1. Server Info Header
	serverInfoContent := ""
	if m.server != nil {
		statusColor := "160"
		statusIcon := "🔴"
		if m.server.Status == "RUNNING" {
			statusColor = "42"
			statusIcon = "🟢"
		} else if m.server.Status == "STARTING" {
			statusColor = "220"
			statusIcon = "🟡"
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("63")).Padding(0, 1)
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

		serverInfoContent = fmt.Sprintf(
			"%s  %s %s  ID: %s  Port: %d  Loader: %s %s  RAM: %d MB",
			titleStyle.Render(m.server.Name),
			statusIcon,
			statusStyle.Render(m.server.Status),
			m.server.ID,
			m.server.Port,
			m.server.Loader,
			m.server.Version,
			m.server.RAM,
		)
	} else {
		serverInfoContent = "Loading server details..."
	}

	// Box Width = ContentWidth + 2 (Border)
	boxWidth := m.viewport.Width + 2

	serverInfoBox := baseStyle.
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(serverInfoContent)

	// 2. Console (Viewport)
	console := baseStyle.
		Width(boxWidth).
		// Height is determined by content (viewport) + border automatically
		Render(m.viewport.View())

	// 3. Footer
	footerContent := fmt.Sprintf(
		"→ %s\n%s",
		m.textInput.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Esc: back • Ctrl+C: quit"),
	)

	footer := baseStyle.
		Width(boxWidth).
		Align(lipgloss.Left).
		Render(footerContent)

	return lipgloss.JoinVertical(lipgloss.Center, serverInfoBox, console, footer)
}

func handleLogs(id string) bool {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(BaseURL)
	if err != nil {
		log.Fatal("Error parsing base URL:", err)
	}
	u.Scheme = "ws"
	wsURL := fmt.Sprintf("%s/ws/servers/%s/console", u.String(), id)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Error connecting to WebSocket. Is the server running? Error: %v", err)
	}
	defer conn.Close()

	sub := make(chan string)

	go func() {
		defer close(sub)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			sub <- string(message)
		}
	}()

	p := tea.NewProgram(
		initialLogModel(id, conn, sub),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	m, err := p.Run()
	if err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}

	if logModel, ok := m.(logModel); ok {
		return logModel.back
	}
	return false
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

	var (
		headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(lipgloss.Color("240"))
		rowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		statusGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		statusRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("160"))
		statusYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		statusBlue   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	)

	fmt.Println()
	fmt.Printf("%s\n", headerStyle.Render(fmt.Sprintf("%-3s %-20s %-8s %-10s %-8s %-12s", "Sts", "Name", "ID", "Loader", "Ver", "Port")))

	for _, s := range servers {
		status := "🔴"
		sStyle := statusRed
		if s.Status == "RUNNING" {
			status = "🟢"
			sStyle = statusGreen
		} else if s.Status == "STARTING" {
			status = "🟡"
			sStyle = statusYellow
		} else if s.Status == "STOPPING" {
			status = "🟠"
			sStyle = statusRed
		} else if s.Status == "CREATING" {
			status = "�"
			sStyle = statusBlue
		}

		sIcon := sStyle.Render(status)
		rest := rowStyle.Render(fmt.Sprintf("%-20s %-8s %-10s %-8s %-12d", s.Name, s.ID, s.Loader, s.Version, s.Port))

		fmt.Printf("%s %s\n", sIcon, rest)
	}
	fmt.Println()
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

func startServer(id string) error {
	reqURL := fmt.Sprintf("%s/servers/%s/start", BaseURL, id)
	resp, err := http.Post(reqURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf("error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start failed: %s", string(body))
	}
	return nil
}

func handleStart(id string) {
	if err := startServer(id); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Start command sent. The server will start in the background.")
}

func stopServer(id string) error {
	reqURL := fmt.Sprintf("%s/servers/%s/stop", BaseURL, id)
	resp, err := http.Post(reqURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf("error connecting to Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop failed: %s", string(body))
	}
	return nil
}

func handleStop(id string) {
	if err := stopServer(id); err != nil {
		log.Fatal(err)
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

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			MarginLeft(2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Align(lipgloss.Center)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Align(lipgloss.Center)
)

type model struct {
	table     table.Model
	servers   []domain.Server
	stats     map[string]domain.ServerStats
	err       error
	width     int
	height    int
	isLoading bool
	message   string
}

type serverDataMsg struct {
	servers []domain.Server
	stats   map[string]domain.ServerStats
}

type errMsg error

func handleDashboard() string {
	columns := []table.Column{
		{Title: "Sts", Width: 3},
		{Title: "ID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "Port", Width: 6},
		{Title: "Ver", Width: 8},
		{Title: "Loader", Width: 10},
		{Title: "CPU", Width: 8},
		{Title: "RAM", Width: 15},
		{Title: "Disk", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{
		table:     t,
		isLoading: true,
		stats:     make(map[string]domain.ServerStats),
	}

	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	finalModel, err := program.Run()
	if err != nil {
		fmt.Printf("Error running dashboard: %v", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(model); ok {
		if m.err != nil {
			if m.err.Error() == "quit" {
				return ""
			}
		}

		if m.message == "navigate_logs" {
			selectedRow := m.table.SelectedRow()
			if len(selectedRow) > 1 {
				return selectedRow[1]
			}
		}
	}

	return ""
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchDataCmd(),
		tickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	varcmd := func() tea.Cmd { return nil }
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.err = fmt.Errorf("quit")
			return m, tea.Quit
		case "s":
			selectedRow := m.table.SelectedRow()
			if len(selectedRow) > 1 {
				id := selectedRow[1]
				// Find server status
				var status string
				for _, s := range m.servers {
					if s.ID == id {
						status = s.Status
						break
					}
				}

				if status == "RUNNING" || status == "STARTING" {
					m.message = fmt.Sprintf("Server %s is already %s", id, status)
				} else {
					go startServer(id)
					m.message = fmt.Sprintf("Starting server %s...", id)
				}

				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return "clear_message"
				})
			}
		case "x":
			selectedRow := m.table.SelectedRow()
			if len(selectedRow) > 1 {
				id := selectedRow[1]
				// Find server status
				var status string
				for _, s := range m.servers {
					if s.ID == id {
						status = s.Status
						break
					}
				}

				if status != "RUNNING" && status != "STARTING" {
					m.message = fmt.Sprintf("Server %s is not running (Status: %s)", id, status)
				} else {
					go stopServer(id)
					m.message = fmt.Sprintf("Stopping server %s...", id)
				}

				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return "clear_message"
				})
			}
		case "enter":
			m.message = "navigate_logs"
			return m, tea.Quit
		case "clear_message":
			m.message = ""
			return m, nil
		}
	case string:
		if msg == "clear_message" {
			m.message = ""
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 10)
		m.table.SetHeight(msg.Height - 10)
	case serverDataMsg:
		m.isLoading = false
		m.servers = msg.servers
		m.stats = msg.stats
		m.updateTable()
		return m, nil
	case tickMsg:
		return m, tea.Batch(fetchDataCmd(), tickCmd())
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, tea.Batch(varcmd(), cmd)
}

func (m *model) updateTable() {
	rows := []table.Row{}
	for _, s := range m.servers {
		status := "🔴"
		if s.Status == "RUNNING" {
			status = "🟢"
		} else if s.Status == "STARTING" {
			status = "🟡"
		} else if s.Status == "STOPPING" {
			status = "🟠"
		} else if s.Status == "CREATING" {
			status = "🔵"
		}

		cpu := "-"
		ram := "-"
		disk := "-"
		if stat, ok := m.stats[s.ID]; ok && (s.Status == "RUNNING" || stat.Disk > 0) {
			if s.Status == "RUNNING" {
				cpu = fmt.Sprintf("%.1f%%", stat.CPU)
				ram = fmt.Sprintf("%s / %dMB", formatBytesShort(int64(stat.RAM)), s.RAM)
			}
			disk = formatBytesShort(stat.Disk)
		}

		rows = append(rows, table.Row{
			status,
			s.ID,
			s.Name,
			fmt.Sprintf("%d", s.Port),
			s.Version,
			s.Loader,
			cpu,
			ram,
			disk,
		})
	}
	m.table.SetRows(rows)
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Render("NAVIGER")
	clock := subHeaderStyle.Render(time.Now().Format("Mon Jan 2 15:04:05"))

	hostInfo := fmt.Sprintf("Daemon: %s  |  Servers: %d", BaseURL, len(m.servers))
	headerBox := baseStyle.
		Width(m.width-4).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Center, title, clock, " ", hostInfo))

	tableContainer := baseStyle.
		Width(m.width - 4).
		Height(m.height - 12).
		Render(m.table.View())

	statusLine := "↑/↓: navigate • s: start • x: stop • enter: logs • q: quit"
	footerText := lipgloss.NewStyle().
		MarginLeft(2).
		Foreground(lipgloss.Color("240")).
		Render(statusLine)

	if m.message != "" {
		footerText = fmt.Sprintf("%s\n%s",
			lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("205")).Bold(true).Render(m.message),
			footerText)
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		headerBox,
		tableContainer,
		footerText,
	)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		serversResp, err := http.Get(BaseURL + "/servers")
		if err != nil {
			return errMsg(err)
		}
		defer serversResp.Body.Close()

		var servers []domain.Server
		if err := json.NewDecoder(serversResp.Body).Decode(&servers); err != nil {
			return errMsg(err)
		}

		statsResp, err := http.Get(BaseURL + "/servers-stats")
		var stats map[string]domain.ServerStats
		if err == nil && statsResp.StatusCode == 200 {
			defer statsResp.Body.Close()
			_ = json.NewDecoder(statsResp.Body).Decode(&stats)
		}

		return serverDataMsg{servers: servers, stats: stats}
	}
}

func formatBytesShort(bytes int64) string {
	if bytes == 0 {
		return "0B"
	}
	const k = 1024
	sizes := []string{"B", "K", "M", "G", "T"}
	i := 0
	fBytes := float64(bytes)
	for fBytes >= k && i < len(sizes)-1 {
		fBytes /= k
		i++
	}
	return fmt.Sprintf("%.1f%s", fBytes, sizes[i])
}
