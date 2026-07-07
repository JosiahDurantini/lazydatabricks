package panels

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/JosiahDurantini/lazydatabricks/internal/bundle"
	"github.com/JosiahDurantini/lazydatabricks/internal/databricks"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FetchRunsForKeyMsg is sent upward to the app to request recent runs.
type FetchRunsForKeyMsg struct{ Key string }

// RunsForKeyMsg is sent back down with the fetched runs.
type RunsForKeyMsg struct {
	Key  string
	Runs []databricks.JobRun
}

// BundleCommandMsg is sent upward to the app to trigger a CLI command.
type BundleCommandMsg struct {
	Dir  string
	Args []string
}

// ── view state ───────────────────────────────────────────────────────────────

type bundleViewState int

const (
	stateBundleList bundleViewState = iota
	stateJobList
)

// ── list items ───────────────────────────────────────────────────────────────

type bundleItem struct{ cfg *bundle.Config }

func (b bundleItem) FilterValue() string { return b.cfg.Bundle.Name }
func (b bundleItem) Title() string       { return b.cfg.Bundle.Name }
func (b bundleItem) Description() string {
	return fmt.Sprintf("%d jobs", len(b.cfg.Resources.Jobs))
}

type jobItem struct {
	key  string
	name string
}

func (j jobItem) FilterValue() string { return j.key }
func (j jobItem) Title() string       { return j.key }
func (j jobItem) Description() string { return "key: " + j.key }

// ── model ────────────────────────────────────────────────────────────────────

type BundlesModel struct {
	configs         []*bundle.Config
	bundleList      list.Model
	jobList         list.Model
	state           bundleViewState
	selected        int // index into configs when in stateJobList
	targets         []string
	target          int
	lastSelectedKey string
	recentRuns      map[string][]databricks.JobRun
	initCmd         tea.Cmd
	err             error
	width           int
	height          int
}

func newList(title string) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return l
}

func NewBundles() BundlesModel {
	m := BundlesModel{
		bundleList: newList("Bundles"),
		jobList:    newList("Jobs"),
	}

	cwd, _ := os.Getwd()
	configs, err := bundle.DetectAll(cwd)
	if err != nil {
		m.err = err
		return m
	}
	m.configs = configs

	if len(configs) == 1 {
		// Single bundle — jump straight into the job list.
		m.initCmd = m.enterBundle(0)
	} else {
		items := make([]list.Item, len(configs))
		for i, cfg := range configs {
			items[i] = bundleItem{cfg}
		}
		m.bundleList.SetItems(items)
	}

	return m
}

func (m BundlesModel) Init() tea.Cmd { return m.initCmd }

func (m BundlesModel) Update(msg tea.Msg) (BundlesModel, tea.Cmd) {
	if m.err != nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.state == stateBundleList {
				if sel, ok := m.bundleList.SelectedItem().(bundleItem); ok {
					for i, cfg := range m.configs {
						if cfg == sel.cfg {
							cmd := m.enterBundle(i)
							return m, cmd
						}
					}
				}
				return m, nil
			}
		case "esc":
			if m.state == stateJobList && len(m.configs) > 1 {
				m.state = stateBundleList
				return m, nil
			}
		case "v":
			if m.state == stateJobList {
				cfg := m.configs[m.selected]
				return m, bundleCmd(cfg.RootDir, "bundle", "validate")
			}
		case "d":
			if m.state == stateJobList {
				cfg := m.configs[m.selected]
				args := []string{"bundle", "deploy"}
				if t := m.currentTarget(); t != "" {
					args = append(args, "--target", t)
				}
				return m, bundleCmd(cfg.RootDir, args...)
			}
		case "r":
			if m.state == stateJobList {
				sel, ok := m.jobList.SelectedItem().(jobItem)
				if !ok || sel.key == "" {
					return m, nil
				}
				cfg := m.configs[m.selected]
				args := []string{"bundle", "run"}
				if t := m.currentTarget(); t != "" {
					args = append(args, "--target", t)
				}
				args = append(args, sel.key)
				return m, bundleCmd(cfg.RootDir, args...)
			}
		case ">":
			if len(m.targets) > 0 {
				m.target = (m.target + 1) % len(m.targets)
			}
			return m, nil
		case "<":
			if len(m.targets) > 0 {
				m.target = (m.target + len(m.targets) - 1) % len(m.targets)
			}
			return m, nil
		}
	}

	// Handle runs arriving back from the app.
	if msg, ok := msg.(RunsForKeyMsg); ok {
		if m.recentRuns == nil {
			m.recentRuns = make(map[string][]databricks.JobRun)
		}
		m.recentRuns[msg.Key] = msg.Runs
		return m, nil
	}

	var cmd tea.Cmd
	if m.state == stateBundleList {
		m.bundleList, cmd = m.bundleList.Update(msg)
	} else {
		prevIdx := m.jobList.Index()
		m.jobList, cmd = m.jobList.Update(msg)
		// Trigger a run fetch when the selected job changes.
		if m.jobList.Index() != prevIdx {
			if newKey := m.selectedJobKey(); newKey != m.lastSelectedKey {
				m.lastSelectedKey = newKey
				return m, tea.Batch(cmd, requestRunsForKey(newKey))
			}
		}
	}
	return m, cmd
}

func (m *BundlesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.bundleList.SetSize(w, h)
	m.jobList.SetSize(w, h)
}

func (m *BundlesModel) enterBundle(idx int) tea.Cmd {
	m.selected = idx
	m.state = stateJobList
	cfg := m.configs[idx]
	m.jobList.Title = cfg.Bundle.Name

	items := make([]list.Item, 0, len(cfg.Resources.Jobs))
	for key, job := range cfg.Resources.Jobs {
		items = append(items, jobItem{key: key, name: job.Name})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].(jobItem).key < items[j].(jobItem).key
	})
	m.jobList.SetItems(items)

	m.targets = nil
	for name := range cfg.Targets {
		m.targets = append(m.targets, name)
	}
	sort.Strings(m.targets)
	m.target = 0

	// Trigger fetch for whichever job is selected first.
	if firstKey := m.selectedJobKey(); firstKey != "" {
		m.lastSelectedKey = firstKey
		return requestRunsForKey(firstKey)
	}
	return nil
}

func (m BundlesModel) selectedJobKey() string {
	if sel, ok := m.jobList.SelectedItem().(jobItem); ok {
		return sel.key
	}
	return ""
}

func requestRunsForKey(key string) tea.Cmd {
	return func() tea.Msg { return FetchRunsForKeyMsg{Key: key} }
}

func (m BundlesModel) currentTarget() string {
	if len(m.targets) == 0 {
		return ""
	}
	return m.targets[m.target]
}

// HelpText returns context-sensitive help for the current state.
func (m BundlesModel) HelpText() string {
	if m.state == stateBundleList {
		return "enter: open bundle"
	}
	parts := []string{"v: validate", "d: deploy", "r: run job", "< >: target"}
	if len(m.configs) > 1 {
		parts = append(parts, "esc: back")
	}
	return strings.Join(parts, "  ")
}

func (m BundlesModel) ViewList() string {
	if m.err != nil {
		return errorStyle.Render(m.err.Error())
	}
	if m.state == stateBundleList {
		return m.bundleList.View()
	}
	return m.jobList.View()
}

func (m BundlesModel) ViewDetail() string {
	if m.err != nil {
		return faintStyle.Render("No bundles found.\nRun from a bundle or bundles directory.")
	}

	if m.state == stateBundleList {
		sel, ok := m.bundleList.SelectedItem().(bundleItem)
		if !ok {
			return faintStyle.Render("No bundle selected.")
		}
		cfg := sel.cfg
		var b strings.Builder
		row := func(label, value string) {
			b.WriteString(detailLabel.Render(label) + "  " + detailValue.Render(value) + "\n")
		}
		row("Bundle", cfg.Bundle.Name)
		row("Root  ", cfg.RootDir)
		row("Jobs  ", fmt.Sprintf("%d", len(cfg.Resources.Jobs)))
		b.WriteString("\n" + faintStyle.Render("enter to open") + "\n")
		return b.String()
	}

	// Job list state
	cfg := m.configs[m.selected]
	var b strings.Builder

	b.WriteString(detailLabel.Render("Bundle") + "  " + detailValue.Render(cfg.Bundle.Name) + "\n")

	target := m.currentTarget()
	if target == "" {
		target = "(none)"
	}
	b.WriteString(detailLabel.Render("Target") + "  " +
		detailValue.Render(target) + "  " +
		faintStyle.Render("< > to switch") + "\n")

	b.WriteString("\n" + detailLabel.Render("Actions") + "\n")
	for _, a := range []struct{ key, desc string }{
		{"v", "validate"},
		{"d", "deploy to target"},
		{"r", "run selected job"},
	} {
		b.WriteString("  " +
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")).Render(a.key) +
			"  " + a.desc + "\n")
	}

	if len(m.configs) > 1 {
		b.WriteString("\n" + faintStyle.Render("esc: back to bundles") + "\n")
	}

	// Recent runs for the selected job.
	if key := m.selectedJobKey(); key != "" {
		runs := m.recentRuns[key]
		b.WriteString("\n" + detailLabel.Render("Recent runs") + "\n")
		if runs == nil {
			b.WriteString(faintStyle.Render("  loading…") + "\n")
		} else if len(runs) == 0 {
			b.WriteString(faintStyle.Render("  no runs found") + "\n")
		} else {
			for _, r := range runs {
				ago := time.Since(r.StartTime).Round(time.Minute)
				b.WriteString(fmt.Sprintf("  %s  %s  %s ago\n",
					statusLabel(r.Status),
					faintStyle.Render(r.Duration.Round(time.Second).String()),
					faintStyle.Render(ago.String()),
				))
			}
		}
	}

	return b.String()
}

func bundleCmd(dir string, args ...string) tea.Cmd {
	return func() tea.Msg {
		return BundleCommandMsg{Dir: dir, Args: args}
	}
}
