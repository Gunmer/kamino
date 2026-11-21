package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// editorFinishedMsg is delivered after the external editor returns.
type editorFinishedMsg struct {
	err error
}

// defaultEditor opens path with $VISUAL, $EDITOR or vi. The editor string may
// contain flags; the path is passed as a positional shell argument.
func defaultEditor(path string) tea.Cmd {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}
	command := exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
	return tea.ExecProcess(command, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}
