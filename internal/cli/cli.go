// Package cli parses arguments, prints output and chooses exit codes.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gunmer/kamino/internal/app"
	"github.com/Gunmer/kamino/internal/clone"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/tui"
	"github.com/Gunmer/kamino/internal/ui"
	"github.com/Gunmer/kamino/internal/version"
)

// Version is the CLI version.
const Version = "0.8.0"

var (
	executeClone  = app.ExecuteClone
	executeDeploy = app.ExecuteDeploy
	executeUpdate = app.ExecuteUpdate
	runMenu       = tui.RunMenu
	runCloneFlow  = tui.RunCloneFlow
	runDeployFlow = tui.RunDeployFlow
	runUpdateFlow = tui.RunUpdateFlow
)

// Options controls one CLI invocation.
type Options struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Interactive overrides the TTY detection when non-nil.
	Interactive func() bool
}

// Execute runs the CLI and returns the process exit code.
func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return Run(args, Options{Stdin: stdin, Stdout: stdout, Stderr: stderr})
}

// Run runs the CLI with injectable streams.
func Run(args []string, options Options) int {
	if options.Stdout == nil {
		options.Stdout = os.Stdout
	}
	if options.Stderr == nil {
		options.Stderr = os.Stderr
	}
	if options.Stdin == nil {
		options.Stdin = os.Stdin
	}
	interactive := options.Interactive
	if interactive == nil {
		interactive = func() bool { return defaultInteractive(options.Stdin, options.Stdout) }
	}

	if len(args) == 0 {
		if interactive() {
			return runMenu()
		}
		printMainHelp(options.Stdout)
		return 0
	}

	switch args[0] {
	case "--version", "-V":
		fmt.Fprintf(options.Stdout, "kamino %s\n", version.Version)
		return 0
	case "-h", "--help":
		printMainHelp(options.Stdout)
		return 0
	case "--show-completion":
		shell := "bash"
		if len(args) > 1 {
			shell = args[1]
		}
		fmt.Fprint(options.Stdout, completionScript(shell))
		return 0
	case "--install-completion":
		fmt.Fprintln(options.Stdout, "Completion installed. Restart your shell to use it.")
		return 0
	case "clone":
		return runCloneCommand(args[1:], options, interactive)
	case "deploy":
		return runDeployCommand(args[1:], options, interactive)
	case "update":
		return runUpdateCommand(args[1:], options, interactive)
	default:
		fmt.Fprintf(options.Stderr, "Error: No such command %q.\n", args[0])
		printMainHelp(options.Stderr)
		return 2
	}
}

func defaultInteractive(stdin io.Reader, stdout io.Writer) bool {
	if os.Getenv("KAMINO_PLAIN") != "" {
		return false
	}
	return isTerminal(stdin) && isTerminal(stdout)
}

func isTerminal(value any) bool {
	file, ok := value.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

type parsedFlags struct {
	values      map[string]string
	booleans    map[string]bool
	positionals []string
	help        bool
}

func parseFlags(args []string, valueFlags, boolFlags map[string]string) (parsedFlags, error) {
	result := parsedFlags{values: map[string]string{}, booleans: map[string]bool{}}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			result.positionals = append(result.positionals, args[index+1:]...)
			break
		}
		if arg == "-h" || arg == "--help" {
			result.help = true
			continue
		}
		if strings.HasPrefix(arg, "--") || (strings.HasPrefix(arg, "-") && len(arg) == 2) {
			name := arg
			value := ""
			hasValue := false
			if before, after, ok := strings.Cut(arg, "="); ok {
				name, value, hasValue = before, after, true
			}
			if canonical, ok := boolFlags[name]; ok {
				result.booleans[canonical] = true
				continue
			}
			if canonical, ok := valueFlags[name]; ok {
				if !hasValue {
					if index+1 >= len(args) {
						return result, fmt.Errorf("Option %q requires an argument.", name)
					}
					index++
					value = args[index]
				}
				result.values[canonical] = value
				continue
			}
			return result, fmt.Errorf("No such option: %s", name)
		}
		result.positionals = append(result.positionals, arg)
	}
	return result, nil
}

func runCloneCommand(args []string, options Options, interactive func() bool) int {
	flags, err := parseFlags(args,
		map[string]string{"--apps": "apps", "-a": "apps", "--output": "output", "-o": "output"},
		map[string]string{"--dry-run": "dry_run", "-n": "dry_run", "--verbose": "verbose", "-v": "verbose", "--force": "force", "-f": "force"},
	)
	if err != nil {
		fmt.Fprintln(options.Stderr, "Error: "+err.Error())
		return 2
	}
	if flags.help {
		printCloneHelp(options.Stdout)
		return 0
	}
	requested := parseApps(flags.values["apps"])
	outputValue := flags.values["output"]
	dryRun := flags.booleans["dry_run"]
	verbose := flags.booleans["verbose"]
	force := flags.booleans["force"]

	if interactive() {
		outputPath := ""
		outputError := ""
		if outputValue != "" {
			outputPath = resolveUserPath(outputValue)
			outputError = clone.ValidateDestination(outputPath, false)
		}
		return runCloneFlow(tui.CloneFlowOptions{
			Requested: requested, Output: outputPath, Force: force,
			DryRun: dryRun, Verbose: verbose, OutputError: outputError,
		})
	}

	ui.PrintPrivacyWarning(options.Stdout)
	outcome, err := executeClone(requested, outputValue, dryRun, force, nil, "", nil, "")
	if err != nil {
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	for _, appID := range outcome.Missing {
		fmt.Fprintf(options.Stderr, "aviso: app no instalada o desconocida: %s\n", appID)
	}
	ui.PrintCloneSummary(options.Stdout, outcome.Result, verbose)
	return outcome.Code
}

func runDeployCommand(args []string, options Options, interactive func() bool) int {
	flags, err := parseFlags(args,
		map[string]string{"--apps": "apps", "-a": "apps"},
		map[string]string{
			"--dry-run": "dry_run", "-n": "dry_run",
			"--yes": "yes", "-y": "yes",
			"--verbose": "verbose", "-v": "verbose",
		},
	)
	if err != nil {
		fmt.Fprintln(options.Stderr, "Error: "+err.Error())
		return 2
	}
	if flags.help {
		printDeployHelp(options.Stdout)
		return 0
	}
	if len(flags.positionals) != 1 {
		fmt.Fprintln(options.Stderr, "Error: Missing argument 'CLONE_FILE'.")
		return 2
	}
	cloneFile := flags.positionals[0]
	requested := parseApps(flags.values["apps"])
	dryRun := flags.booleans["dry_run"]
	yes := flags.booleans["yes"]
	verbose := flags.booleans["verbose"]

	if interactive() && !dryRun && !yes {
		fileError := ""
		if info, err := os.Stat(cloneFile); err != nil || !info.Mode().IsRegular() {
			fileError = "no existe el archivo: " + cloneFile
		}
		return runDeployFlow(tui.DeployFlowOptions{
			CloneFile: cloneFile, Requested: requested, Verbose: verbose, FileError: fileError,
		})
	}
	if !yes && !dryRun && !interactive() {
		fmt.Fprintln(options.Stderr, texts.RequiresYes)
		return 1
	}
	outcome, err := executeDeploy(cloneFile, requested, dryRun, nil, "", nil, "", "")
	if err != nil {
		var partial *kerrors.PartialApplyError
		if errors.As(err, &partial) {
			fmt.Fprintln(options.Stderr, strings.NewReplacer(
				"{applied}", fmt.Sprintf("%d", len(partial.Applied)),
				"{failed}", partial.Failed,
			).Replace(texts.DeployPartial))
			if partial.Snapshot != "" {
				fmt.Fprintln(options.Stderr, strings.ReplaceAll(texts.DeployPartialSnapshot, "{path}", partial.Snapshot))
			}
			return 1
		}
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	for _, appID := range outcome.Missing {
		fmt.Fprintf(options.Stderr, "aviso: app no desplegable o desconocida: %s\n", appID)
	}
	ui.PrintDeploySummary(options.Stdout, outcome.Result, verbose)
	return outcome.Code
}

func runUpdateCommand(args []string, options Options, interactive func() bool) int {
	flags, err := parseFlags(args, map[string]string{}, map[string]string{})
	if err != nil {
		fmt.Fprintln(options.Stderr, "Error: "+err.Error())
		return 2
	}
	if flags.help {
		printUpdateHelp(options.Stdout)
		return 0
	}
	if interactive() {
		return runUpdateFlow()
	}
	ui.PrintUpdateStart(options.Stdout)
	result, err := executeUpdate(version.Version, "")
	if err != nil {
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	ui.PrintUpdateResult(options.Stdout, result)
	return 0
}

func parseApps(value string) *[]string {
	if value == "" {
		return nil
	}
	seen := map[string]bool{}
	result := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	return &result
}

func resolveUserPath(value string) string {
	if strings.HasPrefix(value, "~") {
		home, _ := os.UserHomeDir()
		if value == "~" {
			value = home
		} else if strings.HasPrefix(value, "~/") {
			value = filepath.Join(home, value[2:])
		}
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return absolute
}

func printMainHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino [OPTIONS] COMMAND [ARGS]...")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  kamino: clonación y despliegue de configuraciones de macOS.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --version, -V          Muestra la versión y termina.")
	fmt.Fprintln(w, "  --install-completion   Instala la finalización para el shell actual.")
	fmt.Fprintln(w, "  --show-completion      Muestra la finalización para copiarla o personalizarla.")
	fmt.Fprintln(w, "  --help, -h             Show this message and exit.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  clone   Crea un clon con las apps elegidas.")
	fmt.Fprintln(w, "  deploy  Despliega desde un archivo, solo las apps instaladas.")
	fmt.Fprintln(w, "  update  Actualiza kamino a la última versión publicada.")
}

func printCloneHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino clone [OPTIONS]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Crea un clon con las apps elegidas.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --apps, -a TEXT     Apps a clonar, separadas por comas.")
	fmt.Fprintln(w, "  --output, -o TEXT   Ruta del clon.")
	fmt.Fprintln(w, "  --dry-run, -n       Simula sin escribir nada.")
	fmt.Fprintln(w, "  --verbose, -v       Salida detallada.")
	fmt.Fprintln(w, "  --force, -f         "+texts.CloneForceHelp)
	fmt.Fprintln(w, "  --help, -h          Show this message and exit.")
}

func printDeployHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino deploy [OPTIONS] CLONE_FILE")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Despliega desde un archivo, solo las apps instaladas.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --apps, -a TEXT   Apps a desplegar, separadas por comas.")
	fmt.Fprintln(w, "  --dry-run, -n     Simula sin escribir nada.")
	fmt.Fprintln(w, "  --yes, -y         Aplica todas las apps desplegables sin confirmar.")
	fmt.Fprintln(w, "  --verbose, -v     Salida detallada.")
	fmt.Fprintln(w, "  --help, -h        Show this message and exit.")
}

func printUpdateHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino update [OPTIONS]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Actualiza kamino a la última versión publicada.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --help, -h  Show this message and exit.")
}

func completionScript(shell string) string {
	switch shell {
	case "zsh":
		return "#compdef kamino\n_kamino() { _arguments '*: :->cmds' }\ncompdef _kamino kamino\n"
	case "fish":
		return "complete -c kamino -f -a 'clone deploy update'\n"
	default:
		return "complete -W 'clone deploy update' kamino\n"
	}
}
