package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/detect"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/tui/theme"
	"github.com/Gunmer/kamino/internal/update"
)

// AppOptions injects every environment dependency so tests never touch the
// developer's HOME, clock, terminal or network.
type AppOptions struct {
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
	MenuMode        bool
	Clone           CloneFlowOptions
	Deploy          DeployFlowOptions
	UpdateQuery     func() (update.Check, error)
	UpdateRun       func(progress.Callback, progress.Cancel) (update.Result, error)
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
	cloneCancel  atomic.Bool

	deployOptions DeployFlowOptions
	deploySession *deploy.DeploySession
	deployPlan    *deploy.DeployPlan
	deployResult  *deploy.DeployResult
	deployCancel  atomic.Bool
	conflicts     []*conflictState
	conflictIndex int

	updateQuery  func() (update.Check, error)
	updateRun    func(progress.Callback, progress.Cancel) (update.Result, error)
	updateCancel atomic.Bool

	settings     config.AppConfig
	settingsLoad config.LoadResult

	snapshotTimestamp string
}

type startCloneMsg struct{}
type startDeployMsg struct{}
type startUpdateMsg struct{}
type startSettingsMsg struct{}
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
	home := options.Home
	if home == "" {
		home, _ = os.UserHomeDir()
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
		themeName = os.Getenv("TEXTUAL_THEME")
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
	currentTheme := theme.New(theme.Options{Name: themeName, NoColor: noColor, Unicode: unicode})
	app := &App{
		width: 80, height: 24, theme: currentTheme, frame: newFrame(currentTheme),
		home: home, catalog: options.Catalog, applicationDirs: options.ApplicationDirs,
		pathValue: options.PathValue, cwd: cwd, now: now, phrase: phrase,
		menuMode: options.MenuMode, cloneOptions: options.Clone, deployOptions: options.Deploy,
		cloneForce: options.Clone.Force, cloneDryRun: options.Clone.DryRun,
		cloneVerbose: options.Clone.Verbose, cloneOutput: options.Clone.Output,
		updateQuery: options.UpdateQuery, updateRun: options.UpdateRun,
		snapshotTimestamp: now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	app.menu = newMenuScreen(app)
	app.current = app.menu
	app.layout()
	return app
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
	case replaceMsg:
		a.replace(value.screen)
		a.layout()
		return a, value.screen.Init()
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

func (a *App) replace(screen Screen) {
	a.current = screen
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
	a.cloneCancel.Store(false)
	a.cloneOutput = a.cloneOptions.Output
	destinationError := a.cloneOptions.OutputError
	if a.cloneOutput == "" && destinationError == "" {
		loaded := config.LoadConfig(a.home, "")
		a.cloneOutput = defaultCloneOutput(a.home, loaded.Config, a.now())
	}
	if a.cloneOutput == "" {
		a.cloneOutput = defaultCloneOutput(a.home, config.AppConfig{}, a.now())
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
	a.deployCancel.Store(false)
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
	a.updateCancel.Store(false)
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

func (a *App) loadCatalog() bool {
	if a.catalog != nil {
		return true
	}
	loaded, err := catalog.LoadCatalog(defaults.CatalogDir(a.home))
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
		mission = fmt.Sprintf("despliegue: %d desplegados, %d híbridos", len(a.deployResult.Deployed), len(a.deployResult.Merged))
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

func defaultCloneOutput(home string, value config.AppConfig, now time.Time) string {
	directory := value.CloneOutputDir
	if directory == "" {
		directory = defaults.VaultDir(home)
	}
	return filepath.Join(directory, defaultCloneName(now))
}

func defaultCloneName(now time.Time) string {
	return "clone-" + now.Format("20060102") + ".tar.gz"
}

func (a *App) formatDate(value time.Time) string {
	return value.Format("2006-01-02 15:04")
}

func (a *App) snapshotPath() string {
	directory := defaults.SnapshotsDir(a.home)
	if a.deploySession != nil {
		directory = a.deploySession.SnapshotDirectory()
	}
	base := filepath.Join(directory, a.snapshotTimestamp)
	for index := 0; index <= 1000; index++ {
		candidate := base + ".tar.gz"
		if index > 0 {
			candidate = base + "-" + fmt.Sprintf("%d", index) + ".tar.gz"
		}
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return base + ".tar.gz"
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
