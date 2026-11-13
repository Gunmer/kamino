package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Width returns the visible width of a possibly styled string.
func Width(value string) int { return lipgloss.Width(value) }

// Truncate shortens value to width columns, ending in ellipsis when needed.
// ANSI sequences are preserved.
func Truncate(value string, width int, ellipsis string) string {
	if width <= 0 {
		return ""
	}
	if Width(value) <= width {
		return value
	}
	if ellipsis == "" {
		ellipsis = "…"
	}
	target := width - Width(ellipsis)
	if target < 0 {
		target = 0
	}
	var builder strings.Builder
	visible := 0
	styled := false
	for index := 0; index < len(value); {
		if value[index] == 0x1b {
			end := ansiSequenceEnd(value, index)
			builder.WriteString(value[index:end])
			styled = true
			index = end
			continue
		}
		runeValue, size := utf8.DecodeRuneInString(value[index:])
		runeWidth := Width(string(runeValue))
		if visible+runeWidth > target {
			break
		}
		builder.WriteString(string(runeValue))
		visible += runeWidth
		index += size
	}
	builder.WriteString(ellipsis)
	if styled {
		builder.WriteString("\x1b[0m")
	}
	return builder.String()
}

// TruncatePath abbreviates the home prefix and preserves the end of the path.
func TruncatePath(value, home string, width int, ellipsis string) string {
	display := value
	if home != "" {
		if value == home {
			display = "~"
		} else if strings.HasPrefix(value, home+"/") {
			display = "~/" + strings.TrimPrefix(value, home+"/")
		}
	}
	if Width(display) <= width {
		return display
	}
	if width <= Width(ellipsis) {
		return Truncate(display, width, ellipsis)
	}
	available := width - Width(ellipsis)
	runes := []rune(display)
	used := 0
	start := len(runes)
	for start > 0 {
		runeWidth := Width(string(runes[start-1]))
		if used+runeWidth > available {
			break
		}
		used += runeWidth
		start--
	}
	return ellipsis + string(runes[start:])
}

// PadRight pads value with spaces up to width columns.
func PadRight(value string, width int) string {
	gap := width - Width(value)
	if gap <= 0 {
		return value
	}
	return value + strings.Repeat(" ", gap)
}

// PadLeft pads value on the left up to width columns.
func PadLeft(value string, width int) string {
	gap := width - Width(value)
	if gap <= 0 {
		return value
	}
	return strings.Repeat(" ", gap) + value
}

// Fit returns exactly height lines, each no wider than width.
func Fit(lines []string, width, height int, ellipsis string) string {
	result := make([]string, 0, height)
	for _, line := range lines {
		if height > 0 && len(result) == height {
			break
		}
		result = append(result, Truncate(line, width, ellipsis))
	}
	for len(result) < height {
		result = append(result, "")
	}
	return strings.Join(result, "\n")
}

// displayKeys converts key labels and separators to ASCII when needed.
func displayKeys(value string, unicode bool) string {
	if unicode {
		return value
	}
	replacer := strings.NewReplacer(
		"↑↓", "^v", "←→", "<>", "Mayús+Tab", "Mayus+Tab", "•", "-", "…", "...",
	)
	return replacer.Replace(value)
}

// displaySymbols converts every remaining TUI glyph to its ASCII equivalent.
func displaySymbols(value string, unicode bool) string {
	if unicode {
		return value
	}
	replacer := strings.NewReplacer(
		"↑", "^", "↓", "v", "←", "<", "→", ">", "▌", ">", "●", "*", "○", "o",
		"◉", "*", "─", "-", "·", ".", "⚠", "!", "ℹ", "i", "✕", "x", "▲", "^",
		"▼", "v", "▸", "+", "▾", "v", "…", "...", "•", "-", "—", "-",
	)
	return replacer.Replace(value)
}

// ansiSequenceEnd returns the index just past the ANSI escape starting at start.
func ansiSequenceEnd(value string, start int) int {
	index := start + 1
	if index >= len(value) {
		return len(value)
	}
	if value[index] == '[' {
		index++
		for index < len(value) {
			current := value[index]
			index++
			if current >= 0x40 && current <= 0x7e {
				return index
			}
		}
		return index
	}
	if value[index] == ']' {
		for index < len(value) {
			if value[index] == 0x07 {
				return index + 1
			}
			if value[index] == 0x1b && index+1 < len(value) && value[index+1] == '\\' {
				return index + 2
			}
			index++
		}
		return index
	}
	return index + 1
}
