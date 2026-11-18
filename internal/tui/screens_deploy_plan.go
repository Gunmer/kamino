package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func startDeployPlan(app *App, session *deploy.DeploySession, selected []string) tea.Cmd {
	ctx, done := app.beginOperation()
	return func() tea.Msg {
		events := make(chan operationEvent, 64)
		go func() {
			defer done()
			plan, err := session.Plan(ctx, &selected, false, selected, func(event progress.Event) {
				select {
				case events <- operationEvent{progress: event}:
				default:
				}
			})
			if err == nil {
				events <- operationEvent{plan: plan}
			}
			events <- operationEvent{err: err, done: true}
			close(events)
		}()
		return operationStartedMsg{events: events}
	}
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

func (s *deployConflictsScreen) Flow() string { return texts.FlowDeploy }

func (s *deployConflictsScreen) Title() string { return texts.TitleConflictos }

func (s *deployConflictsScreen) Step() (int, int) { return 3, 4 }

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

func (s *deployDetailScreen) Flow() string { return texts.FlowDeploy }

func (s *deployDetailScreen) Title() string { return texts.TitleConflictos }

func (s *deployDetailScreen) Step() (int, int) { return 3, 4 }

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
