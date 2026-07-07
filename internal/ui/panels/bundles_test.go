package panels

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JosiahDurantini/lazydatabricks/internal/bundle"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func destroyTestModel() BundlesModel {
	cfg := &bundle.Config{RootDir: "/tmp/x"}
	cfg.Bundle.Name = "test-bundle"
	return BundlesModel{
		configs: []*bundle.Config{cfg},
		state:   stateJobList,
	}
}

func typeWord(t *testing.T, m BundlesModel, word string) (BundlesModel, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, r := range word {
		m, cmd = m.Update(key(string(r)))
		if cmd != nil {
			t.Fatalf("typing %q emitted a command early", word)
		}
	}
	return m.Update(key("enter"))
}

func TestBundleDestroyRequiresTypedConfirmation(t *testing.T) {
	m := destroyTestModel()

	// D alone must not run anything.
	m, cmd := m.Update(key("D"))
	if cmd != nil {
		t.Fatal("D emitted a command without confirmation")
	}
	if !m.destroyArmed {
		t.Fatal("D should arm the destroy confirmation")
	}

	// Typing the exact word + enter emits the destroy command.
	m, cmd = typeWord(t, m, destroyConfirmWord)
	if cmd == nil {
		t.Fatal("confirmed destroy emitted no command")
	}
	msg, ok := cmd().(BundleCommandMsg)
	if !ok {
		t.Fatalf("got %T, want BundleCommandMsg", cmd())
	}
	joined := strings.Join(msg.Args, " ")
	if !strings.HasPrefix(joined, "bundle destroy") || !strings.Contains(joined, "--auto-approve") {
		t.Errorf("destroy args = %q", joined)
	}
	if m.destroyArmed {
		t.Error("confirmation should disarm after firing")
	}
}

func TestBundleDestroyAbortsOnWrongWordOrEsc(t *testing.T) {
	m := destroyTestModel()

	m, _ = m.Update(key("D"))
	m, cmd := typeWord(t, m, "destro")
	if cmd != nil {
		t.Error("wrong word + enter must not emit a command")
	}
	if m.destroyArmed {
		t.Error("wrong word + enter should disarm")
	}

	m, _ = m.Update(key("D"))
	m, cmd = m.Update(key("esc"))
	if cmd != nil || m.destroyArmed {
		t.Error("esc should abort the confirmation")
	}

	// While armed, action keys are captured as input, not executed.
	m, _ = m.Update(key("D"))
	m, cmd = m.Update(key("d"))
	if cmd != nil {
		t.Error("action key leaked through while confirmation was armed")
	}
	if m.destroyInput != "d" {
		t.Errorf("destroyInput = %q, want %q", m.destroyInput, "d")
	}
}
