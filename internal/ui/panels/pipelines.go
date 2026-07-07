package panels

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jos/lazydatabricks/internal/databricks"
)

// ── messages ────────────────────────────────────────────────────────────────

type PipelinesLoadedMsg struct{ Pipelines []databricks.PipelineInfo }
type PipelinesErrMsg struct{ Err error }

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
	client  *databricks.Client
	list    list.Model
	spinner spinner.Model
	loaded  bool
	err     error
	width   int
	height  int
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

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.loaded = false
			m.err = nil
			cmds = append(cmds, tea.Batch(m.spinner.Tick, fetchPipelines(m.client)))
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
	return "↑/↓: navigate  r: refresh"
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
