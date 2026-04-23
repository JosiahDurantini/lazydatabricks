package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	logLabelStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Padding(0, 1)
	logRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	logDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	logErrStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

type LogPane struct {
	viewport    viewport.Model
	label       string
	lines       []string
	taskSummary string
	running     bool
	err         error
	visible     bool
}

func NewLogPane() LogPane {
	return LogPane{}
}

func (l *LogPane) Start(label string) {
	l.label = label
	l.lines = nil
	l.taskSummary = ""
	l.running = true
	l.err = nil
	l.visible = true
	l.viewport.SetContent("")
}

func (l *LogPane) SetTaskSummary(s string) {
	l.taskSummary = s
}

func (l *LogPane) AppendLine(line string) {
	l.lines = append(l.lines, line)
	l.viewport.SetContent(strings.Join(l.lines, "\n"))
	l.viewport.GotoBottom()
}

func (l *LogPane) Finish(err error) {
	l.running = false
	l.err = err
}

func (l *LogPane) SetSize(w, h int) {
	l.viewport.Width = w
	taskLines := strings.Count(l.taskSummary, "\n") + 1
	if l.taskSummary == "" {
		taskLines = 0
	}
	// Reserve 1 line for header + space for task summary
	reserved := 1 + taskLines
	if h > reserved {
		l.viewport.Height = h - reserved
	}
	l.viewport.SetContent(strings.Join(l.lines, "\n"))
}

func (l LogPane) IsVisible() bool { return l.visible }

func (l *LogPane) Toggle() { l.visible = !l.visible }

func (l LogPane) Update(msg tea.Msg) (LogPane, tea.Cmd) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

func (l LogPane) View() string {
	var status string
	if l.running {
		status = logRunningStyle.Render("● running")
	} else if l.err != nil {
		status = logErrStyle.Render("✖ " + l.err.Error())
	} else {
		status = logDoneStyle.Render("✔ done")
	}

	header := logLabelStyle.Render(l.label) + "  " + status
	parts := []string{header, l.viewport.View()}
	if l.taskSummary != "" {
		divider := lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", 40))
		parts = append(parts, divider, l.taskSummary)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
