# lazydatabricks

A lazygit-style terminal UI for Databricks — navigate clusters, jobs, pipelines, and notebooks from your terminal.

## Stack

| Layer | Library |
|-------|---------|
| TUI framework | [Bubbletea](https://github.com/charmbracelet/bubbletea) — Elm-style Model/Update/View |
| Styling & layout | [Lipgloss](https://github.com/charmbracelet/lipgloss) |
| Pre-built components | [Bubbles](https://github.com/charmbracelet/bubbles) (list, table, spinner, viewport) |
| Databricks API | [databricks-sdk-go](https://github.com/databricks/databricks-sdk-go) |

## Project layout

```
cmd/lazydatabricks/main.go   — entry point; creates the root model and starts tea.Program
internal/app/app.go          — root Bubbletea model; owns focus state and terminal size
internal/ui/panels/          — one file per panel (clusters, jobs, pipelines, notebooks)
internal/databricks/client.go — thin wrapper around databricks.WorkspaceClient
```

## Bubbletea architecture

Every Bubbletea program has three things:

1. **Model** — a plain Go struct holding all state (no methods that mutate it directly)
2. **Update(msg) → (Model, Cmd)** — receives events (keypresses, API responses), returns a new model and optional async command
3. **View() string** — renders the model to a string; called after every Update

Commands (`tea.Cmd`) are functions that run asynchronously and return a message. Use them for API calls so the UI never blocks.

### Multi-panel pattern

The root `app.Model` holds a `focused` field and delegates key events to the focused panel's own `Update`. Each panel is itself a small model:

```go
type ClustersPanel struct {
    list   list.Model  // from bubbles
    items  []Cluster
    loaded bool
}
```

Panels communicate upward by returning messages the root model handles.

### Layout sizing

Always pass terminal dimensions down to panels via `tea.WindowSizeMsg`. Subtract border widths (usually 2) before passing to child components. Never rely on auto-wrapping.

## Auth

Databricks auth is handled automatically by the SDK in this priority order:

1. `DATABRICKS_HOST` + `DATABRICKS_TOKEN` env vars
2. `~/.databrickscfg` profile (default profile, or `DATABRICKS_CONFIG_PROFILE`)
3. Azure / GCP native auth (if applicable)

No credentials are stored in the repo.

## Dev setup

```bash
# Install Go >= 1.22 (https://go.dev/dl/ or: brew install go)
go mod tidy          # download dependencies
go run ./cmd/lazydatabricks   # run the app

# Logs are written to debug.log in the working directory
tail -f debug.log
```

## Adding a new panel

1. Create `internal/ui/panels/<name>.go` with a model struct implementing `Init / Update / View`
2. Add a `panel<Name>` constant to the `focusedPanel` iota in `internal/app/app.go`
3. Wire it into `app.Model`, `Update` (delegate keypresses), and `View` (render into layout)
4. Add any needed API methods to `internal/databricks/client.go`

## Keybindings (planned)

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus between panels |
| `↑` / `↓` | Navigate list items |
| `Enter` | Select / expand item |
| `r` | Refresh current panel |
| `q` / `Ctrl+C` | Quit |
