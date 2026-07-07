# lazydatabricks

A [lazygit](https://github.com/jesseduffield/lazygit)-style terminal UI for Databricks — browse job runs, deploy asset bundles, manage clusters, and watch DLT pipelines without leaving your terminal.

<!-- TODO: replace with a real screenshot -->
<!-- ![screenshot](docs/screenshot.png) -->

## Panels

- **Job Runs** — recent job runs with live status, timing, and per-task detail
- **Bundles** — detects Databricks Asset Bundles (`databricks.yml`) in your working directory; validate, deploy, and run jobs with streaming CLI output and live run-status polling in a log pane
- **Clusters** — list workspace clusters, start and terminate them (with confirmation)
- **Pipelines** — Delta Live Tables pipelines with state, health, and latest update

## Install

### Homebrew

```bash
brew install JosiahDurantini/tap/lazydatabricks
```

### Go

```bash
go install github.com/JosiahDurantini/lazydatabricks/cmd/lazydatabricks@latest
```

### Binaries

Download a prebuilt binary for Linux or macOS from the [releases page](https://github.com/JosiahDurantini/lazydatabricks/releases).

## Setup

lazydatabricks uses the standard Databricks SDK auth chain — no config of its own:

1. `DATABRICKS_HOST` + `DATABRICKS_TOKEN` environment variables, or
2. a `~/.databrickscfg` profile (pick one with `DATABRICKS_CONFIG_PROFILE`), or
3. Azure / GCP native auth where applicable.

See the [Databricks auth docs](https://docs.databricks.com/dev-tools/auth) for details.

> **Note:** the Bundles panel shells out to the [`databricks` CLI](https://docs.databricks.com/dev-tools/cli/) for `validate` / `deploy` / `run`, so it must be on your `PATH`. The other panels talk to the API directly and don't need it.

Run it from a directory containing a `databricks.yml` (or a directory of bundles) to get the full Bundles experience:

```bash
lazydatabricks
```

## Keybindings

Press `?` in the app for this list.

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Switch panel |
| `↑` / `↓` | Navigate list |
| `r` | Refresh current panel (Bundles: run selected job) |
| `ctrl+l` | Show / focus the log pane |
| `?` | Help overlay |
| `q` / `ctrl+c` | Quit |

**Bundles:** `enter` open bundle · `v` validate · `d` deploy · `r` run job · `<` `>` cycle target · `esc` back

**Clusters:** `s` start · `x` terminate (press `y` to confirm)

## Development

```bash
go run ./cmd/lazydatabricks   # run from source
go test ./...                 # tests
LAZYDATABRICKS_DEBUG=1 lazydatabricks   # write debug.log in the cwd
```

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Lipgloss](https://github.com/charmbracelet/lipgloss), and the [Databricks Go SDK](https://github.com/databricks/databricks-sdk-go). See `CLAUDE.md` for architecture notes.

## License

[MIT](LICENSE)
