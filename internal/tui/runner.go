package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/texts"
)

func runMenu() int {
	app := NewApp(AppOptions{MenuMode: true})
	app.startMenu()
	return runProgram(app)
}

func runCloneFlow(options CloneFlowOptions) int {
	app := NewApp(AppOptions{Clone: options})
	app.startClone()
	return runProgram(app)
}

func runDeployFlow(options DeployFlowOptions) int {
	app := NewApp(AppOptions{Deploy: options})
	app.startDeploy()
	return runProgram(app)
}

func runUpdateFlow() int {
	app := NewApp(AppOptions{})
	app.startUpdate()
	return runProgram(app)
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
