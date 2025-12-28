package ui

import (
	"fmt"
	"naviger/pkg/sdk"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type step int

const (
	stepName step = iota
	stepLoader
	stepVersion
	stepRam
	stepConfirm
)

type createModel struct {
	step      step
	textInput textinput.Model
	list      list.Model
	client    *sdk.Client

	name    string
	version string
	loader  string
	ram     string

	loaders []string

	err      error
	quitting bool
}

type loaderItem string

func (i loaderItem) Title() string       { return string(i) }
func (i loaderItem) Description() string { return "" }
func (i loaderItem) FilterValue() string { return string(i) }

func initialCreateModel(client *sdk.Client) createModel {
	ti := textinput.New()
	ti.Placeholder = "My Awesome Server"
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 30

	return createModel{
		step:      stepName,
		textInput: ti,
		client:    client,
	}
}

func (m createModel) Init() tea.Cmd {
	return textinput.Blink
}

type loadersMsg []string
type versionsMsg []string
type errMsg3 error

func fetchLoaders(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		loaders, err := client.ListLoaders()
		if err != nil {
			return errMsg3(err)
		}
		return loadersMsg(loaders)
	}
}

func fetchVersions(client *sdk.Client, loader string) tea.Cmd {
	return func() tea.Msg {
		versions, err := client.ListLoaderVersions(loader)
		if err != nil {
			return errMsg3(err)
		}
		return versionsMsg(versions)
	}
}

func (m createModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}

		if msg.Type == tea.KeyEnter {
			switch m.step {
			case stepName:
				m.name = m.textInput.Value()
				if m.name == "" {
					return m, nil
				}

				return m, tea.Batch(
					func() tea.Msg { return "loading_loaders" },
					fetchLoaders(m.client),
				)

			case stepLoader:
				if m.list.FilterState() == list.Filtering {
					break
				}
				i, ok := m.list.SelectedItem().(loaderItem)
				if ok {
					m.loader = string(i)
					return m, tea.Batch(
						func() tea.Msg { return "loading_versions" },
						fetchVersions(m.client, m.loader),
					)
				}

			case stepVersion:
				if m.list.FilterState() == list.Filtering {
					break
				}
				i, ok := m.list.SelectedItem().(loaderItem)
				if ok {
					m.version = string(i)
					m.step = stepRam
					m.textInput.Placeholder = "2048"
					m.textInput.SetValue("")
					m.textInput.Focus()
					return m, textinput.Blink
				}

			case stepRam:
				m.ram = m.textInput.Value()
				if _, err := strconv.Atoi(m.ram); err != nil {
					return m, nil
				}
				m.step = stepConfirm
				return m, nil

			case stepConfirm:
				return m, tea.Quit
			}
		}

	case loadersMsg:
		m.loaders = msg
		var items []list.Item
		for _, l := range m.loaders {
			items = append(items, loaderItem(l))
		}
		m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
		m.list.Title = "Select Loader"
		m.list.SetShowHelp(false)
		m.list.SetShowStatusBar(false)
		m.list.SetFilteringEnabled(false)

		h, v := docStyle.GetFrameSize()
		width := 80
		height := 20
		m.list.SetSize(width-h, height-v)

		m.step = stepLoader
		return m, nil

	case versionsMsg:
		var items []list.Item
		for _, v := range msg {
			items = append(items, loaderItem(v))
		}
		m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
		m.list.Title = fmt.Sprintf("Select Version for %s", m.loader)
		m.list.SetShowHelp(false)
		m.list.SetShowStatusBar(false)
		m.list.SetFilteringEnabled(true)

		h, v := docStyle.GetFrameSize()
		width := 80
		height := 20
		m.list.SetSize(width-h, height-v)

		m.step = stepVersion
		return m, nil

	case errMsg3:
		m.err = msg
		return m, tea.Quit

	case tea.WindowSizeMsg:
		if m.step == stepLoader || m.step == stepVersion {
			h, v := docStyle.GetFrameSize()
			m.list.SetSize(msg.Width-h, msg.Height-v)
		}
	}

	if m.step == stepLoader || m.step == stepVersion {
		m.list, cmd = m.list.Update(msg)
	} else if m.step != stepConfirm {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

func (m createModel) View() string {
	if m.quitting {
		return ""
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var s string

	switch m.step {
	case stepName:
		s = fmt.Sprintf(
			"What should we call your server?\n\n%s\n\n%s",
			m.textInput.View(),
			"(esc to quit)",
		)
	case stepLoader:
		if len(m.list.Items()) == 0 {
			return "Loading loaders..."
		}
		return docStyle.Render(m.list.View())
	case stepVersion:
		if len(m.list.Items()) == 0 {
			return "Loading versions..."
		}
		return docStyle.Render(m.list.View())
	case stepRam:
		s = fmt.Sprintf(
			"How much RAM (in MB)?\n\n%s\n\n%s",
			m.textInput.View(),
			"(esc to quit)",
		)
	case stepConfirm:
		s = fmt.Sprintf(
			"Ready to create server:\n\nName: %s\nLoader: %s\nVersion: %s\nRAM: %s MB\n\nPress Enter to confirm or Ctrl+C to cancel.",
			m.name, m.loader, m.version, m.ram,
		)
	}

	return docStyle.Render(s)
}

func RunCreateWizard(client *sdk.Client) (*sdk.CreateServerRequest, bool) {
	m := initialCreateModel(client)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, false
	}

	if m, ok := finalModel.(createModel); ok && !m.quitting && m.step == stepConfirm {
		ram, _ := strconv.Atoi(m.ram)
		return &sdk.CreateServerRequest{
			Name:    m.name,
			Version: m.version,
			Loader:  m.loader,
			Ram:     ram,
		}, true
	}

	return nil, false
}
