package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/texts"
)

const (
	// pathFieldCount is how many leading fields hold editable paths.
	pathFieldCount = 3
	// limitField is the index of the snapshot limit, after the paths.
	limitField = pathFieldCount
	fieldCount = pathFieldCount + 1
	// authoringRow is the action row after the editable fields.
	authoringRow = fieldCount
	// customizeRow follows the authoring row.
	customizeRow = fieldCount + 1
	rowCount     = fieldCount + 2
)

type settingsScreen struct {
	screenBase
	fields     [fieldCount]field
	focus      int
	errorIndex int
	messages   []Message
}

func newSettingsScreen(app *App, initialError string) *settingsScreen {
	loaded := config.LoadConfig(app.home, "")
	app.settingsLoad = loaded
	app.settings = loaded.Config
	screen := &settingsScreen{
		screenBase: screenBase{app: app},
		errorIndex: -1,
		fields: [fieldCount]field{
			{Label: texts.SettingsCatalog, Placeholder: texts.FieldPathPlaceholder},
			{Label: texts.SettingsClones, Placeholder: texts.FieldPathPlaceholder},
			{Label: texts.SettingsSnapshots, Placeholder: texts.FieldPathPlaceholder},
			{Label: texts.FieldLimit},
		},
	}
	screen.fields[0].SetValue(app.displayPath(loaded.Config.CatalogDir))
	screen.fields[1].SetValue(app.displayPath(loaded.Config.CloneOutputDir))
	screen.fields[2].SetValue(app.displayPath(loaded.Config.SnapshotsDir))
	screen.fields[limitField].SetValue(strconv.Itoa(loaded.Config.SnapshotsLimit))
	if loaded.Exists {
		if value, ok := loaded.Raw[config.CatalogKey]; ok {
			screen.fields[0].SetValue(value)
		}
		if value, ok := loaded.Raw[config.CloneKey]; ok {
			screen.fields[1].SetValue(value)
		}
		if value, ok := loaded.Raw[config.SnapshotsKey]; ok {
			screen.fields[2].SetValue(value)
		}
		if value, ok := loaded.Raw[config.SnapshotsLimitKey]; ok {
			screen.fields[limitField].SetValue(value)
		}
	}
	if initialError != "" {
		screen.messages = append(screen.messages, Message{Kind: MessageError, Text: initialError})
	} else if loaded.Error != nil {
		screen.messages = append(screen.messages, Message{Kind: MessageError, Text: loaded.Error.Error()})
	}
	return screen
}

func (s *settingsScreen) Flow() string        { return texts.FlowSettings }
func (s *settingsScreen) Title() string       { return texts.FlowSettings }
func (s *settingsScreen) Step() (int, int)    { return 0, 0 }
func (s *settingsScreen) Messages() []Message { return s.messages }
func (s *settingsScreen) Keys() []key.Binding {
	if s.focus >= authoringRow {
		return settingsAuthoringKeys()
	}
	return settingsKeys(len([]rune(s.fields[s.focus].Value)))
}

func (s *settingsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "tab", "down":
		s.focus = (s.focus + 1) % rowCount
	case "shift+tab", "up":
		s.focus = (s.focus + rowCount - 1) % rowCount
	case "esc":
		return s, popScreen()
	case "ctrl+s":
		s.save()
	case "enter":
		switch s.focus {
		case authoringRow:
			return s, func() tea.Msg { return startAuthoringMsg{} }
		case customizeRow:
			return s, pushScreen(newCustomizeSelectScreen(s.app))
		}
	case "left":
		if s.focus < fieldCount {
			s.fields[s.focus].MoveCursor(-1)
		}
	case "right":
		if s.focus < fieldCount {
			s.fields[s.focus].MoveCursor(1)
		}
	case "backspace":
		if s.focus < fieldCount {
			s.fields[s.focus].Backspace()
			s.fields[s.focus].Invalid = false
		}
	default:
		if s.focus >= fieldCount {
			break
		}
		runes := keyMessage.Runes
		if s.focus == limitField {
			digits := make([]rune, 0, len(runes))
			for _, value := range runes {
				if value >= '0' && value <= '9' {
					digits = append(digits, value)
				}
			}
			runes = digits
		}
		if len(runes) == 0 {
			break
		}
		s.fields[s.focus].Insert(string(runes))
		s.fields[s.focus].Invalid = false
	}
	return s, nil
}

func (s *settingsScreen) save() {
	for index := range s.fields {
		s.fields[index].Invalid = false
	}
	values := make([]string, pathFieldCount)
	keys := []string{config.CatalogKey, config.CloneKey, config.SnapshotsKey}
	for index := range s.fields[:pathFieldCount] {
		resolved, err := config.ResolvePath(s.fields[index].Value, s.app.home, keys[index])
		if err != nil {
			s.errorIndex = index
			s.fields[index].Invalid = true
			s.messages = []Message{{Kind: MessageError, Text: err.Error()}}
			return
		}
		values[index] = resolved
	}
	limit, err := config.ParseSnapshotsLimit(s.fields[limitField].Value)
	if err != nil {
		s.errorIndex = limitField
		s.fields[limitField].Invalid = true
		s.messages = []Message{{Kind: MessageError, Text: err.Error()}}
		return
	}
	value := config.AppConfig{CatalogDir: values[0], CloneOutputDir: values[1], SnapshotsDir: values[2], SnapshotsLimit: limit}
	previousCatalog := s.app.settings.CatalogDir
	if _, err := config.SaveConfig(value, s.app.home, ""); err != nil {
		s.messages = []Message{{Kind: MessageError, Text: err.Error()}}
		return
	}
	if previousCatalog != value.CatalogDir {
		s.app.catalog = nil
	}
	s.errorIndex = -1
	s.fields[limitField].SetValue(strconv.Itoa(limit))
	s.app.settings = value
	s.app.lastMission = texts.SettingsSavedInfo
	s.messages = []Message{{Kind: MessageInfo, Text: texts.SettingsSavedInfo}}
}

func (s *settingsScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(texts.SettingsTitle),
		"  " + s.app.theme.Styles.Tertiary.Render(texts.SettingsSubtitleTUI),
		"",
	}
	labels := make([]string, 0, fieldCount+2)
	for index := range s.fields {
		labels = append(labels, s.fields[index].Label)
	}
	labels = append(labels, texts.SettingsAddApp, texts.SettingsCustomizeApp)
	options := fieldOptions{
		Width:      max(1, width-2),
		LabelWidth: fieldLabelWidth(s.app.theme, labels...),
	}
	for index := range s.fields {
		options.Focused = index == s.focus
		lines = append(lines, "  "+s.fields[index].Render(s.app.theme, options))
	}
	lines = append(lines, "",
		"  "+s.renderActionRow(authoringRow, texts.SettingsAddApp, texts.SettingsAddAppHelp, options),
		"  "+s.renderActionRow(customizeRow, texts.SettingsCustomizeApp, texts.SettingsCustomizeHelp, options))
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *settingsScreen) renderActionRow(row int, labelText, helpText string, options fieldOptions) string {
	marker := "  "
	labelStyle := s.app.theme.Styles.Secondary
	valueStyle := s.app.theme.Styles.Tertiary
	if s.focus == row {
		marker = s.app.theme.Symbols.Cursor + " "
		labelStyle = s.app.theme.Styles.Primary
		valueStyle = s.app.theme.Styles.Secondary
	}
	labelWidth := options.LabelWidth
	if limit := options.Width - Width(marker) - Width(fieldGap) - 1; labelWidth > limit {
		labelWidth = limit
	}
	if labelWidth < 0 {
		labelWidth = 0
	}
	label := PadRight(fieldTruncate(labelText, labelWidth, s.app.theme.Symbols.Ellipsis), labelWidth)
	valueWidth := options.Width - Width(marker) - labelWidth - Width(fieldGap)
	if valueWidth < 1 {
		valueWidth = 1
	}
	value := fieldTruncate(helpText, valueWidth, s.app.theme.Symbols.Ellipsis)
	return marker + labelStyle.Render(label) + fieldGap + valueStyle.Render(value)
}
