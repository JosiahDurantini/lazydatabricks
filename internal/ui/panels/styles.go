package panels

import "github.com/charmbracelet/lipgloss"

var (
	detailLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	detailValue = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	faintStyle  = lipgloss.NewStyle().Faint(true)
)
