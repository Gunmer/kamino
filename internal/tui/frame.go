package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"

	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/tui/theme"
)

const minimumWidth = 80
const minimumHeight = 24

// Frame owns the header, body, message zone, separators, footer and size guard.
type Frame struct {
	theme *theme.Theme
	help  help.Model
}

func newFrame(current *theme.Theme) *Frame {
	model := help.New()
	model.Styles.ShortKey = current.Styles.Secondary
	model.Styles.ShortDesc = current.Styles.Tertiary
	model.Styles.ShortSeparator = current.Styles.Tertiary
	model.Styles.FullKey = current.Styles.Secondary
	model.Styles.FullDesc = current.Styles.Tertiary
	model.Styles.FullSeparator = current.Styles.Tertiary
	model.Styles.Ellipsis = current.Styles.Tertiary
	model.ShortSeparator = "   "
	model.FullSeparator = "   "
	return &Frame{theme: current, help: model}
}

// BodyWidth returns the approved body width for a terminal width.
func BodyWidth(termWidth int) int {
	switch {
	case termWidth < 100:
		width := termWidth - 2*theme.DefaultTokens().Gutter
		if width < 0 {
			return 0
		}
		return width
	case termWidth < 140:
		return 96
	default:
		return 120
	}
}

// BodyHeight returns the body height left by the chrome and the messages.
func BodyHeight(termHeight, messageCount int) int {
	air := 0
	if termHeight >= 26 {
		air = 1
	}
	height := termHeight - 4 - air - messageCount
	if height < 0 {
		return 0
	}
	return height
}

// Render composes one complete frame of exactly width×height cells.
func (f *Frame) Render(screen Screen, width, height int, operation string) string {
	if width < minimumWidth || height < minimumHeight {
		return f.small(width, height, operation)
	}
	messages := compactMessages(screen.Messages())
	bodyWidth := BodyWidth(width)
	bodyHeight := BodyHeight(height, len(messages))

	lines := make([]string, 0, height)
	lines = append(lines, f.header(screen, width))
	lines = append(lines, f.separator(width))
	if height >= 26 {
		lines = append(lines, "")
	}
	body := Fit(strings.Split(screen.View(bodyWidth, bodyHeight), "\n"), bodyWidth, bodyHeight, f.theme.Symbols.Ellipsis)
	gutter := strings.Repeat(" ", (width-bodyWidth)/2)
	for _, line := range strings.Split(body, "\n") {
		lines = append(lines, gutter+line)
	}
	for _, message := range messages {
		lines = append(lines, f.messageLine(message, width))
	}
	lines = append(lines, f.separator(width))
	lines = append(lines, f.footer(screen, width))
	if !f.theme.Unicode {
		for index := range lines {
			lines[index] = displaySymbols(lines[index], false)
		}
	}
	return Fit(lines, width, height, f.theme.Symbols.Ellipsis)
}

func (f *Frame) header(screen Screen, width int) string {
	flow := flowOf(screen)
	title := screen.Title()
	left := texts.AppName
	if flow != "" {
		left += " · " + flow
	}
	if title != "" && title != flow {
		left += " · " + title
	}
	right := ""
	current, total := screen.Step()
	if total > 0 {
		right = strings.NewReplacer(
			"{current}", fmt.Sprintf("%d", current),
			"{total}", fmt.Sprintf("%d", total),
		).Replace(texts.StepFormat) + "   " + f.steps(current, total)
	} else if flow == texts.FlowMenu {
		right = f.capsules()
	}
	if right == "" {
		return Truncate(left, width, f.theme.Symbols.Ellipsis)
	}
	leftWidth := width - Width(right) - 1
	if leftWidth < 0 {
		return Truncate(right, width, f.theme.Symbols.Ellipsis)
	}
	return PadRight(Truncate(left, leftWidth, f.theme.Symbols.Ellipsis), leftWidth) + " " + right
}

func (f *Frame) steps(current, total int) string {
	parts := make([]string, 0, total)
	for index := 1; index <= total; index++ {
		if index <= current {
			parts = append(parts, f.theme.Styles.Accent.Render(f.theme.Symbols.Chosen))
		} else {
			parts = append(parts, f.theme.Styles.Tertiary.Render(f.theme.Symbols.NotChosen))
		}
	}
	return strings.Join(parts, " ")
}

func (f *Frame) capsules() string {
	parts := make([]string, 0, texts.CapsuleTotal)
	for index := 0; index < texts.CapsuleTotal; index++ {
		if index == texts.CapsuleTotal-1 {
			parts = append(parts, f.theme.Styles.Accent.Render(f.theme.Symbols.Original))
		} else {
			parts = append(parts, f.theme.Styles.Tertiary.Render(f.theme.Symbols.CapsuleOff))
		}
	}
	return strings.Join(parts, " ")
}

func (f *Frame) separator(width int) string {
	return f.theme.Styles.Tertiary.Render(strings.Repeat(f.theme.Symbols.Separator, width))
}

func (f *Frame) footer(screen Screen, width int) string {
	suffix := "  " + f.theme.Styles.Secondary.Render("?") + " " + f.theme.Styles.Tertiary.Render(texts.KeyHelp)
	for _, binding := range screen.Keys() {
		for _, value := range binding.Keys() {
			if value == "?" {
				suffix = ""
			}
		}
	}
	available := width - Width(suffix)
	if available < 0 {
		return Truncate(suffix, width, f.theme.Symbols.Ellipsis)
	}
	main := f.help.ShortHelpView(screen.Keys())
	main = displayKeys(main, f.theme.Unicode)
	if main == "" {
		return suffix
	}
	if suffix == "" {
		return Truncate(main, width, f.theme.Symbols.Ellipsis)
	}
	return PadRight(Truncate(main, available, f.theme.Symbols.Ellipsis), available) + suffix
}

func (f *Frame) messageLine(message Message, width int) string {
	icon := f.theme.Symbols.Info
	label := texts.MessageInfoLabel
	style := f.theme.Styles.Info
	switch message.Kind {
	case MessageWarning:
		icon = f.theme.Symbols.Warning
		label = texts.MessageWarningLabel
		style = f.theme.Styles.Warning
	case MessageError:
		icon = f.theme.Symbols.Error
		label = texts.MessageErrorLabel
		style = f.theme.Styles.Error
	}
	prefix := style.Render(icon + " " + label)
	text := "  " + message.Text
	return Truncate(prefix+text, width, f.theme.Symbols.Ellipsis)
}

func (f *Frame) small(width, height int, operation string) string {
	if width < 1 {
		width = 1
	}
	lines := []string{
		f.theme.Styles.Primary.Render(texts.AppName + " · " + texts.FlowSmall),
		f.separator(width),
		"",
		"  " + f.theme.Styles.Error.Render(texts.SmallTitle),
		"  " + strings.NewReplacer("{width}", fmt.Sprintf("%d", width), "{height}", fmt.Sprintf("%d", height)).Replace(texts.SmallMinimum),
		"",
		"  " + texts.SmallBody,
	}
	if operation != "" {
		lines = append(lines, "  "+strings.ReplaceAll(texts.SmallRunning, "{operation}", operation))
	}
	lines = append(lines, "", "  ? "+texts.KeyHelp)
	if !f.theme.Unicode {
		for index := range lines {
			lines[index] = displaySymbols(lines[index], false)
		}
	}
	return Fit(lines, width, height, f.theme.Symbols.Ellipsis)
}

func compactMessages(messages []Message) []Message {
	order := []MessageKind{MessageInfo, MessageWarning, MessageError}
	result := make([]Message, 0, 3)
	for _, kind := range order {
		parts := make([]string, 0)
		for _, message := range messages {
			if message.Kind == kind && strings.TrimSpace(message.Text) != "" {
				parts = append(parts, message.Text)
			}
		}
		if len(parts) > 0 {
			result = append(result, Message{Kind: kind, Text: strings.Join(parts, " · ")})
		}
	}
	return result
}
