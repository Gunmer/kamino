package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/tui/theme"
)

const fieldGap = "  "

// field is the shared single-row text input. Label, value and placeholder are
// content; Cursor is a rune index and Invalid is the validation state of the
// value. Focus is not stored here because the screen owns it.
type field struct {
	Label       string
	Value       string
	Placeholder string
	Cursor      int
	Invalid     bool
}

// SetValue replaces the value and leaves the cursor at its end.
func (f *field) SetValue(value string) {
	f.Value = value
	f.Cursor = len([]rune(value))
}

// Insert writes text at the cursor and advances it by the inserted runes.
func (f *field) Insert(text string) {
	if text == "" {
		return
	}
	inserted := []rune(text)
	f.clampCursor()
	runes := []rune(f.Value)
	result := make([]rune, 0, len(runes)+len(inserted))
	result = append(result, runes[:f.Cursor]...)
	result = append(result, inserted...)
	result = append(result, runes[f.Cursor:]...)
	f.Value = string(result)
	f.Cursor += len(inserted)
}

// Backspace removes the rune before the cursor.
func (f *field) Backspace() {
	f.clampCursor()
	if f.Cursor == 0 {
		return
	}
	runes := []rune(f.Value)
	f.Value = string(append(runes[:f.Cursor-1], runes[f.Cursor:]...))
	f.Cursor--
}

// MoveCursor moves the cursor inside the value.
func (f *field) MoveCursor(delta int) {
	total := len([]rune(f.Value))
	f.Cursor += delta
	if f.Cursor < 0 {
		f.Cursor = 0
	}
	if f.Cursor > total {
		f.Cursor = total
	}
}

func (f *field) clampCursor() {
	total := len([]rune(f.Value))
	if f.Cursor < 0 {
		f.Cursor = 0
	}
	if f.Cursor > total {
		f.Cursor = total
	}
}

// fieldOptions carries the layout decisions of the screen.
type fieldOptions struct {
	Width      int
	LabelWidth int
	Focused    bool
}

// fieldLabelWidth aligns the label column of one screen. With NO_COLOR the
// focused field appends the "editando" suffix, so its width is reserved for
// every row to keep the values aligned.
func fieldLabelWidth(current *theme.Theme, labels ...string) int {
	width := 0
	for _, label := range labels {
		if candidate := Width(label); candidate > width {
			width = candidate
		}
	}
	if current.NoColor {
		width += Width(texts.FieldEditingSuffix)
	}
	return width
}

// Render draws one row: marker, aligned label and visible value. Only the
// focused field draws the editing cursor.
func (f field) Render(current *theme.Theme, options fieldOptions) string {
	width := options.Width
	if width < 1 {
		width = 1
	}
	marker := "  "
	labelText := f.Label
	labelStyle := current.Styles.FieldLabelIdle
	valueStyle := current.Styles.FieldIdle
	if options.Focused {
		marker = current.Symbols.Cursor + " "
		labelStyle = current.Styles.FieldLabelFocus
		valueStyle = current.Styles.FieldFocus
		if current.NoColor {
			labelText += texts.FieldEditingSuffix
		}
	}
	if f.Invalid {
		labelStyle = current.Styles.FieldError
		valueStyle = current.Styles.FieldError
	}

	markerWidth := Width(marker)
	gapWidth := Width(fieldGap)
	labelWidth := options.LabelWidth
	if labelWidth < 0 {
		labelWidth = 0
	}
	if limit := width - markerWidth - gapWidth - 1; labelWidth > limit {
		labelWidth = limit
	}
	if labelWidth < 0 {
		labelWidth = 0
	}
	label := PadRight(fieldTruncate(labelText, labelWidth, current.Symbols.Ellipsis), labelWidth)
	valueWidth := width - markerWidth - labelWidth - gapWidth
	if valueWidth < 1 {
		valueWidth = 1
	}
	renderedMarker := marker
	gap := fieldGap
	if options.Focused {
		renderedMarker = valueStyle.Render(marker)
		gap = valueStyle.Render(fieldGap)
	}
	return renderedMarker + labelStyle.Render(label) + gap + f.renderValue(current, valueWidth, options.Focused, valueStyle)
}

// fieldWindow is the visible slice of the value. Lead and tail record whether
// an ellipsis replaces hidden runes on that side.
type fieldWindow struct {
	start int
	end   int
	lead  bool
	tail  bool
}

func (f field) renderValue(current *theme.Theme, width int, focused bool, style lipgloss.Style) string {
	ellipsis := current.Symbols.Ellipsis
	if ellipsis == "" {
		ellipsis = "…"
	}
	if Width(ellipsis) > width {
		ellipsis = ""
	}
	ellipsisWidth := Width(ellipsis)

	runes := []rune(f.Value)
	if len(runes) == 0 {
		if !focused {
			return style.Render(fieldTruncate(f.Placeholder, width, ellipsis))
		}
		remaining := width - 1
		if remaining < 0 {
			remaining = 0
		}
		return current.Styles.FieldCaret.Render(" ") + style.Render(fieldTruncate(f.Placeholder, remaining, ellipsis))
	}

	cursor := f.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	widths := make([]int, len(runes))
	total := 0
	for index, value := range runes {
		widths[index] = Width(string(value))
		total += widths[index]
	}
	extra := 0
	if focused && cursor == len(runes) {
		extra = 1
	}
	if total+extra <= width {
		return renderFieldValue(current, runes, fieldWindow{start: 0, end: len(runes)}, cursor, focused, style)
	}

	cursorWidth := 1
	if cursor < len(runes) {
		cursorWidth = widths[cursor]
	}
	if focused && cursorWidth > width {
		// A double-width rune that cannot fit becomes the caret cell.
		return current.Styles.FieldCaret.Render(" ")
	}
	window := visibleFieldWindow(widths, cursor, width, extra, ellipsisWidth)
	if window.start == window.end {
		if focused {
			return current.Styles.FieldCaret.Render(" ")
		}
		return style.Render(ellipsis)
	}
	return renderFieldValue(current, runes, window, cursor, focused, style)
}

// visibleFieldWindow picks the rune slice that keeps the cursor visible. It
// never splits a rune, and it reserves the real ellipsis width on each hidden
// side.
func visibleFieldWindow(widths []int, cursor, width, extra, ellipsisWidth int) fieldWindow {
	window := fieldWindow{start: cursor, end: cursor}
	used := extra
	if cursor < len(widths) {
		used += widths[cursor]
		window.end = cursor + 1
	}
	if window.start > 0 {
		window.lead = true
		used += ellipsisWidth
	}
	if window.end < len(widths) {
		window.tail = true
		used += ellipsisWidth
	}
	if used > width {
		// Drop the ellipses before giving up the cursor cell.
		window.lead = false
		window.tail = false
		used = extra
		if cursor < len(widths) {
			used += widths[cursor]
		}
		if used > width {
			return fieldWindow{start: cursor, end: cursor}
		}
	}

	for window.end < len(widths) {
		nextTail := window.end+1 < len(widths)
		delta := widths[window.end]
		if window.tail {
			delta -= ellipsisWidth
		}
		if nextTail {
			delta += ellipsisWidth
		}
		if used+delta > width {
			break
		}
		used += delta
		window.end++
		window.tail = nextTail
	}
	for window.start > 0 {
		nextLead := window.start-1 > 0
		delta := widths[window.start-1]
		if window.lead {
			delta -= ellipsisWidth
		}
		if nextLead {
			delta += ellipsisWidth
		}
		if used+delta > width {
			break
		}
		used += delta
		window.start--
		window.lead = nextLead
	}
	return window
}

func renderFieldValue(current *theme.Theme, runes []rune, window fieldWindow, cursor int, focused bool, style lipgloss.Style) string {
	caret := current.Styles.FieldCaret
	var builder strings.Builder
	if window.lead {
		builder.WriteString(style.Render(current.Symbols.Ellipsis))
	}
	if focused && cursor >= window.start && cursor < window.end {
		if window.start < cursor {
			builder.WriteString(style.Render(string(runes[window.start:cursor])))
		}
		builder.WriteString(caret.Render(string(runes[cursor])))
		if cursor+1 < window.end {
			builder.WriteString(style.Render(string(runes[cursor+1 : window.end])))
		}
	} else {
		builder.WriteString(style.Render(string(runes[window.start:window.end])))
	}
	if window.tail {
		builder.WriteString(style.Render(current.Symbols.Ellipsis))
	}
	if focused && cursor == len(runes) {
		builder.WriteString(caret.Render(" "))
	}
	return builder.String()
}

// fieldTruncate shortens value without splitting runes. It falls back to a
// plain cut when not even the ellipsis fits.
func fieldTruncate(value string, width int, ellipsis string) string {
	if width <= 0 {
		return ""
	}
	if Width(value) <= width {
		return value
	}
	if ellipsis != "" && Width(ellipsis) <= width {
		return Truncate(value, width, ellipsis)
	}
	var builder strings.Builder
	used := 0
	for _, value := range value {
		runeWidth := Width(string(value))
		if used+runeWidth > width {
			break
		}
		builder.WriteRune(value)
		used += runeWidth
	}
	return builder.String()
}
