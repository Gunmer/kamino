package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/texts"
)

// customizeSelectScreen lists the built-in apps so one can be copied into the
// user catalog as an editable draft.
type customizeSelectScreen struct {
	screenBase
	apps     []appChoice
	cursor   int
	messages []Message
}

func newCustomizeSelectScreen(app *App) *customizeSelectScreen {
	screen := &customizeSelectScreen{screenBase: screenBase{app: app}}
	embedded, err := catalog.Embedded()
	if err != nil {
		screen.messages = []Message{{Kind: MessageError, Text: err.Error()}}
		return screen
	}
	for _, id := range embedded.SortedIDs() {
		screen.apps = append(screen.apps, appChoice{ID: id, Name: embedded[id].Name})
	}
	return screen
}

func (s *customizeSelectScreen) Flow() string        { return texts.FlowSettings }
func (s *customizeSelectScreen) Title() string       { return texts.FlowCustomize }
func (s *customizeSelectScreen) Step() (int, int)    { return 0, 0 }
func (s *customizeSelectScreen) Messages() []Message { return s.messages }

func (s *customizeSelectScreen) Keys() []key.Binding {
	return []key.Binding{
		bind([]string{"up", "down"}, "↑/↓", texts.KeyMove),
		bind([]string{"enter"}, "Enter", texts.KeyChoose),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func (s *customizeSelectScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "up":
		if len(s.apps) > 0 {
			s.cursor = (s.cursor + len(s.apps) - 1) % len(s.apps)
		}
	case "down":
		if len(s.apps) > 0 {
			s.cursor = (s.cursor + 1) % len(s.apps)
		}
	case "enter":
		if len(s.apps) > 0 {
			id := s.apps[s.cursor].ID
			return s, func() tea.Msg { return startCustomizeMsg{id: id} }
		}
	case "esc":
		return s, popScreen()
	}
	return s, nil
}

func (s *customizeSelectScreen) View(width, height int) string {
	theme := s.app.theme
	lines := []string{
		"  " + theme.Styles.Primary.Render(texts.CustomizeHeading),
		"  " + theme.Styles.Tertiary.Render(texts.CustomizeIntro),
		"  " + theme.Styles.Tertiary.Render(texts.CustomizeDrift),
		"",
	}
	// Keep the cursor visible when the list is taller than the body.
	visible := max(1, height-len(lines))
	first := 0
	if s.cursor >= visible {
		first = s.cursor - visible + 1
	}
	for index := first; index < len(s.apps) && index < first+visible; index++ {
		marker, style := "  ", theme.Styles.Secondary
		if index == s.cursor {
			marker, style = theme.Symbols.Cursor+" ", theme.Styles.Primary
		}
		lines = append(lines, "  "+marker+style.Render(s.apps[index].Name)+" "+theme.Styles.Tertiary.Render("("+s.apps[index].ID+")"))
	}
	return Fit(lines, width, height, theme.Symbols.Ellipsis)
}
