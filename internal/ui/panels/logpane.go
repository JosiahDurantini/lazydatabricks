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
	width       int
	height      int // total inner height available to the pane
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
	l.recalc()
}

func (l *LogPane) SetTaskSummary(s string) {
	l.taskSummary = s
	// The summary sits below the viewport, so its height changes the
	// viewport's share of the pane.
	l.recalc()
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
	l.width = w
	l.height = h
	l.recalc()
}

// recalc splits the pane's height between the viewport and the fixed parts
// rendered around it (1 header line; divider + task summary when present).
func (l *LogPane) recalc() {
	if l.width == 0 || l.height == 0 {
		return // not sized yet
	}
	l.viewport.Width = l.width
	reserved := 1 // header
	if l.taskSummary != "" {
		reserved += strings.Count(l.taskSummary, "\n") + 2 // summary lines + divider
	}
	vh := l.height - reserved
	if vh < 1 {
		vh = 1
	}
	atBottom := l.viewport.AtBottom()
	l.viewport.Height = vh
	l.viewport.SetContent(strings.Join(l.lines, "\n"))
	if atBottom {
		l.viewport.GotoBottom()
	}
}

func (l LogPane) IsVisible() bool { return l.visible }

func (l *LogPane) Toggle() { l.visible = !l.visible }

func (l LogPane) Update(msg tea.Msg) (LogPane, tea.Cmd) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

func (l LogPane) View() string {
	if l.label == "" && len(l.lines) == 0 {
		return faintStyle.Render("No command output yet — run a bundle action (v / d / r).")
	}

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
