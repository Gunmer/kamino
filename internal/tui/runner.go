package tui

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/texts"
)

func runMenu() int {
	model := NewModel(CloneFlowOptions{})
	model.MenuMode = true
	model.Screen = ScreenMenu
	loaded := config.LoadConfig(model.Home, "")
	if loaded.Error != nil {
		model.startSettings()
	}
	return runModel(model)
}

func runCloneFlow(options CloneFlowOptions) int {
	model := NewModel(options)
	model.startClone()
	return runModel(model)
}

func runDeployFlow(options DeployFlowOptions) int {
	model := NewModel(CloneFlowOptions{})
	model.DeployFile = options.CloneFile
	model.DeployRequested = options.Requested
	model.DeployVerbose = options.Verbose
	model.DeployError = options.FileError
	model.startDeploy()
	return runModel(model)
}

func runUpdateFlow() int {
	model := NewModel(CloneFlowOptions{})
	model.startUpdate()
	return runModel(model)
}

func runModel(model *Model) int {
	stdin := os.Stdin
	stdout := os.Stdout
	if !isTerminalFD(int(stdin.Fd())) || !isTerminalFD(int(stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "la TUI requiere una terminal interactiva")
		return 1
	}
	model.Width, model.Height = terminalSize(int(stdout.Fd()))
	state, err := makeRaw(int(stdin.Fd()))
	if err != nil {
		fmt.Fprintln(os.Stderr, texts.ErrorSmallBody)
		return 1
	}
	defer restoreTerminal(int(stdin.Fd()), state)
	fmt.Fprint(stdout, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(stdout, "\x1b[?25h\x1b[?1049l")

	reader := newKeyReader(int(stdin.Fd()))
	for {
		width, height := terminalSize(int(stdout.Fd()))
		model.Width, model.Height = width, height
		fmt.Fprint(stdout, "\x1b[H\x1b[2J", model.View())
		if model.Quit {
			return model.ExitCode
		}
		key, err := reader.ReadKey()
		if err != nil {
			return 1
		}
		model.Update(key)
		if model.Clipboard != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(model.Clipboard))
			fmt.Fprintf(stdout, "\x1b]52;c;%s\x07", encoded)
			model.Clipboard = ""
		}
	}
}
