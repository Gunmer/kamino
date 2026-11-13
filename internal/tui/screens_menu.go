package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/texts"
)

type menuScreen struct {
	screenBase
	index int
}

func newMenuScreen(app *App) *menuScreen { return &menuScreen{screenBase: screenBase{app: app}} }

func (s *menuScreen) Flow() string     { return texts.FlowMenu }
func (s *menuScreen) Title() string    { return texts.FlowMenu }
func (s *menuScreen) Step() (int, int) { return 0, 0 }

func (s *menuScreen) Keys() []key.Binding { return menuKeys() }

func (s *menuScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "up", "k":
		s.index = (s.index + 4) % 5
	case "down", "j":
		s.index = (s.index + 1) % 5
	case "enter":
		switch s.index {
		case 0:
			return s, func() tea.Msg { return startCloneMsg{} }
		case 1:
			return s, func() tea.Msg { return startDeployMsg{} }
		case 2:
			return s, func() tea.Msg { return startUpdateMsg{} }
		case 3:
			return s, func() tea.Msg { return startSettingsMsg{} }
		case 4:
			return s, quitApp(s.app.exitCode)
		}
	}
	return s, nil
}

func (s *menuScreen) View(width, height int) string {
	lines := make([]string, 0)
	pad := strings.Repeat(" ", s.app.theme.Tokens.PadX)
	showFull := height >= 25
	if s.app.theme.Unicode {
		for _, line := range texts.Wordmark {
			lines = append(lines, pad+s.app.theme.Styles.Wordmark.Render(line))
		}
	}
	if showFull {
		lines = append(lines, "", pad+s.app.theme.Styles.Secondary.Render(texts.MenuSubtitle))
		if s.app.phrase != "" {
			lines = append(lines, pad+s.app.theme.Styles.Tertiary.Render(s.app.phrase))
		}
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	options := []string{texts.MenuClone, texts.MenuDeploy, texts.MenuUpdate, texts.MenuConfig, texts.MenuExit}
	helps := []string{texts.MenuCloneHelp, texts.MenuDeployHelp, texts.MenuUpdateHelp, texts.MenuConfigHelp, texts.MenuExitHelp}
	for index, option := range options {
		marker := "  "
		style := s.app.theme.Styles.Secondary
		if index == s.index {
			marker = s.app.theme.Symbols.Cursor + " "
			style = s.app.theme.Styles.Primary
		}
		line := pad + marker + PadRight(option, 18) + s.app.theme.Styles.Tertiary.Render(helps[index])
		lines = append(lines, style.Render(line))
	}
	if showFull {
		if s.app.lastMission != "" {
			lines = append(lines, "", pad+s.app.theme.Styles.Tertiary.Render(strings.ReplaceAll(texts.MenuLastMission, "{summary}", s.app.lastMission)))
		}
		lines = append(lines, "", pad+s.app.theme.Styles.Warning.Render(texts.MenuPrivacy))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}
