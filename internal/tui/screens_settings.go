package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/texts"
)

type settingsScreen struct {
	screenBase
	fields     [3]string
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
		fields: [3]string{
			app.displayPath(loaded.Config.CatalogDir),
			app.displayPath(loaded.Config.CloneOutputDir),
			app.displayPath(loaded.Config.SnapshotsDir),
		},
	}
	if loaded.Exists {
		if value, ok := loaded.Raw[config.CatalogKey]; ok {
			screen.fields[0] = value
		}
		if value, ok := loaded.Raw[config.CloneKey]; ok {
			screen.fields[1] = value
		}
		if value, ok := loaded.Raw[config.SnapshotsKey]; ok {
			screen.fields[2] = value
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
func (s *settingsScreen) Keys() []key.Binding { return settingsKeys() }

func (s *settingsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "tab", "down":
		s.focus = (s.focus + 1) % 3
	case "shift+tab", "up":
		s.focus = (s.focus + 2) % 3
	case "esc":
		return s, popScreen()
	case "ctrl+s":
		s.save()
	case "backspace":
		runes := []rune(s.fields[s.focus])
		if len(runes) > 0 {
			s.fields[s.focus] = string(runes[:len(runes)-1])
		}
	default:
		if len(keyMessage.Runes) > 0 {
			s.fields[s.focus] += string(keyMessage.Runes)
		}
	}
	return s, nil
}

func (s *settingsScreen) save() {
	values := make([]string, 3)
	keys := []string{config.CatalogKey, config.CloneKey, config.SnapshotsKey}
	for index, field := range s.fields {
		resolved, err := config.ResolvePath(field, s.app.home, keys[index])
		if err != nil {
			s.errorIndex = index
			s.messages = []Message{{Kind: MessageError, Text: err.Error()}}
			return
		}
		values[index] = resolved
	}
	value := config.AppConfig{CatalogDir: values[0], CloneOutputDir: values[1], SnapshotsDir: values[2]}
	if _, err := config.SaveConfig(value, s.app.home, ""); err != nil {
		s.messages = []Message{{Kind: MessageError, Text: err.Error()}}
		return
	}
	s.errorIndex = -1
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
	labels := []string{texts.SettingsCatalog, texts.SettingsClones, texts.SettingsSnapshots}
	hints := []string{texts.SettingsCatalogHint, texts.SettingsCloneHint, texts.SettingsSnapshotsHint}
	for index, label := range labels {
		cursor := "  "
		if index == s.focus {
			cursor = s.app.theme.Symbols.Cursor + " "
		}
		lines = append(lines, "  "+cursor+s.app.theme.Styles.Secondary.Render(label))
		value := TruncatePath(s.fields[index], s.app.home, max(8, width-8), s.app.theme.Symbols.Ellipsis)
		style := s.app.theme.Styles.Secondary
		if index == s.errorIndex {
			style = s.app.theme.Styles.Error
		}
		lines = append(lines, "    "+style.Render(value+"_"))
		lines = append(lines, "    "+s.app.theme.Styles.Tertiary.Render(hints[index]))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}
