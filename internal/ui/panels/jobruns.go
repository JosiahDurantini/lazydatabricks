package panels

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JosiahDurantini/lazydatabricks/internal/databricks"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── messages ────────────────────────────────────────────────────────────────

type JobRunsLoadedMsg struct{ Runs []databricks.JobRun }
type JobRunsErrMsg struct{ Err error }
type RunCancelDoneMsg struct{ Err error }

// ── list.Item adapter ───────────────────────────────────────────────────────

type runItem struct{ run databricks.JobRun }

func (r runItem) FilterValue() string { return r.run.RunName }
func (r runItem) Title() string       { return r.run.RunName }
func (r runItem) Description() string {
	ago := time.Since(r.run.StartTime).Round(time.Second)
	return fmt.Sprintf("%s  •  started %s ago", statusLabel(r.run.Status), ago)
}

// ── styles ───────────────────────────────────────────────────────────────────

var statusColors = map[string]lipgloss.Color{
	"RUNNING":    lipgloss.Color("2"),
	"SUCCESS":    lipgloss.Color("6"),
	"FAILED":     lipgloss.Color("9"),
	"TERMINATED": lipgloss.Color("8"),
	"PENDING":    lipgloss.Color("11"),
	"SKIPPED":    lipgloss.Color("8"),
	"CANCELED":   lipgloss.Color("8"),
	"TIMEDOUT":   lipgloss.Color("3"),
}

func statusLabel(s string) string {
	col, ok := statusColors[s]
	if !ok {
		col = lipgloss.Color("7")
	}
	return lipgloss.NewStyle().Foreground(col).Render(s)
}

// ── model ────────────────────────────────────────────────────────────────────

type JobRunsModel struct {
	client     *databricks.Client
	list       list.Model
	spinner    spinner.Model
	loaded     bool
	err        error
	actionErr  error
	confirming int64 // run ID pending cancel confirmation, 0 if none
	width      int
	height     int
}

// activeRunStates are lifecycle states a run can still be canceled from.
var activeRunStates = map[string]bool{
	"PENDING":     true,
	"RUNNING":     true,
	"QUEUED":      true,
	"BLOCKED":     true,
	"TERMINATING": true,
}

func NewJobRuns(client *databricks.Client) JobRunsModel {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Job Runs"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return JobRunsModel{client: client, list: l, spinner: s}
}

func (m JobRunsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchJobRuns(m.client))
}

func (m JobRunsModel) Update(msg tea.Msg) (JobRunsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case JobRunsLoadedMsg:
		m.loaded = true
		items := make([]list.Item, len(msg.Runs))
		for i, r := range msg.Runs {
			items[i] = runItem{r}
		}
		cmds = append(cmds, m.list.SetItems(items))

	case JobRunsErrMsg:
		m.loaded = true
		m.err = msg.Err

	case RunCancelDoneMsg:
		m.actionErr = msg.Err
		// Refresh so the CANCELED/TERMINATING state shows up.
		cmds = append(cmds, fetchJobRuns(m.client))

	case tea.KeyMsg:
		// A cancel confirmation is pending: y proceeds, anything else aborts.
		if m.confirming != 0 {
			id := m.confirming
			m.confirming = 0
			if msg.String() == "y" {
				return m, cancelRun(m.client, id)
			}
			return m, nil
		}

		switch msg.String() {
		case "r":
			m.loaded = false
			m.err = nil
			m.actionErr = nil
			cmds = append(cmds, tea.Batch(m.spinner.Tick, fetchJobRuns(m.client)))
		case "x":
			if sel, ok := m.list.SelectedItem().(runItem); ok && activeRunStates[sel.run.Status] {
				m.actionErr = nil
				m.confirming = sel.run.RunID
				return m, nil
			}
		}
	}

	if !m.loaded {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// HelpText returns context-sensitive help for the panel.
func (m JobRunsModel) HelpText() string {
	return "↑/↓: navigate  x: cancel run  r: refresh"
}

// Runs exposes the fetched runs so the app can use them for cross-panel filtering.
func (m JobRunsModel) Runs() []databricks.JobRun {
	items := m.list.Items()
	runs := make([]databricks.JobRun, 0, len(items))
	for _, item := range items {
		if ri, ok := item.(runItem); ok {
			runs = append(runs, ri.run)
		}
	}
	return runs
}

func (m *JobRunsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

// ViewList renders the left-hand list panel.
func (m JobRunsModel) ViewList() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	if !m.loaded {
		return m.spinner.View() + " Loading job runs…"
	}
	return m.list.View()
}

// ViewDetail renders the right-hand detail panel for the selected run.
func (m JobRunsModel) ViewDetail() string {
	if !m.loaded || m.err != nil {
		return ""
	}

	selected, ok := m.list.SelectedItem().(runItem)
	if !ok {
		return lipgloss.NewStyle().Faint(true).Render("No run selected")
	}

	r := selected.run
	var b strings.Builder

	row := func(label, value string) {
		fmt.Fprintf(&b, "%s  %s\n", detailLabel.Render(label), detailValue.Render(value))
	}

	row("Run name", r.RunName)
	row("Run ID  ", fmt.Sprintf("%d", r.RunID))
	row("Job ID  ", fmt.Sprintf("%d", r.JobID))
	row("Status  ", statusLabel(r.Status))
	row("Started ", r.StartTime.Format("2006-01-02 15:04:05"))
	row("Duration", r.Duration.Round(time.Second).String())

	if activeRunStates[r.Status] {
		b.WriteString("\n" + faintStyle.Render("x: cancel this run") + "\n")
	}
	if m.confirming != 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).
			Render(fmt.Sprintf("Cancel run %d? press y to confirm, any other key to abort", m.confirming)) + "\n")
	}
	if m.actionErr != nil {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.actionErr.Error()) + "\n")
	}

	return b.String()
}

// ── command ──────────────────────────────────────────────────────────────────

func fetchJobRuns(client *databricks.Client) tea.Cmd {
	return func() tea.Msg {
		runs, err := client.ListJobRuns(context.Background())
		if err != nil {
			return JobRunsErrMsg{err}
		}
		return JobRunsLoadedMsg{runs}
	}
}

func cancelRun(client *databricks.Client, runID int64) tea.Cmd {
	return func() tea.Msg {
		return RunCancelDoneMsg{Err: client.CancelRun(context.Background(), runID)}
	}
}
