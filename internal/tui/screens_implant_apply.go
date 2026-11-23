package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Gunmer/kamino/internal/conflicts"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/implant"
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

type implantConfirmScreen struct {
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

func newImplantConfirmScreen(app *App) *implantConfirmScreen {
	screen := &implantConfirmScreen{
		screenBase: screenBase{app: app},
		actions:    []string{texts.D4Abort, texts.D4Retreat, texts.D4Implant},
		index:      1,
		messages:   []Message{{Kind: MessageWarning, Text: texts.D4CloseApps}},
	}
	screen.build()
	return screen
}

func (s *implantConfirmScreen) Flow() string { return texts.FlowImplant }

func (s *implantConfirmScreen) Title() string { return texts.TitleConfirm }

func (s *implantConfirmScreen) Step() (int, int) { return 4, 4 }

func (s *implantConfirmScreen) Messages() []Message { return s.messages }

func (s *implantConfirmScreen) Keys() []key.Binding { return implantConfirmKeys() }

func (s *implantConfirmScreen) build() {
	app := s.app
	if app.implantPlan == nil {
		return
	}
	decisions := app.implantDecisions()
	if app.implantSession != nil {
		if _, err := app.implantSession.Preview(decisions); err != nil {
			s.previewErr = err.Error()
			s.messages = append(s.messages, Message{Kind: MessageError, Text: err.Error()})
		}
	}
	names := map[string]string{}
	if app.implantSession != nil && app.implantSession.Manifest != nil {
		for _, entry := range app.implantSession.Manifest.Apps {
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
	for _, entry := range app.implantPlan.Entries {
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
		case implant.StatusNew:
			summary.New++
		case implant.StatusIdentical:
			summary.Unchanged++
		case implant.StatusUnsupported:
			summary.Omitted++
		case implant.StatusDifferent:
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

func (s *implantConfirmScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
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
			if s.app.implantSession != nil {
				s.app.implantSession.Close()
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
			if s.app.implantSession == nil {
				s.messages = []Message{{Kind: MessageError, Text: texts.StatusReviewDefault}}
				return s, nil
			}
			s.applying = true
			s.messages = []Message{{Kind: MessageWarning, Text: texts.ImplantApplying}}
			return s, startImplantApply(s.app)
		}
	}
	return s, nil
}

func startImplantApply(app *App) tea.Cmd {
	decisions := app.implantDecisions()
	session := app.implantSession
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
				events <- operationEvent{implant: &result}
			}
			events <- operationEvent{err: err, done: true}
			close(events)
		}()
		return operationStartedMsg{events: events}
	}
}

func (s *implantConfirmScreen) handleApply(event operationEvent) (Screen, tea.Cmd) {
	if event.implant != nil {
		s.app.implantResult = event.implant
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
			result := implant.ImplantResult{Snapshot: partial.Snapshot, Implanted: partial.Applied}
			return s, pushScreen(newImplantSummaryScreen(s.app, &result, true, partial.Failed))
		}
		return s, showError(texts.ErrorApplyingImplant, texts.ErrorNothingWritten, texts.StatusReviewDefault, event.err.Error())
	}
	return s, pushScreen(newImplantSummaryScreen(s.app, s.app.implantResult, false, ""))
}

func (s *implantConfirmScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(texts.ImplantConfirmTitle),
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
		lines = append(lines, "", "  "+s.app.theme.Styles.Warning.Render(texts.ImplantApplying))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *implantConfirmScreen) rowLine(row appSummary, appWidth, numberWidth, gap int) string {
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

func (a *App) implantDecisions() conflicts.ConflictDecisions {
	decisions := conflicts.ConflictDecisions{Decisions: map[string]conflicts.FileDecision{}}
	for _, state := range a.conflicts {
		rows := make([]conflicts.RowDecision, len(state.Rows))
		copy(rows, state.Rows)
		decisions.Set(state.Conflict.Path, conflicts.FileDecision{Action: state.Action, Rows: rows})
	}
	return decisions
}
