package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/texts"
)

type errorScreen struct {
	screenBase
	what       string
	safety     string
	action     string
	detail     string
	showDetail bool
	index      int
	messages   []Message
}

func newErrorScreen(app *App, what, safety, action, detail string) *errorScreen {
	screen := &errorScreen{
		screenBase: screenBase{app: app}, what: what, safety: safety,
		action: action, detail: detail,
	}
	if app.menuMode {
		screen.index = 1
	} else {
		screen.index = 1
	}
	return screen
}

func (s *errorScreen) Flow() string        { return texts.FlowError }
func (s *errorScreen) Title() string       { return texts.FlowError }
func (s *errorScreen) Step() (int, int)    { return 0, 0 }
func (s *errorScreen) Messages() []Message { return s.messages }
func (s *errorScreen) Keys() []key.Binding { return errorKeys(s.app.menuMode) }

func (s *errorScreen) actions() []string {
	values := []string{texts.ErrorActionsDetail}
	if s.app.menuMode {
		values = append(values, texts.ErrorActionsMenu)
	}
	return append(values, texts.ErrorActionsExit)
}

func (s *errorScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	actions := s.actions()
	switch keyMessage.String() {
	case "up", "k", "left":
		s.index = (s.index + len(actions) - 1) % len(actions)
	case "down", "j", "right":
		s.index = (s.index + 1) % len(actions)
	case "t":
		s.showDetail = !s.showDetail
	case "c":
		s.messages = []Message{{Kind: MessageInfo, Text: texts.ErrorCopiedInfo}}
		return s, s.app.copyCommand(s.detail, texts.ErrorCopiedInfo, texts.ErrorCopyError)
	case "esc":
		return s, popScreen()
	case "enter":
		switch s.index {
		case 0:
			s.showDetail = !s.showDetail
		case 1:
			if s.app.menuMode {
				return s, func() tea.Msg { return menuMsg{} }
			}
			return s, quitApp(1)
		case 2:
			if s.app.menuMode {
				return s, quitApp(s.app.exitCode)
			}
			return s, quitApp(1)
		}
	}
	return s, nil
}

func (s *errorScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.ErrorTitle.Render(s.what),
		"",
	}
	if s.safety != "" {
		lines = append(lines, "  "+s.app.theme.Styles.Warning.Render(texts.ErrorSafety+": "+s.safety))
	}
	if s.action != "" {
		lines = append(lines, "  "+texts.ErrorAction+": "+s.action)
	}
	if s.showDetail && s.detail != "" {
		lines = append(lines, "", "  "+s.app.theme.Styles.Secondary.Render(texts.ErrorDetailTitle), "")
		for _, line := range strings.Split(s.detail, "\n") {
			lines = append(lines, "  "+line)
		}
	}
	lines = append(lines, "", "  "+s.actionsLine())
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *errorScreen) actionsLine() string {
	parts := make([]string, 0)
	for index, action := range s.actions() {
		if index == s.index {
			parts = append(parts, s.app.theme.Styles.Primary.Render("[> "+action+"]"))
			continue
		}
		parts = append(parts, "["+action+"]")
	}
	return strings.Join(parts, "   ")
}
