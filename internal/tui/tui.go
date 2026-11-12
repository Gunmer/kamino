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

// DeployFlowOptions is the input shared by the deploy flow.
type DeployFlowOptions struct {
	CloneFile string
	Requested *[]string
	Verbose   bool
	FileError string
}

// RunMenu opens the main menu.
func RunMenu() int { return runMenu() }

// RunCloneFlow opens the clone flow directly.
func RunCloneFlow(options CloneFlowOptions) int { return runCloneFlow(options) }

// RunDeployFlow opens the deploy flow directly.
func RunDeployFlow(options DeployFlowOptions) int { return runDeployFlow(options) }

// RunUpdateFlow opens the update flow.
func RunUpdateFlow() int { return runUpdateFlow() }
