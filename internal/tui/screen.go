package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ScreenID identifies one screen in the navigation stack.
type ScreenID string

const (
	ScreenMenu     ScreenID = "menu"
	ScreenC1       ScreenID = "c1"
	ScreenC2       ScreenID = "c2"
	ScreenC3       ScreenID = "c3"
	ScreenD0       ScreenID = "d0"
	ScreenD1       ScreenID = "d1"
	ScreenD2       ScreenID = "d2"
	ScreenD3       ScreenID = "d3"
	ScreenD4       ScreenID = "d4"
	ScreenD5       ScreenID = "d5"
	ScreenU1       ScreenID = "u1"
	ScreenSettings ScreenID = "settings"
	ScreenHelp     ScreenID = "help"
	ScreenError    ScreenID = "error"
)

// MessageKind is the severity of a message-zone line.
type MessageKind int

const (
	MessageInfo MessageKind = iota
	MessageWarning
	MessageError
)

// Message is one line of the frame message zone.
type Message struct {
	Kind MessageKind
	Text string
}

// Screen is the contract every TUI screen implements. View is pure: it does
// not read the environment, time or disk.
type Screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (Screen, tea.Cmd)
	View(w, h int) string
	Title() string
	Step() (current, total int)
	Keys() []key.Binding
	Messages() []Message
	SetSize(w, h int)
}

// flowScreen is an optional interface used by the frame header.
type flowScreen interface {
	Flow() string
}

type pushMsg struct{ screen Screen }
type popMsg struct{}
type replaceMsg struct{ screen Screen }
type quitMsg struct{ code int }
type errorMsg struct{ what, safety, action, detail string }

func pushScreen(screen Screen) tea.Cmd {
	return func() tea.Msg { return pushMsg{screen: screen} }
}

func popScreen() tea.Cmd {
	return func() tea.Msg { return popMsg{} }
}

func replaceScreen(screen Screen) tea.Cmd {
	return func() tea.Msg { return replaceMsg{screen: screen} }
}

func quitApp(code int) tea.Cmd {
	return func() tea.Msg { return quitMsg{code: code} }
}

func showError(what, safety, action, detail string) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{what: what, safety: safety, action: action, detail: detail}
	}
}

// screenBase holds the state shared by every screen.
type screenBase struct {
	app    *App
	width  int
	height int
}

func (s *screenBase) SetSize(width, height int) {
	s.width, s.height = width, height
}

func (s *screenBase) Init() tea.Cmd { return nil }

func (s *screenBase) Messages() []Message { return nil }

func flowOf(screen Screen) string {
	if value, ok := screen.(flowScreen); ok {
		return value.Flow()
	}
	return ""
}
