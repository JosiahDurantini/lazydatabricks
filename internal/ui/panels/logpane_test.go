package panels

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLogPaneViewFitsHeight(t *testing.T) {
	const width, height = 80, 10

	var l LogPane
	l.SetSize(width, height)
	l.Start("databricks bundle run job")
	for i := 0; i < 30; i++ {
		l.AppendLine(strings.Repeat("x", 20))
	}

	if got := lipgloss.Height(l.View()); got > height {
		t.Errorf("View height without summary = %d, want <= %d", got, height)
	}

	// The task summary sits below the viewport with a divider; the pane must
	// still fit its height budget.
	l.SetTaskSummary("task-a  RUNNING\ntask-b  PENDING\ntask-c  PENDING")
	if got := lipgloss.Height(l.View()); got > height {
		t.Errorf("View height with summary = %d, want <= %d", got, height)
	}
}

func TestLogPaneScrollableAfterResize(t *testing.T) {
	var l LogPane
	l.SetSize(40, 8)
	l.Start("test")
	for i := 0; i < 50; i++ {
		l.AppendLine("line")
	}
	if !l.viewport.AtBottom() {
		t.Error("viewport should follow appended output to the bottom")
	}
	if l.viewport.Height <= 0 {
		t.Errorf("viewport height = %d, want > 0", l.viewport.Height)
	}
}
