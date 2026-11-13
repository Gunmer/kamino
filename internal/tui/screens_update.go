package tui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
	"github.com/Gunmer/kamino/internal/version"
)

type updatePhase int

const (
	updatePhaseChecking updatePhase = iota
	updatePhaseDownloading
	updatePhaseVerifying
	updatePhaseReplacing
	updatePhaseResult
)

type updateQueryDoneMsg struct {
	check update.Check
	err   error
}

type updateScreen struct {
	screenBase
	phase    updatePhase
	check    update.Check
	release  *update.ReleaseInfo
	result   *update.Result
	current  int
	total    int
	events   <-chan operationEvent
	messages []Message
}

func newUpdateScreen(app *App) *updateScreen {
	return &updateScreen{screenBase: screenBase{app: app}, phase: updatePhaseChecking}
}

func (s *updateScreen) Flow() string        { return texts.FlowUpdate }
func (s *updateScreen) Title() string       { return texts.FlowUpdate }
func (s *updateScreen) Step() (int, int)    { return 0, 0 }
func (s *updateScreen) Messages() []Message { return s.messages }
func (s *updateScreen) Keys() []key.Binding { return updateKeys(s.phase) }

func (s *updateScreen) Operation() string {
	switch s.phase {
	case updatePhaseChecking:
		return texts.OperationUpdateChecking
	case updatePhaseDownloading, updatePhaseVerifying:
		return texts.OperationUpdateDownloading
	case updatePhaseReplacing:
		return texts.OperationUpdateReplacing
	}
	return ""
}

func (s *updateScreen) Init() tea.Cmd {
	if s.phase != updatePhaseChecking {
		return nil
	}
	return queryUpdateCmd(s.app)
}

func queryUpdateCmd(app *App) tea.Cmd {
	query := app.updateQuery
	if query == nil {
		query = func() (update.Check, error) {
			return update.QueryUpdate(version.Version, "", nil)
		}
	}
	return func() tea.Msg {
		check, err := query()
		return updateQueryDoneMsg{check: check, err: err}
	}
}

func (s *updateScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch value := msg.(type) {
	case updateQueryDoneMsg:
		if value.err != nil {
			return s, updateErrorCommand(value.err)
		}
		s.check = value.check
		s.release = &value.check.Release
		s.result = &update.Result{
			CurrentVersion: value.check.CurrentVersion,
			LatestVersion:  value.check.Release.Version,
			Target:         value.check.Target,
			Outcome:        value.check.Outcome,
		}
		s.phase = updatePhaseResult
		return s, nil
	case operationStartedMsg:
		s.events = value.events
		return s, waitOperation(value.events)
	case operationEvent:
		return s.handleEvent(value)
	}
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "esc", "ctrl+c":
		switch s.phase {
		case updatePhaseDownloading, updatePhaseVerifying:
			s.app.updateCancel.Store(true)
			s.messages = []Message{{Kind: MessageWarning, Text: texts.UpdateCancelled}}
		case updatePhaseReplacing:
			s.messages = []Message{{Kind: MessageWarning, Text: texts.UpdateCannotCancel}}
		case updatePhaseResult:
			s.finish()
		case updatePhaseChecking:
			return s, popScreen()
		}
	case "enter":
		if s.phase == updatePhaseResult {
			if s.check.ShouldUpdate {
				s.phase = updatePhaseDownloading
				s.current = 0
				s.total = int(s.check.Release.AssetSize)
				s.messages = []Message{{Kind: MessageInfo, Text: texts.UpdateDownloading}}
				return s, startUpdateRun(s.app, s.release)
			}
			s.finish()
		}
	}
	return s, nil
}

func (s *updateScreen) handleEvent(event operationEvent) (Screen, tea.Cmd) {
	if event.progress.Total > 0 {
		s.current = event.progress.Current
		s.total = event.progress.Total
		if event.progress.Phase == progress.Downloading && s.current >= s.total && s.phase == updatePhaseDownloading {
			s.phase = updatePhaseVerifying
		}
	}
	if event.progress.Phase == progress.Replacing {
		s.phase = updatePhaseReplacing
	}
	if event.update != nil {
		s.result = event.update
	}
	if !event.done {
		if s.events != nil {
			return s, waitOperation(s.events)
		}
		return s, nil
	}
	if event.err != nil {
		if s.app.updateCancel.Load() {
			s.phase = updatePhaseResult
			s.messages = []Message{{Kind: MessageWarning, Text: texts.UpdateCancelled}}
			return s, nil
		}
		return s, updateErrorCommand(event.err)
	}
	s.phase = updatePhaseResult
	s.messages = nil
	return s, nil
}

func startUpdateRun(app *App, release *update.ReleaseInfo) tea.Cmd {
	app.updateCancel.Store(false)
	run := app.updateRun
	if run == nil {
		run = func(callback progress.Callback, cancelled progress.Cancel) (update.Result, error) {
			return update.RunUpdate(version.Version, "", release, callback, cancelled)
		}
	}
	return func() tea.Msg {
		events := make(chan operationEvent, 64)
		go func() {
			result, err := run(func(event progress.Event) {
				select {
				case events <- operationEvent{progress: event}:
				default:
				}
			}, func() bool { return app.updateCancel.Load() })
			if err == nil {
				events <- operationEvent{update: &result}
			}
			events <- operationEvent{err: err, done: true}
			close(events)
		}()
		return operationStartedMsg{events: events}
	}
}

func updateErrorCommand(err error) tea.Cmd {
	action := texts.StatusReviewDefault
	var typed *kerrors.UpdateError
	if errors.As(err, &typed) {
		if advice, ok := texts.UpdateErrorAdvice[typed.Kind]; ok {
			action = advice
		}
	}
	return showError(texts.ErrorUpdating, texts.ErrorNothingWritten, action, err.Error())
}

func (s *updateScreen) finish() {
	code := 0
	if s.result != nil && s.result.Outcome == update.OutcomeUpdated {
		code = 0
	}
	s.app.exitCode = code
	if !s.app.menuMode {
		s.app.current = nil
		return
	}
	s.app.lastMission = "actualización: " + string(s.result.Outcome)
	s.app.resetToMenu()
}

func (s *updateScreen) View(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.U1Title)}
	if s.check.CurrentVersion != "" {
		lines = append(lines, "  "+strings.ReplaceAll(texts.U1Current, "{version}", s.check.CurrentVersion))
	}
	if s.release != nil && s.release.Version != "" {
		lines = append(lines, "  "+strings.ReplaceAll(texts.U1Latest, "{version}", s.release.Version))
	}
	lines = append(lines, "")
	switch s.phase {
	case updatePhaseChecking:
		lines = append(lines, "  "+texts.U1Checking)
	case updatePhaseDownloading:
		lines = append(lines, "  "+texts.U1Downloading, "", "  "+progressBar(s.current, s.total, updateBarWidth(width)))
	case updatePhaseVerifying:
		lines = append(lines, "  "+texts.U1Verifying, "", "  "+progressBar(s.current, max(1, s.total), updateBarWidth(width)))
	case updatePhaseReplacing:
		lines = append(lines, "  "+texts.U1Replacing)
	case updatePhaseResult:
		lines = append(lines, "  "+s.resultText())
		lines = append(lines, "", "  "+texts.U1Continue)
	}
	if s.phase == updatePhaseDownloading || s.phase == updatePhaseVerifying {
		lines = append(lines, "", "  "+texts.U1CancelHint)
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func updateBarWidth(width int) int {
	if width > 40 {
		return 40
	}
	if width < 10 {
		return 10
	}
	return width
}

func (s *updateScreen) resultText() string {
	if s.result == nil {
		return ""
	}
	switch s.result.Outcome {
	case update.OutcomeUpdated:
		return strings.NewReplacer(
			"{current}", s.result.CurrentVersion,
			"{latest}", s.result.LatestVersion,
			"{path}", s.app.displayPath(s.result.Target),
		).Replace(texts.U1Updated) + "  " + texts.U1Restart
	case update.OutcomeNewer:
		return strings.NewReplacer("{current}", s.result.CurrentVersion, "{latest}", s.result.LatestVersion).Replace(texts.U1Newer)
	default:
		return strings.NewReplacer("{current}", s.result.CurrentVersion, "{latest}", s.result.LatestVersion).Replace(texts.U1UpToDate)
	}
}
