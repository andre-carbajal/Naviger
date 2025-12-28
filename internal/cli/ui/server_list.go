package ui

import (
	"fmt"
	"naviger/pkg/sdk"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle    = lipgloss.NewStyle().Margin(1, 2)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"})
)

type item struct {
	server sdk.Server
}

func (i item) Title() string { return i.server.Name }
func (i item) Description() string {
	statusIcon := "🔴"
	if i.server.Status == "RUNNING" {
		statusIcon = "🟢"
	} else if i.server.Status == "STARTING" {
		statusIcon = "🟡"
	} else if i.server.Status == "STOPPING" {
		statusIcon = "🟠"
	}
	return fmt.Sprintf("%s %s | ID: %s | Port: %d | %s", statusIcon, i.server.Status, i.server.ID, i.server.Port, i.server.Version)
}
func (i item) FilterValue() string { return i.server.Name + " " + i.server.Status }

type listKeyMap struct {
	start   key.Binding
	stop    key.Binding
	refresh key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

type listModel struct {
	list   list.Model
	client *sdk.Client
	keys   *listKeyMap
	choice *sdk.Server
}

func (m listModel) Init() tea.Cmd {
	return nil
}

type statusMsg string
type serverListMsg []sdk.Server

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, m.keys.start):
			i, ok := m.list.SelectedItem().(item)
			if ok {
				return m, tea.Batch(
					func() tea.Msg {
						if err := m.client.StartServer(i.server.ID); err != nil {
							return statusMsg(fmt.Sprintf("Error starting %s: %v", i.server.Name, err))
						}
						return statusMsg(fmt.Sprintf("Start command sent to %s", i.server.Name))
					},
					m.list.NewStatusMessage(statusStyle.Render(fmt.Sprintf("Starting %s...", i.server.Name))),
				)
			}
		case key.Matches(msg, m.keys.stop):
			i, ok := m.list.SelectedItem().(item)
			if ok {
				return m, tea.Batch(
					func() tea.Msg {
						if err := m.client.StopServer(i.server.ID); err != nil {
							return statusMsg(fmt.Sprintf("Error stopping %s: %v", i.server.Name, err))
						}
						return statusMsg(fmt.Sprintf("Stop command sent to %s", i.server.Name))
					},
					m.list.NewStatusMessage(statusStyle.Render(fmt.Sprintf("Stopping %s...", i.server.Name))),
				)
			}
		case key.Matches(msg, m.keys.refresh):
			return m, refreshList(m.client)
		case msg.String() == "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = &i.server
				return m, tea.Quit
			}
		}
	case statusMsg:
		cmd := m.list.NewStatusMessage(statusStyle.Render(string(msg)))
		return m, tea.Batch(cmd, refreshList(m.client))
	case serverListMsg:
		var items []list.Item
		for _, s := range msg {
			items = append(items, item{server: s})
		}
		cmd := m.list.SetItems(items)
		return m, cmd
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	return docStyle.Render(m.list.View())
}

func refreshList(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		servers, err := client.ListServers()
		if err != nil {
			return nil
		}
		return serverListMsg(servers)
	}
}

func RunServerList(client *sdk.Client) {
	servers, err := client.ListServers()
	if err != nil {
		fmt.Printf("Error listing servers: %v\n", err)
		return
	}

	var items []list.Item
	for _, s := range servers {
		items = append(items, item{server: s})
	}

	keys := newListKeyMap()
	delegate := list.NewDefaultDelegate()

	l := list.New(items, delegate, 0, 0)
	l.Title = "Servers"
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.start, keys.stop, keys.refresh}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.start, keys.stop, keys.refresh}
	}

	m := listModel{
		list:   l,
		client: client,
		keys:   keys,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running list: %v\n", err)
		return
	}

	if m, ok := finalModel.(listModel); ok && m.choice != nil {
		fmt.Println("\nSelected Server Details:")
		fmt.Printf("Name:    %s\n", m.choice.Name)
		fmt.Printf("ID:      %s\n", m.choice.ID)
		fmt.Printf("Status:  %s\n", m.choice.Status)
		fmt.Printf("Port:    %d\n", m.choice.Port)
		fmt.Printf("Version: %s\n", m.choice.Version)
		fmt.Printf("Loader:  %s\n", m.choice.Loader)
		fmt.Printf("RAM:     %d MB\n", m.choice.RAM)
	}
}
