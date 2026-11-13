package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/texts"
)

type helpScreen struct {
	screenBase
	target Screen
}

func newHelpScreen(app *App, target Screen) *helpScreen {
	return &helpScreen{screenBase: screenBase{app: app}, target: target}
}

func (s *helpScreen) Flow() string        { return flowOf(s.target) }
func (s *helpScreen) Title() string       { return texts.TitleAyuda }
func (s *helpScreen) Step() (int, int)    { return s.target.Step() }
func (s *helpScreen) Messages() []Message { return nil }
func (s *helpScreen) Keys() []key.Binding { return helpKeys() }

func (s *helpScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMessage.String() {
	case "enter", "esc", "?", "f1":
		return s, popScreen()
	}
	return s, nil
}

func (s *helpScreen) View(width, height int) string {
	lines := []string{
		"  " + s.app.theme.Styles.Primary.Render(strings.ReplaceAll(texts.HelpBodyTitle, "{screen}", s.target.Title())),
		"",
		"  " + s.app.theme.Styles.Secondary.Render(texts.HelpScreenShortcuts),
	}
	lines = append(lines, bindingRows(s.target.Keys(), width, s.app)...)
	lines = append(lines, "", "  "+s.app.theme.Styles.Secondary.Render(texts.HelpGlobals))
	lines = append(lines, bindingRows(globalKeys(), width, s.app)...)
	steps, current := s.flow()
	if len(steps) > 0 {
		lines = append(lines, "", "  "+s.app.theme.Styles.Secondary.Render(strings.ReplaceAll(texts.HelpFlowTitle, "{flow}", flowOf(s.target))))
		markers := make([]string, 0, len(steps))
		for index, step := range steps {
			name := texts.StepTitles[step]
			if name == "" {
				name = step
			}
			marker := s.app.theme.Symbols.NotChosen
			style := s.app.theme.Styles.Tertiary
			if index+1 == current {
				marker = s.app.theme.Symbols.Chosen
				style = s.app.theme.Styles.Accent
			}
			markers = append(markers, style.Render(marker+" "+name))
		}
		lines = append(lines, "  "+strings.Join(markers, "   "))
	}
	if flowOf(s.target) == texts.FlowDeploy {
		step, _ := s.target.Step()
		if step == 3 {
			lines = append(lines, "", "  "+s.app.theme.Styles.Warning.Render(s.app.theme.Symbols.Warning+" "+texts.DiffSecretsShort))
		}
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *helpScreen) flow() ([]string, int) {
	current, _ := s.target.Step()
	switch flowOf(s.target) {
	case texts.FlowClone:
		return texts.CloneSteps, current
	case texts.FlowDeploy:
		return texts.DeploySteps, current
	}
	return nil, 0
}

func bindingRows(bindings []key.Binding, width int, app *App) []string {
	if len(bindings) == 0 {
		return nil
	}
	gap := app.theme.Tokens.ColGap
	column := (width - gap) / 2
	lines := make([]string, 0)
	for index := 0; index < len(bindings); index += 2 {
		left := displayKeys(bindingCell(bindings[index]), app.theme.Unicode)
		line := "  " + PadRight(Truncate(left, column-2, app.theme.Symbols.Ellipsis), column)
		if index+1 < len(bindings) {
			right := displayKeys(bindingCell(bindings[index+1]), app.theme.Unicode)
			line += strings.Repeat(" ", gap) + Truncate(right, column-2, app.theme.Symbols.Ellipsis)
		}
		lines = append(lines, line)
	}
	return lines
}

func bindingCell(binding key.Binding) string {
	value := binding.Help()
	if value.Key == "" {
		return value.Desc
	}
	if value.Desc == "" {
		return value.Key
	}
	return value.Key + " " + value.Desc
}
