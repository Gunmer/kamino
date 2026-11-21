// Package cli parses arguments, prints output and chooses exit codes.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/Gunmer/kamino/internal/app"
	"github.com/Gunmer/kamino/internal/clone"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/tui"
	"github.com/Gunmer/kamino/internal/ui"
	"github.com/Gunmer/kamino/internal/version"
)

var (
	executeClone        = app.ExecuteClone
	executeDeploy       = app.ExecuteDeploy
	executeUpdate       = app.ExecuteUpdate
	executeAppsList     = app.ExecuteAppsList
	executeAppsValidate = app.ExecuteAppsValidate
	executeAppsNew      = app.ExecuteAppsNew
	runMenu             = tui.RunMenu
	runCloneFlow        = tui.RunCloneFlow
	runDeployFlow       = tui.RunDeployFlow
	runUpdateFlow       = tui.RunUpdateFlow
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
	// The process context: one per invocation, cancelled when the CLI exits.
	ctx := context.Background()
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
		return runCloneCommand(ctx, args[1:], options, interactive)
	case "deploy":
		return runDeployCommand(ctx, args[1:], options, interactive)
	case "update":
		return runUpdateCommand(ctx, args[1:], options, interactive)
	case "apps":
		return runAppsCommand(args[1:], options)
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
	multi       map[string][]string
	booleans    map[string]bool
	positionals []string
	help        bool
}

func parseFlags(args []string, valueFlags, boolFlags map[string]string) (parsedFlags, error) {
	return parseFlagsWithMulti(args, valueFlags, boolFlags, nil)
}

func parseFlagsWithMulti(args []string, valueFlags, boolFlags, multiFlags map[string]string) (parsedFlags, error) {
	result := parsedFlags{
		values: map[string]string{}, multi: map[string][]string{}, booleans: map[string]bool{},
	}
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
			if canonical, ok := multiFlags[name]; ok {
				if !hasValue {
					if index+1 >= len(args) {
						return result, fmt.Errorf("la opción %q requiere un argumento", name)
					}
					index++
					value = args[index]
				}
				result.multi[canonical] = append(result.multi[canonical], value)
				continue
			}
			if canonical, ok := valueFlags[name]; ok {
				if !hasValue {
					if index+1 >= len(args) {
						return result, fmt.Errorf("la opción %q requiere un argumento", name)
					}
					index++
					value = args[index]
				}
				result.values[canonical] = value
				continue
			}
			return result, fmt.Errorf("opción desconocida: %s", name)
		}
		result.positionals = append(result.positionals, arg)
	}
	return result, nil
}

func runCloneCommand(ctx context.Context, args []string, options Options, interactive func() bool) int {
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
			resolved, err := resolveUserPath(outputValue)
			if err != nil {
				fmt.Fprintln(options.Stderr, "Error: "+err.Error())
				return 2
			}
			outputPath = resolved
			outputError = clone.ValidateDestination(outputPath, false)
		}
		return runCloneFlow(tui.CloneFlowOptions{
			Requested: requested, Output: outputPath, Force: force,
			DryRun: dryRun, Verbose: verbose, OutputError: outputError,
		})
	}

	ui.PrintPrivacyWarning(options.Stdout)
	outcome, err := executeClone(ctx, requested, outputValue, dryRun, force, nil, "", nil, "")
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

func runDeployCommand(ctx context.Context, args []string, options Options, interactive func() bool) int {
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
		fmt.Fprintln(options.Stderr, "Error: falta el argumento 'CLONE_FILE'.")
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
	outcome, err := executeDeploy(ctx, cloneFile, requested, dryRun, nil, "", nil, "", "")
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

func runUpdateCommand(ctx context.Context, args []string, options Options, interactive func() bool) int {
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
	result, err := executeUpdate(ctx, version.Version, "")
	if err != nil {
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	ui.PrintUpdateResult(options.Stdout, result)
	return 0
}

func runAppsCommand(args []string, options Options) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printAppsHelp(options.Stdout)
		return 0
	}
	switch args[0] {
	case "list":
		return runAppsListCommand(args[1:], options)
	case "validate":
		return runAppsValidateCommand(args[1:], options)
	case "new":
		return runAppsNewCommand(args[1:], options)
	default:
		fmt.Fprintf(options.Stderr, "Error: No such command %q.\n", args[0])
		printAppsHelp(options.Stderr)
		return 2
	}
}

func runAppsListCommand(args []string, options Options) int {
	flags, err := parseFlags(args, map[string]string{}, map[string]string{})
	if err != nil {
		fmt.Fprintln(options.Stderr, "Error: "+err.Error())
		return 2
	}
	if flags.help {
		printAppsListHelp(options.Stdout)
		return 0
	}
	if len(flags.positionals) != 0 {
		fmt.Fprintln(options.Stderr, "Error: 'apps list' no admite argumentos.")
		return 2
	}
	outcome, err := executeAppsList("")
	if err != nil {
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	printAppsConfigWarning(outcome.ConfigError, outcome.Directory, options.Stderr)
	fmt.Fprintf(options.Stdout, "Catálogo: %s\n\n", outcome.Directory)
	writer := tabwriter.NewWriter(options.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "ORIGEN\tID/ARCHIVO\tNOMBRE\tNOTA")
	for _, entry := range outcome.Entries {
		if entry.Err != nil {
			fmt.Fprintf(writer, "ERROR\t%s\t%s\t%s\n", entry.File, dash(entry.Name), entry.Err.Error())
			continue
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", entry.Source, entry.ID, entry.Name, strings.Join(entry.Warnings, "; "))
	}
	_ = writer.Flush()
	return 0
}

func runAppsValidateCommand(args []string, options Options) int {
	flags, err := parseFlags(args, map[string]string{}, map[string]string{})
	if err != nil {
		fmt.Fprintln(options.Stderr, "Error: "+err.Error())
		return 2
	}
	if flags.help {
		printAppsValidateHelp(options.Stdout)
		return 0
	}
	if len(flags.positionals) > 1 {
		fmt.Fprintln(options.Stderr, "Error: 'apps validate' admite como máximo una ruta.")
		return 2
	}
	path := ""
	if len(flags.positionals) == 1 {
		path = flags.positionals[0]
	}
	outcome, err := executeAppsValidate("", path)
	if err != nil {
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	if path == "" {
		printAppsConfigWarning(outcome.ConfigError, outcome.Directory, options.Stderr)
		fmt.Fprintf(options.Stdout, "Catálogo: %s\n\n", outcome.Directory)
	} else {
		fmt.Fprintf(options.Stdout, "Ruta: %s\n\n", path)
	}
	writer := tabwriter.NewWriter(options.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "ESTADO\tARCHIVO\tID\tNOMBRE\tDETALLE")
	for _, entry := range outcome.Entries {
		status := "OK"
		detail := strings.Join(entry.Warnings, "; ")
		if entry.Err != nil {
			status = "ERROR"
			detail = entry.Err.Error()
		} else if len(entry.Warnings) > 0 {
			status = "AVISO"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", status, entry.File, dash(entry.ID), dash(entry.Name), detail)
	}
	_ = writer.Flush()
	if len(outcome.Entries) == 0 {
		fmt.Fprintln(options.Stdout, "Sin definiciones en el catálogo.")
	}
	if !outcome.Valid {
		return 1
	}
	return 0
}

func runAppsNewCommand(args []string, options Options) int {
	flags, err := parseFlagsWithMulti(args,
		map[string]string{"--command": "command", "--bundle-id": "bundle", "--name": "name"},
		map[string]string{"--force": "force", "--print": "print"},
		map[string]string{"--path": "paths", "--optional-path": "optional_paths", "--search-path": "search_paths"},
	)
	if err != nil {
		fmt.Fprintln(options.Stderr, "Error: "+err.Error())
		return 2
	}
	if flags.help {
		printAppsNewHelp(options.Stdout)
		return 0
	}
	if len(flags.positionals) != 1 {
		fmt.Fprintln(options.Stderr, "Error: falta el argumento 'ID'.")
		return 2
	}
	id := flags.positionals[0]
	command := flags.values["command"]
	bundle := flags.values["bundle"]
	paths := flags.multi["paths"]
	optional := flags.multi["optional_paths"]
	if command == "" && bundle == "" {
		fmt.Fprintln(options.Stderr, "Error: falta --command o --bundle-id.")
		return 2
	}
	if len(paths) == 0 && len(optional) == 0 {
		fmt.Fprintln(options.Stderr, "Error: falta --path o --optional-path.")
		return 2
	}
	outcome, err := executeAppsNew("", app.AppsNewOptions{
		ID: id, Name: flags.values["name"], Command: command, BundleID: bundle,
		Paths: paths, OptionalPaths: optional, SearchPaths: flags.multi["search_paths"],
		Force: flags.booleans["force"], Print: flags.booleans["print"],
	}, nil, "")
	if err != nil {
		fmt.Fprintln(options.Stderr, err.Error())
		return 1
	}
	if flags.booleans["print"] {
		fmt.Fprint(options.Stdout, outcome.Content)
		return 0
	}
	if outcome.ConfigError != nil {
		printAppsConfigWarning(outcome.ConfigError, outcome.Directory, options.Stderr)
	}
	for _, warning := range outcome.Warnings {
		fmt.Fprintln(options.Stderr, "aviso: "+warning)
	}
	fmt.Fprintf(options.Stdout, "creado %s\n", outcome.Path)
	fmt.Fprintf(options.Stdout, "siguiente: kamino apps validate %s\n", outcome.Path)
	return 0
}

func printAppsConfigWarning(err error, directory string, writer io.Writer) {
	if err == nil {
		return
	}
	fmt.Fprintf(writer, "aviso: %v; se usa el catálogo integrado en %s\n", err, directory)
}

func dash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
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

func resolveUserPath(value string) (string, error) {
	if strings.HasPrefix(value, "~") {
		home, err := paths.ResolveHome("")
		if err != nil {
			return "", err
		}
		if value == "~" {
			value = home
		} else if strings.HasPrefix(value, "~/") {
			value = filepath.Join(home, value[2:])
		}
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return absolute, nil
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
	fmt.Fprintln(w, "  --help, -h             Muestra esta ayuda y termina.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  clone   Crea un clon con las apps elegidas.")
	fmt.Fprintln(w, "  deploy  Despliega desde un archivo, solo las apps instaladas.")
	fmt.Fprintln(w, "  update  Actualiza kamino a la última versión publicada.")
	fmt.Fprintln(w, "  apps    Crea y valida definiciones de apps.")
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
	fmt.Fprintln(w, "  --help, -h  Muestra esta ayuda y termina.")
}

func printAppsHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino apps [OPTIONS] COMMAND [ARGS]...")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Crea y valida definiciones de apps.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  list      Muestra el catálogo integrado y el del usuario.")
	fmt.Fprintln(w, "  validate  Valida un archivo, un directorio o el catálogo del usuario.")
	fmt.Fprintln(w, "  new       Crea una definición TOML nueva.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --help, -h  Muestra esta ayuda y termina.")
}

func printAppsListHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino apps list [OPTIONS]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Muestra el catálogo integrado y las definiciones del usuario, con su origen y estado.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --help, -h  Muestra esta ayuda y termina.")
}

func printAppsValidateHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino apps validate [OPTIONS] [RUTA]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Valida un archivo, un directorio o el catálogo del usuario.")
	fmt.Fprintln(w, "  Sale con código 1 si hay errores.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --help, -h  Muestra esta ayuda y termina.")
}

func printAppsNewHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: kamino apps new [OPTIONS] ID")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Crea una definición TOML válida en el catálogo del usuario.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --command TEXT         Ejecutable para detectar la app.")
	fmt.Fprintln(w, "  --bundle-id TEXT       Identificador del bundle de macOS.")
	fmt.Fprintln(w, "  --path TEXT            Ruta global del HOME que se clona; repetible.")
	fmt.Fprintln(w, "  --optional-path TEXT   Ruta opcional; repetible.")
	fmt.Fprintln(w, "  --search-path TEXT     Directorio adicional para buscar el comando; repetible.")
	fmt.Fprintln(w, "  --name TEXT            Nombre visible; por defecto, el id.")
	fmt.Fprintln(w, "  --force                Reemplaza <id>.toml si ya existe.")
	fmt.Fprintln(w, "  --print                Escribe el TOML en stdout sin crear archivos.")
	fmt.Fprintln(w, "  --help, -h             Muestra esta ayuda y termina.")
}

func completionScript(shell string) string {
	switch shell {
	case "zsh":
		return "#compdef kamino\n_kamino() { _arguments '*: :((clone deploy update apps))' }\ncompdef _kamino kamino\n"
	case "fish":
		return "complete -c kamino -f -a 'clone deploy update apps'\n"
	default:
		return "complete -W 'clone deploy update apps' kamino\n"
	}
}
