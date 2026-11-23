package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/texts"
)

func TestC1FilterFieldEditing(t *testing.T) {
	app := testApp(false, true)
	app.cloneApps = fixtureApps()
	screen := newCloneAppsScreen(app, "")
	screen.counting = false
	app.begin(screen)

	send(t, app, keyMsg("/"))
	if !screen.filtering {
		t.Fatal("/ must focus the filter field")
	}
	send(t, app, keyMsg("g"))
	send(t, app, keyMsg("i"))
	send(t, app, keyMsg("t"))
	if screen.filter.Value != "git" || screen.filter.Cursor != 3 {
		t.Fatalf("typing did not edit the filter: %q %d", screen.filter.Value, screen.filter.Cursor)
	}
	send(t, app, keyMsg("left"))
	send(t, app, keyMsg("X"))
	if screen.filter.Value != "giXt" || screen.filter.Cursor != 3 {
		t.Fatalf("middle insertion failed: %q %d", screen.filter.Value, screen.filter.Cursor)
	}
	send(t, app, keyMsg("backspace"))
	if screen.filter.Value != "git" || screen.filter.Cursor != 2 {
		t.Fatalf("middle backspace failed: %q %d", screen.filter.Value, screen.filter.Cursor)
	}
	send(t, app, keyMsg("right"))
	send(t, app, keyMsg("enter"))
	if screen.filtering {
		t.Fatal("Enter must return the focus to the table")
	}
	if visible := screen.visibleApps(); len(visible) != 1 || visible[0].ID != "git" {
		t.Fatalf("the filter was not applied: %#v", visible)
	}

	send(t, app, keyMsg("/"))
	send(t, app, keyMsg("?"))
	if _, ok := app.current.(*helpScreen); !ok {
		t.Fatalf("? must open the help, got %T", app.current)
	}
	if screen.filter.Value != "git" {
		t.Fatalf("the help must not edit the filter: %q", screen.filter.Value)
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*cloneAppsScreen); !ok {
		t.Fatalf("closing the help must return to C1, got %T", app.current)
	}
	send(t, app, keyMsg("esc"))
	if screen.filtering || screen.filter.Value != "" {
		t.Fatalf("Esc must clear and close the filter: %q %v", screen.filter.Value, screen.filtering)
	}
}

func TestC2FieldEditingFocusAndPlaceholder(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	screen := newCloneDestinationScreen(app, "")
	app.begin(screen)

	if screen.path.Cursor != len([]rune(screen.path.Value)) || screen.name.Cursor != len([]rune(screen.name.Value)) {
		t.Fatal("both fields must start with the cursor at the end")
	}
	screen.option = 2
	screen.focus = 2
	screen.path.SetValue("/tmp/base")
	send(t, app, keyMsg("left"))
	send(t, app, keyMsg("X"))
	if screen.path.Value != "/tmp/basXe" || screen.path.Cursor != 9 {
		t.Fatalf("middle insertion failed: %q %d", screen.path.Value, screen.path.Cursor)
	}
	send(t, app, keyMsg("backspace"))
	if screen.path.Value != "/tmp/base" || screen.path.Cursor != 8 {
		t.Fatalf("middle backspace failed: %q %d", screen.path.Value, screen.path.Cursor)
	}
	send(t, app, keyMsg("enter"))
	if screen.focus != 1 {
		t.Fatalf("Enter on the path must focus the name, got %d", screen.focus)
	}

	screen.focus = 2
	screen.path.SetValue("")
	view := screen.View(76, 19)
	plain := stripSGR(view)
	if !strings.Contains(plain, texts.FieldPathPlaceholder) {
		t.Fatalf("empty manual path must show the placeholder: %q", plain)
	}
	if reverseSGR(view) != 1 {
		t.Fatalf("the focused path must draw one cursor: %q", view)
	}

	screen.path.SetValue("/Users/test/" + strings.Repeat("carpeta/", 20) + "final")
	view = screen.View(76, 19)
	if !strings.Contains(stripSGR(view), "…") {
		t.Fatalf("a long path must scroll with an ellipsis: %q", stripSGR(view))
	}
}

func TestC2ErrorMarksFieldAndClearsOnFix(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	screen := newCloneDestinationScreen(app, "")
	app.begin(screen)
	screen.option = 2
	screen.path.SetValue(filepath.Join(home, "destino"))
	screen.focus = 1
	screen.name.SetValue("")

	command := send(t, app, keyMsg("enter"))
	if command != nil {
		t.Fatal("an empty name must not continue")
	}
	if !screen.name.Invalid {
		t.Fatal("the empty name must be marked invalid")
	}
	if len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Kind != MessageError {
		t.Fatalf("the error must go to the message zone: %#v", screen.messages)
	}
	send(t, app, keyMsg("x"))
	if screen.name.Invalid {
		t.Fatal("editing a valid name must clear the invalid mark")
	}
}

func TestD0FilterAndManualFields(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	screen := newImplantFileScreen(app, "")
	app.begin(screen)

	send(t, app, keyMsg("/"))
	if !screen.filtering {
		t.Fatal("/ must focus the D0 filter")
	}
	send(t, app, keyMsg("clone"))
	send(t, app, keyMsg("left"))
	send(t, app, keyMsg("X"))
	if screen.filter.Value != "clonXe" {
		t.Fatalf("middle insertion failed: %q", screen.filter.Value)
	}
	send(t, app, keyMsg("backspace"))
	send(t, app, keyMsg("enter"))
	if screen.filtering || screen.filter.Value != "clone" {
		t.Fatalf("Enter must apply the filter: %q %v", screen.filter.Value, screen.filtering)
	}
	send(t, app, keyMsg("/"))
	send(t, app, keyMsg("esc"))
	if screen.filtering || screen.filter.Value != "" {
		t.Fatalf("Esc must clear and close the filter: %q %v", screen.filter.Value, screen.filtering)
	}

	send(t, app, keyMsg("o"))
	if !screen.manual || screen.manualPath.Value != "" || screen.manualPath.Cursor != 0 {
		t.Fatalf("o must open an empty manual path: %#v", screen.manualPath)
	}
	view := screen.View(76, 19)
	if !strings.Contains(stripSGR(view), texts.FieldManualPlaceholder) {
		t.Fatalf("the manual path must show its placeholder: %q", stripSGR(view))
	}
	send(t, app, keyMsg("/tmp/no-existe.tar.gz"))
	command := send(t, app, keyMsg("enter"))
	if command != nil {
		t.Fatal("a missing path must not continue")
	}
	if !screen.manualPath.Invalid || screen.manualPath.Value != "/tmp/no-existe.tar.gz" {
		t.Fatalf("the invalid path must be kept and marked: %q %v", screen.manualPath.Value, screen.manualPath.Invalid)
	}
	if len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Kind != MessageError {
		t.Fatal("the invalid path must show an ERROR")
	}
	send(t, app, keyMsg("x"))
	if screen.manualPath.Invalid {
		t.Fatal("editing the path must clear the invalid mark")
	}
	for _, message := range screen.messages {
		if message.Kind == MessageError {
			t.Fatalf("editing the path must clear the manual error: %#v", screen.messages)
		}
	}

	screen.manualPath.SetValue("/Users/test/" + strings.Repeat("carpeta/", 20) + "clone.tar.gz")
	view = screen.View(76, 19)
	if !strings.Contains(stripSGR(view), "…") {
		t.Fatalf("a long manual path must scroll: %q", stripSGR(view))
	}
	send(t, app, keyMsg("f1"))
	if _, ok := app.current.(*helpScreen); !ok {
		t.Fatalf("F1 must open the help from the manual path, got %T", app.current)
	}
	if !strings.Contains(screen.manualPath.Value, "clone.tar.gz") {
		t.Fatalf("F1 must not edit the manual path: %q", screen.manualPath.Value)
	}
	navigate(t, app, keyMsg("esc"))

	target := filepath.Join(home, "clone.tar.gz")
	if err := os.WriteFile(target, []byte("clone"), 0o644); err != nil {
		t.Fatal(err)
	}
	screen.manualPath.SetValue(target)
	command = send(t, app, keyMsg("enter"))
	if command == nil {
		t.Fatal("a valid manual path must continue")
	}
	send(t, app, execute(t, command))
	if _, ok := app.current.(*implantAppsScreen); !ok {
		t.Fatalf("expected D1 after a valid path, got %T", app.current)
	}
}

func TestSettingsFieldNavigationEditingAndRows(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	app.startSettings("")
	screen := app.current.(*settingsScreen)

	if screen.focus != 0 {
		t.Fatalf("the focus must start on Catálogo, got %d", screen.focus)
	}
	send(t, app, keyMsg("tab"))
	if screen.focus != 1 {
		t.Fatalf("Tab must move to the next field, got %d", screen.focus)
	}
	send(t, app, keyMsg("shift+tab"))
	if screen.focus != 0 {
		t.Fatalf("Shift+Tab must move to the previous field, got %d", screen.focus)
	}
	send(t, app, keyMsg("down"))
	if screen.focus != 1 {
		t.Fatalf("down must move to the next field, got %d", screen.focus)
	}
	send(t, app, keyMsg("up"))
	if screen.focus != 0 {
		t.Fatalf("up must move to the previous field, got %d", screen.focus)
	}

	screen.fields[0].SetValue("abc")
	send(t, app, keyMsg("left"))
	send(t, app, keyMsg("X"))
	if screen.fields[0].Value != "abXc" {
		t.Fatalf("middle insertion failed: %q", screen.fields[0].Value)
	}
	send(t, app, keyMsg("backspace"))
	if screen.fields[0].Value != "abc" {
		t.Fatalf("middle backspace failed: %q", screen.fields[0].Value)
	}

	screen.fields[0].SetValue("/Users/test/" + strings.Repeat("carpeta/", 20) + "apps")
	screen.focus = 0
	if view := screen.View(76, 19); !strings.Contains(stripSGR(view), "…") {
		t.Fatalf("a long settings path must scroll: %q", stripSGR(view))
	}
	screen.fields[0].SetValue("abc")

	screen.focus = limitField
	screen.fields[limitField].SetValue("5")
	send(t, app, keyMsg("x"))
	if screen.fields[limitField].Value != "5" {
		t.Fatalf("the limit must reject letters: %q", screen.fields[limitField].Value)
	}
	send(t, app, keyMsg("2"))
	if screen.fields[limitField].Value != "52" {
		t.Fatalf("the limit must accept digits: %q", screen.fields[limitField].Value)
	}
	screen.fields[limitField].SetValue("5")

	valid := renderCase(app, screen, 100, 26)
	lines := strings.Split(valid, "\n")
	before := fieldRowIndexes(lines)

	screen.fields[0].SetValue("relative/path")
	send(t, app, keyMsg("ctrl+s"))
	if !screen.fields[0].Invalid {
		t.Fatal("the invalid path must be marked")
	}
	if screen.errorIndex != 0 {
		t.Fatalf("the error must point at Catálogo, got %d", screen.errorIndex)
	}
	withError := renderCase(app, screen, 100, 26)
	after := fieldRowIndexes(strings.Split(withError, "\n"))
	if len(before) != len(after) || len(before) != fieldCount {
		t.Fatalf("unexpected field rows: before=%v after=%v", before, after)
	}
	for index := range before {
		if before[index] != after[index] {
			t.Fatalf("the error moved a field row: before=%v after=%v", before, after)
		}
	}
}

// fieldRowIndexes returns the line of each settings label in order.
func fieldRowIndexes(lines []string) []int {
	labels := []string{texts.SettingsCatalog, texts.SettingsClones, texts.SettingsSnapshots, texts.FieldLimit}
	result := make([]int, 0, len(labels))
	for _, label := range labels {
		for index, line := range lines {
			if strings.Contains(stripSGR(line), label) {
				result = append(result, index)
				break
			}
		}
	}
	return result
}

func TestFieldFootersAndHelpUseTheSameBindings(t *testing.T) {
	app := testApp(false, true)
	app.cloneApps = fixtureApps()
	screen := newCloneAppsScreen(app, "")
	screen.counting = false
	screen.filtering = true
	screen.filter.SetValue("git")
	app.begin(screen)
	plain := stripSGR(renderCase(app, screen, 100, 26))
	for _, part := range []string{"Enter aplicar", "Esc cerrar", "F1/? ayuda"} {
		if !strings.Contains(plain, part) {
			t.Fatalf("the filter footer is missing %q:\n%s", part, plain)
		}
	}
	if strings.Count(plain, "ayuda") != 1 {
		t.Fatalf("the help suffix must not be duplicated:\n%s", plain)
	}

	settingsApp := testApp(false, true)
	settings := newSettingsScreen(settingsApp, "")
	settingsApp.begin(settings)
	settingsPlain := stripSGR(renderCase(settingsApp, settings, 100, 26))
	for _, part := range []string{"Ctrl+S guardar", "Esc volver", "←→ cursor", "F1/? ayuda"} {
		if !strings.Contains(settingsPlain, part) {
			t.Fatalf("the settings footer is missing %q:\n%s", part, settingsPlain)
		}
	}

	help := newHelpScreen(settingsApp, settings)
	helpPlain := stripSGR(renderCase(settingsApp, help, 100, 26))
	for _, part := range []string{"Ctrl+S guardar", "F1/? ayuda"} {
		if !strings.Contains(helpPlain, part) {
			t.Fatalf("the help screen is missing %q:\n%s", part, helpPlain)
		}
	}
	if !strings.Contains(helpPlain, texts.HelpGlobals) {
		t.Fatalf("the help screen lost its global list:\n%s", helpPlain)
	}

	app.begin(screen)
	send(t, app, keyMsg("f1"))
	if _, ok := app.current.(*helpScreen); !ok {
		t.Fatalf("F1 must open the help, got %T", app.current)
	}
	if screen.filter.Value != "git" {
		t.Fatalf("F1 must not insert a character: %q", screen.filter.Value)
	}
}

func TestSingleEditingCursorPerScreen(t *testing.T) {
	cases := []struct {
		name   string
		build  func(*App) Screen
		cursor int
	}{
		{"menu", func(app *App) Screen { return newMenuScreen(app) }, 0},
		{"c1-table", func(app *App) Screen {
			app.cloneApps = fixtureApps()
			screen := newCloneAppsScreen(app, "")
			screen.counting = false
			return screen
		}, 0},
		{"c1-filter", func(app *App) Screen {
			app.cloneApps = fixtureApps()
			screen := newCloneAppsScreen(app, "")
			screen.counting = false
			screen.filtering = true
			screen.filter.SetValue("gh")
			return screen
		}, 1},
		{"c2-options", func(app *App) Screen { return newCloneDestinationScreen(app, "") }, 0},
		{"c2-name", func(app *App) Screen {
			screen := newCloneDestinationScreen(app, "")
			screen.focus = 1
			return screen
		}, 1},
		{"c2-path", func(app *App) Screen {
			screen := newCloneDestinationScreen(app, "")
			screen.option = 2
			screen.focus = 2
			return screen
		}, 1},
		{"c2-dialog", func(app *App) Screen {
			screen := newCloneDestinationScreen(app, "")
			screen.dialog = true
			screen.focus = 1
			return screen
		}, 0},
		{"d0-table", func(app *App) Screen { return newImplantFileScreen(app, "") }, 0},
		{"d0-filter", func(app *App) Screen {
			screen := newImplantFileScreen(app, "")
			screen.filtering = true
			screen.filter.SetValue("clone")
			return screen
		}, 1},
		{"d0-manual", func(app *App) Screen {
			screen := newImplantFileScreen(app, "")
			screen.manual = true
			screen.manualPath.SetValue("/tmp/clone.tar.gz")
			return screen
		}, 1},
		{"settings", func(app *App) Screen { return newSettingsScreen(app, "") }, 1},
	}
	for _, testCase := range cases {
		app := testApp(false, true)
		screen := testCase.build(app)
		view := renderCase(app, screen, 100, 26)
		if got := reverseSGR(view); got != testCase.cursor {
			t.Fatalf("%s: expected %d editing cursors, got %d:\n%s", testCase.name, testCase.cursor, got, view)
		}
	}
	app := testApp(false, true)
	app.cloneApps = fixtureApps()
	table := newCloneAppsScreen(app, "")
	table.counting = false
	view := renderCase(app, table, 100, 26)
	if !strings.Contains(stripSGR(view), "▌") || reverseSGR(view) != 0 {
		t.Fatal("the table selection marker must not be a text cursor")
	}
	help := newHelpScreen(app, newSettingsScreen(app, ""))
	if reverseSGR(renderCase(app, help, 100, 26)) != 0 {
		t.Fatal("the help screen must not draw an editing cursor")
	}
}
