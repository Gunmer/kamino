// Package tui implements the interactive terminal interface.
package tui

// FlowOptions is the input shared by the clone flow.
type CloneFlowOptions struct {
	Requested   *[]string
	Output      string
	Force       bool
	DryRun      bool
	Verbose     bool
	OutputError string
}

// ImplantFlowOptions is the input shared by the implant flow.
type ImplantFlowOptions struct {
	CloneFile string
	Requested *[]string
	Verbose   bool
	FileError string
}

// RunMenu opens the main menu.
func RunMenu() int { return runMenu() }

// RunCloneFlow opens the clone flow directly.
func RunCloneFlow(options CloneFlowOptions) int { return runCloneFlow(options) }

// RunImplantFlow opens the implant flow directly.
func RunImplantFlow(options ImplantFlowOptions) int { return runImplantFlow(options) }

// RunUpdateFlow opens the update flow.
func RunUpdateFlow() int { return runUpdateFlow() }
