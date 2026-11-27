package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/plan"
)

func treeApps() []appChoice {
	files := func(paths ...string) []fileChoice {
		result := make([]fileChoice, 0, len(paths))
		for _, path := range paths {
			result = append(result, fileChoice{Path: path, Size: 100})
		}
		return result
	}
	return []appChoice{
		{ID: "alpha", Name: "alpha", Detected: true, Count: 3, Files: files(".config/alpha/a", ".config/alpha/b", ".config/alpha/c")},
		{ID: "beta", Name: "beta", Detected: true, Count: 2, Files: files(".beta/x", ".beta/y")},
		{ID: "empty", Name: "empty", Detected: true, Count: 0},
		{ID: "gone", Name: "gone"},
	}
}

func treeScreen(t *testing.T) (*App, *cloneAppsScreen) {
	t.Helper()
	app := testApp(false, true)
	app.cloneApps = treeApps()
	screen := newCloneAppsScreen(app, "")
	screen.counting = false
	app.begin(screen)
	return app, screen
}

func TestC1CollapsedRowsAndExpand(t *testing.T) {
	app, screen := treeScreen(t)
	if rows := screen.rows(); len(rows) != 2 {
		t.Fatalf("collapsed list must show only listed apps: %#v", rows)
	}
	send(t, app, keyMsg("right"))
	if rows := screen.rows(); len(rows) != 5 {
		t.Fatalf("expanding must add the files: %#v", rows)
	}
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg("left"))
	if screen.index != 0 || len(screen.rows()) != 2 {
		t.Fatalf("collapsing from a file must return to the app: index=%d", screen.index)
	}
	if app.cloneApps[0].Selected || app.cloneApps[0].includedCount() != 0 {
		t.Fatal("expanding and collapsing must not touch the selection")
	}
}

func TestC1PartialSelection(t *testing.T) {
	app, screen := treeScreen(t)
	send(t, app, keyMsg(" "))
	if app.cloneApps[0].checkState() != checkAll || !app.cloneApps[0].Selected {
		t.Fatal("space on an app must mark every file")
	}
	send(t, app, keyMsg("right"))
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg(" "))
	if app.cloneApps[0].checkState() != checkPartial || !app.cloneApps[0].Selected {
		t.Fatal("unmarking one file must leave the app partial")
	}
	view := screen.View(100, 30)
	if !strings.Contains(view, "[~]") || !strings.Contains(view, "2/3") {
		t.Fatalf("partial state must show [~] and marked/total:\n%s", view)
	}
	send(t, app, keyMsg("up"))
	send(t, app, keyMsg(" "))
	if app.cloneApps[0].checkState() != checkAll {
		t.Fatal("space on a partial app must mark every file")
	}
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg("n"))
	if app.cloneApps[0].Selected || app.cloneApps[0].checkState() != checkNone {
		t.Fatal("n on a file must unmark the files of that app and deselect it")
	}
	if len(screen.selectedIDs()) != 0 {
		t.Fatal("an app with no marked files must not be selected")
	}
}

func TestC1ScopeActionsAndFilter(t *testing.T) {
	app, screen := treeScreen(t)
	send(t, app, keyMsg("a"))
	if !app.cloneApps[0].Selected || !app.cloneApps[1].Selected || app.cloneApps[2].Selected {
		t.Fatal("a on an app must select the visible apps")
	}
	send(t, app, keyMsg("right"))
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg("n"))
	if app.cloneApps[0].Selected || !app.cloneApps[1].Selected {
		t.Fatal("n on a file must only affect that app")
	}
	send(t, app, keyMsg("a"))
	if app.cloneApps[0].checkState() != checkAll {
		t.Fatal("a on a file must mark every file of that app")
	}
	screen.filter.SetValue("a.toml")
	if len(screen.rows()) != 0 {
		t.Fatal("the filter must search only app id and name")
	}
}

func TestC1CountRemovesFocusedApp(t *testing.T) {
	app := testApp(false, true)
	app.cloneApps = []appChoice{
		{ID: "one", Name: "one", Detected: true},
		{ID: "two", Name: "two", Detected: true},
	}
	screen := newCloneAppsScreen(app, "")
	app.begin(screen)
	screen.index = 1
	send(t, app, appsCountedMsg{
		counts: map[string]int{"one": 1, "two": 0},
		files:  map[string][]plan.FileInfo{"one": {{Path: ".one", Size: 1}}},
	})
	if len(screen.rows()) != 1 || screen.index != 0 {
		t.Fatalf("cursor must land on a valid row: index=%d rows=%d", screen.index, len(screen.rows()))
	}
	view := screen.View(100, 24)
	if !strings.Contains(view, "Sin archivos: two") || strings.Contains(view, "[-]") {
		t.Fatalf("app without files must move to the note:\n%s", view)
	}
}

func TestSelectedCloneFiles(t *testing.T) {
	app, _ := treeScreen(t)
	app.cloneApps[0].setAll(true)
	app.cloneApps[0].setFile(1, false)
	app.cloneApps[1].Selected = true // no detail: includes everything
	app.cloneApps[1].Files = nil
	files := app.selectedCloneFiles()
	if got := files["alpha"]; len(got) != 2 || got[0] != ".config/alpha/a" || got[1] != ".config/alpha/c" {
		t.Fatalf("unexpected included paths: %v", got)
	}
	if _, ok := files["beta"]; ok {
		t.Fatal("an app without detail must not be filtered")
	}
}

func TestCloneFlowPlansOnlyMarkedFiles(t *testing.T) {
	home := t.TempDir()
	bin, _, definitions := fakeCatalog(t)
	definitions["test"].Paths = []string{".config/test"}
	for _, name := range []string{"a", "b", "c"} {
		writeHomeFile(t, home, ".config/test/"+name)
	}
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Catalog: definitions, PathValue: bin,
		Now: fixedNow, ThemeName: "ansi-dark", Clone: CloneFlowOptions{DryRun: true},
	})
	app.startClone()
	send(t, app, execute(t, app.Init()))
	send(t, app, keyMsg(" "))
	send(t, app, keyMsg("right"))
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg(" "))
	if choice := app.cloneApps[0]; choice.checkState() != checkPartial || choice.includedCount() != 2 {
		t.Fatalf("expected a partial selection: %#v", choice)
	}
	message, ok := execute(t, planCloneCmd(app)).(clonePlanDoneMsg)
	if !ok || message.err != nil {
		t.Fatalf("unexpected plan result: %#v", message)
	}
	got := message.plan.Plan.Operations
	if len(got) != 2 || got[0].Path != ".config/test/b" || got[1].Path != ".config/test/c" {
		t.Fatalf("plan must contain only the marked files: %#v", got)
	}
}

func writeHomeFile(t *testing.T, home, rel string) {
	t.Helper()
	target := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
