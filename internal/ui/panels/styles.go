package panels

import "github.com/charmbracelet/lipgloss"

var (
	detailLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	detailValue = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	faintStyle  = lipgloss.NewStyle().Faint(true)
)

var clusterStateColors = map[string]lipgloss.Color{
	"RUNNING":     lipgloss.Color("2"),
	"PENDING":     lipgloss.Color("11"),
	"RESTARTING":  lipgloss.Color("11"),
	"RESIZING":    lipgloss.Color("11"),
	"TERMINATING": lipgloss.Color("3"),
	"TERMINATED":  lipgloss.Color("8"),
	"ERROR":       lipgloss.Color("9"),
	"UNKNOWN":     lipgloss.Color("8"),
}

// ClusterStateLabel renders a cluster state with its conventional color.
func ClusterStateLabel(state string) string {
	col, ok := clusterStateColors[state]
	if !ok {
		col = lipgloss.Color("7")
	}
	return lipgloss.NewStyle().Foreground(col).Render(state)
}
