// Package theme centralizes the ANSI palette, symbols and spacing tokens of
// the TUI. Screens never build their own styles.
package theme

import (
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Tokens is the spacing scale shared by every screen.
type Tokens struct {
	Gutter int
	PadX   int
	PadY   int
	Gap    int
	ColGap int
}

// DefaultTokens returns the approved spacing scale.
func DefaultTokens() Tokens {
	return Tokens{Gutter: 2, PadX: 2, PadY: 1, Gap: 1, ColGap: 4}
}

// Symbols holds the visible glyphs for the current terminal mode.
type Symbols struct {
	Cursor     string
	Chosen     string
	NotChosen  string
	Original   string
	CapsuleOff string
	Left       string
	Right      string
	Collapsed  string
	Expanded   string
	Separator  string
	Zero       string
	Warning    string
	Info       string
	Error      string
	Up         string
	Down       string
	NoLine     string
	Ellipsis   string
	Selected   string
	Unselected string
}

// UnicodeSymbols returns the glyphs used with UTF-8 terminals.
func UnicodeSymbols() Symbols {
	return Symbols{
		Cursor: "▌", Chosen: "●", NotChosen: "○", Original: "◉", CapsuleOff: "○",
		Left: "◀", Right: "▶", Collapsed: "▸", Expanded: "▾", Separator: "─",
		Zero: "·", Warning: "⚠", Info: "ℹ", Error: "✕", Up: "▲", Down: "▼",
		NoLine: "— sin línea —", Ellipsis: "…", Selected: "●", Unselected: "○",
	}
}

// ASCIISymbols returns the deterministic fallback glyphs.
func ASCIISymbols() Symbols {
	return Symbols{
		Cursor: ">", Chosen: "*", NotChosen: "o", Original: "*", CapsuleOff: "o",
		Left: "<", Right: ">", Collapsed: "+", Expanded: "v", Separator: "-",
		Zero: ".", Warning: "!", Info: "i", Error: "x", Up: "^", Down: "v",
		NoLine: "-- sin línea --", Ellipsis: "...", Selected: "*", Unselected: "o",
	}
}

// Styles is the complete style set. Every field is built from the renderer so
// golden files are deterministic.
type Styles struct {
	Primary      lipgloss.Style
	Secondary    lipgloss.Style
	Tertiary     lipgloss.Style
	Accent       lipgloss.Style
	Success      lipgloss.Style
	Info         lipgloss.Style
	Warning      lipgloss.Style
	Error        lipgloss.Style
	SuccessTitle lipgloss.Style
	ErrorTitle   lipgloss.Style
	Selection    lipgloss.Style
	Header       lipgloss.Style
	TableHead    lipgloss.Style
	Wordmark     lipgloss.Style
	Chosen       lipgloss.Style
	Discarded    lipgloss.Style
}

// Theme is the injected presentation configuration.
type Theme struct {
	Name    string
	NoColor bool
	Unicode bool
	Tokens  Tokens
	Symbols Symbols
	Styles  Styles
}

// Options controls New.
type Options struct {
	Name    string
	NoColor bool
	Unicode bool
}

// New builds a theme with an ANSI-only renderer.
func New(options Options) *Theme {
	name := options.Name
	if name != "ansi-light" {
		name = "ansi-dark"
	}
	renderer := lipgloss.NewRenderer(io.Discard)
	renderer.SetColorProfile(termenv.ANSI)
	renderer.SetHasDarkBackground(name != "ansi-light")

	style := renderer.NewStyle
	color := func(value lipgloss.TerminalColor) lipgloss.Style {
		if options.NoColor {
			return style()
		}
		return style().Foreground(value)
	}

	tertiary := color(lipgloss.ANSIColor(8))
	if name != "ansi-light" && !options.NoColor {
		tertiary = style().Faint(true)
	}
	discarded := style().Faint(true)
	if name == "ansi-light" {
		discarded = color(lipgloss.ANSIColor(8))
	}

	symbols := UnicodeSymbols()
	if !options.Unicode {
		symbols = ASCIISymbols()
	}

	return &Theme{
		Name: name, NoColor: options.NoColor, Unicode: options.Unicode,
		Tokens: DefaultTokens(), Symbols: symbols,
		Styles: Styles{
			Primary:      style().Bold(true),
			Secondary:    style(),
			Tertiary:     tertiary,
			Accent:       color(lipgloss.ANSIColor(6)),
			Success:      color(lipgloss.ANSIColor(2)),
			Info:         color(lipgloss.ANSIColor(2)),
			Warning:      color(lipgloss.ANSIColor(3)),
			Error:        color(lipgloss.ANSIColor(1)),
			SuccessTitle: color(lipgloss.ANSIColor(2)).Bold(true),
			ErrorTitle:   color(lipgloss.ANSIColor(1)).Bold(true),
			Selection:    style().Bold(true),
			Header:       style().Bold(true),
			TableHead:    style().Bold(true),
			Wordmark:     color(lipgloss.ANSIColor(6)).Bold(true),
			Chosen:       style(),
			Discarded:    discarded,
		},
	}
}
