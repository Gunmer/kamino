package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
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
	filter     string
	filtering  bool
	manual     bool
	manualPath string
	messages   []Message
}

func newDeployFileScreen(app *App, initialError string) *deployFileScreen {
	screen := &deployFileScreen{screenBase: screenBase{app: app}}
	screen.candidates, screen.messages = discoverDeployCandidates(app.home, app.cwd)
	if initialError != "" {
		screen.messages = append(screen.messages, Message{Kind: MessageError, Text: initialError})
	}
	return screen
}

func discoverDeployCandidates(home, cwd string) ([]deployCandidate, []Message) {
	sources := []struct {
		path   string
		origin string
	}{
		{defaults.VaultDir(home), texts.DeployOriginDefault},
		{cwd, texts.DeployOriginCWD},
	}
	seen := map[string]bool{}
	candidates := make([]deployCandidate, 0)
	messages := make([]Message, 0)
	for _, source := range sources {
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

func (s *deployFileScreen) Flow() string        { return texts.FlowDeploy }
func (s *deployFileScreen) Title() string       { return texts.TitleClon }
func (s *deployFileScreen) Step() (int, int)    { return 1, 4 }
func (s *deployFileScreen) Messages() []Message { return s.messages }
func (s *deployFileScreen) Keys() []key.Binding { return deployFileKeys(s.manual, s.filtering) }

func (s *deployFileScreen) visible() []deployCandidate {
	if s.filter == "" {
		return s.candidates
	}
	result := make([]deployCandidate, 0)
	query := strings.ToLower(s.filter)
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
			s.manualPath = ""
		case "enter":
			if strings.TrimSpace(s.manualPath) == "" {
				s.messages = []Message{{Kind: MessageError, Text: texts.DeployNoFile}}
				return s, nil
			}
			if info, err := os.Stat(s.manualPath); err != nil || !info.Mode().IsRegular() {
				s.messages = []Message{{Kind: MessageError, Text: strings.ReplaceAll(texts.D0ManualError, "{path}", s.manualPath)}}
				return s, nil
			}
			return s, pushScreen(newDeployAppsScreen(s.app, s.manualPath, true))
		case "backspace":
			runes := []rune(s.manualPath)
			if len(runes) > 0 {
				s.manualPath = string(runes[:len(runes)-1])
			}
		default:
			if len(keyMessage.Runes) > 0 {
				s.manualPath += string(keyMessage.Runes)
			}
		}
		return s, nil
	}
	if s.filtering {
		switch keyMessage.String() {
		case "esc":
			s.filtering = false
			s.filter = ""
		case "enter":
			s.filtering = false
		case "backspace":
			runes := []rune(s.filter)
			if len(runes) > 0 {
				s.filter = string(runes[:len(runes)-1])
			}
		default:
			if len(keyMessage.Runes) > 0 {
				s.filter += string(keyMessage.Runes)
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
	case "o":
		s.manual = true
		s.manualPath = ""
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

func (s *deployFileScreen) View(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.D0Title)}
	if s.manual {
		lines = append(lines, "", "  "+texts.D0ManualHint, "  "+s.manualPath+"_")
		return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
	}
	if s.filter != "" || s.filtering {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1FilterLine, "{value}", s.filter))
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
}

func newDeployAppsScreen(app *App, cloneFile string, preparing bool) *deployAppsScreen {
	screen := &deployAppsScreen{
		screenBase: screenBase{app: app}, cloneFile: cloneFile, preparing: preparing,
		skipSelection: app.deployOptions.Requested != nil && len(*app.deployOptions.Requested) > 0,
	}
	return screen
}

func (s *deployAppsScreen) Flow() string        { return texts.FlowDeploy }
func (s *deployAppsScreen) Title() string       { return texts.TitleApps }
func (s *deployAppsScreen) Step() (int, int)    { return 2, 4 }
func (s *deployAppsScreen) Messages() []Message { return s.messages }
func (s *deployAppsScreen) Keys() []key.Binding { return deployAppsKeys(s.preparing || s.planning) }

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
		Path: app.pathValue, SnapshotsDir: loaded.Config.SnapshotsDir,
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
				s.app.deployCancel.Store(true)
				s.messages = []Message{{Kind: MessageWarning, Text: texts.D1CancelRunning}}
				return s, nil
			}
			return s, popScreen()
		case "ctrl+c":
			if s.planning {
				s.app.deployCancel.Store(true)
				s.messages = []Message{{Kind: MessageWarning, Text: texts.D1CancelRunning}}
			} else {
				return s, popScreen()
			}
		}
		return s, nil
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
		s.planning = true
		s.current = 0
		s.total = 0
		s.messages = []Message{{Kind: MessageWarning, Text: texts.D1Verifying}}
		return s, startDeployPlan(s.app, s.app.deploySession, selected)
	}
	return s, nil
}

func startDeployPlan(app *App, session *deploy.DeploySession, selected []string) tea.Cmd {
	app.deployCancel.Store(false)
	return func() tea.Msg {
		events := make(chan operationEvent, 64)
		go func() {
			plan, err := session.Plan(&selected, false, selected, func(event progress.Event) {
				select {
				case events <- operationEvent{progress: event}:
				default:
				}
			}, func() bool { return app.deployCancel.Load() })
			if err == nil {
				events <- operationEvent{plan: plan}
			}
			events <- operationEvent{err: err, done: true}
			close(events)
		}()
		return operationStartedMsg{events: events}
	}
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
		if s.app.deployCancel.Load() {
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

type conflictState struct {
	Conflict conflicts.Conflict
	Action   conflicts.FileAction
	Rows     []conflicts.RowDecision
}

func newConflictStates(plan *deploy.DeployPlan) []*conflictState {
	states := make([]*conflictState, 0, len(plan.Conflicts))
	for _, conflict := range plan.Conflicts {
		action := conflicts.ActionReview
		rows := conflicts.ReviewAll(conflict, conflicts.RowClone).Rows
		if conflict.LocalText == nil || conflict.CloneText == nil {
			// Whole-file conflicts cannot be reviewed by rows: the person must
			// choose apply or omit explicitly.
			action = ""
			rows = nil
		}
		states = append(states, &conflictState{Conflict: conflict, Action: action, Rows: rows})
	}
	return states
}

type deployConflictsScreen struct {
	screenBase
	index    int
	messages []Message
}

func newDeployConflictsScreen(app *App) *deployConflictsScreen {
	screen := &deployConflictsScreen{screenBase: screenBase{app: app}}
	if len(app.conflicts) == 0 {
		screen.messages = []Message{{Kind: MessageInfo, Text: texts.D2NoConflicts}}
	}
	return screen
}

func (s *deployConflictsScreen) Flow() string        { return texts.FlowDeploy }
func (s *deployConflictsScreen) Title() string       { return texts.TitleConflictos }
func (s *deployConflictsScreen) Step() (int, int)    { return 3, 4 }
func (s *deployConflictsScreen) Messages() []Message { return s.messages }
func (s *deployConflictsScreen) Keys() []key.Binding { return deployConflictsKeys() }

func (s *deployConflictsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	states := s.app.conflicts
	switch keyMessage.String() {
	case "up", "k":
		if len(states) > 0 {
			s.index = (s.index + len(states) - 1) % len(states)
			s.app.conflictIndex = s.index
		}
	case "down", "j":
		if len(states) > 0 {
			s.index = (s.index + 1) % len(states)
			s.app.conflictIndex = s.index
		}
	case "c":
		s.setAction(s.index, conflicts.ActionApply)
	case "o":
		s.setAction(s.index, conflicts.ActionOmit)
	case "C":
		for index := range states {
			s.setAction(index, conflicts.ActionApply)
		}
	case "O":
		for index := range states {
			s.setAction(index, conflicts.ActionOmit)
		}
	case "r":
		if len(states) > 0 {
			s.app.conflictIndex = s.index
			s.setAction(s.index, conflicts.ActionReview)
			return s, pushScreen(newDeployDetailScreen(s.app))
		}
	case "enter":
		if missing := s.missing(); len(missing) > 0 {
			s.messages = []Message{{Kind: MessageWarning, Text: strings.ReplaceAll(texts.D2Pending, "{paths}", strings.Join(missing, ", "))}}
			return s, nil
		}
		return s, pushScreen(newDeployConfirmScreen(s.app))
	case "esc":
		return s, popScreen()
	}
	return s, nil
}

func (s *deployConflictsScreen) setAction(index int, action conflicts.FileAction) {
	if index < 0 || index >= len(s.app.conflicts) {
		return
	}
	state := s.app.conflicts[index]
	state.Action = action
	if action == conflicts.ActionReview && len(state.Rows) != conflicts.RowCount(state.Conflict) {
		state.Rows = conflicts.ReviewAll(state.Conflict, conflicts.RowClone).Rows
	}
}

func (s *deployConflictsScreen) missing() []string {
	missing := make([]string, 0)
	for _, state := range s.app.conflicts {
		if state.Action == "" {
			missing = append(missing, state.Conflict.Path)
			continue
		}
		if state.Action == conflicts.ActionReview {
			if state.Conflict.LocalText == nil || state.Conflict.CloneText == nil {
				missing = append(missing, state.Conflict.Path)
				continue
			}
			if conflicts.RowCount(state.Conflict) > 0 && len(state.Rows) != conflicts.RowCount(state.Conflict) {
				missing = append(missing, state.Conflict.Path)
			}
		}
	}
	return missing
}

func (s *deployConflictsScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(texts.DeployConflictsTitle),
	}
	if s.app.deployPlan != nil {
		counts := deployCounts(s.app.deployPlan)
		notices := len(s.app.deployPlan.Notices)
		if s.app.deploySession != nil {
			notices += len(s.app.deploySession.Warnings)
		}
		summary := strings.NewReplacer(
			"{new}", fmt.Sprintf("%d", counts["new"]),
			"{perfect}", fmt.Sprintf("%d", counts["perfect"]),
			"{unsupported}", fmt.Sprintf("%d", counts["unsupported"]),
			"{notices}", fmt.Sprintf("%d", notices),
		).Replace(texts.D2Summary)
		lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(summary))
	}
	lines = append(lines, "")
	widthTable := width
	if widthTable > 74 {
		widthTable = 74
	}
	fileWidth := widthTable - 34
	if fileWidth < 12 {
		fileWidth = 12
	}
	lines = append(lines, "  "+PadRight(texts.D2ColumnFile, fileWidth)+"  "+PadLeft(texts.D2ColumnBlocks, 7)+"  "+PadLeft(texts.D2ColumnLines, 7)+"  "+PadRight(texts.D2ColumnAction, 24))
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, widthTable-4))))
	maxRows := height - 6
	if maxRows < 3 {
		maxRows = 3
	}
	start, end := window(s.index, len(s.app.conflicts), maxRows)
	if start > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D2ScrollUp, "{count}", fmt.Sprintf("%d", start)))
	}
	for index := start; index < end; index++ {
		state := s.app.conflicts[index]
		cursor := "  "
		style := s.app.theme.Styles.Secondary
		if index == s.index {
			cursor = s.app.theme.Symbols.Cursor + " "
			style = s.app.theme.Styles.Primary
		}
		blocks := len(state.Conflict.Hunks)
		linesCount := conflicts.RowCount(state.Conflict)
		blockText := fmt.Sprintf("%d", blocks)
		lineText := fmt.Sprintf("%d", linesCount)
		if blocks == 0 {
			blockText = s.app.theme.Symbols.Zero
		}
		if linesCount == 0 {
			lineText = s.app.theme.Symbols.Zero
		}
		line := cursor + PadRight(Truncate(s.app.displayPath(state.Conflict.Destination), fileWidth, s.app.theme.Symbols.Ellipsis), fileWidth) + "  " + PadLeft(blockText, 7) + "  " + PadLeft(lineText, 7) + "  " + PadRight(s.actionLabel(state), 24)
		lines = append(lines, style.Render(line))
	}
	if end < len(s.app.conflicts) {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D2ScrollDown, "{count}", fmt.Sprintf("%d", len(s.app.conflicts)-end)))
	}
	if len(s.app.conflicts) == 0 {
		lines = append(lines, "  "+s.app.theme.Styles.Success.Render(texts.D2NoConflicts))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func deployCounts(plan *deploy.DeployPlan) map[string]int {
	result := map[string]int{"new": 0, "perfect": len(plan.Unchanged), "unsupported": len(plan.Omitted)}
	seen := map[string]bool{}
	for _, entry := range plan.Entries {
		if seen[entry.Path] {
			continue
		}
		seen[entry.Path] = true
		if entry.Status == deploy.StatusNew {
			result["new"]++
		}
	}
	return result
}

func (s *deployConflictsScreen) actionLabel(state *conflictState) string {
	switch state.Action {
	case conflicts.ActionApply:
		return texts.D2ActionApply
	case conflicts.ActionOmit:
		return texts.D2ActionOmit
	case conflicts.ActionReview:
		if state.Conflict.LocalText == nil || state.Conflict.CloneText == nil {
			return texts.D2ActionReview
		}
		chosen := 0
		for _, decision := range state.Rows {
			if decision == conflicts.RowClone {
				chosen++
			}
		}
		return strings.NewReplacer(
			"{chosen}", fmt.Sprintf("%d", chosen),
			"{total}", fmt.Sprintf("%d", conflicts.RowCount(state.Conflict)),
		).Replace(texts.D2Hybrid)
	case "":
		return texts.D2ActionPending
	}
	return texts.D2ActionReview
}

type fileRow struct {
	LocalNo       int
	CloneNo       int
	LocalText     string
	CloneText     string
	Kind          conflicts.RowKind
	DecisionIndex int
	Block         int
	Changed       bool
}

type deployDetailScreen struct {
	screenBase
	conflictIndex int
	index         int
	whole         bool
	wholeIndex    int
	rows          []fileRow
	digits        int
	messages      []Message
}

func newDeployDetailScreen(app *App) *deployDetailScreen {
	screen := &deployDetailScreen{screenBase: screenBase{app: app}, conflictIndex: app.conflictIndex}
	if screen.conflictIndex >= len(app.conflicts) {
		screen.conflictIndex = 0
	}
	if len(app.conflicts) > 0 {
		state := app.conflicts[screen.conflictIndex]
		screen.whole = len(state.Conflict.Hunks) == 0 || state.Conflict.LocalText == nil || state.Conflict.CloneText == nil
		if state.Action == conflicts.ActionOmit {
			screen.wholeIndex = 1
		}
		screen.messages = []Message{{Kind: MessageWarning, Text: texts.DiffSecretsShort}}
		screen.rows = buildFileRows(state.Conflict)
		screen.digits = lineDigits(state.Conflict)
		for index, row := range screen.rows {
			if row.DecisionIndex >= 0 {
				screen.index = index
				break
			}
		}
	}
	return screen
}

func lineDigits(conflict conflicts.Conflict) int {
	value := max(len(splitDisplayLines(textValue(conflict.LocalText))), len(splitDisplayLines(textValue(conflict.CloneText))))
	digits := len(strconv.Itoa(max(1, value)))
	if digits < 2 {
		digits = 2
	}
	return digits
}

func (s *deployDetailScreen) Flow() string        { return texts.FlowDeploy }
func (s *deployDetailScreen) Title() string       { return texts.TitleConflictos }
func (s *deployDetailScreen) Step() (int, int)    { return 3, 4 }
func (s *deployDetailScreen) Messages() []Message { return s.messages }
func (s *deployDetailScreen) Keys() []key.Binding { return deployDetailKeys(s.whole) }

func (s *deployDetailScreen) state() *conflictState {
	if len(s.app.conflicts) == 0 {
		return nil
	}
	return s.app.conflicts[s.conflictIndex]
}

func (s *deployDetailScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	state := s.state()
	if state == nil {
		return s, popScreen()
	}
	if s.whole {
		return s.updateWhole(state, keyMessage)
	}
	rows := s.rows
	if len(rows) == 0 {
		return s, popScreen()
	}
	switch keyMessage.String() {
	case "up", "k":
		s.index = (s.index + len(rows) - 1) % len(rows)
	case "down", "j":
		s.index = (s.index + 1) % len(rows)
	case "left":
		s.setRow(rows[s.index], conflicts.RowLocal)
	case "right", " ":
		s.setRow(rows[s.index], conflicts.RowClone)
	case "c":
		s.setBlock(state, rows[s.index].Block, conflicts.RowClone)
	case "o":
		s.setBlock(state, rows[s.index].Block, conflicts.RowLocal)
	case "C":
		s.setAll(state, conflicts.RowClone)
	case "O":
		s.setAll(state, conflicts.RowLocal)
	case "tab":
		s.moveBlock(rows, 1)
	case "shift+tab":
		s.moveBlock(rows, -1)
	case "esc":
		if message := validateHybrid(state); message != "" {
			s.messages = []Message{{Kind: MessageError, Text: message}}
			return s, nil
		}
		return s, popScreen()
	}
	return s, nil
}

func (s *deployDetailScreen) updateWhole(state *conflictState, keyMessage tea.KeyMsg) (Screen, tea.Cmd) {
	switch keyMessage.String() {
	case "up", "k":
		s.wholeIndex = (s.wholeIndex + 1) % 2
	case "down", "j":
		s.wholeIndex = (s.wholeIndex + 1) % 2
	case "enter":
		if s.wholeIndex == 0 {
			state.Action = conflicts.ActionApply
		} else {
			state.Action = conflicts.ActionOmit
		}
		return s, popScreen()
	case "esc":
		if state.Action == conflicts.ActionReview {
			state.Action = ""
		}
		return s, popScreen()
	}
	return s, nil
}

func (s *deployDetailScreen) setRow(row fileRow, decision conflicts.RowDecision) {
	state := s.state()
	if state == nil || row.DecisionIndex < 0 || row.DecisionIndex >= len(state.Rows) {
		return
	}
	state.Rows[row.DecisionIndex] = decision
}

func (s *deployDetailScreen) setBlock(state *conflictState, block int, decision conflicts.RowDecision) {
	if block < 0 {
		return
	}
	decisionIndex := 0
	for index, hunk := range state.Conflict.Hunks {
		for _, row := range hunk.Rows {
			if row.Kind == conflicts.RowEqual {
				continue
			}
			if index == block {
				state.Rows[decisionIndex] = decision
			}
			decisionIndex++
		}
	}
}

func (s *deployDetailScreen) setAll(state *conflictState, decision conflicts.RowDecision) {
	for index := range state.Rows {
		state.Rows[index] = decision
	}
}

func (s *deployDetailScreen) moveBlock(rows []fileRow, direction int) {
	if len(rows) == 0 {
		return
	}
	currentBlock := rows[s.index].Block
	for step := 1; step <= len(rows); step++ {
		position := (s.index + direction*step + len(rows)) % len(rows)
		if rows[position].Changed && rows[position].Block != currentBlock {
			s.index = position
			return
		}
	}
}

func validateHybrid(state *conflictState) string {
	if state.Action != conflicts.ActionReview || state.Conflict.LocalText == nil || state.Conflict.CloneText == nil {
		return ""
	}
	merged, err := conflicts.Compose(*state.Conflict.LocalText, *state.Conflict.CloneText, state.Rows)
	if err != nil {
		return err.Error()
	}
	validation := conflicts.ValidateMerge(state.Conflict.Path, *state.Conflict.LocalText, *state.Conflict.CloneText, merged)
	if validation.Status == "invalid" {
		return strings.ReplaceAll(texts.D3InvalidHybrid, "{message}", validation.Message)
	}
	return ""
}

func buildFileRows(conflict conflicts.Conflict) []fileRow {
	if conflict.LocalText == nil || conflict.CloneText == nil {
		return nil
	}
	localLines := splitDisplayLines(*conflict.LocalText)
	cloneLines := splitDisplayLines(*conflict.CloneText)
	rows := make([]fileRow, 0, len(localLines)+len(cloneLines))
	localCursor, cloneCursor := 0, 0
	decision := 0
	for block, hunk := range conflict.Hunks {
		for localCursor < hunk.LocalStart-1 && cloneCursor < hunk.CloneStart-1 {
			rows = append(rows, fileRow{
				LocalNo: localCursor + 1, CloneNo: cloneCursor + 1,
				LocalText: localLines[localCursor], CloneText: cloneLines[cloneCursor],
				Kind: conflicts.RowEqual, DecisionIndex: -1, Block: -1,
			})
			localCursor++
			cloneCursor++
		}
		for _, hunkRow := range hunk.Rows {
			row := fileRow{Kind: hunkRow.Kind, Block: block, DecisionIndex: -1}
			if hunkRow.Kind != conflicts.RowCloneOnly {
				row.LocalNo = localCursor + 1
				row.LocalText = trimLine(hunkRow.Local)
				localCursor++
			}
			if hunkRow.Kind != conflicts.RowLocalOnly {
				row.CloneNo = cloneCursor + 1
				row.CloneText = trimLine(hunkRow.Clone)
				cloneCursor++
			}
			if hunkRow.Kind != conflicts.RowEqual {
				row.Changed = true
				row.DecisionIndex = decision
				decision++
			}
			rows = append(rows, row)
		}
	}
	for localCursor < len(localLines) && cloneCursor < len(cloneLines) {
		rows = append(rows, fileRow{
			LocalNo: localCursor + 1, CloneNo: cloneCursor + 1,
			LocalText: localLines[localCursor], CloneText: cloneLines[cloneCursor],
			Kind: conflicts.RowEqual, DecisionIndex: -1, Block: -1,
		})
		localCursor++
		cloneCursor++
	}
	return rows
}

func splitDisplayLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	if value == "" {
		return nil
	}
	lines := strings.Split(value, "\n")
	if strings.HasSuffix(value, "\n") {
		lines = lines[:len(lines)-1]
	}
	for index := range lines {
		lines[index] = strings.TrimSuffix(lines[index], "\r")
	}
	return lines
}

func trimLine(value string) string {
	return strings.TrimSuffix(strings.TrimSuffix(value, "\n"), "\r")
}

func (s *deployDetailScreen) View(width, height int) string {
	state := s.state()
	if state == nil {
		return Fit([]string{texts.DeployConflictsTitle}, width, height, s.app.theme.Symbols.Ellipsis)
	}
	if s.whole {
		return s.viewWhole(state, width, height)
	}
	rows := s.rows
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(s.app.displayPath(state.Conflict.Destination)),
		"",
	}
	leftCount, rightCount := chosenCounts(rows, state)
	column := (width - s.app.theme.Tokens.ColGap) / 2
	right := width - column - s.app.theme.Tokens.ColGap
	lines = append(lines,
		"  "+PadRight(texts.D3Original, column-2)+strings.Repeat(" ", s.app.theme.Tokens.ColGap)+texts.D3Clone,
		"  "+PadRight(strings.ReplaceAll(texts.D3LinesChosen, "{count}", fmt.Sprintf("%d", leftCount)), column-2)+strings.Repeat(" ", s.app.theme.Tokens.ColGap)+strings.ReplaceAll(texts.D3LinesChosen, "{count}", fmt.Sprintf("%d", rightCount)),
	)
	block := rows[s.index].Block
	if block >= 0 {
		lines = append(lines, "  "+strings.NewReplacer("{index}", fmt.Sprintf("%d", block+1), "{total}", fmt.Sprintf("%d", len(state.Conflict.Hunks))).Replace(texts.D3Block))
	} else {
		lines = append(lines, "")
	}
	maxRows := height - 9
	if maxRows < 3 {
		maxRows = 3
	}
	start, end := window(s.index, len(rows), maxRows)
	if start > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D3ScrollUp, "{count}", fmt.Sprintf("%d", start)))
	}
	digits := s.digits
	for index := start; index < end; index++ {
		row := rows[index]
		leftCell := s.cell(state, row, false, digits, column, index == s.index)
		rightCell := s.cell(state, row, true, digits, right, false)
		lines = append(lines, PadRight(leftCell, column)+strings.Repeat(" ", s.app.theme.Tokens.ColGap)+rightCell)
	}
	if end < len(rows) {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D3ScrollDown, "{count}", fmt.Sprintf("%d", len(rows)-end)))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *deployDetailScreen) viewWhole(state *conflictState, width, height int) string {
	reason := texts.D3BinaryReason
	switch state.Conflict.Kind {
	case conflicts.KindLarge:
		reason = texts.D3LargeReason
	case conflicts.KindLink:
		reason = texts.D3LinkReason
	}
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(texts.D3WholeTitle),
		"",
		"  " + strings.ReplaceAll(texts.D3PreviewLine, "{path}", s.app.displayPath(state.Conflict.Destination)),
		"  " + strings.ReplaceAll(texts.D3WholeReason, "{reason}", reason),
		"",
	}
	options := []string{texts.D3ApplyClone, texts.D3KeepOriginal}
	for index, option := range options {
		if index == s.wholeIndex {
			lines = append(lines, "  "+s.app.theme.Styles.Primary.Render(s.app.theme.Symbols.Cursor+" "+option))
		} else {
			lines = append(lines, "    "+option)
		}
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func textValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func chosenCounts(rows []fileRow, state *conflictState) (int, int) {
	left, right := 0, 0
	for _, row := range rows {
		if !row.Changed || row.DecisionIndex < 0 || row.DecisionIndex >= len(state.Rows) {
			continue
		}
		if state.Rows[row.DecisionIndex] == conflicts.RowClone {
			if row.CloneNo > 0 {
				right++
			}
		} else if row.LocalNo > 0 {
			left++
		}
	}
	return left, right
}

func (s *deployDetailScreen) cell(state *conflictState, row fileRow, cloneSide bool, digits, width int, focused bool) string {
	number := row.LocalNo
	text := row.LocalText
	if cloneSide {
		number = row.CloneNo
		text = row.CloneText
	}
	marker := " "
	style := s.app.theme.Styles.Secondary
	if row.Changed && row.DecisionIndex >= 0 && row.DecisionIndex < len(state.Rows) {
		decision := state.Rows[row.DecisionIndex]
		chosen := decision == conflicts.RowClone
		if !cloneSide {
			chosen = decision == conflicts.RowLocal
		}
		if chosen {
			marker = s.app.theme.Symbols.Chosen
			style = s.app.theme.Styles.Chosen
		} else {
			marker = s.app.theme.Symbols.NotChosen
			style = s.app.theme.Styles.Discarded
		}
	}
	numberText := strings.Repeat(" ", digits)
	if number > 0 {
		numberText = PadLeft(fmt.Sprintf("%d", number), digits)
	}
	if number == 0 {
		text = s.app.theme.Symbols.NoLine
	}
	prefix := "  "
	if focused {
		prefix = s.app.theme.Symbols.Cursor + " "
	}
	cell := prefix + numberText + " " + marker + " " + text
	return style.Render(Truncate(cell, width, s.app.theme.Symbols.Ellipsis))
}

type appSummary struct {
	ID        string
	Name      string
	New       int
	Replaced  int
	Hybrid    int
	Omitted   int
	Unchanged int
}

type deployConfirmScreen struct {
	screenBase
	rows       []appSummary
	totals     appSummary
	protected  int
	snapshot   string
	actions    []string
	index      int
	applying   bool
	events     <-chan operationEvent
	messages   []Message
	previewErr string
}

func newDeployConfirmScreen(app *App) *deployConfirmScreen {
	screen := &deployConfirmScreen{
		screenBase: screenBase{app: app},
		actions:    []string{texts.D4Abort, texts.D4Retreat, texts.D4Deploy},
		index:      1,
		messages:   []Message{{Kind: MessageWarning, Text: texts.D4CloseApps}},
	}
	screen.build()
	return screen
}

func (s *deployConfirmScreen) Flow() string        { return texts.FlowDeploy }
func (s *deployConfirmScreen) Title() string       { return texts.TitleConfirm }
func (s *deployConfirmScreen) Step() (int, int)    { return 4, 4 }
func (s *deployConfirmScreen) Messages() []Message { return s.messages }
func (s *deployConfirmScreen) Keys() []key.Binding { return deployConfirmKeys() }

func (s *deployConfirmScreen) build() {
	app := s.app
	if app.deployPlan == nil {
		return
	}
	decisions := app.deployDecisions()
	if app.deploySession != nil {
		if _, err := app.deploySession.Preview(decisions); err != nil {
			s.previewErr = err.Error()
			s.messages = append(s.messages, Message{Kind: MessageError, Text: err.Error()})
		}
	}
	names := map[string]string{}
	if app.deploySession != nil && app.deploySession.Manifest != nil {
		for _, entry := range app.deploySession.Manifest.Apps {
			names[entry.ID] = entry.Name
			if names[entry.ID] == "" {
				names[entry.ID] = entry.ID
			}
		}
	}
	byApp := map[string]*appSummary{}
	order := make([]string, 0)
	seen := map[string]bool{}
	conflictByPath := map[string]*conflictState{}
	for _, state := range app.conflicts {
		conflictByPath[state.Conflict.Path] = state
	}
	for _, entry := range app.deployPlan.Entries {
		if seen[entry.Path] {
			continue
		}
		seen[entry.Path] = true
		summary, ok := byApp[entry.AppID]
		if !ok {
			name := names[entry.AppID]
			if name == "" {
				name = entry.AppID
			}
			if definition, found := app.catalog[entry.AppID]; found && definition.Name != "" {
				name = definition.Name
			}
			summary = &appSummary{ID: entry.AppID, Name: name}
			byApp[entry.AppID] = summary
			order = append(order, entry.AppID)
		}
		switch entry.Status {
		case deploy.StatusNew:
			summary.New++
		case deploy.StatusIdentical:
			summary.Unchanged++
		case deploy.StatusUnsupported:
			summary.Omitted++
		case deploy.StatusDifferent:
			state := conflictByPath[entry.Path]
			if state == nil {
				summary.Replaced++
				continue
			}
			switch state.Action {
			case conflicts.ActionApply:
				summary.Replaced++
			case conflicts.ActionOmit:
				summary.Omitted++
			case conflicts.ActionReview:
				if state.Conflict.LocalText == nil || state.Conflict.CloneText == nil {
					summary.Replaced++
					continue
				}
				merged, err := conflicts.Compose(*state.Conflict.LocalText, *state.Conflict.CloneText, state.Rows)
				if err != nil {
					summary.Omitted++
					continue
				}
				if merged == *state.Conflict.LocalText {
					summary.Unchanged++
					continue
				}
				allClone := true
				for _, decision := range state.Rows {
					if decision != conflicts.RowClone {
						allClone = false
						break
					}
				}
				if allClone {
					summary.Replaced++
				} else {
					summary.Hybrid++
				}
			default:
				summary.Omitted++
			}
		}
	}
	s.rows = make([]appSummary, 0, len(order))
	for _, id := range order {
		summary := *byApp[id]
		s.rows = append(s.rows, summary)
		s.totals.New += summary.New
		s.totals.Replaced += summary.Replaced
		s.totals.Hybrid += summary.Hybrid
		s.totals.Omitted += summary.Omitted
		s.totals.Unchanged += summary.Unchanged
	}
	s.protected = s.totals.Replaced + s.totals.Hybrid
	if s.protected > 0 {
		s.snapshot = app.snapshotPath()
	}
}

func (s *deployConfirmScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if s.applying {
		if event, ok := msg.(operationEvent); ok {
			return s.handleApply(event)
		}
		if started, ok := msg.(operationStartedMsg); ok {
			s.events = started.events
			return s, waitOperation(started.events)
		}
		if keyMessage, ok := msg.(tea.KeyMsg); ok && (keyMessage.String() == "ctrl+c" || keyMessage.String() == "esc") {
			s.app.deployCancel.Store(true)
			s.messages = []Message{{Kind: MessageWarning, Text: texts.StatusCancelling}}
			return s, nil
		}
		return s, nil
	}
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "left", "h":
		s.index = (s.index + len(s.actions) - 1) % len(s.actions)
	case "right", "l":
		s.index = (s.index + 1) % len(s.actions)
	case "esc":
		return s, popScreen()
	case "enter":
		switch s.index {
		case 0:
			if s.app.deploySession != nil {
				s.app.deploySession.Close()
			}
			if s.app.menuMode {
				return s, func() tea.Msg { return menuMsg{} }
			}
			return s, quitApp(0)
		case 1:
			return s, popScreen()
		case 2:
			if s.previewErr != "" {
				s.messages = []Message{{Kind: MessageError, Text: s.previewErr}}
				return s, nil
			}
			if s.app.deploySession == nil {
				s.messages = []Message{{Kind: MessageError, Text: texts.StatusReviewDefault}}
				return s, nil
			}
			s.applying = true
			s.messages = []Message{{Kind: MessageWarning, Text: texts.DeployApplying}}
			return s, startDeployApply(s.app)
		}
	}
	return s, nil
}

func startDeployApply(app *App) tea.Cmd {
	app.deployCancel.Store(false)
	decisions := app.deployDecisions()
	session := app.deploySession
	return func() tea.Msg {
		events := make(chan operationEvent, 64)
		go func() {
			result, err := session.Apply(decisions, func(event progress.Event) {
				select {
				case events <- operationEvent{progress: event}:
				default:
				}
			}, func() bool { return app.deployCancel.Load() })
			if err == nil {
				events <- operationEvent{deploy: &result}
			}
			events <- operationEvent{err: err, done: true}
			close(events)
		}()
		return operationStartedMsg{events: events}
	}
}

func (s *deployConfirmScreen) handleApply(event operationEvent) (Screen, tea.Cmd) {
	if event.deploy != nil {
		s.app.deployResult = event.deploy
	}
	if !event.done {
		if s.events != nil {
			return s, waitOperation(s.events)
		}
		return s, nil
	}
	s.applying = false
	if event.err != nil {
		if s.app.deployCancel.Load() {
			s.messages = []Message{{Kind: MessageWarning, Text: texts.StatusPreparingCancelled}}
			return s, nil
		}
		var partial *kerrors.PartialApplyError
		if errors.As(event.err, &partial) {
			result := deploy.DeployResult{Snapshot: partial.Snapshot, Deployed: partial.Applied}
			return s, pushScreen(newDeploySummaryScreen(s.app, &result, true, partial.Failed))
		}
		return s, showError(texts.ErrorApplyingDeploy, texts.ErrorNothingWritten, texts.StatusReviewDefault, event.err.Error())
	}
	return s, pushScreen(newDeploySummaryScreen(s.app, s.app.deployResult, false, ""))
}

func (s *deployConfirmScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(texts.DeployConfirmTitle),
		"",
	}
	numberWidth := 10
	if width >= 90 {
		numberWidth = 13
	}
	gap := 1
	appWidth := width - 2 - 5*(gap+numberWidth)
	if appWidth < 12 {
		appWidth = 12
	}
	header := "  " + PadRight(texts.D4ColumnApp, appWidth)
	for _, column := range []string{texts.D4ColumnNew, texts.D4ColumnReplaced, texts.D4ColumnHybrid, texts.D4ColumnOmitted, texts.D4ColumnUnchanged} {
		header += strings.Repeat(" ", gap) + PadLeft(Truncate(column, numberWidth, s.app.theme.Symbols.Ellipsis), numberWidth)
	}
	lines = append(lines, header)
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, width-4))))
	maxRows := height - 11
	if maxRows < 3 {
		maxRows = 3
	}
	start, end := window(0, len(s.rows), maxRows)
	if start > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D4ScrollUp, "{count}", fmt.Sprintf("%d", start)))
	}
	for index := start; index < end; index++ {
		lines = append(lines, s.rowLine(s.rows[index], appWidth, numberWidth, gap))
	}
	if end < len(s.rows) {
		lines = append(lines, "  "+strings.ReplaceAll(texts.D4ScrollDown, "{count}", fmt.Sprintf("%d", len(s.rows)-end)))
	}
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, width-4))))
	lines = append(lines, s.rowLine(s.totals, appWidth, numberWidth, gap))
	lines = append(lines, "")
	protected := strings.ReplaceAll(texts.D4Protected, "{count}", fmt.Sprintf("%d", s.protected))
	snapshot := texts.D4NoSnapshot
	if s.protected > 0 {
		snapshot = strings.ReplaceAll(texts.D4Snapshot, "{path}", s.app.displayPath(s.snapshot))
	}
	lines = append(lines, "  "+protected+"    "+snapshot, "")
	actions := make([]string, 0, len(s.actions))
	for index, action := range s.actions {
		if index == s.index {
			actions = append(actions, s.app.theme.Styles.Primary.Render("> "+action+" <"))
		} else {
			actions = append(actions, action)
		}
	}
	lines = append(lines, "  "+strings.Join(actions, "        "))
	if s.applying {
		lines = append(lines, "", "  "+s.app.theme.Styles.Warning.Render(texts.DeployApplying))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *deployConfirmScreen) rowLine(row appSummary, appWidth, numberWidth, gap int) string {
	cells := []int{row.New, row.Replaced, row.Hybrid, row.Omitted, row.Unchanged}
	line := "  " + PadRight(Truncate(row.Name, appWidth, s.app.theme.Symbols.Ellipsis), appWidth)
	for _, cell := range cells {
		value := fmt.Sprintf("%d", cell)
		if cell == 0 {
			value = s.app.theme.Symbols.Zero
		}
		line += strings.Repeat(" ", gap) + PadLeft(value, numberWidth)
	}
	if row.ID == "" {
		line = "  " + PadRight(texts.D4Total, appWidth)
		for _, cell := range cells {
			value := fmt.Sprintf("%d", cell)
			if cell == 0 {
				value = s.app.theme.Symbols.Zero
			}
			line += strings.Repeat(" ", gap) + PadLeft(value, numberWidth)
		}
	}
	return line
}

func (a *App) deployDecisions() conflicts.ConflictDecisions {
	decisions := conflicts.ConflictDecisions{Decisions: map[string]conflicts.FileDecision{}}
	for _, state := range a.conflicts {
		rows := make([]conflicts.RowDecision, len(state.Rows))
		copy(rows, state.Rows)
		decisions.Set(state.Conflict.Path, conflicts.FileDecision{Action: state.Action, Rows: rows})
	}
	return decisions
}

type deploySummaryScreen struct {
	screenBase
	result     *deploy.DeployResult
	incomplete bool
	failed     string
	messages   []Message
}

func newDeploySummaryScreen(app *App, result *deploy.DeployResult, incomplete bool, failed string) *deploySummaryScreen {
	screen := &deploySummaryScreen{
		screenBase: screenBase{app: app}, result: result, incomplete: incomplete, failed: failed,
	}
	if incomplete {
		screen.messages = []Message{{Kind: MessageError, Text: strings.NewReplacer(
			"{applied}", fmt.Sprintf("%d", len(result.Deployed)),
			"{failed}", failed,
		).Replace(texts.D5IncompleteInfo)}}
	} else if result != nil && (len(result.Omitted) > 0 || len(result.MissingRequested) > 0) {
		screen.messages = []Message{{Kind: MessageWarning, Text: strings.ReplaceAll(texts.DeployExitLine, "{code}", "3")}}
	}
	return screen
}

func (s *deploySummaryScreen) Flow() string        { return texts.FlowDeploy }
func (s *deploySummaryScreen) Step() (int, int)    { return 0, 0 }
func (s *deploySummaryScreen) Messages() []Message { return s.messages }
func (s *deploySummaryScreen) Keys() []key.Binding {
	return deploySummaryKeys(s.result != nil && s.result.Snapshot != "")
}

func (s *deploySummaryScreen) Title() string {
	if s.incomplete {
		return texts.D5Incomplete
	}
	return texts.D5Title
}

func (s *deploySummaryScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "c":
		if s.result != nil && s.result.Snapshot != "" {
			s.messages = []Message{{Kind: MessageInfo, Text: texts.D5CopyInfo}}
			return s, s.app.copyCommand(s.result.Snapshot, texts.D5CopyInfo, texts.D5CopyError)
		}
	case "enter", "esc":
		if s.incomplete {
			s.app.finish(1, "")
			return s, nil
		}
		s.app.finishDeploy()
		return s, nil
	}
	return s, nil
}

func (s *deploySummaryScreen) View(width, height int) string {
	style := s.app.theme.Styles.SuccessTitle
	if s.incomplete {
		style = s.app.theme.Styles.ErrorTitle
	}
	lines := []string{"  " + style.Render(s.Title()), "", "  " + s.app.theme.Styles.Secondary.Render(texts.DeploySummaryTitle)}
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, width-4))))
	rows := [][2]string{
		{texts.D5Deployed, "0"}, {texts.D5Hybrids, "0"}, {texts.D5Omitted, "0"},
		{texts.D5Unchanged, "0"}, {texts.D5NotInstalled, "0"}, {texts.D5Protected, "0"},
	}
	if s.result != nil {
		rows[0][1] = fmt.Sprintf("%d", len(s.result.Deployed))
		rows[1][1] = fmt.Sprintf("%d", len(s.result.Merged))
		rows[2][1] = fmt.Sprintf("%d", len(s.result.Omitted))
		rows[3][1] = fmt.Sprintf("%d", len(s.result.Unchanged))
		rows[4][1] = fmt.Sprintf("%d", len(s.result.NotInstalled))
		rows[5][1] = fmt.Sprintf("%d", s.result.SnapshotFiles)
	}
	for _, row := range rows {
		lines = append(lines, "  "+PadRight(row[0], max(4, width-14))+PadLeft(row[1], 10))
	}
	lines = append(lines, "")
	if s.result != nil && s.result.Snapshot != "" {
		lines = append(lines,
			"  "+strings.ReplaceAll(texts.D5Snapshot, "{path}", s.app.displayPath(s.result.Snapshot)),
			"  "+strings.ReplaceAll(texts.D5Undo, "{path}", s.app.displayPath(s.result.Snapshot)),
			"", "  c "+texts.KeyCopySnapshot,
		)
	} else {
		lines = append(lines, "  "+texts.D5NoSnapshot)
	}
	if s.incomplete {
		lines = append(lines, "", "  "+s.app.theme.Styles.Warning.Render(texts.D5Recovery))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}
