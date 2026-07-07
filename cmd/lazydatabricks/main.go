package main

import (
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jos/lazydatabricks/internal/app"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println("lazydatabricks " + version)
			return
		}
	}

	// Debug logging is opt-in so a published binary doesn't litter the cwd.
	if os.Getenv("LAZYDATABRICKS_DEBUG") != "" {
		f, err := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(f)
			defer f.Close()
		}
	} else {
		log.SetOutput(io.Discard)
	}

	model, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, `lazydatabricks: could not connect to Databricks: %v

Configure credentials one of these ways:
  • export DATABRICKS_HOST and DATABRICKS_TOKEN
  • create a profile in ~/.databrickscfg (and optionally set DATABRICKS_CONFIG_PROFILE)
  • use Azure/GCP native auth if your workspace supports it

Docs: https://docs.databricks.com/dev-tools/auth
`, err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazydatabricks: %v\n", err)
		os.Exit(1)
	}
}
