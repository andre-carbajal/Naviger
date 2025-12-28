package ui

import (
	"fmt"
	"log"
	"naviger/pkg/sdk"
	"os"
	"os/signal"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

var (
	logBaseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		MarginLeft(2)
)

type logModel struct {
	sub       chan string
	conn      *websocket.Conn
	viewport  viewport.Model
	textInput textinput.Model
	err       error
	ready     bool
	serverID  string
	server    *sdk.Server
	content   string
	quitting  bool
	back      bool
	client    *sdk.Client
}

func initialLogModel(id string, conn *websocket.Conn, sub chan string, client *sdk.Client) logModel {
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
		client:    client,
	}
}

func (m logModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForLog(m.sub),
		getServerDetails(m.client, m.serverID),
		tickCmd(),
	)
}

type logMsg string
type errMsg2 error
type serverDetailsMsg *sdk.Server

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

func getServerDetails(client *sdk.Client, id string) tea.Cmd {
	return func() tea.Msg {
		srv, err := client.GetServer(id)
		if err != nil {
			return errMsg2(err)
		}
		return serverDetailsMsg(srv)
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
		headerHeight := 3
		footerHeight := 4
		verticalMarginHeight := headerHeight + footerHeight + 2

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
	case tickMsg:
		return m, tea.Batch(getServerDetails(m.client, m.serverID), tickCmd())
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m logModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

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

	boxWidth := m.viewport.Width + 2

	serverInfoBox := logBaseStyle.
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(serverInfoContent)

	console := logBaseStyle.
		Width(boxWidth).
		Render(m.viewport.View())

	footerContent := fmt.Sprintf(
		"→ %s\n%s",
		m.textInput.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Esc: back • Ctrl+C: quit"),
	)

	footer := logBaseStyle.
		Width(boxWidth).
		Align(lipgloss.Left).
		Render(footerContent)

	return lipgloss.JoinVertical(lipgloss.Center, serverInfoBox, console, footer)
}

func RunLogs(client *sdk.Client, id string) bool {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	wsURL, err := client.GetWebSocketURL(fmt.Sprintf("/ws/servers/%s/console", id))
	if err != nil {
		log.Fatal("Error parsing base URL:", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("Error connecting to logs: %v\nPress Enter to continue...", err)
		fmt.Scanln()
		return true
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
		initialLogModel(id, conn, sub, client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	m, err := p.Run()
	if err != nil {
		log.Printf("Error running logs UI: %v", err)
		return true
	}

	if logModel, ok := m.(logModel); ok {
		return logModel.back
	}
	return false
}
