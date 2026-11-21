package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/detect"
	"github.com/Gunmer/kamino/internal/texts"
)

type authoringPhase int

const (
	authoringPhaseIntro authoringPhase = iota
	authoringPhaseEditing
	authoringPhaseConfirm
	authoringPhaseError
	authoringPhaseDone
)

type appAuthoringScreen struct {
	screenBase
	directory   string
	draftPath   string
	phase       authoringPhase
	messages    []Message
	createdPath string
	definition  *catalog.AppDefinition
	content     []byte
	override    bool
	overwrite   bool
	detected    bool
	committed   bool
}

func newAppAuthoringScreen(app *App, directory, draftPath string) *appAuthoringScreen {
	return &appAuthoringScreen{
		screenBase: screenBase{app: app},
		directory:  directory,
		draftPath:  draftPath,
		phase:      authoringPhaseIntro,
	}
}

func (s *appAuthoringScreen) Flow() string        { return texts.FlowSettings }
func (s *appAuthoringScreen) Title() string       { return texts.FlowAuthoring }
func (s *appAuthoringScreen) Step() (int, int)    { return 0, 0 }
func (s *appAuthoringScreen) Messages() []Message { return s.messages }

func (s *appAuthoringScreen) Keys() []key.Binding {
	switch s.phase {
	case authoringPhaseEditing:
		return []key.Binding{bind([]string{"esc", "ctrl+c"}, "Esc/Ctrl+C", texts.KeyCancel)}
	case authoringPhaseConfirm:
		return []key.Binding{
			bind([]string{"enter"}, "Enter", texts.KeySave),
			bind([]string{"esc"}, "Esc", texts.KeyBack),
		}
	case authoringPhaseDone:
		return []key.Binding{bind([]string{"enter", "esc"}, "Enter/Esc", texts.KeyContinue)}
	default:
		return []key.Binding{
			bind([]string{"enter", "e"}, "Enter/e", texts.KeyEdit),
			bind([]string{"esc"}, "Esc", texts.KeyDiscard),
		}
	}
}

func (s *appAuthoringScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch value := msg.(type) {
	case editorFinishedMsg:
		return s.finishEditor(value.err)
	case tea.KeyMsg:
		keyName := value.String()
		switch s.phase {
		case authoringPhaseIntro, authoringPhaseError:
			switch keyName {
			case "enter", "e":
				return s.edit()
			case "esc":
				return s.discard()
			}
		case authoringPhaseConfirm:
			switch keyName {
			case "enter":
				return s.commit()
			case "esc":
				s.phase = authoringPhaseError
				s.messages = []Message{{Kind: MessageWarning, Text: texts.AuthoringKeepDraft}}
			}
		case authoringPhaseDone:
			if keyName == "enter" || keyName == "esc" {
				return s, popScreen()
			}
		}
	}
	return s, nil
}

func (s *appAuthoringScreen) edit() (Screen, tea.Cmd) {
	s.phase = authoringPhaseEditing
	s.messages = []Message{{Kind: MessageInfo, Text: texts.AuthoringEditing}}
	return s, s.app.editor(s.draftPath)
}

func (s *appAuthoringScreen) finishEditor(editorErr error) (Screen, tea.Cmd) {
	if editorErr != nil {
		return s.fail(strings.ReplaceAll(texts.AuthoringEditorFailed, "{message}", editorErr.Error()))
	}
	data, err := os.ReadFile(s.draftPath)
	if err != nil {
		return s.fail(err.Error())
	}
	definition, err := catalog.Parse("borrador", data)
	if err != nil {
		return s.fail(err.Error())
	}
	results, err := catalog.InspectDir(s.directory)
	if err != nil {
		return s.fail(err.Error())
	}
	targetName := definition.ID + ".toml"
	for _, result := range results {
		if result.Definition == nil {
			continue
		}
		if result.ID == definition.ID && result.File != targetName {
			return s.fail(fmt.Sprintf("otro archivo ya declara el id %q: %s", definition.ID, result.File))
		}
	}
	s.definition = definition
	s.content = data
	s.overwrite = authoringFileExists(filepath.Join(s.directory, targetName))
	s.override = false
	if embedded, err := catalog.Embedded(); err == nil {
		_, s.override = embedded[definition.ID]
	}
	if s.overwrite || s.override {
		s.phase = authoringPhaseConfirm
		s.messages = nil
		return s, nil
	}
	return s.commit()
}

func (s *appAuthoringScreen) commit() (Screen, tea.Cmd) {
	result, err := catalog.CreateDefinition(s.directory, s.content, s.overwrite)
	if err != nil {
		return s.fail(err.Error())
	}
	_ = os.Remove(s.draftPath)
	s.committed = true
	s.createdPath = result.Path
	s.definition = result.Definition
	s.detected = detect.IsInstalled(result.Definition, detect.Options{
		Home: s.app.home, ApplicationDirs: s.app.applicationDirs, Path: s.app.pathValue,
	})
	s.app.catalog = nil
	s.app.lastMission = "app añadida: " + result.Definition.ID
	s.phase = authoringPhaseDone
	s.messages = nil
	if !s.app.loadCatalog() {
		return s, nil
	}
	return s, nil
}

func (s *appAuthoringScreen) fail(message string) (Screen, tea.Cmd) {
	s.phase = authoringPhaseError
	s.messages = []Message{{Kind: MessageError, Text: message}}
	return s, nil
}

func (s *appAuthoringScreen) discard() (Screen, tea.Cmd) {
	if !s.committed && s.draftPath != "" {
		_ = os.Remove(s.draftPath)
	}
	return s, popScreen()
}

func (s *appAuthoringScreen) View(width, height int) string {
	var lines []string
	switch s.phase {
	case authoringPhaseEditing:
		lines = []string{
			"  " + s.app.theme.Styles.Primary.Render(texts.AuthoringHeading),
			"",
			"  " + s.app.theme.Styles.Secondary.Render(texts.AuthoringEditing),
			"  " + s.app.theme.Styles.Tertiary.Render(strings.ReplaceAll(texts.AuthoringDraft, "{path}", s.app.displayPath(s.draftPath))),
		}
	case authoringPhaseConfirm:
		lines = []string{"  " + s.app.theme.Styles.Primary.Render(texts.AuthoringConfirmTitle), ""}
		if s.override {
			lines = append(lines, "  "+s.app.theme.Styles.Warning.Render(strings.ReplaceAll(texts.AuthoringOverride, "{id}", s.definition.ID)))
		}
		if s.overwrite {
			lines = append(lines, "  "+s.app.theme.Styles.Warning.Render(strings.ReplaceAll(texts.AuthoringOverwrite, "{path}", s.app.displayPath(s.createdTarget()))))
		}
		lines = append(lines, "", "  "+s.app.theme.Styles.Tertiary.Render(texts.AuthoringKeepDraft))
	case authoringPhaseError:
		lines = []string{
			"  " + s.app.theme.Styles.Error.Render(texts.AuthoringErrorTitle),
			"",
			"  " + s.app.theme.Styles.Tertiary.Render(strings.ReplaceAll(texts.AuthoringDraftHint, "{path}", s.app.displayPath(s.draftPath))),
			"  " + s.app.theme.Styles.Tertiary.Render(texts.AuthoringKeepDraft),
		}
	case authoringPhaseDone:
		lines = []string{
			"  " + s.app.theme.Styles.Primary.Render(texts.AuthoringDoneTitle),
			"",
			"  " + s.app.theme.Styles.Secondary.Render(strings.ReplaceAll(texts.AuthoringCreated, "{path}", s.app.displayPath(s.createdPath))),
			"  " + s.app.theme.Styles.Tertiary.Render(texts.AuthoringDetected),
		}
		if !s.detected {
			lines[3] = "  " + s.app.theme.Styles.Tertiary.Render(texts.AuthoringNotDetected)
		}
	default:
		lines = []string{
			"  " + s.app.theme.Styles.Primary.Render(texts.AuthoringHeading),
			"  " + s.app.theme.Styles.Tertiary.Render(texts.AuthoringIntro),
			"",
			"  " + s.app.theme.Styles.Secondary.Render(strings.ReplaceAll(texts.AuthoringCatalog, "{path}", s.app.displayPath(s.directory))),
			"  " + s.app.theme.Styles.Tertiary.Render(strings.ReplaceAll(texts.AuthoringDraft, "{path}", s.app.displayPath(s.draftPath))),
			"",
			"  " + s.app.theme.Styles.Tertiary.Render(texts.AuthoringRules),
			"  " + s.app.theme.Styles.Tertiary.Render(texts.AuthoringEdit),
		}
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func (s *appAuthoringScreen) createdTarget() string {
	if s.definition == nil {
		return s.directory
	}
	return filepath.Join(s.directory, s.definition.ID+".toml")
}

func authoringFileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
