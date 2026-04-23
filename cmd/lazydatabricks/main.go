package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jos/lazydatabricks/internal/app"
)

func main() {
	// Optional: write logs to a file during development
	f, err := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(f)
		defer f.Close()
	}

	model, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialise app: %v", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("error running program: %v", err)
	}
}
