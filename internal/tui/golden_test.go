package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/texts"
)

var updateGolden = flag.Bool("update", false, "update golden files")

type goldenMode struct {
	name    string
	noColor bool
	unicode bool
}

var (
	modeColor   = goldenMode{name: "color", unicode: true}
	modeNoColor = goldenMode{name: "no-color", noColor: true, unicode: true}
	modeASCII   = goldenMode{name: "ascii", unicode: false}
)

type goldenCase struct {
	name     string
	build    func(*App) Screen
	sizes    [][2]int
	variants bool
	small    bool
}

func standardSizes() [][2]int {
	return [][2]int{{80, 24}, {100, 26}, {140, 40}, {200, 50}}
}

func renderCase(app *App, screen Screen, width, height int) string {
	app.width, app.height = width, height
	app.current = screen
	app.layout()
	return app.View()
}

func runGolden(t *testing.T, testCase goldenCase) {
	t.Helper()
	modes := []goldenMode{modeColor}
	if testCase.variants {
		modes = append(modes, modeNoColor, modeASCII)
	}
	for _, mode := range modes {
		for _, size := range testCase.sizes {
			width, height := size[0], size[1]
			if width == 0 {
				width = 100
			}
			if height == 0 {
				height = 26
			}
			name := testCase.name
			if testCase.small {
				width, height = 79, 23
			}
			app := testApp(mode.noColor, mode.unicode)
			screen := testCase.build(app)
			view := renderCase(app, screen, width, height)
			path := goldenPath(name, mode, width, height)
			assertGolden(t, path, view)
		}
	}
}

func goldenPath(name string, mode goldenMode, width, height int) string {
	directory := filepath.Join("testdata", "golden", name)
	if mode.name != "color" {
		directory = filepath.Join(directory, mode.name)
	}
	return filepath.Join(directory, fmt.Sprintf("%dx%d.golden", width, height))
}

func assertGolden(t *testing.T, path, value string) {
	t.Helper()
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s; run go test ./internal/tui -run Golden -update: %v", path, err)
	}
	expected := strings.TrimSuffix(string(data), "\n")
	if expected != value {
		t.Fatalf("golden mismatch %s\n--- expected ---\n%s\n--- actual ---\n%s", path, expected, value)
	}
}

func goldenMenu() goldenCase {
	return goldenCase{
		name: "menu", sizes: [][2]int{{80, 24}, {80, 26}, {80, 30}, {80, 40}}, variants: true,
		build: func(app *App) Screen {
			app.lastMission = "clonación completada · 20 archivos"
			return newMenuScreen(app)
		},
	}
}

func goldenC1() goldenCase {
	return goldenCase{
		name: "c1", sizes: standardSizes(), variants: true,
		build: func(app *App) Screen {
			app.cloneApps = fixtureApps()
			screen := newCloneAppsScreen(app, "")
			screen.counting = false
			return screen
		},
	}
}

func goldenC2() goldenCase {
	return goldenCase{
		name: "c2", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			return newCloneDestinationScreen(app, "")
		},
	}
}

func goldenC2Manual() goldenCase {
	return goldenCase{
		name: "c2-manual", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newCloneDestinationScreen(app, "")
			screen.option = 2
			screen.focus = 2
			screen.path = "/Users/test/Descargas"
			return screen
		},
	}
}

func goldenC2Dialog() goldenCase {
	return goldenCase{
		name: "c2-dialog", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newCloneDestinationScreen(app, "")
			screen.dialog = true
			screen.dialogIndex = 2
			screen.focus = 1
			screen.existingAt = "/Users/test/.config/kamino/vault/clone-20261113.tar.gz"
			screen.existingMod = "2026-11-12 10:30"
			screen.existingSize = "1.2 MB"
			return screen
		},
	}
}

func goldenC3Review() goldenCase {
	return goldenCase{
		name: "c3-review", sizes: standardSizes(), variants: true,
		build: func(app *App) Screen {
			app.clonePlan = fixtureClonePlan()
			screen := newCloneReviewScreen(app)
			screen.messages = []Message{{Kind: MessageWarning, Text: texts.SecretsShort}}
			return screen
		},
	}
}

func goldenC3Writing() goldenCase {
	return goldenCase{
		name: "c3-writing", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			app.clonePlan = fixtureClonePlan()
			screen := newCloneReviewScreen(app)
			screen.phase = clonePhaseWriting
			screen.current = 12
			screen.total = 20
			screen.messages = []Message{{Kind: MessageWarning, Text: "Ctrl+C cancela antes del renombrado; el clon anterior queda intacto."}}
			return screen
		},
	}
}

func goldenC3Done() goldenCase {
	return goldenCase{
		name: "c3-done", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			app.clonePlan = fixtureClonePlan()
			app.cloneResult = fixtureCloneResult()
			screen := newCloneReviewScreen(app)
			screen.phase = clonePhaseDone
			screen.messages = []Message{{Kind: MessageWarning, Text: strings.ReplaceAll(texts.C3ExitWarning, "{count}", "3")}}
			return screen
		},
	}
}

func goldenC3DryRun() goldenCase {
	return goldenCase{
		name: "c3-dryrun", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			planned := fixtureClonePlan()
			app.clonePlan = planned
			result := *fixtureCloneResult()
			result.DryRun = true
			app.cloneResult = &result
			screen := newCloneReviewScreen(app)
			screen.phase = clonePhaseDone
			return screen
		},
	}
}

func goldenD0() goldenCase {
	return goldenCase{
		name: "d0", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			return &deployFileScreen{
				screenBase: screenBase{app: app}, candidates: fixtureDeployCandidates(),
			}
		},
	}
}

func goldenD1() goldenCase {
	return goldenCase{
		name: "d1", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			app.deploySession = &deploy.DeploySession{NotInstalledApps: []archive.AppEntry{
				{ID: "iterm2", Name: "iterm2"}, {ID: "yazi", Name: "yazi"},
			}}
			apps := fixtureApps()[:7]
			for index := range apps {
				apps[index].Selected = false
			}
			return &deployAppsScreen{
				screenBase: screenBase{app: app}, apps: apps,
				messages: []Message{{Kind: MessageWarning, Text: texts.D1CloseApps}},
			}
		},
	}
}

func goldenD2() goldenCase {
	return goldenCase{
		name: "d2", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			app.conflicts = fixtureConflictStates()
			app.deployPlan = fixtureDeployPlan(app.conflicts)
			return newDeployConflictsScreen(app)
		},
	}
}

func goldenD3() goldenCase {
	return goldenCase{
		name: "d3", sizes: standardSizes(), variants: true,
		build: func(app *App) Screen {
			app.conflicts = fixtureConflictStates()
			app.conflictIndex = 0
			app.deployPlan = fixtureDeployPlan(app.conflicts)
			return newDeployDetailScreen(app)
		},
	}
}

func goldenD4() goldenCase {
	return goldenCase{
		name: "d4", sizes: standardSizes(), variants: true,
		build: func(app *App) Screen {
			states := fixtureConflictStates()
			states[0].Rows[0] = conflicts.RowLocal
			states[1].Action = conflicts.ActionApply
			app.conflicts = states
			app.deployPlan = fixtureDeployPlan(states)
			return newDeployConfirmScreen(app)
		},
	}
}

func goldenD5() goldenCase {
	return goldenCase{
		name: "d5", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			return newDeploySummaryScreen(app, fixtureDeployResult(), false, "")
		},
	}
}

func goldenSettings() goldenCase {
	return goldenCase{
		name: "settings", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			return newSettingsScreen(app, "")
		},
	}
}

func goldenSettingsFocusOne() goldenCase {
	return goldenCase{
		name: "settings-focus1", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newSettingsScreen(app, "")
			screen.focus = 1
			return screen
		},
	}
}

func goldenSettingsFocusTwo() goldenCase {
	return goldenCase{
		name: "settings-focus2", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newSettingsScreen(app, "")
			screen.focus = 2
			return screen
		},
	}
}

func goldenSettingsError() goldenCase {
	return goldenCase{
		name: "settings-error", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newSettingsScreen(app, "")
			screen.focus = 0
			screen.messages = []Message{{Kind: MessageError, Text: "clone.output_dir: usa una ruta absoluta o que empiece por ~"}}
			return screen
		},
	}
}

func goldenU1Query() goldenCase {
	return goldenCase{
		name: "u1-query", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			return newUpdateScreen(app)
		},
	}
}

func goldenU1Download() goldenCase {
	return goldenCase{
		name: "u1-download", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newUpdateScreen(app)
			screen.phase = updatePhaseDownloading
			screen.check = fixtureUpdateCheck()
			screen.release = &screen.check.Release
			screen.current = 7 * 1024 * 1024
			screen.total = 12 * 1024 * 1024
			screen.messages = []Message{{Kind: MessageInfo, Text: texts.UpdateDownloading}}
			return screen
		},
	}
}

func goldenU1Result() goldenCase {
	return goldenCase{
		name: "u1-result", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			screen := newUpdateScreen(app)
			screen.phase = updatePhaseResult
			screen.check = fixtureUpdateCheck()
			screen.check.ShouldUpdate = false
			screen.release = &screen.check.Release
			screen.result = fixtureUpdateResult()
			return screen
		},
	}
}

func goldenHelp() goldenCase {
	return goldenCase{
		name: "help", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			app.conflicts = fixtureConflictStates()
			app.deployPlan = fixtureDeployPlan(app.conflicts)
			target := newDeployConflictsScreen(app)
			return newHelpScreen(app, target)
		},
	}
}

func goldenError() goldenCase {
	return goldenCase{
		name: "error", sizes: [][2]int{{100, 26}}, variants: true,
		build: func(app *App) Screen {
			app.menuMode = true
			return newErrorScreen(app, "No se pudo planificar el clon", texts.ErrorNothingWritten,
				"Revisa el destino y vuelve a intentarlo.",
				"open /Users/test/.config/kamino/vault: permission denied")
		},
	}
}

func goldenSmall() goldenCase {
	return goldenCase{
		name: "small", small: true, sizes: [][2]int{{79, 23}},
		build: func(app *App) Screen {
			return newMenuScreen(app)
		},
	}
}

type messageScreen struct {
	screenBase
	body     string
	messages []Message
}

func (s *messageScreen) Flow() string                     { return "Mensajes" }
func (s *messageScreen) Title() string                    { return "Mensajes" }
func (s *messageScreen) Step() (int, int)                 { return 0, 0 }
func (s *messageScreen) Update(tea.Msg) (Screen, tea.Cmd) { return s, nil }
func (s *messageScreen) Keys() []key.Binding              { return nil }
func (s *messageScreen) Messages() []Message              { return s.messages }
func (s *messageScreen) View(width, height int) string {
	return Fit([]string{s.body}, width, height, s.app.theme.Symbols.Ellipsis)
}

func goldenMessages() goldenCase {
	return goldenCase{
		name: "messages", sizes: [][2]int{{100, 26}},
		build: func(app *App) Screen {
			return &messageScreen{
				screenBase: screenBase{app: app}, body: "  Cuerpo de prueba",
				messages: []Message{
					{Kind: MessageInfo, Text: "Ruta copiada al portapapeles."},
					{Kind: MessageWarning, Text: "El clon no está cifrado y puede contener secretos."},
					{Kind: MessageError, Text: "No se pudo copiar la ruta."},
				},
			}
		},
	}
}

func TestGoldenMenu(t *testing.T) { runGolden(t, goldenMenu()) }
func TestGoldenC1(t *testing.T)   { runGolden(t, goldenC1()) }
func TestGoldenC2(t *testing.T) {
	runGolden(t, goldenC2())
	runGolden(t, goldenC2Manual())
	runGolden(t, goldenC2Dialog())
}
func TestGoldenC3(t *testing.T) {
	runGolden(t, goldenC3Review())
	runGolden(t, goldenC3Writing())
	runGolden(t, goldenC3Done())
	runGolden(t, goldenC3DryRun())
}
func TestGoldenD0D1D2(t *testing.T) {
	runGolden(t, goldenD0())
	runGolden(t, goldenD1())
	runGolden(t, goldenD2())
}
func TestGoldenD3(t *testing.T) { runGolden(t, goldenD3()) }
func TestGoldenD4(t *testing.T) { runGolden(t, goldenD4()) }
func TestGoldenD5(t *testing.T) { runGolden(t, goldenD5()) }
func TestGoldenSettings(t *testing.T) {
	runGolden(t, goldenSettings())
	runGolden(t, goldenSettingsFocusOne())
	runGolden(t, goldenSettingsFocusTwo())
	runGolden(t, goldenSettingsError())
}
func TestGoldenU1(t *testing.T) {
	runGolden(t, goldenU1Query())
	runGolden(t, goldenU1Download())
	runGolden(t, goldenU1Result())
}
func TestGoldenHelp(t *testing.T)     { runGolden(t, goldenHelp()) }
func TestGoldenError(t *testing.T)    { runGolden(t, goldenError()) }
func TestGoldenSmall(t *testing.T)    { runGolden(t, goldenSmall()) }
func TestGoldenMessages(t *testing.T) { runGolden(t, goldenMessages()) }

func TestContract(t *testing.T) {
	cases := []goldenCase{
		goldenMenu(), goldenC1(), goldenC2(), goldenC2Manual(), goldenC2Dialog(),
		goldenC3Review(), goldenC3Writing(), goldenC3Done(), goldenD3(), goldenD4(),
		goldenD5(), goldenSettings(), goldenSettingsFocusOne(), goldenSettingsFocusTwo(),
		goldenD0(), goldenD1(), goldenD2(),
		goldenSettingsError(), goldenU1Query(), goldenU1Download(), goldenU1Result(),
		goldenHelp(), goldenError(), goldenMessages(),
	}
	for _, testCase := range cases {
		for _, mode := range []goldenMode{modeColor, modeNoColor, modeASCII} {
			for _, size := range testCase.sizes {
				app := testApp(mode.noColor, mode.unicode)
				screen := testCase.build(app)
				view := renderCase(app, screen, size[0], size[1])
				assertContract(t, testCase.name, mode, size[0], size[1], view)
			}
		}
	}
	small := goldenSmall()
	view := renderCase(testApp(false, true), small.build(testApp(false, true)), 79, 23)
	assertContract(t, "small", modeColor, 79, 23, view)
}

func assertContract(t *testing.T, name string, mode goldenMode, width, height int, view string) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Fatalf("%s/%s/%dx%d: expected %d lines, got %d", name, mode.name, width, height, height, len(lines))
	}
	if strings.Contains(view, "\r") {
		t.Fatalf("%s: view contains carriage returns", name)
	}
	if strings.Contains(view, "\t") {
		t.Fatalf("%s: view contains tabs", name)
	}
	for index, line := range lines {
		if Width(line) > width {
			t.Fatalf("%s/%dx%d line %d is %d wide: %q", name, width, height, index+1, Width(line), line)
		}
	}
	if strings.Contains(view, "\x1b[38;2;") || strings.Contains(view, "\x1b[48;2;") {
		t.Fatalf("%s: truecolor sequence found", name)
	}
	allowed := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	if strings.Contains(allowed.ReplaceAllString(view, ""), "\x1b") {
		t.Fatalf("%s: non-SGR escape found", name)
	}
	if mode.noColor {
		colors := regexp.MustCompile(`\x1b\[(3[0-7]|9[0-7])(;[0-9;]*)?m`)
		if colors.MatchString(view) {
			t.Fatalf("%s: NO_COLOR view contains a color sequence", name)
		}
	}
}

func TestMessagesZone(t *testing.T) {
	app := testApp(false, true)
	screen := goldenMessages().build(app)
	view := renderCase(app, screen, 100, 26)
	info := strings.Index(view, texts.MessageInfoLabel)
	warning := strings.Index(view, texts.MessageWarningLabel)
	errorLabel := strings.Index(view, texts.MessageErrorLabel)
	if info < 0 || warning < info || errorLabel < warning {
		t.Fatalf("messages are not ordered INFO, AVISO, ERROR:\n%s", view)
	}
}
