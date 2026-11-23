package tui

import (
	"fmt"
	"strings"

	"github.com/Gunmer/kamino/internal/implant"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type implantSummaryScreen struct {
	screenBase
	result     *implant.ImplantResult
	incomplete bool
	failed     string
	messages   []Message
}

func newImplantSummaryScreen(app *App, result *implant.ImplantResult, incomplete bool, failed string) *implantSummaryScreen {
	screen := &implantSummaryScreen{
		screenBase: screenBase{app: app}, result: result, incomplete: incomplete, failed: failed,
	}
	if incomplete {
		screen.messages = []Message{{Kind: MessageError, Text: strings.NewReplacer(
			"{applied}", fmt.Sprintf("%d", len(result.Implanted)),
			"{failed}", failed,
		).Replace(texts.D5IncompleteInfo)}}
	} else if result != nil && implant.ResultCode(*result) == 3 {
		screen.messages = []Message{{Kind: MessageWarning, Text: strings.ReplaceAll(texts.ImplantExitLine, "{code}", "3")}}
	}
	if result != nil {
		for _, outcome := range result.Installs {
			if outcome.Status == implant.InstallFailed {
				screen.messages = append(screen.messages, Message{Kind: MessageWarning, Text: strings.NewReplacer(
					"{app}", outcome.AppID, "{reason}", outcome.Reason,
				).Replace(texts.ImplantInstallFailed)})
			}
		}
	}
	return screen
}

func (s *implantSummaryScreen) Flow() string { return texts.FlowImplant }

func (s *implantSummaryScreen) Step() (int, int) { return 0, 0 }

func (s *implantSummaryScreen) Messages() []Message { return s.messages }

func (s *implantSummaryScreen) Keys() []key.Binding {
	return implantSummaryKeys(s.result != nil && s.result.Snapshot != "")
}

func (s *implantSummaryScreen) Title() string {
	if s.incomplete {
		return texts.D5Incomplete
	}
	return texts.D5Title
}

func (s *implantSummaryScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "c":
		if s.result != nil && s.result.Snapshot != "" {
			s.messages = []Message{{Kind: MessageInfo, Text: texts.D5CopyInfo}}
			return s, s.app.copyCommand(s.result.Snapshot, texts.D5CopyInfo, texts.D5CopyError)
		}
	case "enter", "esc":
		if s.incomplete {
			s.app.finish(1, "")
			return s, nil
		}
		s.app.finishImplant()
		return s, nil
	}
	return s, nil
}

func (s *implantSummaryScreen) View(width, height int) string {
	style := s.app.theme.Styles.SuccessTitle
	if s.incomplete {
		style = s.app.theme.Styles.ErrorTitle
	}
	lines := []string{"  " + style.Render(s.Title()), "", "  " + s.app.theme.Styles.Secondary.Render(texts.ImplantSummaryTitle)}
	lines = append(lines, "  "+s.app.theme.Styles.Tertiary.Render(strings.Repeat(s.app.theme.Symbols.Separator, max(0, width-4))))
	rows := [][2]string{
		{texts.D5Implanted, "0"}, {texts.D5Hybrids, "0"}, {texts.D5Omitted, "0"},
		{texts.D5Unchanged, "0"}, {texts.D5NotInstalled, "0"}, {texts.D5Protected, "0"},
	}
	if s.result != nil {
		rows[0][1] = fmt.Sprintf("%d", len(s.result.Implanted))
		rows[1][1] = fmt.Sprintf("%d", len(s.result.Merged))
		rows[2][1] = fmt.Sprintf("%d", len(s.result.Omitted))
		rows[3][1] = fmt.Sprintf("%d", len(s.result.Unchanged))
		rows[4][1] = fmt.Sprintf("%d", len(s.result.NotInstalled))
		rows[5][1] = fmt.Sprintf("%d", s.result.SnapshotFiles)
	}
	for _, row := range rows {
		lines = append(lines, "  "+PadRight(row[0], max(4, width-14))+PadLeft(row[1], 10))
	}
	lines = append(lines, "")
	if s.result != nil && s.result.Snapshot != "" {
		lines = append(lines,
			"  "+strings.ReplaceAll(texts.D5Snapshot, "{path}", s.app.displayPath(s.result.Snapshot)),
			"  "+strings.ReplaceAll(texts.D5Undo, "{path}", s.app.displayPath(s.result.Snapshot)),
			"", "  c "+texts.KeyCopySnapshot,
		)
	} else {
		lines = append(lines, "  "+texts.D5NoSnapshot)
	}
	if s.incomplete {
		lines = append(lines, "", "  "+s.app.theme.Styles.Warning.Render(texts.D5Recovery))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}
