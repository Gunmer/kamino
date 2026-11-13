package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/texts"
)

func TestDeployCandidatesOrderAndSources(t *testing.T) {
	home := t.TempDir()
	cwd := filepath.Join(home, ".config", "kamino", "vault")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(cwd, "clone-20261113.tar.gz")
	second := filepath.Join(cwd, "clone-20261112.tar.gz")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("clone"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(first, time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(second, time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC), time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	candidates, messages := discoverDeployCandidates(home, cwd)
	if len(candidates) != 2 {
		t.Fatalf("same directory must be deduplicated, got %d", len(candidates))
	}
	if candidates[0].Path != first || candidates[1].Path != second {
		t.Fatalf("candidates are not sorted by date: %#v", candidates)
	}
	if len(messages) != 0 {
		t.Fatalf("unexpected messages: %#v", messages)
	}
}

func TestDeployCandidatesReportUnreadableSource(t *testing.T) {
	home := t.TempDir()
	vault := filepath.Join(home, ".config", "kamino", "vault")
	if err := os.MkdirAll(filepath.Dir(vault), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vault, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, messages := discoverDeployCandidates(home, t.TempDir())
	if len(messages) == 0 || messages[0].Kind != MessageWarning {
		t.Fatalf("unreadable source must produce a warning: %#v", messages)
	}
}

func TestDeployAppsSelectionAndPreparation(t *testing.T) {
	app := testApp(false, true)
	apps := fixtureApps()[:3]
	for index := range apps {
		apps[index].Selected = false
	}
	screen := &deployAppsScreen{screenBase: screenBase{app: app}, apps: apps}
	app.begin(screen)
	send(t, app, keyMsg(" "))
	if !screen.apps[0].Selected {
		t.Fatal("space must select the focused app")
	}
	send(t, app, keyMsg("n"))
	if screen.apps[0].Selected {
		t.Fatal("n must deselect all")
	}
	send(t, app, keyMsg("enter"))
	if len(screen.messages) == 0 || screen.messages[0].Kind != MessageWarning {
		t.Fatal("enter without selection must warn")
	}
	screen.apps[0].Selected = true
	send(t, app, keyMsg("enter"))
	if screen.planning {
		t.Fatal("planning without a session must not start")
	}
}

func TestDeployPreparedMessageBuildsApps(t *testing.T) {
	app := testApp(false, true)
	app.deployOptions.Requested = &[]string{"herdr"}
	screen := &deployAppsScreen{screenBase: screenBase{app: app}, preparing: true}
	app.begin(screen)
	session := &deploy.DeploySession{
		DeployableApps:   []archive.AppEntry{{ID: "herdr", Name: "herdr", Files: make([]archive.FileEntry, 3)}},
		NotInstalledApps: []archive.AppEntry{{ID: "yazi", Name: "yazi"}},
		Warnings:         []string{"aviso"},
	}
	send(t, app, deployPreparedMsg{session: session})
	if screen.preparing || len(screen.apps) != 1 {
		t.Fatalf("prepared message not applied: %#v", screen)
	}
	if !screen.apps[0].Selected || screen.apps[0].Count != 3 {
		t.Fatalf("requested app must be preselected: %#v", screen.apps[0])
	}
	if len(screen.messages) == 0 {
		t.Fatal("close-apps warning missing")
	}
}

func TestDeployAppsSkipsSelectionWithRequested(t *testing.T) {
	app := testApp(false, true)
	requested := []string{"herdr"}
	app.deployOptions.Requested = &requested
	screen := newDeployAppsScreen(app, "/tmp/clone.tar.gz", true)
	app.begin(screen)
	session := &deploy.DeploySession{
		DeployableApps: []archive.AppEntry{{ID: "herdr", Name: "herdr", Files: make([]archive.FileEntry, 3)}},
	}
	command := send(t, app, deployPreparedMsg{session: session})
	if !screen.planning || command == nil {
		t.Fatalf("--apps must skip the selection table: planning=%v command=%v", screen.planning, command)
	}
	if !screen.apps[0].Selected {
		t.Fatal("requested app must stay selected")
	}
}

func TestNonTextConflictStartsUndecided(t *testing.T) {
	binary := conflicts.Conflict{
		AppID: "test", Path: ".config/bin", Destination: "/Users/test/.config/bin",
		Kind: conflicts.KindBinary,
	}
	plan := &deploy.DeployPlan{Conflicts: []conflicts.Conflict{binary}}
	states := newConflictStates(plan)
	if len(states) != 1 || states[0].Action != "" {
		t.Fatalf("non-text conflict must wait for an explicit decision, got %#v", states)
	}
}

func TestD3DeletionKeepsOriginalSide(t *testing.T) {
	local := "a\nb\nold\nc\n"
	cloneText := "a\nb\nc\n"
	conflict := makeConflict("test", ".config/test", local, cloneText)
	state := &conflictState{Conflict: conflict, Action: conflicts.ActionReview, Rows: conflicts.ReviewAll(conflict, conflicts.RowClone).Rows}
	app := testApp(false, true)
	app.conflicts = []*conflictState{state}
	screen := newDeployDetailScreen(app)
	rows := screen.rows
	var removal fileRow
	for _, row := range rows {
		if row.Kind == conflicts.RowLocalOnly {
			removal = row
		}
	}
	if removal.DecisionIndex < 0 {
		t.Fatal("expected a deletion row")
	}
	state.Rows[removal.DecisionIndex] = conflicts.RowLocal
	left, right := chosenCounts(rows, state)
	if left != 1 || right != 0 {
		t.Fatalf("deletion must count the original side: %d %d", left, right)
	}
	state.Rows[removal.DecisionIndex] = conflicts.RowClone
	left, right = chosenCounts(rows, state)
	if left != 0 || right != 0 {
		t.Fatalf("choosing the missing side must omit the line: %d %d", left, right)
	}
}

func TestD3ArrowKeysChooseVersion(t *testing.T) {
	app := testApp(false, true)
	app.conflicts = fixtureConflictStates()
	app.conflictIndex = 0
	screen := newDeployDetailScreen(app)
	app.begin(screen)
	state := screen.state()
	row := screen.rows[screen.index]
	send(t, app, keyMsg("left"))
	if state.Rows[row.DecisionIndex] != conflicts.RowLocal {
		t.Fatal("left must choose the original side")
	}
	send(t, app, keyMsg("right"))
	if state.Rows[row.DecisionIndex] != conflicts.RowClone {
		t.Fatal("right must choose the clone side")
	}
}

func TestD3BlockAndFileActions(t *testing.T) {
	app := testApp(false, true)
	app.conflicts = fixtureConflictStates()
	app.conflictIndex = 0
	screen := newDeployDetailScreen(app)
	app.begin(screen)
	state := screen.state()
	firstBlock := screen.rows[screen.index].Block
	send(t, app, keyMsg("o"))
	for _, row := range screen.rows {
		if row.Changed && row.Block == firstBlock && state.Rows[row.DecisionIndex] != conflicts.RowLocal {
			t.Fatal("o must put the current block on the original side")
		}
		if row.Changed && row.Block != firstBlock && state.Rows[row.DecisionIndex] != conflicts.RowClone {
			t.Fatal("o must not touch other blocks")
		}
	}
	send(t, app, keyMsg("tab"))
	nextBlock := screen.rows[screen.index].Block
	if nextBlock == firstBlock {
		t.Fatalf("tab must move to another block, still %d", nextBlock)
	}
	send(t, app, keyMsg("c"))
	for _, row := range screen.rows {
		if row.Changed && row.Block == nextBlock && state.Rows[row.DecisionIndex] != conflicts.RowClone {
			t.Fatal("c must put the current block on the clone side")
		}
	}
	send(t, app, keyMsg("O"))
	for _, decision := range state.Rows {
		if decision != conflicts.RowLocal {
			t.Fatal("O must put the whole file on the original side")
		}
	}
}

func TestD3ScrollFollowsCursor(t *testing.T) {
	app := testApp(false, true)
	app.conflicts = fixtureConflictStates()
	app.conflictIndex = 0
	screen := newDeployDetailScreen(app)
	screen.index = len(screen.rows) - 1
	view := renderCase(app, screen, 100, 24)
	if !strings.Contains(view, "arriba") {
		t.Fatalf("expected an upper scroll indicator:\n%s", view)
	}
	if screen.index == 0 {
		t.Fatal("cursor did not move")
	}
}

func TestD2WholeFileReviewIsStillPending(t *testing.T) {
	app := testApp(false, true)
	app.conflicts = []*conflictState{{
		Conflict: conflicts.Conflict{AppID: "test", Path: ".config/bin", Destination: "/Users/test/.config/bin", Kind: conflicts.KindBinary},
		Action:   conflicts.ActionReview,
	}}
	screen := newDeployConflictsScreen(app)
	app.begin(screen)
	command := send(t, app, keyMsg("enter"))
	if command != nil || len(screen.messages) == 0 || screen.messages[0].Kind != MessageWarning {
		t.Fatal("a whole-file review without a choice must block confirmation")
	}
}

func TestD5IncompleteVariant(t *testing.T) {
	app := testApp(false, true)
	result := &deploy.DeployResult{Deployed: []string{"a"}, Snapshot: "/Users/test/snapshot.tar.gz"}
	app.deployResult = result
	screen := newDeploySummaryScreen(app, result, true, ".config/test")
	app.begin(screen)
	if len(screen.messages) == 0 || screen.messages[0].Kind != MessageError {
		t.Fatal("incomplete deploy must show an ERROR")
	}
	view := renderCase(app, screen, 100, 26)
	if !strings.Contains(view, texts.D5Incomplete) {
		t.Fatalf("missing incomplete title:\n%s", view)
	}
	send(t, app, keyMsg("enter"))
	if app.ExitCode() != 1 {
		t.Fatalf("incomplete deploy must exit 1, got %d", app.ExitCode())
	}
}

func TestExitCodeInMenuMode(t *testing.T) {
	app := testApp(false, true)
	app.menuMode = true
	app.clonePlan = fixtureClonePlan()
	app.cloneResult = fixtureCloneResult()
	screen := newCloneReviewScreen(app)
	screen.phase = clonePhaseDone
	app.begin(screen)
	send(t, app, keyMsg("enter"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatalf("expected menu after finishing, got %T", app.current)
	}
	if app.ExitCode() != 3 {
		t.Fatalf("expected code 3, got %d", app.ExitCode())
	}
	menu := app.current.(*menuScreen)
	menu.index = 4
	command := send(t, app, keyMsg("enter"))
	message := execute(t, command)
	if _, ok := message.(quitMsg); !ok {
		t.Fatalf("menu exit must quit with the last code: %#v", message)
	}
	if message.(quitMsg).code != 3 {
		t.Fatalf("menu exit must preserve code 3, got %d", message.(quitMsg).code)
	}
}

func TestCloneSkipsDestinationWithOutput(t *testing.T) {
	home := t.TempDir()
	requested := []string{"test"}
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark",
		Clone: CloneFlowOptions{
			Requested: &requested, Output: filepath.Join(home, "clone.tar.gz"),
		},
	})
	app.catalog = fixtureCatalog()
	app.startClone()
	if _, ok := app.current.(*cloneReviewScreen); !ok {
		t.Fatalf("--output must skip C2 and open C3, got %T", app.current)
	}
	if len(app.stack) < 2 {
		t.Fatal("back stack must keep apps and destination")
	}
}
