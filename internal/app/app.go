package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jos/lazydatabricks/internal/bundle"
	"github.com/jos/lazydatabricks/internal/databricks"
	"github.com/jos/lazydatabricks/internal/ui/panels"
)

type runDetailMsg struct{ detail databricks.RunDetail }
type clusterInfoMsg struct {
	clusterID string
	info      databricks.ClusterInfo
}

type focusedPanel int

const (
	panelJobRuns focusedPanel = iota
	panelBundles
)

const totalPanels = 2

var (
	paneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))

	paneBorderFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("12"))

	detailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)

	logPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(0, 1)

	helpStyle        = lipgloss.NewStyle().Faint(true)
	detailValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	tabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			Padding(0, 1)

	tabInactive = lipgloss.NewStyle().
			Faint(true).
			Padding(0, 1)
)

type Model struct {
	client        *databricks.Client
	jobRuns       panels.JobRunsModel
	bundles       panels.BundlesModel
	logPane       panels.LogPane
	logFocused    bool
	logCh         <-chan string
	activeRunID   int64
	lastRunDetail databricks.RunDetail
	clusterCache  map[string]databricks.ClusterInfo
	focused       focusedPanel
	width         int
	height        int
}

func New() (Model, error) {
	client, err := databricks.NewClient()
	if err != nil {
		return Model{}, err
	}
	return Model{
		client:  client,
		jobRuns: panels.NewJobRuns(client),
		bundles: panels.NewBundles(),
		logPane: panels.NewLogPane(),
	}, nil
}

func (m Model) Init() tea.Cmd {
	return m.jobRuns.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSizes()
		return m, nil

	case tea.KeyMsg:
		// When log pane is focused it gets all keys; esc/ctrl+l exit focus.
		if m.logFocused {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "ctrl+l":
				m.logFocused = false
				return m, nil
			}
			var cmd tea.Cmd
			m.logPane, cmd = m.logPane.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focused = focusedPanel((int(m.focused) + 1) % totalPanels)
			return m, nil
		case "shift+tab":
			m.focused = focusedPanel((int(m.focused) + totalPanels - 1) % totalPanels)
			return m, nil
		case "ctrl+l":
			if m.logPane.IsVisible() {
				// Already visible — focus it for scrolling.
				m.logFocused = true
			} else {
				m.logPane.Toggle()
				m.syncSizes()
			}
			return m, nil
		}

	// Bundle panel wants to run a CLI command — start streaming it.
	case panels.BundleCommandMsg:
		m.logPane.Start(fmt.Sprintf("databricks %v", msg.Args))
		m.syncSizes()
		cmds = append(cmds, bundle.StartCommand(msg.Dir, msg.Args...))

	// Process has started and handed us the line channel.
	case bundle.CommandStartMsg:
		m.logCh = msg.Lines
		cmds = append(cmds, bundle.WaitForLine(m.logCh))

	// One more line from the stream — append, scan for run ID, request next.
	case bundle.LineMsg:
		m.logPane.AppendLine(string(msg))
		if m.activeRunID == 0 {
			if id := bundle.ExtractRunID(string(msg)); id > 0 {
				m.activeRunID = id
				cmds = append(cmds, pollRunDetail(m.client, id))
			}
		}
		cmds = append(cmds, bundle.WaitForLine(m.logCh))

	// Stream closed — command is done.
	case bundle.CommandDoneMsg:
		m.logPane.Finish(msg.Err)
		m.logCh = nil

	// SDK poll returned task status.
	case runDetailMsg:
		m.lastRunDetail = msg.detail
		m.logPane.SetTaskSummary(formatTaskSummary(msg.detail, m.clusterCache))
		// Fetch cluster info for any cluster IDs we haven't seen yet.
		for _, t := range msg.detail.Tasks {
			if t.ClusterID != "" {
				if _, ok := m.clusterCache[t.ClusterID]; !ok {
					cmds = append(cmds, fetchClusterInfo(m.client, t.ClusterID))
				}
			}
		}
		if !msg.detail.Done {
			cmds = append(cmds, pollRunDetail(m.client, m.activeRunID))
		} else {
			m.activeRunID = 0
		}

	// Cluster info arrived — update cache and refresh task summary.
	case clusterInfoMsg:
		if m.clusterCache == nil {
			m.clusterCache = make(map[string]databricks.ClusterInfo)
		}
		m.clusterCache[msg.clusterID] = msg.info
		m.logPane.SetTaskSummary(formatTaskSummary(m.lastRunDetail, m.clusterCache))

	// Bundle panel wants recent runs for a job key — filter from in-memory data.
	case panels.FetchRunsForKeyMsg:
		runs := filterRunsByKey(m.jobRuns.Runs(), msg.Key)
		var cmd tea.Cmd
		m.bundles, cmd = m.bundles.Update(panels.RunsForKeyMsg{Key: msg.Key, Runs: runs})
		cmds = append(cmds, cmd)
	}

	// Route to focused panel.
	switch m.focused {
	case panelJobRuns:
		var cmd tea.Cmd
		m.jobRuns, cmd = m.jobRuns.Update(msg)
		cmds = append(cmds, cmd)
	case panelBundles:
		var cmd tea.Cmd
		m.bundles, cmd = m.bundles.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Log pane always handles scroll events.
	var cmd tea.Cmd
	m.logPane, cmd = m.logPane.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	listW, detailW, mainH := m.mainDimensions()

	var listContent, detailContent string
	switch m.focused {
	case panelJobRuns:
		listContent = m.jobRuns.ViewList()
		detailContent = m.jobRuns.ViewDetail()
	case panelBundles:
		listContent = m.bundles.ViewList()
		detailContent = m.bundles.ViewDetail()
	}

	listPane := paneBorderFocused.Width(listW).Height(mainH).Render(listContent)
	detail := detailPaneStyle.Width(detailW).Height(mainH).Render(detailContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, listPane, detail)

	rows := []string{m.tabBar(), body}

	if m.logPane.IsVisible() {
		logInnerH := m.logHeight() - 2
		logW := m.width - 4
		m.logPane.SetSize(logW, logInnerH)
		style := logPaneStyle
		if m.logFocused {
			style = logPaneStyle.BorderForeground(lipgloss.Color("11"))
		}
		rows = append(rows, style.Width(m.width-2).Height(m.logHeight()).Render(m.logPane.View()))
	}

	rows = append(rows, m.helpBar())

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) tabBar() string {
	tabs := []struct {
		label string
		panel focusedPanel
	}{
		{"Job Runs", panelJobRuns},
		{"Bundles", panelBundles},
	}
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		if m.focused == t.panel {
			parts[i] = tabActive.Render(t.label)
		} else {
			parts[i] = tabInactive.Render(t.label)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m Model) helpBar() string {
	if m.logFocused {
		return helpStyle.Render("↑/↓/pgup/pgdn: scroll log  esc: back  ctrl+c: quit")
	}
	base := "tab: switch panel  ctrl+l: " + logHint(m.logPane.IsVisible()) + "  q: quit"
	switch m.focused {
	case panelBundles:
		return helpStyle.Render(m.bundles.HelpText() + "  " + base)
	default:
		return helpStyle.Render("↑/↓: navigate  r: refresh  " + base)
	}
}

func logHint(visible bool) string {
	if visible {
		return "focus log"
	}
	return "show log"
}

func (m *Model) syncSizes() {
	listW, _, mainH := m.mainDimensions()
	m.jobRuns.SetSize(listW-2, mainH-2)
	m.bundles.SetSize(listW-2, mainH-2)
}

func (m Model) logHeight() int {
	return m.height * 30 / 100
}

func fetchClusterInfo(client *databricks.Client, clusterID string) tea.Cmd {
	return func() tea.Msg {
		info, err := client.GetCluster(context.Background(), clusterID)
		if err != nil {
			return nil
		}
		return clusterInfoMsg{clusterID: clusterID, info: info}
	}
}

func pollRunDetail(client *databricks.Client, runID int64) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		detail, err := client.GetRunDetail(context.Background(), runID)
		if err != nil {
			// Return a done detail so we stop polling on error.
			return runDetailMsg{databricks.RunDetail{Done: true}}
		}
		return runDetailMsg{detail}
	}
}

func formatTaskSummary(d databricks.RunDetail, clusters map[string]databricks.ClusterInfo) string {
	if len(d.Tasks) == 0 {
		return ""
	}

	stateColors := map[string]string{
		"RUNNING":    "2",
		"SUCCESS":    "6",
		"FAILED":     "9",
		"TERMINATED": "6",
		"PENDING":    "8",
		"BLOCKED":    "8",
		"TIMEDOUT":   "3",
	}
	clusterStateColors := map[string]string{
		"RUNNING":     "2",
		"PENDING":     "11",
		"RESTARTING":  "11",
		"RESIZING":    "11",
		"TERMINATING": "3",
		"TERMINATED":  "8",
		"ERROR":       "9",
	}

	badge := func(state, result string) string {
		s := result
		if s == "" {
			s = state
		}
		col := stateColors[s]
		if col == "" {
			col = "7"
		}
		sym := "○"
		switch s {
		case "RUNNING":
			sym = "●"
		case "SUCCESS":
			sym = "✔"
		case "FAILED", "TIMEDOUT":
			sym = "✖"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(sym + " " + s)
	}

	faint := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	for _, t := range d.Tasks {
		dur := "—"
		if t.Duration > 0 {
			dur = t.Duration.String()
		}
		b.WriteString(fmt.Sprintf("  %-28s %s  %s\n",
			t.Key, badge(t.State, t.Result), faint.Render(dur)))

		// State message (e.g. "Waiting for cluster to be provisioned")
		if t.StateMessage != "" {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				faint.Render("  └"),
				faint.Render(t.StateMessage)))
		}

		// Cluster line
		if t.ClusterID != "" {
			if info, ok := clusters[t.ClusterID]; ok {
				col := clusterStateColors[info.State]
				if col == "" {
					col = "7"
				}
				clusterState := lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(info.State)
				b.WriteString(fmt.Sprintf("  %s cluster: %s  %s  %s\n",
					faint.Render("  └"),
					detailValueStyle.Render(info.ClusterName),
					clusterState,
					faint.Render(info.NodeTypeID),
				))
			} else {
				b.WriteString(fmt.Sprintf("  %s cluster: %s\n",
					faint.Render("  └"),
					faint.Render(t.ClusterID+" (fetching…)")))
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// filterRunsByKey returns runs whose name contains the job key (case-insensitive).
func filterRunsByKey(runs []databricks.JobRun, key string) []databricks.JobRun {
	key = strings.ToLower(key)
	var out []databricks.JobRun
	for _, r := range runs {
		if strings.Contains(strings.ToLower(r.RunName), key) {
			out = append(out, r)
			if len(out) == 5 {
				break
			}
		}
	}
	return out
}

func (m Model) mainDimensions() (listW, detailW, innerH int) {
	const (
		tabBarH = 1
		helpH   = 1
		borderH = 2
	)
	logH := 0
	if m.logPane.IsVisible() {
		logH = m.logHeight() + borderH
	}
	listW = m.width * 2 / 5
	detailW = m.width - listW - borderH
	innerH = m.height - tabBarH - helpH - borderH - logH
	return
}
