package tui

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/app"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/detect"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/tui/theme"
	"github.com/Gunmer/kamino/internal/update"
	"github.com/Gunmer/kamino/internal/version"
)

// AppOptions injects every environment dependency so tests never touch the
// developer's HOME, clock, terminal or network.
type AppOptions struct {
	// Context is the parent context of every cancellable operation. The TUI
	// entrypoint creates it once; a nil value means context.Background().
	Context         context.Context
	Home            string
	Catalog         catalog.Catalog
	ApplicationDirs []string
	PathValue       string
	Cwd             string
	Now             func() time.Time
	ThemeName       string
	NoColor         bool
	Unicode         *bool
	Phrase          string
	Version         string
	MenuMode        bool
	Clone           CloneFlowOptions
	Deploy          DeployFlowOptions
	UpdateQuery     func(context.Context) (update.Check, error)
	UpdateRun       func(context.Context, progress.Callback) (update.Result, error)
	// Editor opens an authoring draft. Tests inject a fake; nil uses the
	// user's $VISUAL, $EDITOR or vi.
	Editor func(path string) tea.Cmd
}

// App is the Bubble Tea root: it owns the navigation stack, the shared flow
// state and the frame.
type App struct {
	width  int
	height int
	theme  *theme.Theme
	frame  *Frame
	menu   Screen

	stack    []Screen
	current  Screen
	menuMode bool

	home            string
	catalog         catalog.Catalog
	applicationDirs []string
	pathValue       string
	cwd             string
	now             func() time.Time
	phrase          string
	version         string
	lastMission     string
	exitCode        int

	cloneOptions CloneFlowOptions
	cloneApps    []appChoice
	clonePlan    *clone.ClonePlan
	cloneResult  *clone.CloneResult
	cloneOutput  string
	cloneForce   bool
	cloneDryRun  bool
	cloneVerbose bool

	deployOptions DeployFlowOptions
	deploySession *deploy.DeploySession
	deployPlan    *deploy.DeployPlan
	deployResult  *deploy.DeployResult
	conflicts     []*conflictState
	conflictIndex int

	updateQuery func(context.Context) (update.Check, error)
	updateRun   func(context.Context, progress.Callback) (update.Result, error)
	editor      func(string) tea.Cmd

	settings     config.AppConfig
	settingsLoad config.LoadResult

	snapshotTimestamp string

	parent       context.Context
	cancelMu     sync.Mutex
	cancelOp     context.CancelFunc
	cancelSerial int
}

type startCloneMsg struct{}
type startDeployMsg struct{}
type startUpdateMsg struct{}
type startSettingsMsg struct{}
type startAuthoringMsg struct{}
type menuMsg struct{}

type operationStartedMsg struct {
	events <-chan operationEvent
}

type operationEvent struct {
	progress progress.Event
	clone    *clone.CloneResult
	deploy   *deploy.DeployResult
	plan     *deploy.DeployPlan
	update   *update.Result
	err      error
	done     bool
}

// NewApp builds the root model with injected dependencies.
func NewApp(options AppOptions) *App {
	// An unresolvable home leaves the field empty: every operation that needs it
	// propagates the resolution error instead of resolving against the cwd.
	home, _ := paths.ResolveHome(options.Home)
	parent := options.Context
	if parent == nil {
		parent = context.Background()
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	cwd := options.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	themeName := options.ThemeName
	if themeName == "" {
		themeName = os.Getenv("KAMINO_THEME")
	}
	noColor := options.NoColor || os.Getenv("NO_COLOR") != ""
	unicode := detectUnicode()
	if options.Unicode != nil {
		unicode = *options.Unicode
	}
	phrase := options.Phrase
	if phrase == "" && len(texts.Phrases) > 0 {
		phrase = texts.Phrases[int(now().UnixNano())%len(texts.Phrases)]
	}
	buildVersion := options.Version
	if buildVersion == "" {
		buildVersion = version.Version
	}
	currentTheme := theme.New(theme.Options{Name: themeName, NoColor: noColor, Unicode: unicode})
	app := &App{
		width: 80, height: 24, theme: currentTheme, frame: newFrame(currentTheme),
		home: home, catalog: options.Catalog, applicationDirs: options.ApplicationDirs,
		pathValue: options.PathValue, cwd: cwd, now: now, phrase: phrase, version: buildVersion,
		menuMode: options.MenuMode, cloneOptions: options.Clone, deployOptions: options.Deploy,
		cloneForce: options.Clone.Force, cloneDryRun: options.Clone.DryRun,
		cloneVerbose: options.Clone.Verbose, cloneOutput: options.Clone.Output,
		updateQuery: options.UpdateQuery, updateRun: options.UpdateRun,
		editor: options.Editor, snapshotTimestamp: now().UTC().Format("2006-01-02T15:04:05Z"), parent: parent,
	}
	if app.editor == nil {
		app.editor = defaultEditor
	}
	app.menu = newMenuScreen(app)
	app.current = app.menu
	app.layout()
	return app
}

// beginOperation creates the cancellable context of one operation and returns
// the function that releases it. The cancel function is stored so Esc and
// ctrl+c can stop the work from any screen.
func (a *App) beginOperation() (context.Context, func()) {
	ctx, cancel := context.WithCancel(a.parent)
	a.cancelMu.Lock()
	previous := a.cancelOp
	a.cancelSerial++
	serial := a.cancelSerial
	a.cancelOp = cancel
	a.cancelMu.Unlock()
	if previous != nil {
		previous()
	}
	return ctx, func() {
		a.cancelMu.Lock()
		if a.cancelSerial == serial {
			a.cancelOp = nil
		}
		a.cancelMu.Unlock()
		cancel()
	}
}

// cancelOperation asks the running operation to stop.
func (a *App) cancelOperation() {
	a.cancelMu.Lock()
	cancel := a.cancelOp
	a.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// cancelled reports whether err is a cooperative cancellation.
func cancelled(err error) bool {
	var target *kerrors.OperationCancelled
	return errors.As(err, &target)
}

func detectUnicode() bool {
	if os.Getenv("KAMINO_NO_LOGO") != "" {
		return false
	}
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LC_CTYPE")
	}
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	if locale == "" || locale == "C" || locale == "POSIX" {
		return false
	}
	lower := strings.ToLower(locale)
	return strings.Contains(lower, "utf-8") || strings.Contains(lower, "utf8")
}

// Init starts the current screen.
func (a *App) Init() tea.Cmd {
	if a.current == nil {
		return nil
	}
	return a.current.Init()
}

// Update routes messages through the frame and the current screen.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch value := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = value.Width, value.Height
		a.layout()
		return a, nil
	case pushMsg:
		a.push(value.screen)
		a.layout()
		return a, value.screen.Init()
	case popMsg:
		a.pop()
		a.layout()
		if a.current == nil {
			return a, tea.Quit
		}
		return a, nil
	case quitMsg:
		a.exitCode = value.code
		return a, tea.Quit
	case errorMsg:
		a.push(newErrorScreen(a, value.what, value.safety, value.action, value.detail))
		a.layout()
		return a, nil
	case startCloneMsg:
		a.startClone()
		return a, a.Init()
	case startDeployMsg:
		a.startDeploy()
		return a, a.Init()
	case startUpdateMsg:
		a.startUpdate()
		return a, a.Init()
	case startSettingsMsg:
		a.startSettings("")
		return a, a.Init()
	case startAuthoringMsg:
		a.startAuthoring()
		return a, a.Init()
	case menuMsg:
		a.resetToMenu()
		return a, nil
	case clipboardMsg:
		// The screen that requested the copy already shows the result.
		return a, nil
	}
	if keyMessage, ok := msg.(tea.KeyMsg); ok && a.current != nil {
		if _, isHelp := a.current.(*helpScreen); !isHelp {
			switch keyMessage.String() {
			case "?", "f1":
				a.openHelp()
				return a, nil
			}
		}
	}
	if a.current == nil {
		return a, nil
	}
	previous := a.current
	next, command := previous.Update(msg)
	if next != nil && a.current == previous {
		a.current = next
	}
	if a.current == nil {
		return a, tea.Quit
	}
	a.layout()
	return a, command
}

// View renders the current frame.
func (a *App) View() string {
	if a.current == nil {
		return ""
	}
	return a.frame.Render(a.current, a.width, a.height, a.operationText())
}

// ExitCode returns the code the program must exit with.
func (a *App) ExitCode() int { return a.exitCode }

func (a *App) layout() {
	if a.current == nil {
		return
	}
	messages := compactMessages(a.current.Messages())
	a.current.SetSize(BodyWidth(a.width), BodyHeight(a.height, len(messages)))
}

func (a *App) push(screen Screen) {
	if a.current != nil {
		a.stack = append(a.stack, a.current)
	}
	a.current = screen
}

func (a *App) pop() {
	if len(a.stack) == 0 {
		if a.menuMode {
			a.resetToMenu()
			return
		}
		a.exitCode = 1
		a.current = nil
		return
	}
	a.current = a.stack[len(a.stack)-1]
	a.stack = a.stack[:len(a.stack)-1]
}

func (a *App) resetToMenu() {
	a.stack = nil
	a.menu = newMenuScreen(a)
	a.current = a.menu
}

func (a *App) begin(screen Screen) {
	if a.menuMode {
		a.menu = newMenuScreen(a)
		a.stack = []Screen{a.menu}
	} else {
		a.stack = nil
	}
	a.current = screen
}

func (a *App) openHelp() {
	a.push(newHelpScreen(a, a.current))
}

func (a *App) operationText() string {
	if value, ok := a.current.(interface{ Operation() string }); ok {
		return value.Operation()
	}
	return ""
}

func (a *App) displayPath(value string) string {
	if value == "" {
		return ""
	}
	if value == a.home {
		return "~"
	}
	if strings.HasPrefix(value, a.home+string(os.PathSeparator)) {
		return "~/" + strings.TrimPrefix(value, a.home+string(os.PathSeparator))
	}
	return value
}

func (a *App) copyCommand(value, success, failure string) tea.Cmd {
	if value == "" {
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(value))
	return func() tea.Msg {
		_, err := os.Stdout.WriteString("\x1b]52;c;" + encoded + "\x07")
		if err != nil {
			return clipboardMsg{text: failure, kind: MessageError}
		}
		return clipboardMsg{text: success, kind: MessageInfo}
	}
}

type clipboardMsg struct {
	text string
	kind MessageKind
}

func (a *App) startMenu() {
	loaded := config.LoadConfig(a.home, "")
	a.menuMode = true
	if loaded.Error != nil {
		a.settingsLoad = loaded
		a.begin(newSettingsScreen(a, loaded.Error.Error()))
		return
	}
	a.resetToMenu()
}

func (a *App) startClone() {
	if !a.loadCatalog() {
		return
	}
	selected := map[string]bool{}
	if a.cloneOptions.Requested != nil {
		for _, id := range *a.cloneOptions.Requested {
			selected[id] = true
		}
	}
	apps := make([]appChoice, 0)
	for _, id := range a.catalog.SortedIDs() {
		definition := a.catalog[id]
		detected := detect.IsInstalled(definition, detect.Options{
			Home: a.home, ApplicationDirs: a.applicationDirs, Path: a.pathValue,
		})
		apps = append(apps, appChoice{ID: id, Name: definition.Name, Detected: detected, Selected: selected[id]})
	}
	a.cloneApps = apps
	a.cloneForce = a.cloneOptions.Force
	a.cloneDryRun = a.cloneOptions.DryRun
	a.cloneVerbose = a.cloneOptions.Verbose
	a.cloneResult = nil
	a.clonePlan = nil
	a.cloneOutput = a.cloneOptions.Output
	destinationError := a.cloneOptions.OutputError
	if a.cloneOutput == "" && destinationError == "" {
		loaded := config.LoadConfig(a.home, "")
		name, err := defaultCloneOutput(a.home, loaded.Config, a.now())
		if err != nil {
			destinationError = err.Error()
		}
		a.cloneOutput = name
	}
	appsScreen := newCloneAppsScreen(a, destinationError)
	destination := newCloneDestinationScreen(a, destinationError)
	if a.cloneOptions.Requested != nil && len(*a.cloneOptions.Requested) > 0 {
		if a.cloneOutput != "" && destinationError == "" {
			a.begin(newCloneReviewScreen(a))
			a.stack = append(a.stack, appsScreen, destination)
			return
		}
		a.begin(destination)
		a.stack = append(a.stack, appsScreen)
		return
	}
	a.begin(appsScreen)
}

func (a *App) startDeploy() {
	if !a.loadCatalog() {
		return
	}
	a.deploySession = nil
	a.deployPlan = nil
	a.deployResult = nil
	fileScreen := newDeployFileScreen(a, "")
	if a.deployOptions.FileError != "" {
		fileScreen.messages = []Message{{Kind: MessageError, Text: a.deployOptions.FileError}}
	}
	if a.deployOptions.CloneFile != "" && a.deployOptions.FileError == "" {
		a.begin(newDeployAppsScreen(a, a.deployOptions.CloneFile, true))
		a.stack = append(a.stack, fileScreen)
		return
	}
	a.begin(fileScreen)
}

func (a *App) startUpdate() {
	a.begin(newUpdateScreen(a))
}

func (a *App) startSettings(initialError string) {
	loaded := config.LoadConfig(a.home, "")
	a.settingsLoad = loaded
	a.settings = loaded.Config
	message := initialError
	if message == "" && loaded.Error != nil {
		message = loaded.Error.Error()
	}
	a.begin(newSettingsScreen(a, message))
}

func (a *App) startAuthoring() {
	directory, err := app.CatalogDir(a.home)
	if err != nil {
		a.push(newErrorScreen(a, "no se pudo crear el borrador", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error()))
		return
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		a.push(newErrorScreen(a, "no se pudo crear el borrador", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error()))
		return
	}
	file, err := os.CreateTemp(directory, ".kamino-draft-*.toml.tmp")
	if err != nil {
		a.push(newErrorScreen(a, "no se pudo crear el borrador", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error()))
		return
	}
	draft := file.Name()
	content, renderErr := catalog.RenderTemplate(catalog.TemplateOptions{Skeleton: true})
	if renderErr == nil {
		_, renderErr = file.WriteString(content)
	}
	if closeErr := file.Close(); renderErr == nil {
		renderErr = closeErr
	}
	if renderErr != nil {
		_ = os.Remove(draft)
		a.push(newErrorScreen(a, "no se pudo crear el borrador", texts.ErrorNothingWritten, texts.StatusReviewDefault, renderErr.Error()))
		return
	}
	a.push(newAppAuthoringScreen(a, directory, draft))
}

func (a *App) loadCatalog() bool {
	if a.catalog != nil {
		return true
	}
	directory, err := app.CatalogDir(a.home)
	if err != nil {
		a.push(newErrorScreen(a, "no se pudo cargar el catálogo", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error()))
		return false
	}
	loaded, err := catalog.LoadCatalog(directory)
	if err != nil {
		a.push(newErrorScreen(a, "no se pudo cargar el catálogo", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error()))
		return false
	}
	a.catalog = loaded
	return true
}

func (a *App) finishClone() {
	code := 0
	if a.cloneResult != nil && len(a.cloneResult.Plan.Skipped) > 0 {
		code = 3
	}
	mission := ""
	if a.cloneResult != nil {
		mission = fmt.Sprintf("clon: %d archivos", len(a.cloneResult.Plan.Operations))
	}
	a.finish(code, mission)
}

func (a *App) finishDeploy() {
	code := 0
	if a.deployResult != nil {
		code = deploy.ResultCode(*a.deployResult)
	}
	mission := ""
	if a.deployResult != nil {
		mission = fmt.Sprintf("implantación: %d implantados, %d híbridos", len(a.deployResult.Deployed), len(a.deployResult.Merged))
	}
	if a.deploySession != nil {
		a.deploySession.Close()
	}
	a.finish(code, mission)
}

func (a *App) finish(code int, mission string) {
	a.exitCode = code
	if !a.menuMode {
		a.current = nil
		return
	}
	a.stack = nil
	a.menu = newMenuScreen(a)
	a.current = a.menu
	if mission != "" {
		a.lastMission = mission
	}
}

func defaultCloneOutput(home string, value config.AppConfig, now time.Time) (string, error) {
	directory := value.CloneOutputDir
	if directory == "" {
		resolved, err := defaults.VaultDir(home)
		if err != nil {
			return "", err
		}
		directory = resolved
	}
	return filepath.Join(directory, defaultCloneName(now)), nil
}

func defaultCloneName(now time.Time) string {
	return "clone-" + now.Format("20060102") + ".tar.gz"
}

func (a *App) formatDate(value time.Time) string {
	return value.Format("2006-01-02 15:04")
}

func (a *App) snapshotPath() (string, error) {
	directory := ""
	if a.deploySession != nil {
		resolved, err := a.deploySession.SnapshotDirectory()
		if err != nil {
			return "", err
		}
		directory = resolved
	} else {
		resolved, err := defaults.SnapshotsDir(a.home)
		if err != nil {
			return "", err
		}
		directory = resolved
	}
	base := filepath.Join(directory, a.snapshotTimestamp)
	for index := 0; index <= 1000; index++ {
		candidate := base + ".tar.gz"
		if index > 0 {
			candidate = base + "-" + fmt.Sprintf("%d", index) + ".tar.gz"
		}
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return base + ".tar.gz", nil
}

func waitOperation(events <-chan operationEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return operationEvent{done: true}
		}
		return event
	}
}
