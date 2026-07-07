package panels

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jos/lazydatabricks/internal/databricks"
)

// ── messages ────────────────────────────────────────────────────────────────

type ClustersLoadedMsg struct{ Clusters []databricks.ClusterInfo }
type ClustersErrMsg struct{ Err error }
type ClusterActionDoneMsg struct {
	Action string
	Err    error
}

// ── list.Item adapter ───────────────────────────────────────────────────────

type clusterItem struct{ cluster databricks.ClusterInfo }

func (c clusterItem) FilterValue() string { return c.cluster.ClusterName }
func (c clusterItem) Title() string       { return c.cluster.ClusterName }
func (c clusterItem) Description() string {
	return fmt.Sprintf("%s  •  %s", ClusterStateLabel(c.cluster.State), c.cluster.NodeTypeID)
}

// ── model ────────────────────────────────────────────────────────────────────

type ClustersModel struct {
	client     *databricks.Client
	list       list.Model
	spinner    spinner.Model
	loaded     bool
	err        error
	actionErr  error
	confirming string // cluster ID pending terminate confirmation, "" if none
	width      int
	height     int
}

func NewClusters(client *databricks.Client) ClustersModel {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Clusters"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return ClustersModel{client: client, list: l, spinner: s}
}

func (m ClustersModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchClusters(m.client))
}

func (m ClustersModel) Update(msg tea.Msg) (ClustersModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ClustersLoadedMsg:
		m.loaded = true
		items := make([]list.Item, len(msg.Clusters))
		for i, c := range msg.Clusters {
			items[i] = clusterItem{c}
		}
		cmds = append(cmds, m.list.SetItems(items))

	case ClustersErrMsg:
		m.loaded = true
		m.err = msg.Err

	case ClusterActionDoneMsg:
		m.actionErr = msg.Err
		// Refresh so the new state (PENDING/TERMINATING) shows up.
		cmds = append(cmds, fetchClusters(m.client))

	case tea.KeyMsg:
		sel, hasSel := m.list.SelectedItem().(clusterItem)

		// A terminate confirmation is pending: y proceeds, anything else cancels.
		if m.confirming != "" {
			id := m.confirming
			m.confirming = ""
			if msg.String() == "y" {
				return m, clusterAction(m.client, "terminate", id)
			}
			return m, nil
		}

		switch msg.String() {
		case "r":
			m.loaded = false
			m.err = nil
			m.actionErr = nil
			cmds = append(cmds, tea.Batch(m.spinner.Tick, fetchClusters(m.client)))
		case "s":
			if hasSel {
				m.actionErr = nil
				return m, clusterAction(m.client, "start", sel.cluster.ClusterID)
			}
		case "x":
			if hasSel {
				m.actionErr = nil
				m.confirming = sel.cluster.ClusterID
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

func (m *ClustersModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

// HelpText returns context-sensitive help for the panel.
func (m ClustersModel) HelpText() string {
	return "↑/↓: navigate  s: start  x: terminate  r: refresh"
}

func (m ClustersModel) ViewList() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	if !m.loaded {
		return m.spinner.View() + " Loading clusters…"
	}
	return m.list.View()
}

func (m ClustersModel) ViewDetail() string {
	if !m.loaded || m.err != nil {
		return ""
	}

	selected, ok := m.list.SelectedItem().(clusterItem)
	if !ok {
		return faintStyle.Render("No cluster selected")
	}

	c := selected.cluster
	var b strings.Builder

	row := func(label, value string) {
		fmt.Fprintf(&b, "%s  %s\n", detailLabel.Render(label), detailValue.Render(value))
	}

	row("Name    ", c.ClusterName)
	row("ID      ", c.ClusterID)
	fmt.Fprintf(&b, "%s  %s\n", detailLabel.Render("State   "), ClusterStateLabel(c.State))
	if c.StateMessage != "" {
		b.WriteString("  " + faintStyle.Render(c.StateMessage) + "\n")
	}
	row("Node    ", c.NodeTypeID)
	row("Workers ", fmt.Sprintf("%d", c.NumWorkers))
	row("Spark   ", c.SparkVersion)
	if c.AutoterminationMinutes > 0 {
		row("Autoterm", fmt.Sprintf("%d min", c.AutoterminationMinutes))
	}
	row("Creator ", c.CreatorUserName)

	if m.confirming != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).
			Render(fmt.Sprintf("Terminate %s? press y to confirm, any other key to cancel", c.ClusterName)) + "\n")
	}
	if m.actionErr != nil {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.actionErr.Error()) + "\n")
	}

	return b.String()
}

// ── commands ─────────────────────────────────────────────────────────────────

func fetchClusters(client *databricks.Client) tea.Cmd {
	return func() tea.Msg {
		clusters, err := client.ListClusters(context.Background())
		if err != nil {
			return ClustersErrMsg{err}
		}
		return ClustersLoadedMsg{clusters}
	}
}

func clusterAction(client *databricks.Client, action, clusterID string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "start":
			err = client.StartCluster(context.Background(), clusterID)
		case "terminate":
			err = client.StopCluster(context.Background(), clusterID)
		}
		return ClusterActionDoneMsg{Action: action, Err: err}
	}
}
