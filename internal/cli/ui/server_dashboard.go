package ui

import (
	"fmt"
	"naviger/pkg/sdk"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			MarginLeft(2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Align(lipgloss.Center)
)

type serverListItem struct {
	id          string
	title       string
	description string
}

func (i serverListItem) FilterValue() string { return i.title + " " + i.description }
func (i serverListItem) Title() string       { return i.title }
func (i serverListItem) Description() string { return i.description }

type model struct {
	list      list.Model
	servers   []sdk.Server
	stats     map[string]sdk.ServerStats
	err       error
	width     int
	height    int
	isLoading bool
	message   string
	client    *sdk.Client
	wizard    tea.Model
	mode      dashboardMode
}

type dashboardMode int

const (
	ViewDashboard dashboardMode = iota
	ViewWizard
)

type serverDataMsg struct {
	servers []sdk.Server
	stats   map[string]sdk.ServerStats
}

type errMsg error

func RunServerDashboard(client *sdk.Client) string {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Servers"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = headerStyle

	m := model{
		list:      l,
		isLoading: true,
		stats:     make(map[string]sdk.ServerStats),
		client:    client,
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
			if m.mode == ViewDashboard {
				i := m.list.SelectedItem()
				if i != nil {
					return i.(serverListItem).id
				}
			}
		}
	}

	return ""
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchDataCmd(m.client),
		tickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	varcmd := func() tea.Cmd { return nil }
	var cmd tea.Cmd

	if m.mode == ViewWizard {
		var wCmd tea.Cmd
		m.wizard, wCmd = m.wizard.Update(msg)

		switch msg.(type) {
		case WizardDoneMsg:
			m.mode = ViewDashboard
			m.message = "Server creation started!"
			return m, tea.Batch(fetchDataCmd(m.client), tickCmd(), func() tea.Msg { return "clear_message" })
		case WizardCancelMsg:
			m.mode = ViewDashboard
			m.message = "Server creation cancelled."
			return m, tea.Batch(tickCmd(), func() tea.Msg { return "clear_message" })
		case tea.WindowSizeMsg:
		}

		return m, wCmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "q":
			m.err = fmt.Errorf("quit")
			return m, tea.Quit
		case "ctrl+c", "esc":
			if m.list.FilterState() != list.Filtering {
				m.err = fmt.Errorf("quit")
				return m, tea.Quit
			}
		case "c":
			m.mode = ViewWizard
			wm := NewWizardModel(m.client, m.width, m.height)
			m.wizard = wm
			return m, wm.Init()
		case "s":
			i := m.list.SelectedItem()
			if i != nil {
				itm := i.(serverListItem)
				id := itm.id
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
					go m.client.StartServer(id)
					m.message = fmt.Sprintf("Starting server %s...", id)
				}

				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return "clear_message"
				})
			}
		case "x":
			i := m.list.SelectedItem()
			if i != nil {
				itm := i.(serverListItem)
				id := itm.id
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
					go m.client.StopServer(id)
					m.message = fmt.Sprintf("Stopping server %s...", id)
				}

				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return "clear_message"
				})
			}
		case "d":
			i := m.list.SelectedItem()
			if i != nil {
				itm := i.(serverListItem)
				id := itm.id
				var name string
				for _, s := range m.servers {
					if s.ID == id {
						name = s.Name
						break
					}
				}
				m.message = fmt.Sprintf("Are you sure you want to delete server '%s' (%s)? (y/n)", name, id)
				return m, nil
			}
		case "y":
			if m.message != "" && len(m.message) > 6 && m.message[:26] == "Are you sure you want to d" {
				i := m.list.SelectedItem()
				if i != nil {
					itm := i.(serverListItem)
					id := itm.id
					go m.client.DeleteServer(id)
					m.message = fmt.Sprintf("Deleting server %s...", id)
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
						return "clear_message"
					})
				}
			}
		case "n":
			if m.message != "" && len(m.message) > 6 && m.message[:26] == "Are you sure you want to d" {
				m.message = "Deletion cancelled."
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
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 12)
		if m.mode == ViewWizard {
		}
	case serverDataMsg:
		m.isLoading = false
		m.servers = msg.servers
		m.stats = msg.stats
		m.updateList()
		return m, nil
	case tickMsg:
		return m, tea.Batch(fetchDataCmd(m.client), tickCmd())
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.list, cmd = m.list.Update(msg)
	return m, tea.Batch(varcmd(), cmd)
}

func (m *model) updateList() {
	var items []list.Item
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
		if stat, ok := m.stats[s.ID]; ok {
			cpu = fmt.Sprintf("%.1f%%", stat.CPU)
			ram = fmt.Sprintf("%s / %dMB", formatBytesShort(int64(stat.RAM)), s.RAM)
			disk = formatBytesShort(stat.Disk)
		}

		title := fmt.Sprintf("%s %s (%s)", status, s.Name, s.ID)
		desc := fmt.Sprintf("Port: %d | Ver: %s | CPU: %s | RAM: %s | Disk: %s",
			s.Port, s.Version, cpu, ram, disk)

		items = append(items, serverListItem{
			id:          s.ID,
			title:       title,
			description: desc,
		})
	}
	m.list.SetItems(items)
}

func (m model) View() string {
	if m.mode == ViewWizard {
		return m.wizard.View()
	}

	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Render("NAVIGER")

	var totalCPU float64
	var totalRAM int64
	var totalDisk int64
	for _, stat := range m.stats {
		totalCPU += stat.CPU
		totalRAM += int64(stat.RAM)
		totalDisk += stat.Disk
	}

	hostInfo := fmt.Sprintf("Daemon: %s  |  Servers: %d  |  CPU: %.1f%%  |  RAM: %s  |  Disk: %s",
		m.client.BaseURL(),
		len(m.servers),
		totalCPU,
		formatBytesShort(totalRAM),
		formatBytesShort(totalDisk))
	headerBox := baseStyle.
		Width(m.width-4).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Center, title, " ", hostInfo))

	listContainer := baseStyle.
		Width(m.width - 4).
		Height(m.height - 12).
		Render(m.list.View())

	statusLine := "c: create • s: start • x: stop • d: delete • enter: logs • q/esc: quit"
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
		listContainer,
		footerText,
	)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchDataCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		servers, err := client.ListServers()
		if err != nil {
			return errMsg(err)
		}

		stats, err := client.GetServerStats()
		if err != nil {
			stats = make(map[string]sdk.ServerStats)
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

func (m model) additionalKeyMap() []key.Binding {
	return []key.Binding{}
}
