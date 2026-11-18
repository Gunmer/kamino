package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/texts"
)

func runMenu() int {
	app := newRootApp(AppOptions{MenuMode: true})
	app.startMenu()
	return runProgram(app)
}

func runCloneFlow(options CloneFlowOptions) int {
	app := newRootApp(AppOptions{Clone: options})
	app.startClone()
	return runProgram(app)
}

func runDeployFlow(options DeployFlowOptions) int {
	app := newRootApp(AppOptions{Deploy: options})
	app.startDeploy()
	return runProgram(app)
}

func runUpdateFlow() int {
	app := newRootApp(AppOptions{})
	app.startUpdate()
	return runProgram(app)
}

// newRootApp builds the model with the context that owns every TUI operation.
func newRootApp(options AppOptions) *App {
	options.Context = context.Background()
	return NewApp(options)
}

func runProgram(app *App) int {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		fmt.Fprintln(os.Stderr, texts.TTYRequired)
		return 1
	}
	program := tea.NewProgram(app, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	final, err := program.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if value, ok := final.(*App); ok {
		return value.ExitCode()
	}
	return 0
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
