package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

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

func (s *deployConfirmScreen) Flow() string { return texts.FlowDeploy }

func (s *deployConfirmScreen) Title() string { return texts.TitleConfirm }

func (s *deployConfirmScreen) Step() (int, int) { return 4, 4 }

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
		path, err := app.snapshotPath()
		if err != nil {
			s.messages = append(s.messages, Message{Kind: MessageError, Text: err.Error()})
		} else {
			s.snapshot = path
		}
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
			s.app.cancelOperation()
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
	decisions := app.deployDecisions()
	session := app.deploySession
	ctx, done := app.beginOperation()
	return func() tea.Msg {
		events := make(chan operationEvent, 64)
		go func() {
			defer done()
			result, err := session.Apply(ctx, decisions, func(event progress.Event) {
				select {
				case events <- operationEvent{progress: event}:
				default:
				}
			})
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
		if cancelled(event.err) {
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
	if s.protected > 0 && s.snapshot != "" {
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
