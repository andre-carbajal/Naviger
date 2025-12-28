package ui

import "github.com/charmbracelet/lipgloss"

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1).
			Align(lipgloss.Center)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	footerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			Align(lipgloss.Center)
)
