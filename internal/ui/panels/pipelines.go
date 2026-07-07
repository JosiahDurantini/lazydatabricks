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

type PipelinesLoadedMsg struct{ Pipelines []databricks.PipelineInfo }
type PipelinesErrMsg struct{ Err error }
type PipelineStopDoneMsg struct{ Err error }

// ── list.Item adapter ───────────────────────────────────────────────────────

type pipelineItem struct{ pipeline databricks.PipelineInfo }

func (p pipelineItem) FilterValue() string { return p.pipeline.Name }
func (p pipelineItem) Title() string       { return p.pipeline.Name }
func (p pipelineItem) Description() string {
	desc := PipelineStateLabel(p.pipeline.State)
	if p.pipeline.LatestUpdate != "" {
		desc += "  •  last update " + p.pipeline.LatestUpdate
	}
	return desc
}

// ── model ────────────────────────────────────────────────────────────────────

type PipelinesModel struct {
	client     *databricks.Client
	list       list.Model
	spinner    spinner.Model
	loaded     bool
	err        error
	actionErr  error
	confirming string // pipeline ID pending stop confirmation, "" if none
	width      int
	height     int
}

// stoppablePipelineStates are states with an update in flight worth stopping.
var stoppablePipelineStates = map[string]bool{
	"RUNNING":    true,
	"STARTING":   true,
	"DEPLOYING":  true,
	"RESETTING":  true,
	"RECOVERING": true,
}

func NewPipelines(client *databricks.Client) PipelinesModel {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Pipelines"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return PipelinesModel{client: client, list: l, spinner: s}
}

func (m PipelinesModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchPipelines(m.client))
}

func (m PipelinesModel) Update(msg tea.Msg) (PipelinesModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PipelinesLoadedMsg:
		m.loaded = true
		items := make([]list.Item, len(msg.Pipelines))
		for i, p := range msg.Pipelines {
			items[i] = pipelineItem{p}
		}
		cmds = append(cmds, m.list.SetItems(items))

	case PipelinesErrMsg:
		m.loaded = true
		m.err = msg.Err

	case PipelineStopDoneMsg:
		m.actionErr = msg.Err
		// Refresh so the STOPPING/IDLE state shows up.
		cmds = append(cmds, fetchPipelines(m.client))

	case tea.KeyMsg:
		// A stop confirmation is pending: y proceeds, anything else aborts.
		if m.confirming != "" {
			id := m.confirming
			m.confirming = ""
			if msg.String() == "y" {
				return m, stopPipeline(m.client, id)
			}
			return m, nil
		}

		switch msg.String() {
		case "r":
			m.loaded = false
			m.err = nil
			m.actionErr = nil
			cmds = append(cmds, tea.Batch(m.spinner.Tick, fetchPipelines(m.client)))
		case "x":
			if sel, ok := m.list.SelectedItem().(pipelineItem); ok && stoppablePipelineStates[sel.pipeline.State] {
				m.actionErr = nil
				m.confirming = sel.pipeline.PipelineID
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

func (m *PipelinesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

// HelpText returns context-sensitive help for the panel.
func (m PipelinesModel) HelpText() string {
	return "↑/↓: navigate  x: stop update  r: refresh"
}

func (m PipelinesModel) ViewList() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	if !m.loaded {
		return m.spinner.View() + " Loading pipelines…"
	}
	if len(m.list.Items()) == 0 {
		return faintStyle.Render("No pipelines in this workspace.")
	}
	return m.list.View()
}

func (m PipelinesModel) ViewDetail() string {
	if !m.loaded || m.err != nil {
		return ""
	}

	selected, ok := m.list.SelectedItem().(pipelineItem)
	if !ok {
		return faintStyle.Render("No pipeline selected")
	}

	p := selected.pipeline
	var b strings.Builder

	row := func(label, value string) {
		fmt.Fprintf(&b, "%s  %s\n", detailLabel.Render(label), detailValue.Render(value))
	}

	row("Name   ", p.Name)
	row("ID     ", p.PipelineID)
	fmt.Fprintf(&b, "%s  %s\n", detailLabel.Render("State  "), PipelineStateLabel(p.State))
	if p.Health != "" {
		row("Health ", p.Health)
	}
	if p.LatestUpdate != "" {
		update := p.LatestUpdate
		if !p.LatestUpdateTime.IsZero() {
			update += fmt.Sprintf("  (%s ago)", time.Since(p.LatestUpdateTime).Round(time.Minute))
		}
		row("Update ", update)
	}
	if p.ClusterID != "" {
		row("Cluster", p.ClusterID)
	}
	row("Creator", p.CreatorUserName)

	if stoppablePipelineStates[p.State] {
		b.WriteString("\n" + faintStyle.Render("x: stop the active update") + "\n")
	}
	if m.confirming != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).
			Render(fmt.Sprintf("Stop %s? press y to confirm, any other key to abort", p.Name)) + "\n")
	}
	if m.actionErr != nil {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.actionErr.Error()) + "\n")
	}

	return b.String()
}

// ── command ──────────────────────────────────────────────────────────────────

func fetchPipelines(client *databricks.Client) tea.Cmd {
	return func() tea.Msg {
		pipelines, err := client.ListPipelines(context.Background())
		if err != nil {
			return PipelinesErrMsg{err}
		}
		return PipelinesLoadedMsg{pipelines}
	}
}

func stopPipeline(client *databricks.Client, pipelineID string) tea.Cmd {
	return func() tea.Msg {
		return PipelineStopDoneMsg{Err: client.StopPipeline(context.Background(), pipelineID)}
	}
}
