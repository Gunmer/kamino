package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type deployCandidate struct {
	Path   string
	Name   string
	Origin string
	Date   time.Time
	Size   int64
}

type deployFileScreen struct {
	screenBase
	candidates []deployCandidate
	index      int
	filter     field
	filtering  bool
	manual     bool
	manualPath field
	messages   []Message
}

func newDeployFileScreen(app *App, initialError string) *deployFileScreen {
	screen := &deployFileScreen{
		screenBase: screenBase{app: app},
		filter:     field{Label: texts.FieldFilter, Placeholder: texts.FieldFilterPlaceholder},
		manualPath: field{Label: texts.FieldPath, Placeholder: texts.FieldManualPlaceholder},
	}
	screen.candidates, screen.messages = discoverDeployCandidates(app.home, app.cwd)
	if initialError != "" {
		screen.messages = append(screen.messages, Message{Kind: MessageError, Text: initialError})
	}
	return screen
}

func discoverDeployCandidates(home, cwd string) ([]deployCandidate, []Message) {
	messages := make([]Message, 0)
	vault, err := defaults.VaultDir(home)
	if err != nil {
		messages = append(messages, Message{Kind: MessageError, Text: err.Error()})
	}
	sources := []struct {
		path   string
		origin string
	}{
		{vault, texts.DeployOriginDefault},
		{cwd, texts.DeployOriginCWD},
	}
	seen := map[string]bool{}
	candidates := make([]deployCandidate, 0)
	for _, source := range sources {
		if source.path == "" {
			continue
		}
		entries, err := os.ReadDir(source.path)
		if err != nil {
			if !os.IsNotExist(err) {
				messages = append(messages, Message{Kind: MessageWarning, Text: strings.NewReplacer(
					"{origin}", source.origin, "{path}", source.path, "{message}", err.Error(),
				).Replace(texts.DeploySourceUnreadable)})
			}
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar.gz") {
				continue
			}
			path := filepath.Join(source.path, entry.Name())
			absolute, err := filepath.Abs(path)
			if err == nil {
				path = absolute
			}
			if seen[path] {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			seen[path] = true
			candidates = append(candidates, deployCandidate{
				Path: path, Name: entry.Name(), Origin: source.origin,
				Date: info.ModTime(), Size: info.Size(),
			})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].Date.Equal(candidates[right].Date) {
			return candidates[left].Path < candidates[right].Path
		}
		return candidates[left].Date.After(candidates[right].Date)
	})
	return candidates, messages
}

func (s *deployFileScreen) Flow() string { return texts.FlowDeploy }

func (s *deployFileScreen) Title() string { return texts.TitleClon }

func (s *deployFileScreen) Step() (int, int) { return 1, 4 }

func (s *deployFileScreen) Messages() []Message { return s.messages }

func (s *deployFileScreen) Keys() []key.Binding {
	length := 0
	if s.manual {
		length = len([]rune(s.manualPath.Value))
	} else if s.filtering {
		length = len([]rune(s.filter.Value))
	}
	return deployFileKeys(s.manual, s.filtering, length)
}

func (s *deployFileScreen) visible() []deployCandidate {
	if s.filter.Value == "" {
		return s.candidates
	}
	result := make([]deployCandidate, 0)
	query := strings.ToLower(s.filter.Value)
	for _, candidate := range s.candidates {
		if strings.Contains(strings.ToLower(candidate.Path), query) || strings.Contains(strings.ToLower(candidate.Origin), query) {
			result = append(result, candidate)
		}
	}
	return result
}

func (s *deployFileScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if s.manual {
		switch keyMessage.String() {
		case "esc":
			s.manual = false
			s.manualPath.SetValue("")
		case "enter":
			path := strings.TrimSpace(s.manualPath.Value)
			if path == "" {
				s.manualPath.Invalid = true
				s.messages = []Message{{Kind: MessageError, Text: texts.DeployNoFile}}
				return s, nil
			}
			if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
				s.manualPath.Invalid = true
				s.messages = []Message{{Kind: MessageError, Text: strings.ReplaceAll(texts.D0ManualError, "{path}", path)}}
				return s, nil
			}
			return s, pushScreen(newDeployAppsScreen(s.app, path, true))
		case "left":
			s.manualPath.MoveCursor(-1)
		case "right":
			s.manualPath.MoveCursor(1)
		case "backspace":
			s.manualPath.Backspace()
			s.clearManualError()
		default:
			if len(keyMessage.Runes) > 0 {
				s.manualPath.Insert(string(keyMessage.Runes))
				s.clearManualError()
			}
		}
		return s, nil
	}
	if s.filtering {
		switch keyMessage.String() {
		case "esc":
			s.filtering = false
			s.filter.SetValue("")
		case "enter":
			s.filtering = false
		case "left":
			s.filter.MoveCursor(-1)
		case "right":
			s.filter.MoveCursor(1)
		case "backspace":
			s.filter.Backspace()
		default:
			if len(keyMessage.Runes) > 0 {
				s.filter.Insert(string(keyMessage.Runes))
			}
		}
		return s, nil
	}
	visible := s.visible()
	switch keyMessage.String() {
	case "up", "k":
		if len(visible) > 0 {
			s.index = (s.index + len(visible) - 1) % len(visible)
		}
	case "down", "j":
		if len(visible) > 0 {
			s.index = (s.index + 1) % len(visible)
		}
	case "/":
		s.filtering = true
		s.filter.Cursor = len([]rune(s.filter.Value))
	case "o":
		s.manual = true
		s.manualPath.SetValue("")
		s.manualPath.Invalid = false
	case "enter":
		if len(visible) == 0 {
			s.messages = []Message{{Kind: MessageWarning, Text: texts.DeployNoFile}}
			return s, nil
		}
		return s, pushScreen(newDeployAppsScreen(s.app, visible[s.index%len(visible)].Path, true))
	case "esc":
		return s, popScreen()
	}
	return s, nil
}

func (s *deployFileScreen) clearManualError() {
	s.manualPath.Invalid = false
	kept := make([]Message, 0, len(s.messages))
	for _, message := range s.messages {
		if message.Kind != MessageError {
			kept = append(kept, message)
		}
	}
	s.messages = kept
}

func (s *deployFileScreen) View(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.D0Title)}
	if s.manual {
		options := fieldOptions{
			Width:      max(1, width-2),
			LabelWidth: fieldLabelWidth(s.app.theme, texts.FieldPath),
			Focused:    true,
		}
		lines = append(lines, "", "  "+s.manualPath.Render(s.app.theme, options))
		return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
	}
	if s.filter.Value != "" || s.filtering {
		options := fieldOptions{
			Width:      max(1, width-2),
			LabelWidth: fieldLabelWidth(s.app.theme, texts.FieldFilter),
			Focused:    s.filtering,
		}
		lines = append(lines, "  "+s.filter.Render(s.app.theme, options))
	}
	lines = append(lines, "")
	tableWidth := width
	if tableWidth > 64 {
		tableWidth = 64
	}
	fileWidth := tableWidth - 34
	if fileWidth < 12 {
		fileWidth = 12
	}
	lines = append(lines, "  "+PadRight(texts.D0ColumnFile, fileWidth)+"  "+PadRight(texts.D0ColumnOrigin, 16)+"  "+PadLeft(texts.D0ColumnSize, 9))
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, tableWidth-4))))
	visible := s.visible()
	maxRows := height - 9
	if maxRows < 3 {
		maxRows = 3
	}
	start, end := window(s.index, len(visible), maxRows)
	if start > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D0ScrollUp, "{count}", fmt.Sprintf("%d", start)))
	}
	for index := start; index < end; index++ {
		candidate := visible[index]
		cursor := "  "
		style := s.app.theme.Styles.Secondary
		if index == s.index {
			cursor = s.app.theme.Symbols.Cursor + " "
			style = s.app.theme.Styles.Primary
		}
		line := cursor + PadRight(Truncate(candidate.Name, fileWidth, s.app.theme.Symbols.Ellipsis), fileWidth) + "  " + PadRight(candidate.Origin, 16) + "  " + PadLeft(texts.SizeLabel(candidate.Size), 9)
		lines = append(lines, style.Render(line))
	}
	if end < len(visible) {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D0ScrollDown, "{count}", fmt.Sprintf("%d", len(visible)-end)))
	}
	if len(visible) == 0 {
		lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(texts.D0NoClones))
	}
	if len(visible) > 0 {
		candidate := visible[s.index%len(visible)]
		lines = append(lines, "",
			"  "+strings.ReplaceAll(texts.D0Preview, "{path}", s.app.displayPath(candidate.Path)),
			"  "+strings.NewReplacer("{date}", s.app.formatDate(candidate.Date), "{size}", texts.SizeLabel(candidate.Size)).Replace(texts.D0PreviewDate),
		)
	}
	lines = append(lines, "", "  "+texts.D0OtherFile)
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

type deployPreparedMsg struct {
	session *deploy.DeploySession
	err     error
}

type deployAppsScreen struct {
	screenBase
	cloneFile     string
	preparing     bool
	planning      bool
	skipSelection bool
	index         int
	apps          []appChoice
	messages      []Message
	current       int
	total         int
	events        <-chan operationEvent
	plan          *deploy.DeployPlan
	// installing shows the install question before the plan; pendingSelected
	// keeps the apps the user chose to deploy meanwhile.
	installing      bool
	installChoices  []installChoice
	installIndex    int
	pendingSelected []string
}

func newDeployAppsScreen(app *App, cloneFile string, preparing bool) *deployAppsScreen {
	screen := &deployAppsScreen{
		screenBase: screenBase{app: app}, cloneFile: cloneFile, preparing: preparing,
		skipSelection: app.deployOptions.Requested != nil && len(*app.deployOptions.Requested) > 0,
	}
	return screen
}

func (s *deployAppsScreen) Flow() string { return texts.FlowDeploy }

func (s *deployAppsScreen) Title() string { return texts.TitleApps }

func (s *deployAppsScreen) Step() (int, int) { return 2, 4 }

func (s *deployAppsScreen) Messages() []Message { return s.messages }

func (s *deployAppsScreen) Keys() []key.Binding {
	if s.installing {
		return deployInstallKeys()
	}
	return deployAppsKeys(s.preparing || s.planning)
}

func (s *deployAppsScreen) Operation() string {
	if s.preparing {
		return texts.OperationDeployPreparing
	}
	if s.planning {
		if s.total > 0 {
			return strings.NewReplacer("{current}", fmt.Sprintf("%d", s.current), "{total}", fmt.Sprintf("%d", s.total)).Replace(texts.OperationDeployProgress)
		}
		return texts.OperationDeployPlanning
	}
	return ""
}

func (s *deployAppsScreen) Init() tea.Cmd {
	if s.preparing {
		return prepareDeployCmd(s.app, s.cloneFile)
	}
	return nil
}

func prepareDeployCmd(app *App, cloneFile string) tea.Cmd {
	loaded := config.LoadConfig(app.home, "")
	options := deploy.Options{
		Home: app.home, Catalog: app.catalog, ApplicationDirs: app.applicationDirs,
		Path: app.pathValue, SnapshotsDir: loaded.Config.SnapshotsDir, SnapshotsLimit: loaded.Config.SnapshotsLimit,
	}
	return func() tea.Msg {
		session, err := deploy.PrepareDeploy(cloneFile, options)
		return deployPreparedMsg{session: session, err: err}
	}
}

func (s *deployAppsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch value := msg.(type) {
	case deployPreparedMsg:
		s.preparing = false
		if value.err != nil {
			return s, showError(texts.ErrorInvalidClone, texts.ErrorNothingWritten, texts.TitleRetryAdvice, value.err.Error())
		}
		s.app.deploySession = value.session
		requested := map[string]bool{}
		if s.app.deployOptions.Requested != nil {
			for _, id := range *s.app.deployOptions.Requested {
				requested[id] = true
			}
		}
		s.apps = make([]appChoice, 0, len(value.session.DeployableApps))
		for _, item := range value.session.DeployableApps {
			s.apps = append(s.apps, appChoice{
				ID: item.ID, Name: item.Name, Detected: true, Selected: requested[item.ID],
				Count: len(item.Files),
			})
		}
		s.messages = append(s.messages, Message{Kind: MessageWarning, Text: texts.D1CloseApps})
		for _, warning := range value.session.Warnings {
			s.messages = append(s.messages, Message{Kind: MessageWarning, Text: warning})
		}
		for _, notice := range value.session.Notices {
			s.messages = append(s.messages, Message{Kind: MessageInfo, Text: notice})
		}
		if s.skipSelection {
			selected := make([]string, 0, len(s.apps))
			for _, choice := range s.apps {
				if choice.Selected {
					selected = append(selected, choice.ID)
				}
			}
			s.planning = true
			s.messages = []Message{{Kind: MessageWarning, Text: texts.D1Verifying}}
			return s, startDeployPlan(s.app, s.app.deploySession, selected)
		}
		return s, nil
	case operationStartedMsg:
		s.events = value.events
		return s, waitOperation(value.events)
	case operationEvent:
		return s.handlePlanEvent(value)
	}
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if s.preparing || s.planning {
		switch keyMessage.String() {
		case "esc":
			if s.planning {
				s.app.cancelOperation()
				s.messages = []Message{{Kind: MessageWarning, Text: texts.D1CancelRunning}}
				return s, nil
			}
			return s, popScreen()
		case "ctrl+c":
			if s.planning {
				s.app.cancelOperation()
				s.messages = []Message{{Kind: MessageWarning, Text: texts.D1CancelRunning}}
			} else {
				return s, popScreen()
			}
		}
		return s, nil
	}
	if s.installing {
		return s.updateInstall(keyMessage)
	}
	switch keyMessage.String() {
	case "up", "k":
		if len(s.apps) > 0 {
			s.index = (s.index + len(s.apps) - 1) % len(s.apps)
		}
	case "down", "j":
		if len(s.apps) > 0 {
			s.index = (s.index + 1) % len(s.apps)
		}
	case " ":
		if len(s.apps) > 0 {
			s.apps[s.index%len(s.apps)].Selected = !s.apps[s.index%len(s.apps)].Selected
		}
	case "a":
		for index := range s.apps {
			s.apps[index].Selected = true
		}
	case "n":
		for index := range s.apps {
			s.apps[index].Selected = false
		}
	case "esc":
		return s, popScreen()
	case "enter":
		selected := make([]string, 0)
		for _, choice := range s.apps {
			if choice.Selected {
				selected = append(selected, choice.ID)
			}
		}
		if len(selected) == 0 {
			s.messages = []Message{{Kind: MessageWarning, Text: texts.D1NoneSelected}}
			return s, nil
		}
		if s.app.deploySession == nil {
			return s, nil
		}
		if s.beginInstallQuestion(selected) {
			return s, nil
		}
		s.planning = true
		s.current = 0
		s.total = 0
		s.messages = []Message{{Kind: MessageWarning, Text: texts.D1Verifying}}
		return s, startDeployPlan(s.app, s.app.deploySession, selected)
	}
	return s, nil
}

func (s *deployAppsScreen) handlePlanEvent(event operationEvent) (Screen, tea.Cmd) {
	if event.progress.Total > 0 {
		s.current = event.progress.Current
		s.total = event.progress.Total
	}
	if event.plan != nil {
		s.plan = event.plan
	}
	if !event.done {
		if s.events != nil {
			return s, waitOperation(s.events)
		}
		return s, nil
	}
	s.planning = false
	if event.err != nil {
		if cancelled(event.err) {
			s.messages = []Message{{Kind: MessageWarning, Text: texts.StatusPreparingCancelled}}
			return s, nil
		}
		return s, showError(texts.ErrorPreparingDeploy, texts.ErrorNothingWritten, texts.StatusReviewDefault, event.err.Error())
	}
	if s.plan == nil {
		return s, showError(texts.ErrorPreparingDeploy, texts.ErrorNothingWritten, texts.StatusReviewDefault, texts.ErrorEmptyPlan)
	}
	s.app.deployPlan = s.plan
	s.app.conflicts = newConflictStates(s.plan)
	s.app.conflictIndex = 0
	return s, pushScreen(newDeployConflictsScreen(s.app))
}

func (s *deployAppsScreen) View(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.D1Title)}
	if s.preparing {
		lines = append(lines, "", "  "+texts.D1Preparing, "  "+texts.D1Verifying)
		return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
	}
	if s.installing {
		return s.viewInstall(width, height)
	}
	if s.planning {
		value := texts.D1Verifying
		if s.total > 0 {
			value = fmt.Sprintf("Verificando y clasificando · %d de %d archivos", s.current, s.total)
		}
		lines = append(lines, "", "  "+value, "  "+progressBar(s.current, max(1, s.total), max(8, width-8)))
		return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
	}
	lines = append(lines, "")
	widthTable := width
	if widthTable > 60 {
		widthTable = 60
	}
	appWidth := widthTable - 17
	if appWidth < 10 {
		appWidth = 10
	}
	lines = append(lines, "  "+PadRight(texts.D1ColumnSelect, 3)+"  "+PadRight(texts.D1ColumnApp, appWidth)+"  "+PadLeft(texts.D1ColumnFiles, 8))
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, widthTable-4))))
	maxRows := height - 7
	if maxRows < 3 {
		maxRows = 3
	}
	start, end := window(s.index, len(s.apps), maxRows)
	if start > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1ScrollUp, "{count}", fmt.Sprintf("%d", start)))
	}
	for index := start; index < end; index++ {
		choice := s.apps[index]
		cursor := "  "
		style := s.app.theme.Styles.Secondary
		if index == s.index {
			cursor = s.app.theme.Symbols.Cursor + " "
			style = s.app.theme.Styles.Primary
		}
		checkbox := "[ ]"
		if choice.Selected {
			checkbox = "[x]"
		}
		line := cursor + checkbox + "  " + PadRight(Truncate(choice.Name, appWidth, s.app.theme.Symbols.Ellipsis), appWidth) + "  " + PadLeft(fmt.Sprintf("%d", choice.Count), 8)
		lines = append(lines, style.Render(line))
	}
	if end < len(s.apps) {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1ScrollDown, "{count}", fmt.Sprintf("%d", len(s.apps)-end)))
	}
	if len(s.apps) == 0 {
		lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(texts.D1NoApps))
	}
	var notInstalled []string
	if s.app.deploySession != nil {
		for _, item := range s.app.deploySession.NotInstalledApps {
			notInstalled = append(notInstalled, item.Name)
		}
	}
	lines = append(lines, "")
	if len(notInstalled) > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D1NoBase, "{apps}", strings.Join(notInstalled, ", ")))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}
