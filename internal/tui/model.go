package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/detect"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
)

// Screen identifies a TUI screen.
type Screen string

const (
	ScreenMenu     Screen = "menu"
	ScreenC1       Screen = "c1"
	ScreenC2       Screen = "c2"
	ScreenC3       Screen = "c3"
	ScreenC4       Screen = "c4"
	ScreenD0       Screen = "d0"
	ScreenD1       Screen = "d1"
	ScreenD2       Screen = "d2"
	ScreenD3       Screen = "d3"
	ScreenD4       Screen = "d4"
	ScreenD5       Screen = "d5"
	ScreenU1       Screen = "u1"
	ScreenSettings Screen = "settings"
	ScreenHelp     Screen = "help"
	ScreenError    Screen = "error"
)

type appChoice struct {
	ID       string
	Name     string
	Detected bool
	Selected bool
	Count    int
}

type conflictState struct {
	Conflict  conflicts.Conflict
	Action    conflicts.FileAction
	Rows      []conflicts.RowDecision
	HunkIndex int
	RowIndex  int
}

// UIError is the four-part visible error.
type UIError struct {
	What   string
	Safety string
	Action string
	Detail string
}

// Model is the complete interactive state.
type Model struct {
	Screen          Screen
	Stack           []Screen
	Width           int
	Height          int
	Theme           string
	NoColor         bool
	Home            string
	Catalog         catalog.Catalog
	ApplicationDirs []string
	PathValue       string

	MenuIndex   int
	MenuMode    bool
	LastMission string
	Phrase      string

	CloneRequested       *[]string
	CloneOutput          string
	CloneSkipDestination bool
	CloneForce           bool
	CloneDryRun          bool
	CloneVerbose         bool
	CloneOutputError     string
	CloneApps            []appChoice
	CloneIndex           int
	CloneFilter          string
	CloneFiltering       bool
	ClonePlan            *clone.ClonePlan
	CloneResult          *clone.CloneResult
	CloneError           string
	CloneShowSkipped     bool
	CloneShowFiles       bool
	CloneWriting         bool
	ReplaceDialog        bool
	ReplaceIndex         int

	DeployFile         string
	DeployRequested    *[]string
	DeployVerbose      bool
	DeployError        string
	DeployFiles        []string
	DeployIndex        int
	DeployApps         []appChoice
	DeploySession      *deploy.DeploySession
	DeployPlan         *deploy.DeployPlan
	Conflicts          []conflictState
	ConflictIndex      int
	ConfirmIndex       int
	DeployResult       *deploy.DeployResult
	DeployManual       bool
	DeployFilter       string
	DeployFiltering    bool
	DeployPreparing    bool
	DeployCannotCancel bool

	UpdateResult  *update.Result
	UpdateRelease *update.ReleaseInfo
	UpdateCheck   update.Check
	UpdateError   string

	Settings       config.AppConfig
	SettingsFields [3]string
	SettingsIndex  int
	SettingsError  string
	SettingsLoaded config.LoadResult

	HelpReturn  Screen
	Error       *UIError
	ErrorReturn Screen
	ErrorDetail bool
	Message     string
	Clipboard   string
	Quit        bool
	ExitCode    int

	UpdateCheckFunc func() (update.Result, error)
	Now             func() time.Time
}

// NewModel builds an initial model.
func NewModel(options CloneFlowOptions) *Model {
	home, _ := os.UserHomeDir()
	theme := os.Getenv("TEXTUAL_THEME")
	if theme != "ansi-light" {
		theme = "ansi-dark"
	}
	phrases := texts.Phrases
	phrase := ""
	if len(phrases) > 0 {
		phrase = phrases[int(time.Now().UnixNano())%len(phrases)]
	}
	model := &Model{
		Screen: ScreenMenu, Width: 80, Height: 24, Theme: theme,
		Home: home, NoColor: os.Getenv("NO_COLOR") != "", Phrase: phrase,
		Now: time.Now,
	}
	model.CloneRequested = options.Requested
	model.CloneOutput = options.Output
	model.CloneSkipDestination = options.Output != ""
	model.CloneForce = options.Force
	model.CloneDryRun = options.DryRun
	model.CloneVerbose = options.Verbose
	model.CloneOutputError = options.OutputError
	return model
}

// Update applies one key press.
func (m *Model) Update(key Key) {
	if m.Screen == ScreenHelp {
		if key.Kind == KeyEsc || key.Kind == KeyEnter || key.Kind == KeyF1 || key.Kind == KeyRune && key.Rune == '?' {
			m.Screen = m.HelpReturn
			m.Stack = popScreen(m.Stack)
		}
		return
	}
	if m.Width < 80 || m.Height < 24 {
		if key.Kind == KeyRune && key.Rune == '?' {
			m.openHelp()
		}
		return
	}
	if m.Error != nil {
		switch {
		case key.Kind == KeyEsc:
			m.Error = nil
			m.Screen = m.ErrorReturn
		case key.Kind == KeyRune && key.Rune == 't':
			m.ErrorDetail = !m.ErrorDetail
		case key.Kind == KeyRune && key.Rune == 'c':
			m.Clipboard = m.Error.Detail
			m.Message = texts.ErrorCopied
		}
		return
	}
	if key.Kind == KeyF1 || (key.Kind == KeyRune && key.Rune == '?') {
		m.openHelp()
		return
	}
	switch m.Screen {
	case ScreenMenu:
		m.updateMenu(key)
	case ScreenC1:
		m.updateCloneApps(key)
	case ScreenC2:
		m.updateCloneDestination(key)
	case ScreenC3:
		m.updateCloneReview(key)
	case ScreenC4:
		m.updateCloneSummary(key)
	case ScreenD0:
		m.updateDeployFile(key)
	case ScreenD1:
		m.updateDeployApps(key)
	case ScreenD2:
		m.updateDeployConflicts(key)
	case ScreenD3:
		m.updateDeployDetail(key)
	case ScreenD4:
		m.updateDeployConfirm(key)
	case ScreenD5:
		m.updateDeploySummary(key)
	case ScreenU1:
		m.updateUpdate(key)
	case ScreenSettings:
		m.updateSettings(key)
	}
}

func (m *Model) push(next Screen) {
	m.Stack = append(m.Stack, m.Screen)
	m.Screen = next
}

func popScreen(stack []Screen) []Screen {
	if len(stack) == 0 {
		return stack
	}
	return stack[:len(stack)-1]
}

func (m *Model) back() {
	if len(m.Stack) == 0 {
		m.Screen = ScreenMenu
		return
	}
	m.Screen = m.Stack[len(m.Stack)-1]
	m.Stack = m.Stack[:len(m.Stack)-1]
}

func (m *Model) openHelp() {
	m.HelpReturn = m.Screen
	m.Screen = ScreenHelp
}

func (m *Model) showError(what, safety, action, detail string) {
	m.Error = &UIError{What: what, Safety: safety, Action: action, Detail: detail}
	m.ErrorReturn = m.Screen
	m.Screen = ScreenError
	m.ErrorDetail = false
}

func (m *Model) updateMenu(key Key) {
	options := 5
	switch key.Kind {
	case KeyUp:
		m.MenuIndex = (m.MenuIndex + options - 1) % options
	case KeyDown:
		m.MenuIndex = (m.MenuIndex + 1) % options
	case KeyEnter:
		switch m.MenuIndex {
		case 0:
			m.startClone()
		case 1:
			m.startDeploy()
		case 2:
			m.startUpdate()
		case 3:
			m.startSettings()
		case 4:
			m.Quit = true
		}
	}
}

func (m *Model) startClone() {
	m.loadCatalog()
	m.CloneApps = make([]appChoice, 0)
	selected := map[string]bool{}
	if m.CloneRequested != nil {
		for _, id := range *m.CloneRequested {
			selected[id] = true
		}
	}
	for _, id := range m.Catalog.SortedIDs() {
		definition := m.Catalog[id]
		detected := detectInstalled(definition, m.Home, m.ApplicationDirs, m.PathValue)
		count := 0
		if detected {
			if counted, err := plan.CountFiles([]*catalog.AppDefinition{definition}, m.Home, nil); err == nil {
				count = counted.Counts[id]
			}
		}
		choice := appChoice{ID: id, Name: definition.Name, Detected: detected, Selected: selected[id], Count: count}
		m.CloneApps = append(m.CloneApps, choice)
	}
	if m.CloneOutput == "" {
		loaded := config.LoadConfig(m.Home, "")
		if loaded.Config.CloneOutputDir != "" {
			m.CloneOutput = filepath.Join(loaded.Config.CloneOutputDir, clone.DefaultName(m.Now()))
		}
	}
	m.Screen = ScreenC1
	m.Stack = nil
	if m.CloneRequested != nil && len(*m.CloneRequested) > 0 {
		if m.CloneSkipDestination && m.CloneOutput != "" && m.CloneOutputError == "" {
			m.planClone()
		} else {
			m.Screen = ScreenC2
		}
	}
}

func (m *Model) loadCatalog() {
	if m.Catalog != nil {
		return
	}
	loaded, err := catalog.LoadCatalog(defaults.CatalogDir(m.Home))
	if err != nil {
		m.showError("no se pudo cargar el catálogo", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error())
		return
	}
	m.Catalog = loaded
}

func (m *Model) startDeploy() {
	m.loadCatalog()
	m.DeploySession = nil
	m.DeployResult = nil
	m.DeployError = ""
	m.DeployFiles = discoverClones(m.Home)
	m.DeployIndex = 0
	m.Screen = ScreenD0
	m.Stack = nil
	if m.DeployFile != "" {
		if info, err := os.Stat(m.DeployFile); err == nil && info.Mode().IsRegular() {
			m.openDeployFile(m.DeployFile)
			return
		}
		m.DeployError = "no existe el archivo: " + m.DeployFile
	}
}

func discoverClones(home string) []string {
	directories := []string{
		defaults.VaultDir(home),
		".",
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
	}
	seen := map[string]bool{}
	result := make([]string, 0)
	for _, directory := range directories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar.gz") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			if absolute, err := filepath.Abs(path); err == nil {
				path = absolute
			}
			if !seen[path] {
				seen[path] = true
				result = append(result, path)
			}
		}
	}
	sort.Strings(result)
	return result
}

func (m *Model) openDeployFile(path string) {
	m.DeployFile = path
	m.DeployError = ""
	session, err := deploy.PrepareDeploy(path, deploy.Options{Home: m.Home, Catalog: m.Catalog, ApplicationDirs: m.ApplicationDirs, Path: m.PathValue})
	if err != nil {
		m.showError("el archivo no es un clon válido", texts.ErrorNothingWritten, texts.TitleRetryAdvice, err.Error())
		return
	}
	m.DeploySession = session
	m.DeployApps = make([]appChoice, 0)
	selected := map[string]bool{}
	if m.DeployRequested != nil {
		for _, id := range *m.DeployRequested {
			selected[id] = true
		}
	}
	for _, item := range session.DeployableApps {
		m.DeployApps = append(m.DeployApps, appChoice{ID: item.ID, Name: item.Name, Detected: true, Selected: selected[item.ID] || m.DeployRequested == nil})
	}
	m.Screen = ScreenD1
}

func (m *Model) startUpdate() {
	m.Screen = ScreenU1
	m.UpdateError = ""
	m.UpdateResult = nil
	if m.UpdateCheckFunc != nil {
		result, err := m.UpdateCheckFunc()
		if err != nil {
			m.UpdateError = err.Error()
		} else {
			m.UpdateResult = &result
		}
	} else {
		m.startRealUpdate()
	}
}

func (m *Model) startSettings() {
	m.Screen = ScreenSettings
	m.SettingsIndex = 0
	loaded := config.LoadConfig(m.Home, "")
	m.SettingsLoaded = loaded
	m.Settings = loaded.Config
	m.SettingsError = ""
	if loaded.Error != nil {
		m.SettingsError = loaded.Error.Error()
	}
	m.SettingsFields = [3]string{
		displayPath(loaded.Config.CatalogDir, m.Home),
		displayPath(loaded.Config.CloneOutputDir, m.Home),
		displayPath(loaded.Config.SnapshotsDir, m.Home),
	}
	if loaded.Exists {
		if value, ok := loaded.Raw[config.CatalogKey]; ok {
			m.SettingsFields[0] = value
		}
		if value, ok := loaded.Raw[config.CloneKey]; ok {
			m.SettingsFields[1] = value
		}
		if value, ok := loaded.Raw[config.SnapshotsKey]; ok {
			m.SettingsFields[2] = value
		}
	}
}

func displayPath(value, home string) string {
	if strings.HasPrefix(value, home+string(filepath.Separator)) {
		return "~/" + filepath.ToSlash(strings.TrimPrefix(value, home+string(filepath.Separator)))
	}
	return value
}

func detectInstalled(definition *catalog.AppDefinition, home string, applicationDirs []string, pathValue string) bool {
	return detect.IsInstalled(definition, detect.Options{Home: home, ApplicationDirs: applicationDirs, Path: pathValue})
}
