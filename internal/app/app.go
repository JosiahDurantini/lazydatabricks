package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jos/lazydatabricks/internal/databricks"
)

// focusedPanel tracks which panel currently receives keyboard input.
type focusedPanel int

const (
	panelClusters focusedPanel = iota
	panelJobs
	panelPipelines
	panelNotebooks
)

// Model is the root Bubbletea model — it owns all panel state.
type Model struct {
	client  *databricks.Client
	focused focusedPanel
	width   int
	height  int
}

// New creates the root model and initialises the Databricks client.
func New() (Model, error) {
	client, err := databricks.NewClient()
	if err != nil {
		return Model{}, err
	}
	return Model{client: client}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		// Tab cycles focus between panels
		case "tab":
			m.focused = focusedPanel((int(m.focused) + 1) % 4)
		case "shift+tab":
			m.focused = focusedPanel((int(m.focused) + 3) % 4)
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	return "lazydatabricks — press q to quit\n(panels coming soon)"
}
