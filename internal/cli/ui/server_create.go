package ui

import (
	"fmt"
	"naviger/pkg/sdk"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WizardStep int

const (
	StepName WizardStep = iota
	StepLoader
	StepVersion
	StepRAM
	StepConfirm
)

type WizardModel struct {
	client          *sdk.Client
	step            WizardStep
	nameInput       textinput.Model
	ramInput        textinput.Model
	loaderList      list.Model
	versionList     list.Model
	selectedLoader  string
	selectedVersion string
	width           int
	height          int
	err             error
	creating        bool
}

type WizardDoneMsg struct{}
type WizardCancelMsg struct{}

type loadersMsg []string
type versionsMsg []string
type serverCreatedMsg struct{}

func NewWizardModel(client *sdk.Client, width, height int) WizardModel {
	tiName := textinput.New()
	tiName.Placeholder = "My Awesome Server"
	tiName.Focus()
	tiName.CharLimit = 32
	tiName.Width = 30

	tiRam := textinput.New()
	tiRam.Placeholder = "2048"
	tiRam.CharLimit = 6
	tiRam.Width = 10

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	l.Title = "Select Loader"
	l.SetShowHelp(false)

	v := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	v.Title = "Select Version"
	v.SetShowHelp(false)

	return WizardModel{
		client:      client,
		step:        StepName,
		nameInput:   tiName,
		ramInput:    tiRam,
		loaderList:  l,
		versionList: v,
		width:       width,
		height:      height,
	}
}

func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.creating {
		switch msg.(type) {
		case serverCreatedMsg:
			return m, func() tea.Msg { return WizardDoneMsg{} }
		case errMsg:
			m.creating = false
			m.err = msg.(error)
			return m, nil
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.step > StepName {
				m.step--
				return m, nil
			}
			return m, func() tea.Msg { return WizardCancelMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	case loadersMsg:
		var items []list.Item
		for _, l := range msg {
			items = append(items, item(l))
		}
		m.loaderList.SetItems(items)
		m.loaderList.SetSize(m.width-8, m.height-14)
		m.step = StepLoader
		return m, nil
	case versionsMsg:
		var items []list.Item
		for _, v := range msg {
			items = append(items, item(v))
		}
		m.versionList.SetItems(items)
		m.versionList.SetSize(m.width-8, m.height-14)
		m.step = StepVersion
		m.versionList.ResetSelected()
		return m, nil
	case errMsg:
		m.err = msg
		return m, nil
	}

	switch m.step {
	case StepName:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				if m.nameInput.Value() == "" {
					m.err = fmt.Errorf("name cannot be empty")
					return m, nil
				}
				m.err = nil
				return m, fetchLoaders(m.client)
			}
		}
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd

	case StepLoader:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.loaderList.SelectedItem().(item)
				if ok {
					m.selectedLoader = string(i)
					return m, fetchVersions(m.client, m.selectedLoader)
				}
			}
		}
		m.loaderList, cmd = m.loaderList.Update(msg)
		return m, cmd

	case StepVersion:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.versionList.SelectedItem().(item)
				if ok {
					m.selectedVersion = string(i)
					m.step = StepRAM
					m.ramInput.Focus()
					return m, textinput.Blink
				}
			}
		}
		m.versionList, cmd = m.versionList.Update(msg)
		return m, cmd

	case StepRAM:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				val, err := strconv.Atoi(m.ramInput.Value())
				if err != nil || val <= 0 {
					m.err = fmt.Errorf("invalid RAM amount")
					return m, nil
				}
				m.err = nil
				m.step = StepConfirm
				return m, nil
			}
		}
		m.ramInput, cmd = m.ramInput.Update(msg)
		return m, cmd

	case StepConfirm:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "y" || msg.Type == tea.KeyEnter {
				ram, _ := strconv.Atoi(m.ramInput.Value())
				m.creating = true
				return m, createServer(m.client, sdk.CreateServerRequest{
					Name:    m.nameInput.Value(),
					Loader:  m.selectedLoader,
					Version: m.selectedVersion,
					Ram:     ram,
				})
			} else if msg.String() == "n" {
				return m, func() tea.Msg { return WizardCancelMsg{} }
			}
		}
	}

	return m, nil
}

func (m WizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("CREATE NEW SERVER")

	stepTitle := ""
	content := ""

	if m.err != nil {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n\n", m.err))
	}

	switch m.step {
	case StepName:
		stepTitle = "Enter Server Name"
		content += fmt.Sprintf("\n%s", m.nameInput.View())
	case StepLoader:
		stepTitle = "Select Loader"
		content += "\n" + m.loaderList.View()
	case StepVersion:
		stepTitle = fmt.Sprintf("Select Version for %s", m.selectedLoader)
		content += "\n" + m.versionList.View()
	case StepRAM:
		stepTitle = "Enter RAM (MB)"
		content += fmt.Sprintf("\n%s", m.ramInput.View())
	case StepConfirm:
		stepTitle = "Confirm Creation"
		content += fmt.Sprintf("\nName: %s\nLoader: %s\nVersion: %s\nRAM: %s MB\n\n(y/n)",
			m.nameInput.Value(), m.selectedLoader, m.selectedVersion, m.ramInput.Value())
	}

	if m.creating {
		content = fmt.Sprintf("\n\nCreating server '%s'...\nPlease wait.", m.nameInput.Value())
	}

	headerBox := baseStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Padding(1).
		Render(titleStyle.Render(stepTitle))

	mainContainer := baseStyle.
		Width(m.width - 4).
		Height(m.height - 12).
		Align(lipgloss.Center).
		Render(content)

	statusLine := "esc: back/cancel • enter: next"
	footerBox := footerStyle.
		Width(m.width - 4).
		Render(statusLine)

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		headerBox,
		mainContainer,
		footerBox,
	)
}

type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

func fetchLoaders(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		loaders, err := client.ListLoaders()
		if err != nil {
			return errMsg(err)
		}
		return loadersMsg(loaders)
	}
}

func fetchVersions(client *sdk.Client, loader string) tea.Cmd {
	return func() tea.Msg {
		versions, err := client.ListLoaderVersions(loader)
		if err != nil {
			return errMsg(err)
		}
		return versionsMsg(versions)
	}
}

func createServer(client *sdk.Client, req sdk.CreateServerRequest) tea.Cmd {
	return func() tea.Msg {
		err := client.CreateServer(req)
		if err != nil {
			return errMsg(err)
		}
		return serverCreatedMsg{}
	}
}
