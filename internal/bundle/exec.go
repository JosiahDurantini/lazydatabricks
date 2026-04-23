package bundle

import (
	"bufio"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	// matches /run/12345 or /runs/12345 in a URL
	reRunID = regexp.MustCompile(`/runs?/(\d+)`)
	// matches run_id: 12345 in structured output
	reRunIDKV = regexp.MustCompile(`run.id[:\s]+(\d+)`)
)

// CommandStartMsg is returned when a process launches — holds the line channel.
type CommandStartMsg struct {
	Label string
	Lines <-chan string
}

// LineMsg carries one streamed output line.
type LineMsg string

// CommandDoneMsg signals the process has exited.
type CommandDoneMsg struct{ Err error }

// StartCommand launches "databricks <args...>" in dir and immediately returns
// a CommandStartMsg with a channel of output lines. The channel closes when
// the process exits.
func StartCommand(dir string, args ...string) tea.Cmd {
	return func() tea.Msg {
		linesCh := make(chan string, 64)

		go func() {
			cmd := exec.Command("databricks", args...)
			cmd.Dir = dir

			pr, pw := io.Pipe()
			cmd.Stdout = pw
			cmd.Stderr = pw

			if err := cmd.Start(); err != nil {
				pw.Close()
				linesCh <- "error starting process: " + err.Error()
				close(linesCh)
				return
			}

			// Close writer once process exits so the reader gets EOF.
			go func() {
				cmd.Wait()
				pw.Close()
			}()

			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				linesCh <- scanner.Text()
			}
			close(linesCh)
		}()

		label := "databricks " + strings.Join(args, " ")
		return CommandStartMsg{Label: label, Lines: linesCh}
	}
}

// ExtractRunID tries to parse a Databricks run ID from a line of CLI output.
// Returns 0 if none found.
func ExtractRunID(line string) int64 {
	for _, re := range []*regexp.Regexp{reRunID, reRunIDKV} {
		if m := re.FindStringSubmatch(line); len(m) > 1 {
			id, err := strconv.ParseInt(m[1], 10, 64)
			if err == nil && id > 0 {
				return id
			}
		}
	}
	return 0
}

// WaitForLine reads the next line from ch. Dispatch this again after each
// LineMsg to drain the stream one line at a time.
func WaitForLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return CommandDoneMsg{}
		}
		return LineMsg(line)
	}
}
