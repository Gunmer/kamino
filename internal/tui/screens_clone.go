package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
)

type appChoice struct {
	ID       string
	Name     string
	Detected bool
	Selected bool
	Count    int
	// Files lists the files the clone flow can include for the app; empty until
	// the count finishes.
	Files []fileChoice
	// Installable marks a missing app (implant) that has a recipe for this Mac.
	Installable bool
}

// fileChoice is one file of an app in the clone flow and whether it is included.
type fileChoice struct {
	Path     string
	Size     int64
	Link     bool
	Included bool
}

type checkState int

const (
	checkNone checkState = iota
	checkPartial
	checkAll
)

// includedCount returns how many files of the app are marked.
func (c appChoice) includedCount() int {
	total := 0
	for _, file := range c.Files {
		if file.Included {
			total++
		}
	}
	return total
}

// checkState derives the checkbox from the files. Without file detail (the
// count has not finished) it falls back to Selected.
func (c appChoice) checkState() checkState {
	if len(c.Files) == 0 {
		if c.Selected {
			return checkAll
		}
		return checkNone
	}
	switch included := c.includedCount(); {
	case included == 0:
		return checkNone
	case included == len(c.Files):
		return checkAll
	default:
		return checkPartial
	}
}

// setAll marks or unmarks every file and keeps Selected in sync.
func (c *appChoice) setAll(included bool) {
	for index := range c.Files {
		c.Files[index].Included = included
	}
	c.Selected = included
}

// setFile marks or unmarks one file; the app counts as selected while any file
// remains included.
func (c *appChoice) setFile(index int, included bool) {
	c.Files[index].Included = included
	c.Selected = c.includedCount() > 0
}

// includedPaths returns the marked paths, or nil when there is no file detail.
func (c appChoice) includedPaths() []string {
	if len(c.Files) == 0 {
		return nil
	}
	result := make([]string, 0, len(c.Files))
	for _, file := range c.Files {
		if file.Included {
			result = append(result, file.Path)
		}
	}
	return result
}

// stateWidth is the width of the ESTADO column of the implant apps screen.
const stateWidth = 12

// selectable reports whether the row can be marked in the implant apps screen:
// an installed app, or a missing one that kamino knows how to install.
func (c appChoice) selectable() bool { return c.Detected || c.Installable }

type appsCountedMsg struct {
	counts map[string]int
	files  map[string][]plan.FileInfo
	err    error
}

// cloneRow is one line of the C1 tree: an app (file == -1) or one of its files.
// app indexes App.cloneApps.
type cloneRow struct {
	app  int
	file int
}

type cloneAppsScreen struct {
	screenBase
	index     int
	filter    field
	filtering bool
	counting  bool
	expanded  map[string]bool
	messages  []Message
}

func newCloneAppsScreen(app *App, destinationError string) *cloneAppsScreen {
	screen := &cloneAppsScreen{
		screenBase: screenBase{app: app}, counting: true, expanded: map[string]bool{},
		filter: field{Label: texts.FieldFilter, Placeholder: texts.FieldFilterPlaceholder},
	}
	if destinationError != "" {
		screen.messages = append(screen.messages, Message{Kind: MessageError, Text: destinationError})
	}
	return screen
}

func (s *cloneAppsScreen) Flow() string        { return texts.FlowClone }
func (s *cloneAppsScreen) Title() string       { return texts.TitleApps }
func (s *cloneAppsScreen) Step() (int, int)    { return 1, 3 }
func (s *cloneAppsScreen) Messages() []Message { return s.messages }
func (s *cloneAppsScreen) Keys() []key.Binding {
	return cloneAppsKeys(s.filtering, len([]rune(s.filter.Value)))
}

func (s *cloneAppsScreen) Init() tea.Cmd {
	if !s.counting {
		return nil
	}
	return countAppsCmd(s.app)
}

func (s *cloneAppsScreen) Operation() string {
	if s.counting {
		return texts.OperationCountFiles
	}
	return ""
}

func countAppsCmd(app *App) tea.Cmd {
	definitions := make([]*catalog.AppDefinition, 0, len(app.cloneApps))
	for _, choice := range app.cloneApps {
		if !choice.Detected {
			continue
		}
		if definition, ok := app.catalog[choice.ID]; ok {
			definitions = append(definitions, definition)
		}
	}
	home := app.home
	ctx, done := app.beginOperation()
	return func() tea.Msg {
		defer done()
		counted, err := plan.CountFiles(ctx, definitions, home)
		if err != nil {
			return appsCountedMsg{err: err}
		}
		return appsCountedMsg{counts: counted.Counts, files: counted.Files}
	}
}

func (s *cloneAppsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch value := msg.(type) {
	case appsCountedMsg:
		s.counting = false
		if value.err != nil {
			s.messages = []Message{{Kind: MessageError, Text: strings.ReplaceAll(texts.C1CountError, "{message}", value.err.Error())}}
			return s, nil
		}
		for index := range s.app.cloneApps {
			choice := &s.app.cloneApps[index]
			count, ok := value.counts[choice.ID]
			if !ok {
				continue
			}
			choice.Count = count
			choice.Files = make([]fileChoice, 0, len(value.files[choice.ID]))
			for _, file := range value.files[choice.ID] {
				choice.Files = append(choice.Files, fileChoice{Path: file.Path, Size: file.Size, Link: file.Link, Included: choice.Selected})
			}
			if count == 0 {
				choice.Selected = false
			}
		}
		s.clampIndex()
		return s, nil
	}
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if s.filtering {
		switch keyMessage.String() {
		case "esc":
			s.filtering = false
			s.filter.SetValue("")
		case "enter":
			s.filtering = false
		case "backspace":
			s.filter.Backspace()
		case "left":
			s.filter.MoveCursor(-1)
		case "right":
			s.filter.MoveCursor(1)
		default:
			if len(keyMessage.Runes) > 0 {
				s.filter.Insert(string(keyMessage.Runes))
			}
		}
		return s, nil
	}
	rows := s.rows()
	switch keyMessage.String() {
	case "up", "k":
		if len(rows) > 0 {
			s.index = (s.index + len(rows) - 1) % len(rows)
		}
	case "down", "j":
		if len(rows) > 0 {
			s.index = (s.index + 1) % len(rows)
		}
	case "right", "l":
		s.expand(rows, true)
	case "left", "h":
		s.expand(rows, false)
	case "/":
		s.filtering = true
		s.filter.Cursor = len([]rune(s.filter.Value))
	case " ":
		s.toggle(rows)
	case "a":
		s.setScope(rows, true)
	case "n":
		s.setScope(rows, false)
	case "enter":
		if len(s.selectedIDs()) == 0 {
			s.messages = []Message{{Kind: MessageWarning, Text: texts.CloneNoSelection}}
			return s, nil
		}
		if s.app.cloneOptions.Output != "" && s.app.cloneOptions.OutputError == "" {
			return s, pushScreen(newCloneReviewScreen(s.app))
		}
		return s, pushScreen(newCloneDestinationScreen(s.app, ""))
	case "esc":
		return s, popScreen()
	}
	return s, nil
}

// visibleApps lists the detected apps that match the filter. Once the count has
// finished, apps without files are left out (the view reports them in a note).
func (s *cloneAppsScreen) visibleApps() []appChoice {
	result := make([]appChoice, 0)
	for _, choice := range s.app.cloneApps {
		if choice.Detected && s.matches(choice) && (s.counting || choice.Count > 0) {
			result = append(result, choice)
		}
	}
	return result
}

func (s *cloneAppsScreen) matches(choice appChoice) bool {
	if s.filter.Value == "" {
		return true
	}
	query := strings.ToLower(s.filter.Value)
	return strings.Contains(strings.ToLower(choice.ID), query) || strings.Contains(strings.ToLower(choice.Name), query)
}

// rows flattens the visible apps and the files of the expanded ones.
func (s *cloneAppsScreen) rows() []cloneRow {
	rows := make([]cloneRow, 0, len(s.app.cloneApps))
	for index, choice := range s.app.cloneApps {
		if !choice.Detected || !s.matches(choice) || (!s.counting && choice.Count == 0) {
			continue
		}
		rows = append(rows, cloneRow{app: index, file: -1})
		if s.expanded[choice.ID] {
			for file := range choice.Files {
				rows = append(rows, cloneRow{app: index, file: file})
			}
		}
	}
	return rows
}

// clampIndex keeps the cursor on a valid row after the list changes.
func (s *cloneAppsScreen) clampIndex() {
	if rows := len(s.rows()); s.index >= rows {
		s.index = max(0, rows-1)
	}
}

// expand opens or closes the focused app; on a file row, closing returns the
// cursor to its app.
func (s *cloneAppsScreen) expand(rows []cloneRow, open bool) {
	if len(rows) == 0 {
		return
	}
	row := rows[s.index%len(rows)]
	id := s.app.cloneApps[row.app].ID
	if open {
		if row.file == -1 && len(s.app.cloneApps[row.app].Files) > 0 {
			s.expanded[id] = true
		}
		return
	}
	delete(s.expanded, id)
	for position, candidate := range rows {
		if candidate.app == row.app && candidate.file == -1 {
			s.index = position
			return
		}
	}
}

// toggle flips the focused file, or every file of the focused app.
func (s *cloneAppsScreen) toggle(rows []cloneRow) {
	if len(rows) == 0 {
		return
	}
	row := rows[s.index%len(rows)]
	choice := &s.app.cloneApps[row.app]
	switch {
	case row.file >= 0:
		choice.setFile(row.file, !choice.Files[row.file].Included)
	case len(choice.Files) == 0:
		choice.Selected = !choice.Selected
	default:
		choice.setAll(choice.checkState() != checkAll)
	}
}

// setScope marks or unmarks the visible apps, or, with the cursor on a file,
// the files of that app.
func (s *cloneAppsScreen) setScope(rows []cloneRow, included bool) {
	if len(rows) == 0 {
		return
	}
	row := rows[s.index%len(rows)]
	if row.file >= 0 {
		s.app.cloneApps[row.app].setAll(included)
		return
	}
	for _, candidate := range rows {
		if candidate.file != -1 {
			continue
		}
		choice := &s.app.cloneApps[candidate.app]
		if len(choice.Files) == 0 {
			choice.Selected = included
			continue
		}
		choice.setAll(included)
	}
}

func (s *cloneAppsScreen) selectedIDs() []string {
	return s.app.selectedCloneIDs()
}

func (s *cloneAppsScreen) View(width, height int) string {
	visible := s.visibleApps()
	rows := s.rows()
	selected := 0
	for _, choice := range visible {
		if choice.Selected && choice.Count > 0 {
			selected++
		}
	}
	counter := strings.NewReplacer(
		"{selected}", fmt.Sprintf("%d", selected),
		"{total}", fmt.Sprintf("%d", len(visible)),
	).Replace(texts.C1Counter)
	title := texts.C1Title
	if s.counting {
		title = texts.C1CountLoading
	}
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(title) + "    " + s.app.theme.Styles.Tertiary.Render(counter),
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
	if tableWidth > 60 {
		tableWidth = 60
	}
	appWidth := tableWidth - 17
	if appWidth < 10 {
		appWidth = 10
	}
	lines = append(lines, "  "+PadRight(texts.C1ColumnSelect, 3)+"  "+PadRight(texts.C1ColumnApp, appWidth)+"  "+PadLeft(texts.C1ColumnFiles, 8))
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, tableWidth-4))))
	notDetected := make([]string, 0)
	noFiles := make([]string, 0)
	for _, choice := range s.app.cloneApps {
		switch {
		case !choice.Detected:
			notDetected = append(notDetected, choice.Name)
		case !s.counting && choice.Count == 0:
			noFiles = append(noFiles, choice.Name)
		}
	}
	maxRows := height - 7
	if len(noFiles) > 0 {
		maxRows--
	}
	if maxRows < 3 {
		maxRows = 3
	}
	start, end := window(s.index, len(rows), maxRows)
	if start > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1ScrollUp, "{count}", fmt.Sprintf("%d", start)))
	}
	for index := start; index < end; index++ {
		lines = append(lines, s.renderRow(rows[index], index == s.index, appWidth))
	}
	if end < len(rows) {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1ScrollDown, "{count}", fmt.Sprintf("%d", len(rows)-end)))
	}
	lines = append(lines, "")
	if len(notDetected) > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1NotDetected, "{apps}", strings.Join(notDetected, ", ")))
	}
	if len(noFiles) > 0 {
		lines = append(lines, "  "+strings.ReplaceAll(texts.C1NoFilesNote, "{apps}", strings.Join(noFiles, ", ")))
	}
	if len(visible) == 0 && !s.counting {
		lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(texts.C1NoApps))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

// renderRow draws an app row or, indented under it, one of its files.
func (s *cloneAppsScreen) renderRow(row cloneRow, focused bool, appWidth int) string {
	choice := s.app.cloneApps[row.app]
	cursor := "  "
	if focused {
		cursor = s.app.theme.Symbols.Cursor + " "
	}
	style := s.app.theme.Styles.Secondary
	if focused {
		style = s.app.theme.Styles.Primary
	}
	ellipsis := s.app.theme.Symbols.Ellipsis
	if row.file >= 0 {
		file := choice.Files[row.file]
		checkbox := "[ ]"
		if file.Included {
			checkbox = "[x]"
		}
		path := TruncatePath(file.Path, "", appWidth-2, ellipsis)
		return style.Render(cursor + "  " + checkbox + "  " + PadRight(path, appWidth-2) + "  " + PadLeft(texts.SizeLabel(file.Size), 8))
	}
	checkbox := "[ ]"
	state := choice.checkState()
	switch state {
	case checkAll:
		checkbox = "[x]"
	case checkPartial:
		checkbox = "[~]"
	}
	count := fmt.Sprintf("%d", choice.Count)
	switch {
	case s.counting:
		count = ellipsis
	case state == checkPartial:
		count = fmt.Sprintf("%d/%d", choice.includedCount(), len(choice.Files))
	}
	return style.Render(cursor + checkbox + "  " + PadRight(Truncate(choice.Name, appWidth, ellipsis), appWidth) + "  " + PadLeft(count, 8))
}

type cloneDestinationScreen struct {
	screenBase
	option       int
	focus        int
	path         field
	name         field
	vault        string
	dialog       bool
	dialogIndex  int
	existingAt   string
	existingMod  string
	existingSize string
	messages     []Message
}

func newCloneDestinationScreen(app *App, outputError string) *cloneDestinationScreen {
	configValue := config.LoadConfig(app.home, "").Config
	vault := configValue.CloneOutputDir
	if vault == "" {
		resolved, err := defaults.VaultDir(app.home)
		if err != nil {
			outputError = err.Error()
		}
		vault = resolved
	}
	screen := &cloneDestinationScreen{
		screenBase: screenBase{app: app},
		vault:      vault,
		path:       field{Label: texts.FieldDestination, Placeholder: texts.FieldPathPlaceholder},
		name:       field{Label: texts.FieldName, Placeholder: texts.FieldNamePlaceholder},
	}
	screen.name.SetValue(defaultCloneName(app.now()))
	screen.path.SetValue(vault)
	if app.cloneOptions.Output != "" {
		output := app.cloneOptions.Output
		if outputError != "" {
			screen.option = 2
			screen.focus = 2
			screen.path.SetValue(filepath.Dir(output))
			screen.name.SetValue(filepath.Base(output))
			screen.messages = []Message{{Kind: MessageError, Text: outputError}}
		} else {
			screen.path.SetValue(filepath.Dir(output))
			screen.name.SetValue(filepath.Base(output))
			screen.option = 2
		}
	}
	if len(screen.messages) == 0 {
		screen.messages = screen.persistentMessages()
	}
	return screen
}

func (s *cloneDestinationScreen) Flow() string        { return texts.FlowClone }
func (s *cloneDestinationScreen) Title() string       { return texts.TitleDestino }
func (s *cloneDestinationScreen) Step() (int, int)    { return 2, 3 }
func (s *cloneDestinationScreen) Messages() []Message { return s.messages }
func (s *cloneDestinationScreen) Keys() []key.Binding {
	length := 0
	if s.focus == 1 {
		length = len([]rune(s.name.Value))
	}
	if s.focus == 2 {
		length = len([]rune(s.path.Value))
	}
	return cloneDestinationKeys(s.option, s.focus, s.dialog, length)
}

func (s *cloneDestinationScreen) directory() string {
	if s.option == 1 {
		return s.app.cwd
	}
	if s.option == 2 {
		return s.path.Value
	}
	return s.vault
}

func (s *cloneDestinationScreen) fullPath() string {
	directory := s.directory()
	if directory == "" {
		return ""
	}
	return filepath.Join(directory, s.name.Value)
}

func (s *cloneDestinationScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if s.dialog {
		return s.updateDialog(keyMessage)
	}
	switch keyMessage.String() {
	case "up", "k":
		if s.focus == 0 {
			s.option = (s.option + 2) % 3
		}
	case "down", "j":
		if s.focus == 0 {
			s.option = (s.option + 1) % 3
		}
	case "tab":
		s.cycleFocus(1)
	case "shift+tab":
		s.cycleFocus(-1)
	case "esc":
		return s, popScreen()
	case "enter":
		return s.accept()
	case "left":
		s.moveCursor(-1)
	case "right":
		s.moveCursor(1)
	case "backspace":
		s.backspace()
	default:
		if len(keyMessage.Runes) > 0 {
			s.insert(string(keyMessage.Runes))
		}
	}
	s.validate()
	return s, nil
}

func (s *cloneDestinationScreen) cycleFocus(direction int) {
	limit := 1
	if s.option == 2 {
		limit = 2
	}
	s.focus = (s.focus + direction + limit + 1) % (limit + 1)
}

func (s *cloneDestinationScreen) insert(value string) {
	switch s.focus {
	case 1:
		s.name.Insert(value)
	case 2:
		s.path.Insert(value)
	}
}

func (s *cloneDestinationScreen) backspace() {
	switch s.focus {
	case 1:
		s.name.Backspace()
	case 2:
		s.path.Backspace()
	}
}

func (s *cloneDestinationScreen) moveCursor(delta int) {
	switch s.focus {
	case 1:
		s.name.MoveCursor(delta)
	case 2:
		s.path.MoveCursor(delta)
	}
}

func (s *cloneDestinationScreen) accept() (Screen, tea.Cmd) {
	if s.focus == 0 {
		if s.option == 2 {
			s.focus = 2
			return s, nil
		}
		s.focus = 1
		s.validate()
		return s, nil
	}
	if s.focus == 2 {
		if strings.TrimSpace(s.path.Value) == "" {
			s.path.Invalid = true
			s.messages = append(s.persistentMessages(), Message{Kind: MessageError, Text: texts.C2EmptyPath})
			return s, nil
		}
		s.path.Invalid = false
		s.focus = 1
		return s, nil
	}
	full := s.fullPath()
	if full == "" || strings.TrimSpace(s.name.Value) == "" {
		s.name.Invalid = true
		s.messages = append(s.persistentMessages(), Message{Kind: MessageError, Text: texts.C2EmptyPath})
		return s, nil
	}
	if message := clone.ValidateDestination(full, true); message != "" {
		s.name.Invalid = true
		s.messages = append(s.persistentMessages(), Message{Kind: MessageError, Text: message})
		return s, nil
	}
	s.name.Invalid = false
	if !s.app.cloneForce {
		if info, err := os.Lstat(full); err == nil && !info.IsDir() {
			s.dialog = true
			s.dialogIndex = 2
			s.existingAt = full
			s.existingMod = s.app.formatDate(info.ModTime())
			s.existingSize = texts.SizeLabel(info.Size())
			return s, nil
		}
	}
	s.app.cloneOutput = full
	return s, pushScreen(newCloneReviewScreen(s.app))
}

func (s *cloneDestinationScreen) updateDialog(keyMessage tea.KeyMsg) (Screen, tea.Cmd) {
	switch keyMessage.String() {
	case "up", "k":
		s.dialogIndex = (s.dialogIndex + 2) % 3
	case "down", "j":
		s.dialogIndex = (s.dialogIndex + 1) % 3
	case "esc":
		s.dialog = false
		s.focus = 1
	case "enter":
		switch s.dialogIndex {
		case 0:
			s.app.cloneForce = true
			s.dialog = false
			s.app.cloneOutput = s.fullPath()
			return s, pushScreen(newCloneReviewScreen(s.app))
		case 1:
			if free, err := clone.FreeSuffix(s.fullPath()); err == nil {
				s.name.SetValue(filepath.Base(free))
			}
			s.dialog = false
			s.focus = 1
		case 2:
			s.dialog = false
			s.focus = 1
		}
	}
	return s, nil
}

func (s *cloneDestinationScreen) validate() {
	if s.focus == 0 {
		return
	}
	s.path.Invalid = false
	s.name.Invalid = false
	s.messages = s.persistentMessages()
	full := s.fullPath()
	if value := clone.ValidateDestination(full, true); value != "" {
		if s.focus == 2 {
			s.path.Invalid = true
		} else {
			s.name.Invalid = true
		}
		s.messages = append(s.messages, Message{Kind: MessageError, Text: value})
	}
}

func (s *cloneDestinationScreen) persistentMessages() []Message {
	messages := []Message{{Kind: MessageWarning, Text: texts.SecretsShort}}
	if s.app.cloneForce {
		messages = append(messages, Message{Kind: MessageWarning, Text: texts.C2ForceNotice})
	}
	return messages
}

func (s *cloneDestinationScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(texts.CloneDestinationTitle),
		"",
		"  " + s.app.theme.Styles.Secondary.Render(texts.C2FolderTitle),
	}
	vault := s.vault
	options := []string{texts.C2VaultOption, texts.C2CurrentOption, texts.C2ManualOption}
	values := []string{s.app.displayPath(vault) + "/", s.app.displayPath(s.app.cwd), ""}
	for index, option := range options {
		marker := s.app.theme.Symbols.NotChosen
		style := s.app.theme.Styles.Secondary
		if index == s.option {
			marker = s.app.theme.Symbols.Chosen
			style = s.app.theme.Styles.Primary
		}
		cursor := "  "
		if s.focus == 0 && index == s.option {
			cursor = s.app.theme.Symbols.Cursor + " "
		}
		line := cursor + marker + " " + PadRight(option, 22)
		if values[index] != "" {
			line += s.app.theme.Styles.Tertiary.Render(TruncatePath(values[index], s.app.home, max(4, width-Width(line)-4), s.app.theme.Symbols.Ellipsis))
		}
		lines = append(lines, style.Render(line))
	}
	fieldOpts := fieldOptions{
		Width:      max(1, width-2),
		LabelWidth: fieldLabelWidth(s.app.theme, texts.FieldDestination, texts.FieldName),
	}
	lines = append(lines, "")
	if s.option == 2 {
		fieldOpts.Focused = s.focus == 2 && !s.dialog
		lines = append(lines, "  "+s.path.Render(s.app.theme, fieldOpts))
	}
	fieldOpts.Focused = s.focus == 1 && !s.dialog
	lines = append(lines, "  "+s.name.Render(s.app.theme, fieldOpts))
	lines = append(lines, "", "  "+strings.ReplaceAll(texts.C2PathLine, "{path}", s.app.displayPath(s.fullPath())))
	if s.dialog {
		lines = append(lines, "", "  "+s.app.theme.Styles.Warning.Render(texts.C2ExistingTitle))
		lines = append(lines, "  "+strings.ReplaceAll(texts.C2PathLine, "{path}", s.app.displayPath(s.existingAt)))
		lines = append(lines, "  "+strings.ReplaceAll(texts.C2ExistingDate, "{date}", s.existingMod)+"   "+strings.ReplaceAll(texts.C2ExistingSize, "{size}", s.existingSize))
		lines = append(lines, "")
		dialogOptions := []string{texts.C2Overwrite, texts.C2Rename, texts.C2Back}
		for index, option := range dialogOptions {
			if index == s.dialogIndex {
				lines = append(lines, "  "+s.app.theme.Styles.Primary.Render(s.app.theme.Symbols.Cursor+" "+option))
			} else {
				lines = append(lines, "    "+option)
			}
		}
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

type clonePhase int

const (
	clonePhasePlanning clonePhase = iota
	clonePhaseReady
	clonePhaseWriting
	clonePhaseDone
	clonePhaseCancelled
)

type clonePlanDoneMsg struct {
	plan *clone.ClonePlan
	err  error
}

type cloneReviewScreen struct {
	screenBase
	phase       clonePhase
	plan        *clone.ClonePlan
	result      *clone.CloneResult
	showSkipped bool
	showFiles   bool
	current     int
	total       int
	events      <-chan operationEvent
	messages    []Message
}

func newCloneReviewScreen(app *App) *cloneReviewScreen {
	screen := &cloneReviewScreen{screenBase: screenBase{app: app}, plan: app.clonePlan, result: app.cloneResult}
	switch {
	case app.cloneResult != nil:
		screen.phase = clonePhaseDone
	case app.clonePlan != nil:
		screen.phase = clonePhaseReady
	default:
		screen.phase = clonePhasePlanning
	}
	return screen
}

func (s *cloneReviewScreen) Flow() string        { return texts.FlowClone }
func (s *cloneReviewScreen) Step() (int, int)    { return 3, 3 }
func (s *cloneReviewScreen) Messages() []Message { return s.messages }
func (s *cloneReviewScreen) Keys() []key.Binding {
	return cloneReviewKeys(s.phase, s.hasSkipped(), s.canCopy())
}

func (s *cloneReviewScreen) Title() string {
	if s.phase == clonePhaseDone {
		return texts.TitleResultado
	}
	return texts.TitleRevision
}

func (s *cloneReviewScreen) Operation() string {
	if s.phase == clonePhaseWriting {
		return strings.NewReplacer("{current}", fmt.Sprintf("%d", s.current), "{total}", fmt.Sprintf("%d", s.total)).Replace(texts.OperationCloneWriting)
	}
	return ""
}

func (s *cloneReviewScreen) hasSkipped() bool {
	return s.plan != nil && len(s.plan.Plan.Skipped) > 0 || s.result != nil && len(s.result.Plan.Skipped) > 0
}

func (s *cloneReviewScreen) canCopy() bool {
	return s.phase == clonePhaseDone && s.result != nil && !s.result.DryRun && s.result.Output != ""
}

func (s *cloneReviewScreen) Init() tea.Cmd {
	if s.phase == clonePhasePlanning {
		return planCloneCmd(s.app)
	}
	return nil
}

func planCloneCmd(app *App) tea.Cmd {
	selected := app.selectedCloneIDs()
	options := clone.PlanOptions{
		Home: app.home, Selected: &selected, Files: app.selectedCloneFiles(), Output: app.cloneOutput, Catalog: app.catalog,
		ApplicationDirs: app.applicationDirs, Path: app.pathValue, Now: app.now(),
	}
	ctx, done := app.beginOperation()
	return func() tea.Msg {
		defer done()
		planned, err := clone.PlanClone(ctx, options)
		return clonePlanDoneMsg{plan: planned, err: err}
	}
}

func (s *cloneReviewScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch value := msg.(type) {
	case tea.KeyMsg:
		return s.updateKey(value)
	case clonePlanDoneMsg:
		if value.err != nil {
			if cancelled(value.err) {
				s.phase = clonePhaseCancelled
				s.messages = []Message{{Kind: MessageWarning, Text: texts.C3CancelledInfo}}
				return s, nil
			}
			return s, showError(texts.ErrorPlanningClone, texts.ErrorNothingWritten, texts.StatusReviewDefault, value.err.Error())
		}
		s.plan = value.plan
		s.app.clonePlan = value.plan
		s.phase = clonePhaseReady
		s.messages = []Message{{Kind: MessageWarning, Text: texts.SecretsShort}}
		return s, nil
	case operationStartedMsg:
		s.events = value.events
		return s, waitOperation(value.events)
	case operationEvent:
		return s.handleOperation(value)
	}
	return s, nil
}

func (s *cloneReviewScreen) updateKey(keyMessage tea.KeyMsg) (Screen, tea.Cmd) {
	switch keyMessage.String() {
	case "ctrl+c":
		switch s.phase {
		case clonePhasePlanning:
			s.app.cancelOperation()
			s.messages = []Message{{Kind: MessageWarning, Text: texts.CloneCancelling}}
		case clonePhaseReady:
			s.phase = clonePhaseCancelled
			s.messages = []Message{{Kind: MessageWarning, Text: texts.C3CancelledInfo}}
		case clonePhaseWriting:
			s.app.cancelOperation()
			s.messages = []Message{{Kind: MessageWarning, Text: texts.CloneCancelling}}
		}
		return s, nil
	}
	switch s.phase {
	case clonePhaseWriting:
		return s, nil
	case clonePhaseCancelled:
		return s, popScreen()
	case clonePhaseDone:
		switch keyMessage.String() {
		case "d":
			s.showSkipped = !s.showSkipped
		case "c":
			if s.canCopy() {
				s.messages = []Message{{Kind: MessageInfo, Text: texts.C3CopyInfo}}
				return s, s.app.copyCommand(s.result.Output, texts.C3CopyInfo, texts.C3CopyError)
			}
		case "enter", "esc":
			s.app.finishClone()
			return s, nil
		}
	case clonePhaseReady:
		switch keyMessage.String() {
		case "d":
			s.showSkipped = !s.showSkipped
		case "f":
			s.showFiles = !s.showFiles
		case "esc":
			return s, popScreen()
		case "enter":
			if s.app.cloneDryRun {
				result := clone.CloneResult{Plan: s.plan.Plan, Manifest: s.plan.Manifest, DryRun: true, PlanOutput: s.plan.Output}
				s.result = &result
				s.app.cloneResult = &result
				s.phase = clonePhaseDone
				s.messages = nil
				return s, nil
			}
			s.phase = clonePhaseWriting
			s.current = 0
			s.total = s.plan.FileCount()
			s.messages = []Message{{Kind: MessageWarning, Text: texts.C3WritingWarning}}
			return s, startCloneWrite(s.app, s.plan)
		}
	case clonePhasePlanning:
		if keyMessage.String() == "esc" {
			s.app.cancelOperation()
			return s, popScreen()
		}
	}
	return s, nil
}

func (s *cloneReviewScreen) handleOperation(event operationEvent) (Screen, tea.Cmd) {
	if event.progress.Total > 0 {
		s.current = event.progress.Current
		s.total = event.progress.Total
	}
	if event.clone != nil {
		s.result = event.clone
		s.app.cloneResult = event.clone
	}
	if !event.done {
		if s.events != nil {
			return s, waitOperation(s.events)
		}
		return s, nil
	}
	s.phase = clonePhaseDone
	if event.err != nil {
		if cancelled(event.err) {
			s.phase = clonePhaseCancelled
			s.messages = []Message{{Kind: MessageWarning, Text: texts.C3CancelledInfo}}
			return s, nil
		}
		return s, showError(texts.ErrorWritingClone, texts.ErrorCloneWritten, texts.StatusReviewDefault, event.err.Error())
	}
	if s.result != nil {
		if len(s.result.Plan.Skipped) > 0 {
			s.messages = []Message{{Kind: MessageWarning, Text: strings.ReplaceAll(texts.C3ExitWarning, "{count}", fmt.Sprintf("%d", len(s.result.Plan.Skipped)))}}
		} else {
			s.messages = nil
		}
	}
	return s, nil
}

func startCloneWrite(app *App, plan *clone.ClonePlan) tea.Cmd {
	ctx, done := app.beginOperation()
	return func() tea.Msg {
		events := make(chan operationEvent, 64)
		go func() {
			defer done()
			result, err := clone.WriteClone(ctx, plan, app.home, app.cloneForce, nil, func(event progress.Event) {
				select {
				case events <- operationEvent{progress: event}:
				default:
				}
			})
			if err == nil {
				events <- operationEvent{clone: &result}
			}
			events <- operationEvent{err: err, done: true}
			close(events)
		}()
		return operationStartedMsg{events: events}
	}
}

func (s *cloneReviewScreen) View(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.C3TitleReview)}
	switch s.phase {
	case clonePhasePlanning:
		lines = append(lines, "  "+texts.C3ReviewBody)
	case clonePhaseWriting:
		lines = append(lines, "  "+texts.C3WritingBody)
	case clonePhaseDone:
		lines = append(lines, "  "+texts.C3DoneBody)
	default:
		lines = append(lines, "  "+texts.C3ReviewBody)
	}
	lines = append(lines, "", "  "+s.app.theme.Styles.Secondary.Render(texts.C3PlanTitle), "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, tableWidth(width)-4))))
	plan := s.plan
	if s.result != nil {
		plan = &clone.ClonePlan{Plan: s.result.Plan, Manifest: s.result.Manifest, Output: s.result.Output}
	}
	tableW := tableWidth(width)
	if plan == nil {
		lines = append(lines, s.planRow(tableW, texts.C3Apps, "-"), s.planRow(tableW, texts.C3Files, "-"), s.planRow(tableW, texts.C3Size, "-"), s.planRow(tableW, texts.C3Path, "-"), s.stateRow(width))
		return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
	}
	appCount := len(plan.Apps)
	if s.result != nil && s.result.Manifest != nil {
		appCount = len(s.result.Manifest.Apps)
	}
	lines = append(lines,
		s.planRow(tableW, texts.C3Apps, fmt.Sprintf("%d", appCount)),
		s.planRow(tableW, texts.C3Files, fmt.Sprintf("%d", plan.FileCount())),
		s.planRow(tableW, texts.C3Size, texts.SizeLabel(plan.Size())),
		s.planRow(tableW, texts.C3Path, TruncatePath(s.app.displayPath(plan.Output), s.app.home, max(8, tableW-16), s.app.theme.Symbols.Ellipsis)),
		s.stateRow(width),
	)
	lines = append(lines, "")
	if s.phase == clonePhaseReady {
		lines = append(lines, "  + "+strings.ReplaceAll(texts.CloneSkippedTitle, "", "")+" ("+fmt.Sprintf("%d", len(plan.Plan.Skipped))+")     + "+texts.CloneIncludedTitle+" ("+fmt.Sprintf("%d", plan.FileCount())+")")
	}
	if s.phase == clonePhaseDone {
		if len(plan.Plan.Skipped) > 0 {
			lines = append(lines, "  d "+texts.CloneSkippedTitle+" ("+fmt.Sprintf("%d", len(plan.Plan.Skipped))+")")
		}
		if s.canCopy() {
			lines = append(lines, "  c "+texts.KeyCopyPath)
		}
	}
	if s.showSkipped {
		lines = append(lines, "", "  "+s.app.theme.Styles.Secondary.Render(texts.CloneSkippedTitle))
		for _, skipped := range plan.Plan.Skipped {
			lines = append(lines, "  "+strings.NewReplacer("{path}", skipped.Path, "{reason}", skipped.Reason).Replace(texts.C3SkippedReason))
		}
	}
	if s.showFiles {
		lines = append(lines, "", "  "+s.app.theme.Styles.Secondary.Render(texts.CloneIncludedTitle))
		currentApp := ""
		for _, operation := range plan.Plan.Operations {
			if operation.AppID != currentApp {
				currentApp = operation.AppID
				name := currentApp
				if definition, ok := s.app.catalog[currentApp]; ok && definition.Name != "" {
					name = definition.Name
				}
				lines = append(lines, "  "+s.app.theme.Styles.Secondary.Render(name))
			}
			lines = append(lines, strings.ReplaceAll(texts.C3IncludedPath, "{path}", operation.Path))
		}
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *cloneReviewScreen) planRow(width int, label, value string) string {
	return "  " + PadRight(label, 10) + PadLeft(Truncate(value, max(4, width-14), s.app.theme.Symbols.Ellipsis), max(4, width-12))
}

func tableWidth(width int) int {
	if width > 72 {
		return 72
	}
	return width
}

func (s *cloneReviewScreen) stateRow(width int) string {
	value := texts.C3Planning
	switch s.phase {
	case clonePhaseReady:
		value = texts.C3Ready
	case clonePhaseWriting:
		value = strings.NewReplacer("{current}", fmt.Sprintf("%d", s.current), "{total}", fmt.Sprintf("%d", s.total)).Replace(texts.C3WritingFiles)
		barWidth := width - 50
		if barWidth > 30 {
			barWidth = 30
		}
		if barWidth < 8 {
			barWidth = 8
		}
		value += "   " + progressBar(s.current, s.total, barWidth)
	case clonePhaseDone:
		value = texts.C3Completed
		if s.result != nil && s.result.DryRun {
			value = texts.C3DryRunDone
		}
	case clonePhaseCancelled:
		value = texts.C3Cancelled
	}
	return "  " + PadRight(texts.C3State, 10) + "  " + Truncate(value, max(4, width-14), s.app.theme.Symbols.Ellipsis)
}

func progressBar(current, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	percent := current * 100 / total
	inner := width - 2
	if inner < 4 {
		inner = 4
	}
	filled := inner * current / total
	if filled > inner {
		filled = inner
	}
	bar := strings.Repeat("=", filled)
	if filled < inner {
		bar += ">"
		bar += strings.Repeat(" ", inner-filled-1)
	}
	return fmt.Sprintf("[%s] %d%%", bar, percent)
}

func window(index, count, rows int) (int, int) {
	if rows <= 0 || count == 0 {
		return 0, 0
	}
	if index >= count {
		index = count - 1
	}
	start := index - rows + 1
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > count {
		end = count
		start = end - rows
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func (a *App) selectedCloneIDs() []string {
	result := make([]string, 0)
	for _, choice := range a.cloneApps {
		if choice.Selected && choice.Detected {
			result = append(result, choice.ID)
		}
	}
	return result
}

// selectedCloneFiles maps each selected app that has file detail to its marked
// paths. Apps without detail (a clone started with --apps) include every file.
func (a *App) selectedCloneFiles() map[string][]string {
	result := map[string][]string{}
	for _, choice := range a.cloneApps {
		if choice.Selected && choice.Detected && len(choice.Files) > 0 {
			result[choice.ID] = choice.includedPaths()
		}
	}
	return result
}
