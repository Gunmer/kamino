package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/texts"
)

func TestSettingsOpensAuthoringAndCancelRemovesDraft(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true})
	app.startMenu()
	menu := app.current.(*menuScreen)
	menu.index = 3
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	settings, ok := app.current.(*settingsScreen)
	if !ok {
		t.Fatalf("expected settings, got %T", app.current)
	}
	settings.focus = authoringRow
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	screen, ok := app.current.(*appAuthoringScreen)
	if !ok {
		t.Fatalf("expected the authoring screen, got %T", app.current)
	}
	if _, err := os.Stat(screen.draftPath); err != nil {
		t.Fatalf("draft was not created: %v", err)
	}
	results, err := catalog.InspectDir(filepath.Dir(screen.draftPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("the draft must be invisible to the catalog: %#v", results)
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*settingsScreen); !ok {
		t.Fatalf("discarding must return to settings, got %T", app.current)
	}
	if _, err := os.Stat(screen.draftPath); !os.IsNotExist(err) {
		t.Fatal("discarding must remove the draft")
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatalf("leaving settings must return to the menu, got %T", app.current)
	}
}

func TestAuthoringUntouchedDraftDoesNotSave(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true,
		Editor: func(string) tea.Cmd { return func() tea.Msg { return editorFinishedMsg{} } },
	})
	screen := startAuthoringForTest(t, app)
	command := send(t, app, keyMsg("enter"))
	if command == nil {
		t.Fatal("editing must return an editor command")
	}
	send(t, app, execute(t, command))
	if screen.phase != authoringPhaseError {
		t.Fatalf("an untouched draft must fail validation, got %v", screen.phase)
	}
	if len(screen.messages) == 0 || screen.messages[0].Kind != MessageError {
		t.Fatalf("validation failure must show an ERROR: %#v", screen.messages)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "kamino", "apps", "miapp.toml")); !os.IsNotExist(err) {
		t.Fatal("an untouched draft must not create a definition")
	}
	if _, err := os.Stat(screen.draftPath); err != nil {
		t.Fatal("the draft must remain for re-editing")
	}
}

func TestAuthoringValidDraftCreatesFileAndReloads(t *testing.T) {
	home := t.TempDir()
	content := validAuthoringContent("miapp")
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true,
		Editor: writeAuthoringEditor(content),
	})
	screen := startAuthoringForTest(t, app)
	command := send(t, app, keyMsg("enter"))
	send(t, app, execute(t, command))
	if screen.phase != authoringPhaseDone {
		t.Fatalf("expected a completed definition, got %v (%#v)", screen.phase, screen.messages)
	}
	target := filepath.Join(home, ".config", "kamino", "apps", "miapp.toml")
	if data, err := os.ReadFile(target); err != nil || !strings.Contains(string(data), `id = "miapp"`) {
		t.Fatalf("definition was not written: %v", err)
	}
	if app.catalog == nil || app.catalog["miapp"] == nil {
		t.Fatal("the catalog must be reloaded after creating a definition")
	}
	if !strings.Contains(app.lastMission, "app añadida") {
		t.Fatalf("last mission was not updated: %q", app.lastMission)
	}
	if _, err := os.Stat(screen.draftPath); !os.IsNotExist(err) {
		t.Fatal("the draft must be removed after a successful commit")
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*settingsScreen); !ok {
		t.Fatalf("expected settings after finishing, got %T", app.current)
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatalf("expected the menu after leaving settings, got %T", app.current)
	}
}

func TestAuthoringOverrideRequiresConfirmation(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true,
		Editor: writeAuthoringEditor(validAuthoringContent("git")),
	})
	screen := startAuthoringForTest(t, app)
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	if screen.phase != authoringPhaseConfirm {
		t.Fatalf("an embedded override must ask for confirmation, got %v", screen.phase)
	}
	target := filepath.Join(home, ".config", "kamino", "apps", "git.toml")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("confirmation must not write before acceptance")
	}
	send(t, app, keyMsg("esc"))
	if screen.phase != authoringPhaseError {
		t.Fatalf("rejecting must return to the error state, got %v", screen.phase)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("rejecting must not write the definition")
	}
	send(t, app, execute(t, send(t, app, keyMsg("e"))))
	if screen.phase != authoringPhaseConfirm {
		t.Fatalf("re-editing must ask again, got %v", screen.phase)
	}
	send(t, app, keyMsg("enter"))
	if screen.phase != authoringPhaseDone {
		t.Fatalf("accepting must write the definition, got %v (%#v)", screen.phase, screen.messages)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("accepted override was not written")
	}
}

func TestAuthoringOverwriteRequiresConfirmation(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".config", "kamino", "apps")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	old := authoringContent("miapp", "Vieja")
	if err := os.WriteFile(filepath.Join(directory, "miapp.toml"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true,
		Editor: writeAuthoringEditor(authoringContent("miapp", "Nueva")),
	})
	screen := startAuthoringForTest(t, app)
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	if screen.phase != authoringPhaseConfirm {
		t.Fatalf("an existing file must ask for confirmation, got %v", screen.phase)
	}
	if data, _ := os.ReadFile(filepath.Join(directory, "miapp.toml")); !strings.Contains(string(data), "Vieja") {
		t.Fatal("the existing file must stay intact before confirmation")
	}
	send(t, app, keyMsg("enter"))
	if screen.phase != authoringPhaseDone {
		t.Fatalf("accepting must overwrite the file, got %v (%#v)", screen.phase, screen.messages)
	}
	if data, _ := os.ReadFile(filepath.Join(directory, "miapp.toml")); !strings.Contains(string(data), "Nueva") {
		t.Fatal("the accepted overwrite was not written")
	}
}

func TestAuthoringEditorFailureKeepsDraft(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true,
		Editor: func(string) tea.Cmd {
			return func() tea.Msg { return editorFinishedMsg{err: os.ErrNotExist} }
		},
	})
	screen := startAuthoringForTest(t, app)
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	if screen.phase != authoringPhaseError {
		t.Fatalf("an editor failure must keep the error screen, got %v", screen.phase)
	}
	if _, err := os.Stat(screen.draftPath); err != nil {
		t.Fatal("the draft must be kept after an editor failure")
	}
	view := stripSGR(renderCase(app, screen, 100, 26))
	if !strings.Contains(view, app.displayPath(screen.draftPath)) {
		t.Fatalf("the draft path must be visible: %s", view)
	}
}

func TestSettingsAddAppRowFocusAndView(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	app.startSettings("")
	screen := app.current.(*settingsScreen)
	screen.focus = limitField
	send(t, app, keyMsg("tab"))
	if screen.focus != authoringRow {
		t.Fatalf("Tab must reach the Añadir app row, got %d", screen.focus)
	}
	view := stripSGR(renderCase(app, screen, 80, 24))
	if !strings.Contains(view, texts.SettingsAddApp) || !strings.Contains(view, texts.SettingsAddAppHelp) {
		t.Fatalf("the Añadir app row is missing:\n%s", view)
	}
	before := screen.fields[limitField].Value
	send(t, app, keyMsg("x"))
	if screen.fields[limitField].Value != before {
		t.Fatal("typing on the action row must not edit a field")
	}
	send(t, app, keyMsg("shift+tab"))
	if screen.focus != limitField {
		t.Fatalf("Shift+Tab must return to the limit, got %d", screen.focus)
	}
}

func TestSettingsCatalogDirReloadsCatalog(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(home, "kamino-apps")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	content := validAuthoringContent("miapp")
	if err := os.WriteFile(filepath.Join(custom, "miapp.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true})
	app.startSettings("")
	screen := app.current.(*settingsScreen)
	screen.fields[0].SetValue(custom)
	send(t, app, keyMsg("ctrl+s"))
	if app.catalog != nil {
		t.Fatal("saving a new catalog.dir must invalidate the cached catalog")
	}
	app.startClone()
	if app.catalog == nil || app.catalog["miapp"] == nil {
		t.Fatalf("the catalog was not loaded from the new directory: %#v", app.catalog)
	}
	found := false
	for _, choice := range app.cloneApps {
		if choice.ID == "miapp" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the new app is missing from C1: %#v", app.cloneApps)
	}
}

func TestSettingsInvalidCatalogDirShowsErrorWithoutTouchingFiles(t *testing.T) {
	home := t.TempDir()
	badDir := filepath.Join(home, "bad-apps")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badFile := filepath.Join(badDir, "bad.toml")
	if err := os.WriteFile(badFile, []byte("not toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark", MenuMode: true})
	app.startSettings("")
	screen := app.current.(*settingsScreen)
	screen.fields[0].SetValue(badDir)
	send(t, app, keyMsg("ctrl+s"))
	if app.catalog != nil {
		t.Fatal("saving a new catalog.dir must invalidate the cache")
	}
	app.startClone()
	if _, ok := app.current.(*errorScreen); !ok {
		t.Fatalf("an invalid new catalog must show the catalog error, got %T", app.current)
	}
	if data, _ := os.ReadFile(badFile); string(data) != "not toml" {
		t.Fatal("the catalog files must not be modified")
	}
}

func TestTUICatalogUsesConfiguredAndDefaultDirs(t *testing.T) {
	home := t.TempDir()
	defaultDir := filepath.Join(home, ".config", "kamino", "apps")
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaultDir, "defaultapp.toml"), []byte(validAuthoringContent("defaultapp")), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	app.startClone()
	if app.catalog == nil || app.catalog["defaultapp"] == nil {
		t.Fatal("the default catalog directory must keep working")
	}

	custom := filepath.Join(home, "custom-apps")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, "customapp.toml"), []byte(validAuthoringContent("customapp")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "kamino"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "kamino", "config.toml"), []byte("[catalog]\ndir = \""+custom+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app = NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	app.startClone()
	if app.catalog == nil || app.catalog["customapp"] == nil {
		t.Fatal("the configured catalog directory must be used")
	}
}

func startAuthoringForTest(t *testing.T, app *App) *appAuthoringScreen {
	t.Helper()
	app.startMenu()
	app.startSettings("")
	app.startAuthoring()
	screen, ok := app.current.(*appAuthoringScreen)
	if !ok {
		t.Fatalf("expected the authoring screen, got %T", app.current)
	}
	return screen
}

func writeAuthoringEditor(content string) func(string) tea.Cmd {
	return func(path string) tea.Cmd {
		return func() tea.Msg {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return editorFinishedMsg{err: err}
			}
			return editorFinishedMsg{}
		}
	}
}

func validAuthoringContent(id string) string {
	return authoringContent(id, id)
}

func authoringContent(id, name string) string {
	content, err := catalog.RenderTemplate(catalog.TemplateOptions{
		ID: id, Name: name, Command: id, Paths: []string{".config/" + id},
	})
	if err != nil {
		panic(err)
	}
	return content
}
