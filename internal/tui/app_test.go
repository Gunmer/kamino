package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
)

func keyMsg(value string) tea.KeyMsg {
	switch value {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	}
}

func send(t *testing.T, app *App, msg tea.Msg) tea.Cmd {
	t.Helper()
	_, command := app.Update(msg)
	return command
}

func execute(t *testing.T, command tea.Cmd) tea.Msg {
	t.Helper()
	if command == nil {
		return nil
	}
	return command()
}

func navigate(t *testing.T, app *App, msg tea.Msg) {
	t.Helper()
	command := send(t, app, msg)
	if command == nil {
		return
	}
	send(t, app, execute(t, command))
}

func TestStackPreservesCursorAndFilter(t *testing.T) {
	app := testApp(false, true)
	app.cloneApps = fixtureApps()
	first := newCloneAppsScreen(app, "")
	first.counting = false
	first.index = 3
	first.filter = "git"
	app.begin(first)
	second := newCloneDestinationScreen(app, "")
	app.push(second)
	app.pop()
	if app.current != first {
		t.Fatal("pop must restore the same screen")
	}
	if first.index != 3 || first.filter != "git" {
		t.Fatalf("cursor/filter lost: %d %q", first.index, first.filter)
	}
}

func TestHelpFromQuestionAndF1(t *testing.T) {
	app := testApp(false, true)
	app.begin(newMenuScreen(app))
	command := send(t, app, keyMsg("?"))
	if command != nil {
		t.Fatal("help must not run a command")
	}
	if _, ok := app.current.(*helpScreen); !ok {
		t.Fatalf("expected help, got %T", app.current)
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatalf("expected menu after closing help, got %T", app.current)
	}
	send(t, app, keyMsg("f1"))
	if _, ok := app.current.(*helpScreen); !ok {
		t.Fatal("F1 must open help")
	}
}

func TestSmallTerminalKeepsState(t *testing.T) {
	app := testApp(false, true)
	app.cloneApps = fixtureApps()
	screen := newCloneAppsScreen(app, "")
	screen.counting = false
	screen.index = 4
	screen.filter = "git"
	app.begin(screen)
	app.width, app.height = 79, 23
	if view := app.View(); !strings.Contains(view, texts.SmallTitle) {
		t.Fatalf("expected small guard:\n%s", view)
	}
	app.width, app.height = 100, 26
	view := app.View()
	if strings.Contains(view, texts.SmallTitle) {
		t.Fatal("guard must disappear after resize")
	}
	if screen.index != 4 || screen.filter != "git" {
		t.Fatal("resize must not touch screen state")
	}
}

func TestErrorScreenHasFourParts(t *testing.T) {
	app := testApp(false, true)
	app.begin(newMenuScreen(app))
	send(t, app, errorMsg{
		what: "No se pudo planificar el clon", safety: texts.ErrorNothingWritten,
		action: "Revisa el destino.", detail: "open: permission denied",
	})
	errorScreen, ok := app.current.(*errorScreen)
	if !ok {
		t.Fatalf("expected error screen, got %T", app.current)
	}
	view := renderCase(app, errorScreen, 100, 26)
	for _, part := range []string{errorScreen.what, texts.ErrorSafety, texts.ErrorAction} {
		if !strings.Contains(view, part) {
			t.Fatalf("missing %q in error view", part)
		}
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatalf("expected menu after error, got %T", app.current)
	}
}

func TestMenuNavigationAndNoSingleKeyShortcuts(t *testing.T) {
	app := testApp(false, true)
	app.begin(newMenuScreen(app))
	menu := app.current.(*menuScreen)
	send(t, app, keyMsg("down"))
	if menu.index != 1 {
		t.Fatalf("expected index 1, got %d", menu.index)
	}
	send(t, app, keyMsg("c"))
	if menu.index != 1 {
		t.Fatal("single-key shortcuts must not run menu options")
	}
	navigate(t, app, keyMsg("esc"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatal("escape must not close the menu")
	}
}

func TestC1SelectionFilterAndWarnings(t *testing.T) {
	app := testApp(false, true)
	app.cloneApps = fixtureApps()
	for index := range app.cloneApps {
		app.cloneApps[index].Selected = false
	}
	screen := newCloneAppsScreen(app, "")
	screen.counting = false
	app.begin(screen)
	if screen.index != 0 {
		t.Fatal("cursor must start at the first app")
	}
	send(t, app, keyMsg(" "))
	if !app.cloneApps[0].Selected {
		t.Fatal("space must select the focused app")
	}
	if screen.index != 0 {
		t.Fatal("space must not move the cursor")
	}
	send(t, app, keyMsg("n"))
	if app.cloneApps[0].Selected || app.cloneApps[1].Selected {
		t.Fatal("n must deselect visible rows")
	}
	send(t, app, keyMsg("enter"))
	if len(screen.messages) == 0 || screen.messages[0].Kind != MessageWarning {
		t.Fatal("enter without selection must warn")
	}
	screen.index = 4
	send(t, app, keyMsg(" "))
	if app.cloneApps[4].Selected {
		t.Fatal("an app with zero files must not be selectable")
	}
	screen.filtering = false
	send(t, app, keyMsg("/"))
	send(t, app, keyMsg("g"))
	send(t, app, keyMsg("h"))
	send(t, app, keyMsg("enter"))
	if screen.filter != "gh" || screen.filtering {
		t.Fatalf("filter not applied: %q %v", screen.filter, screen.filtering)
	}
	send(t, app, keyMsg("a"))
	if !app.cloneApps[1].Selected {
		t.Fatal("a must select visible matches")
	}
	if app.cloneApps[2].Selected {
		t.Fatal("a must not select filtered-out rows")
	}
}

func TestC2OptionsFocusAndValidation(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, Catalog: fixtureCatalog(),
		ThemeName: "ansi-dark", Phrase: "frase",
	})
	screen := newCloneDestinationScreen(app, "")
	app.begin(screen)
	if screen.option != 0 || screen.focus != 0 {
		t.Fatalf("unexpected initial focus: %d %d", screen.option, screen.focus)
	}
	send(t, app, keyMsg("enter"))
	if screen.focus != 1 {
		t.Fatal("enter on an option must focus the name input")
	}
	send(t, app, keyMsg("tab"))
	if screen.focus != 0 {
		t.Fatalf("tab must return to the options, got %d", screen.focus)
	}
	screen.option = 2
	screen.path = filepath.Join(home, "Destino")
	screen.focus = 2
	send(t, app, keyMsg("tab"))
	if screen.focus == 2 {
		t.Fatalf("tab must move the focus away from the path, got %d", screen.focus)
	}
	screen.focus = 1
	screen.name = ""
	command := send(t, app, keyMsg("enter"))
	if command != nil || len(screen.messages) == 0 {
		t.Fatal("invalid name must keep the screen and show an error")
	}
	if screen.messages[len(screen.messages)-1].Kind != MessageError {
		t.Fatal("invalid name must be an ERROR")
	}
}

func TestC2ExistingDialogAndFreeSuffix(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	target := filepath.Join(home, "clone-20261113.tar.gz")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	screen := newCloneDestinationScreen(app, "")
	screen.option = 2
	screen.path = home
	screen.focus = 1
	screen.name = "clone-20261113.tar.gz"
	screen.accept()
	if !screen.dialog || screen.dialogIndex != 2 {
		t.Fatalf("expected dialog with Volver selected: %#v", screen)
	}
	screen.updateDialog(keyMsg("up"))
	screen.updateDialog(keyMsg("enter"))
	if screen.dialog {
		t.Fatal("Cambiar nombre must close the dialog")
	}
	if screen.name != "clone-20261113-1.tar.gz" {
		t.Fatalf("unexpected free suffix: %q", screen.name)
	}
}

func TestC2OverwriteConfirmsReplacement(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	target := filepath.Join(home, "clone-20261113.tar.gz")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	screen := newCloneDestinationScreen(app, "")
	screen.option = 2
	screen.path = home
	screen.focus = 1
	screen.name = "clone-20261113.tar.gz"
	screen.accept()
	screen.updateDialog(keyMsg("up"))
	screen.updateDialog(keyMsg("up"))
	_, command := screen.updateDialog(keyMsg("enter"))
	if !app.cloneForce || screen.dialog || command == nil {
		t.Fatalf("Sobrescribir must force and continue: force=%v dialog=%v", app.cloneForce, screen.dialog)
	}
}

func TestC2InvalidOutputPrefill(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark",
		Clone: CloneFlowOptions{Output: filepath.Join(home, "invalido", "x.tar.gz"), OutputError: "directorio ausente"},
	})
	screen := newCloneDestinationScreen(app, app.cloneOptions.OutputError)
	if screen.option != 2 || screen.focus != 2 {
		t.Fatalf("invalid --output must open manual folder input: %d %d", screen.option, screen.focus)
	}
	if screen.name != "x.tar.gz" {
		t.Fatalf("name must be prefilled, got %q", screen.name)
	}
}

func TestC2ForceSkipsDialog(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark",
		Clone: CloneFlowOptions{Force: true},
	})
	target := filepath.Join(home, "clone-20261113.tar.gz")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	screen := newCloneDestinationScreen(app, "")
	screen.option = 2
	screen.path = home
	screen.focus = 1
	screen.name = "clone-20261113.tar.gz"
	_, command := screen.accept()
	if screen.dialog {
		t.Fatal("--force must skip the dialog")
	}
	if command == nil {
		t.Fatal("accept must navigate to review")
	}
}

func TestCloneFlowDryRun(t *testing.T) {
	home := t.TempDir()
	bin, _, definitions := fakeCatalog(t)
	if err := os.MkdirAll(filepath.Join(home, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "test", "config"), []byte("value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Catalog: definitions, PathValue: bin,
		Now: fixedNow, ThemeName: "ansi-dark", Clone: CloneFlowOptions{DryRun: true},
	})
	app.startClone()
	if _, ok := app.current.(*cloneAppsScreen); !ok {
		t.Fatalf("expected C1, got %T", app.current)
	}
	command := app.Init()
	send(t, app, execute(t, command))
	if app.cloneApps[0].Count != 1 {
		t.Fatalf("count command did not update C1: %#v", app.cloneApps[0])
	}
	command = send(t, app, keyMsg(" "))
	_ = command
	command = send(t, app, keyMsg("enter"))
	send(t, app, execute(t, command))
	if _, ok := app.current.(*cloneDestinationScreen); !ok {
		t.Fatalf("expected C2, got %T", app.current)
	}
	command = send(t, app, keyMsg("enter"))
	send(t, app, execute(t, command))
	if _, ok := app.current.(*cloneDestinationScreen); !ok {
		t.Fatalf("expected C2, got %T", app.current)
	}
	send(t, app, keyMsg("enter"))
	command = send(t, app, keyMsg("enter"))
	planCommand := send(t, app, execute(t, command))
	review, ok := app.current.(*cloneReviewScreen)
	if !ok {
		t.Fatalf("expected C3, got %T", app.current)
	}
	send(t, app, execute(t, planCommand))
	if review.phase != clonePhaseReady {
		t.Fatalf("expected ready, got %v", review.phase)
	}
	send(t, app, keyMsg("enter"))
	if review.phase != clonePhaseDone || app.cloneResult == nil || !app.cloneResult.DryRun {
		t.Fatalf("expected dry-run result: %v %#v", review.phase, app.cloneResult)
	}
	command = send(t, app, keyMsg("enter"))
	if command == nil || app.ExitCode() != 0 {
		t.Fatalf("expected quit with code 0, got %v %d", command, app.ExitCode())
	}
}

func TestCloneWriteAndExitThree(t *testing.T) {
	home := t.TempDir()
	bin, _, definitions := fakeCatalog(t)
	if err := os.MkdirAll(filepath.Join(home, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "test", "config"), []byte("value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(AppOptions{
		Home: home, Cwd: home, Catalog: definitions, PathValue: bin,
		Now: fixedNow, ThemeName: "ansi-dark",
	})
	app.startClone()
	send(t, app, execute(t, app.Init()))
	send(t, app, keyMsg(" "))
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	send(t, app, keyMsg("enter"))
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	review, ok := app.current.(*cloneReviewScreen)
	if !ok {
		t.Fatalf("expected C3, got %T", app.current)
	}
	send(t, app, execute(t, app.Init()))
	if review.phase != clonePhaseReady {
		t.Fatalf("expected ready, got %v", review.phase)
	}
	command := send(t, app, keyMsg("enter"))
	message := execute(t, command)
	command = send(t, app, message)
	for command != nil {
		message = execute(t, command)
		command = send(t, app, message)
		if review.phase == clonePhaseDone || review.phase == clonePhaseCancelled {
			break
		}
	}
	if review.phase != clonePhaseDone || app.cloneResult == nil {
		t.Fatalf("expected completed clone: %v %#v", review.phase, app.cloneResult)
	}
	if _, err := os.Stat(app.cloneResult.Output); err != nil {
		t.Fatalf("clone was not written: %v", err)
	}
	send(t, app, keyMsg("enter"))
	if app.ExitCode() != 0 && app.ExitCode() != 3 {
		t.Fatalf("unexpected exit code: %d", app.ExitCode())
	}
}

func TestCloneCancelBeforeWrite(t *testing.T) {
	app := testApp(false, true)
	app.clonePlan = fixtureClonePlan()
	app.cloneOutput = app.clonePlan.Output
	screen := newCloneReviewScreen(app)
	app.begin(screen)
	send(t, app, keyMsg("ctrl+c"))
	if screen.phase != clonePhaseCancelled {
		t.Fatalf("expected cancelled phase, got %v", screen.phase)
	}
	navigate(t, app, keyMsg("esc"))
	if app.current != nil {
		t.Fatalf("direct flow must exit after cancel, got %T", app.current)
	}
	if app.ExitCode() != 1 {
		t.Fatalf("unexpected exit code: %d", app.ExitCode())
	}
}

func TestC3ExitThreeAndCopy(t *testing.T) {
	app := testApp(false, true)
	app.clonePlan = fixtureClonePlan()
	app.cloneResult = fixtureCloneResult()
	screen := newCloneReviewScreen(app)
	screen.phase = clonePhaseDone
	screen.messages = []Message{{Kind: MessageWarning, Text: "3 omitidos"}}
	app.begin(screen)
	send(t, app, keyMsg("c"))
	if len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Text != texts.C3CopyInfo {
		t.Fatal("copy must report INFO in the message zone")
	}
	send(t, app, keyMsg("enter"))
	if app.ExitCode() != 3 {
		t.Fatalf("expected exit code 3, got %d", app.ExitCode())
	}
}

func TestD3FileRowsCRLFAndFinalNewline(t *testing.T) {
	conflict := makeConflict("test", ".config/test", "a\r\nb\r\nc", "a\r\nB\r\nc")
	rows := buildFileRows(conflict)
	if len(rows) == 0 {
		t.Fatal("expected rows")
	}
	for _, row := range rows {
		if strings.Contains(row.LocalText, "\r") || strings.Contains(row.CloneText, "\r") {
			t.Fatalf("row has carriage return: %#v", row)
		}
	}
	last := rows[len(rows)-1]
	if last.LocalText != "c" || last.CloneText != "c" {
		t.Fatalf("final line without newline lost: %#v", last)
	}
}

func TestD3InsertionOmissionAndCounters(t *testing.T) {
	local := "a\nb\nc\n"
	cloneText := "a\nb\nnew\nc\n"
	conflict := makeConflict("test", ".config/test", local, cloneText)
	state := &conflictState{Conflict: conflict, Action: conflicts.ActionReview, Rows: conflicts.ReviewAll(conflict, conflicts.RowClone).Rows}
	app := testApp(false, true)
	app.conflicts = []*conflictState{state}
	screen := newDeployDetailScreen(app)
	if len(state.Rows) != 1 {
		t.Fatalf("expected one decision, got %d", len(state.Rows))
	}
	rows := screen.rows
	var insertion fileRow
	for _, row := range rows {
		if row.Kind == conflicts.RowCloneOnly {
			insertion = row
		}
	}
	if insertion.DecisionIndex < 0 {
		t.Fatal("expected an insertion row")
	}
	state.Rows[insertion.DecisionIndex] = conflicts.RowLocal
	left, right := chosenCounts(screen.rows, state)
	if left != 0 || right != 0 {
		t.Fatalf("omitting the only inserted line must not count: %d %d", left, right)
	}
}

func TestD3SingleDecisionAndLongLine(t *testing.T) {
	local := "line one\nline two\n"
	cloneText := "line one changed\nline two\n"
	conflict := makeConflict("test", ".config/test", local, cloneText)
	state := &conflictState{Conflict: conflict, Action: conflicts.ActionReview, Rows: conflicts.ReviewAll(conflict, conflicts.RowClone).Rows}
	app := testApp(false, true)
	app.conflicts = []*conflictState{state}
	screen := newDeployDetailScreen(app)
	first := screen.rows[0]
	screen.setRow(first, conflicts.RowLocal)
	for index, row := range screen.rows {
		if index != 0 && row.Changed && state.Rows[row.DecisionIndex] != conflicts.RowClone {
			t.Fatal("changing one row must not change the others")
		}
	}
	long := strings.Repeat("x", 200)
	row := fileRow{LocalNo: 1, LocalText: long, CloneNo: 1, CloneText: long, Kind: conflicts.RowEqual}
	cell := screen.cell(state, row, false, 4, 20, false)
	if Width(cell) > 20 || !strings.Contains(cell, "…") {
		t.Fatalf("long line not truncated: %q", cell)
	}
}

func TestD3ViewportOnlyRendersVisibleRows(t *testing.T) {
	localLines := make([]string, 0, 20000)
	cloneLines := make([]string, 0, 20000)
	hunkRows := make([]conflicts.HunkRow, 0, 20000)
	for index := 0; index < 20000; index++ {
		line := "line-" + strconv.Itoa(index)
		localLines = append(localLines, line)
		cloneLines = append(cloneLines, line+"-clone")
		hunkRows = append(hunkRows, conflicts.HunkRow{Kind: conflicts.RowReplace, Local: line + "\n", Clone: line + "-clone\n"})
	}
	local := strings.Join(localLines, "\n") + "\n"
	cloneText := strings.Join(cloneLines, "\n") + "\n"
	conflict := conflicts.Conflict{
		AppID: "test", Path: ".config/large", Destination: "/Users/test/.config/large",
		Kind: conflicts.KindText, LocalText: &local, CloneText: &cloneText,
		Hunks: []conflicts.Hunk{{Index: 1, LocalStart: 1, LocalEnd: 20001, CloneStart: 1, CloneEnd: 20001, Rows: hunkRows}},
	}
	state := &conflictState{Conflict: conflict, Action: conflicts.ActionReview, Rows: conflicts.ReviewAll(conflict, conflicts.RowClone).Rows}
	app := testApp(false, true)
	app.conflicts = []*conflictState{state}
	screen := newDeployDetailScreen(app)
	view := renderCase(app, screen, 100, 30)
	if !strings.Contains(view, "line-0") {
		t.Fatal("visible window missing first row")
	}
	if strings.Contains(view, "line-19999") {
		t.Fatal("view rendered off-screen rows")
	}
}

func TestD4AggregationTotalsAndDedup(t *testing.T) {
	states := fixtureConflictStates()
	states[0].Rows[0] = conflicts.RowLocal
	states[1].Action = conflicts.ActionApply
	app := testApp(false, true)
	app.conflicts = states
	app.deployPlan = fixtureDeployPlan(states)
	app.deployPlan.Entries = append(app.deployPlan.Entries, deploy.DeployEntry{
		AppID: "otra", Path: states[0].Conflict.Path, Status: deploy.StatusNew,
	})
	screen := newDeployConfirmScreen(app)
	if screen.totals.New != 1 {
		t.Fatalf("shared path must be counted once, totals=%#v", screen.totals)
	}
	if screen.totals.Hybrid != 1 || screen.totals.Replaced != 1 {
		t.Fatalf("unexpected categories: %#v", screen.totals)
	}
	if screen.protected != 2 {
		t.Fatalf("protected must be replacements plus hybrids, got %d", screen.protected)
	}
	if screen.index != 1 {
		t.Fatalf("Retirada must be preselected, got %d", screen.index)
	}
	send(t, app, keyMsg("d"))
	if screen.applying {
		t.Fatal("no single key may deploy")
	}
}

func TestD4IdenticalHybridIsUnchanged(t *testing.T) {
	states := fixtureConflictStates()
	state := states[0]
	for index := range state.Rows {
		state.Rows[index] = conflicts.RowLocal
	}
	app := testApp(false, true)
	app.conflicts = states
	app.deployPlan = fixtureDeployPlan(states)
	screen := newDeployConfirmScreen(app)
	if screen.rows[0].Hybrid != 0 {
		t.Fatalf("an identical hybrid must not be a hybrid: %#v", screen.rows[0])
	}
	if screen.rows[0].Unchanged != 1 {
		t.Fatalf("an identical hybrid must count as unchanged: %#v", screen.rows[0])
	}
}

func TestD2ActionsAndMissingDecision(t *testing.T) {
	app := testApp(false, true)
	app.conflicts = fixtureConflictStates()
	app.deployPlan = fixtureDeployPlan(app.conflicts)
	screen := newDeployConflictsScreen(app)
	app.begin(screen)
	send(t, app, keyMsg("c"))
	if app.conflicts[0].Action != conflicts.ActionApply {
		t.Fatal("c must apply the clone")
	}
	send(t, app, keyMsg("o"))
	if app.conflicts[0].Action != conflicts.ActionOmit {
		t.Fatal("o must keep the original")
	}
	app.conflicts[1].Action = ""
	send(t, app, keyMsg("enter"))
	if len(screen.messages) == 0 || screen.messages[0].Kind != MessageWarning {
		t.Fatal("a missing decision must warn")
	}
}

func TestD5CategoriesCopyAndCode(t *testing.T) {
	app := testApp(false, true)
	result := fixtureDeployResult()
	app.deployResult = result
	screen := newDeploySummaryScreen(app, result, false, "")
	app.begin(screen)
	view := renderCase(app, screen, 100, 26)
	for _, label := range []string{texts.D5Deployed, texts.D5Hybrids, texts.D5Omitted, texts.D5Unchanged, texts.D5NotInstalled, texts.D5Protected} {
		if !strings.Contains(view, label) {
			t.Fatalf("missing %q in summary", label)
		}
	}
	send(t, app, keyMsg("c"))
	if len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Kind != MessageInfo {
		t.Fatal("copying the snapshot must report INFO")
	}
	send(t, app, keyMsg("enter"))
	if app.ExitCode() != 3 {
		t.Fatalf("expected code 3 for omitted files, got %d", app.ExitCode())
	}
}

func TestSettingsEditSaveAndError(t *testing.T) {
	home := t.TempDir()
	app := NewApp(AppOptions{Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark"})
	app.startSettings("")
	screen := app.current.(*settingsScreen)
	target := filepath.Join(home, "catalog")
	screen.fields[0] = target
	send(t, app, keyMsg("ctrl+s"))
	if len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Kind != MessageInfo {
		t.Fatalf("save must report INFO: %#v", screen.messages)
	}
	loaded := config.LoadConfig(home, "")
	if loaded.Config.CatalogDir != target {
		t.Fatalf("config not saved: %#v", loaded.Config)
	}
	screen.fields[0] = "relative/path"
	send(t, app, keyMsg("ctrl+s"))
	if len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Kind != MessageError {
		t.Fatal("invalid path must report ERROR")
	}
}

func TestUpdateResultAndFinish(t *testing.T) {
	app := NewApp(AppOptions{
		MenuMode: true, Home: "/Users/test", Cwd: "/Users/test", Now: fixedNow, ThemeName: "ansi-dark",
		UpdateQuery: func() (update.Check, error) {
			check := fixtureUpdateCheck()
			check.ShouldUpdate = false
			return check, nil
		},
	})
	app.startUpdate()
	screen := app.current.(*updateScreen)
	send(t, app, execute(t, app.Init()))
	if screen.phase != updatePhaseResult || screen.result == nil {
		t.Fatalf("unexpected update state: %v %#v", screen.phase, screen.result)
	}
	send(t, app, keyMsg("enter"))
	if _, ok := app.current.(*menuScreen); !ok {
		t.Fatalf("expected menu after update, got %T", app.current)
	}
}

func TestUpdateProgressAndCancellation(t *testing.T) {
	app := NewApp(AppOptions{
		MenuMode: true, Home: "/Users/test", Cwd: "/Users/test", Now: fixedNow, ThemeName: "ansi-dark",
		UpdateQuery: func() (update.Check, error) {
			return fixtureUpdateCheck(), nil
		},
		UpdateRun: func(callback progress.Callback, cancelled progress.Cancel) (update.Result, error) {
			progress.Emit(callback, progress.Downloading, 5, 10, "")
			if cancelled() {
				return update.Result{}, errors.New("cancelada")
			}
			progress.Emit(callback, progress.Replacing, 0, 1, "")
			return *fixtureUpdateResult(), nil
		},
	})
	app.startUpdate()
	screen := app.current.(*updateScreen)
	send(t, app, execute(t, app.Init()))
	command := send(t, app, keyMsg("enter"))
	if screen.phase != updatePhaseDownloading {
		t.Fatalf("expected downloading, got %v", screen.phase)
	}
	send(t, app, execute(t, command))
	command = send(t, app, operationEvent{progress: progress.Event{Phase: progress.Downloading, Current: 5, Total: 10}})
	if screen.phase != updatePhaseDownloading || screen.current != 5 {
		t.Fatalf("progress not applied: %v %d", screen.phase, screen.current)
	}
	send(t, app, execute(t, command))
	send(t, app, operationEvent{progress: progress.Event{Phase: progress.Replacing}})
	if screen.phase != updatePhaseReplacing {
		t.Fatalf("expected replacing, got %v", screen.phase)
	}
}

func TestCtrlCAfterNoReturnIsIgnored(t *testing.T) {
	app := testApp(false, true)
	app.clonePlan = fixtureClonePlan()
	screen := newCloneReviewScreen(app)
	screen.phase = clonePhaseWriting
	screen.current = 10
	screen.total = 20
	app.begin(screen)
	send(t, app, keyMsg("ctrl+c"))
	if !app.cloneCancel.Load() {
		t.Fatal("Ctrl+C must request cancellation while writing")
	}
	send(t, app, operationEvent{err: nil, done: true, clone: fixtureCloneResult()})
	if screen.phase != clonePhaseDone {
		t.Fatalf("a finished rename must complete even after Ctrl+C: %v", screen.phase)
	}
}

func TestDeployAggregationWithSession(t *testing.T) {
	source := t.TempDir()
	cloneFile := filepath.Join(t.TempDir(), "clone.tar.gz")
	buildClone(t, source, cloneFile)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".config", "test", "config"), []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, _, definitions := fakeCatalog(t)
	session, err := deploy.PrepareDeploy(cloneFile, deploy.Options{
		Home: destination, Catalog: definitions, Path: bin,
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := []string{"test"}
	plan, err := session.Plan(&selected, true, selected, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	states := newConflictStates(plan)
	app := NewApp(AppOptions{Home: destination, Cwd: destination, Now: fixedNow, ThemeName: "ansi-dark"})
	app.catalog = definitions
	app.deploySession = session
	app.deployPlan = plan
	app.conflicts = states
	screen := newDeployConfirmScreen(app)
	if screen.previewErr != "" {
		t.Fatalf("Preview rejected the decisions: %s", screen.previewErr)
	}
	if screen.totals.Replaced+screen.totals.Hybrid == 0 {
		t.Fatalf("expected a protected write: %#v", screen.totals)
	}
}

func TestDeployFlowEndToEnd(t *testing.T) {
	source := t.TempDir()
	cloneFile := filepath.Join(t.TempDir(), "clone.tar.gz")
	buildClone(t, source, cloneFile)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".config", "test", "config"), []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, _, definitions := fakeCatalog(t)
	app := NewApp(AppOptions{
		Home: destination, Cwd: destination, Catalog: definitions, PathValue: bin,
		Now: fixedNow, ThemeName: "ansi-dark", Deploy: DeployFlowOptions{CloneFile: cloneFile},
	})
	app.startDeploy()
	if _, ok := app.current.(*deployAppsScreen); !ok {
		t.Fatalf("expected D1 preparing, got %T", app.current)
	}
	send(t, app, execute(t, app.Init()))
	if _, ok := app.current.(*deployAppsScreen); !ok {
		t.Fatalf("expected D1 apps, got %T", app.current)
	}
	send(t, app, keyMsg("a"))
	command := send(t, app, keyMsg("enter"))
	message := execute(t, command)
	command = send(t, app, message)
	for command != nil {
		message = execute(t, command)
		command = send(t, app, message)
		if _, ok := app.current.(*deployConflictsScreen); ok {
			break
		}
	}
	if _, ok := app.current.(*deployConflictsScreen); !ok {
		t.Fatalf("expected D2, got %T", app.current)
	}
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	if _, ok := app.current.(*deployConfirmScreen); !ok {
		t.Fatalf("expected D4, got %T", app.current)
	}
	confirm := app.current.(*deployConfirmScreen)
	confirm.index = 2
	command = send(t, app, keyMsg("enter"))
	message = execute(t, command)
	command = send(t, app, message)
	for command != nil {
		message = execute(t, command)
		command = send(t, app, message)
		if _, ok := app.current.(*deploySummaryScreen); ok {
			break
		}
	}
	if _, ok := app.current.(*deploySummaryScreen); !ok {
		t.Fatalf("expected D5, got %T", app.current)
	}
	data, err := os.ReadFile(filepath.Join(destination, ".config", "test", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "value=2\n" {
		t.Fatalf("unexpected deployed content: %q", data)
	}
	send(t, app, keyMsg("enter"))
	if app.ExitCode() != 0 {
		t.Fatalf("unexpected exit code: %d", app.ExitCode())
	}
}

func fakeCatalog(t *testing.T) (string, string, catalog.Catalog) {
	t.Helper()
	bin := t.TempDir()
	command := "kamino-tui-test"
	if err := os.WriteFile(filepath.Join(bin, command), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	definitions := catalog.Catalog{
		"test": {ID: "test", Name: "Test", Command: &command, Paths: []string{".config/test/config"}},
	}
	return bin, command, definitions
}

func buildClone(t *testing.T, home, cloneFile string) {
	t.Helper()
	data := []byte("value=2\n")
	digest := sha256.Sum256(data)
	value := hex.EncodeToString(digest[:])
	manifest := &archive.Manifest{
		FormatVersion: 1, ToolVersion: "0.9.0", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		MacOSVersion: "", Arch: "arm64", Hostname: "h",
		Apps: []archive.AppEntry{{ID: "test", Name: "Test", Files: []archive.FileEntry{{
			Path: ".config/test/config", SHA256: &value, Size: int64(len(data)), Mode: "0644",
		}}}},
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "test", "config"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := archive.WriteArchive(cloneFile, manifest, archive.WriteOptions{SourceRoot: home}); err != nil {
		t.Fatal(err)
	}
}
