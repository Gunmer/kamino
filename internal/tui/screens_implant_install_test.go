package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/implant"
	"github.com/Gunmer/kamino/internal/install"
	"github.com/Gunmer/kamino/internal/texts"
)

// noToolRunner never finds brew and never runs anything, so no test here can
// reach a real installer even if a command is executed.
type noToolRunner struct{ runs int }

func (r *noToolRunner) Lookup(string) bool { return false }

func (r *noToolRunner) Run(context.Context, string, ...string) ([]byte, error) {
	r.runs++
	return nil, errors.New("no debe ejecutarse")
}

// installableSession prepares a real implant session in which the app "demo" is
// missing and has an installer recipe from the catalog.
func installableSession(t *testing.T) (*implant.ImplantSession, string) {
	return installableSessionWith(t, []install.Recipe{
		{Method: install.MethodInstaller, URL: "https://example.com/demo-install.sh"},
	})
}

func installableSessionWith(t *testing.T, recipes []install.Recipe) (*implant.ImplantSession, string) {
	t.Helper()
	sourceHome := t.TempDir()
	home := t.TempDir()
	bin := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sourceHome, ".config", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, ".config", "demo", "config"), []byte("value=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("value=2\n"))
	digest := hex.EncodeToString(sum[:])
	manifest := &archive.Manifest{
		FormatVersion: 1, ToolVersion: "0.15.0", CreatedAt: "2026-01-01T00:00:00Z", Arch: "arm64", Hostname: "h",
		Apps: []archive.AppEntry{{ID: "demo", Name: "Demo", Files: []archive.FileEntry{{
			Path: ".config/demo/config", SHA256: &digest, Size: 8, Mode: "0644",
		}}}},
	}
	cloneFile := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := archive.WriteArchive(context.Background(), cloneFile, manifest, archive.WriteOptions{SourceRoot: sourceHome}); err != nil {
		t.Fatal(err)
	}
	command := "kamino-demo"
	definitions := catalog.Catalog{"demo": {
		ID: "demo", Name: "Demo", Command: &command, Paths: []string{".config/demo/config"},
		Install: recipes,
	}}
	session, err := implant.PrepareImplant(cloneFile, implant.Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return session, home
}

func installTestScreen(t *testing.T) (*App, *implantAppsScreen, *noToolRunner) {
	t.Helper()
	session, _ := installableSession(t)
	t.Cleanup(session.Close)
	app := testApp(false, true)
	runner := &noToolRunner{}
	app.installRunner = runner
	app.implantSession = session
	apps := []appChoice{{ID: "demo", Name: "Demo", Installable: true, Selected: true, Count: 1}}
	screen := &implantAppsScreen{screenBase: screenBase{app: app}, apps: apps}
	app.begin(screen)
	return app, screen, runner
}

func TestInstallQuestionAppearsBeforePlanWhenAppHasRecipe(t *testing.T) {
	app, screen, _ := installTestScreen(t)
	send(t, app, keyMsg("enter"))
	if !screen.installing || screen.planning {
		t.Fatalf("debe preguntar antes de planificar: installing=%v planning=%v", screen.installing, screen.planning)
	}
	view := screen.View(100, 26)
	if !strings.Contains(view, texts.D1InstallQ) || !strings.Contains(view, "installer") || !strings.Contains(view, "https://example.com/demo-install.sh") {
		t.Fatalf("la pregunta debe mostrar método y URL exacta:\n%s", view)
	}
}

func TestInstallQuestionDeclineKeepsImplantGoing(t *testing.T) {
	app, screen, runner := installTestScreen(t)
	screen.apps = append(screen.apps, appChoice{ID: "other", Name: "Other", Detected: true, Selected: true, Count: 1})
	send(t, app, keyMsg("enter"))
	send(t, app, keyMsg("n"))
	send(t, app, keyMsg("enter"))
	if screen.installing || !screen.planning {
		t.Fatalf("rechazar todas las instalaciones debe seguir con el plan: installing=%v planning=%v", screen.installing, screen.planning)
	}
	if runner.runs != 0 {
		t.Fatal("rechazar no debe ejecutar nada")
	}
}

func TestInstallQuestionDeclineWithNothingElseWarns(t *testing.T) {
	app, screen, _ := installTestScreen(t)
	send(t, app, keyMsg("enter"))
	send(t, app, keyMsg("n"))
	send(t, app, keyMsg("enter"))
	if screen.planning || screen.installing {
		t.Fatal("sin apps que implantar no debe iniciarse un plan vacío")
	}
	if len(screen.messages) == 0 || screen.messages[0].Text != texts.D1NoneSelected {
		t.Fatalf("debe avisar de que no hay nada seleccionado: %v", screen.messages)
	}
}

func TestInstallQuestionEscReturnsWithoutCancelling(t *testing.T) {
	app, screen, _ := installTestScreen(t)
	send(t, app, keyMsg("enter"))
	send(t, app, keyMsg("esc"))
	if screen.installing || screen.planning {
		t.Fatalf("esc debe volver a la elección de apps: installing=%v planning=%v", screen.installing, screen.planning)
	}
	if !screen.apps[0].Selected {
		t.Fatal("la elección de apps debe conservarse")
	}
}

func TestInstallQuestionSpaceMarksAndConfirmStartsPlan(t *testing.T) {
	app, screen, _ := installTestScreen(t)
	send(t, app, keyMsg("enter"))
	if !screen.installChoices[0].Selected {
		t.Fatal("las apps marcadas en D1 llegan marcadas a la confirmación")
	}
	send(t, app, keyMsg(" "))
	send(t, app, keyMsg(" "))
	if !screen.installChoices[0].Selected {
		t.Fatal("espacio alterna la marca de la instalación")
	}
	send(t, app, keyMsg("enter"))
	if screen.installing || !screen.planning {
		t.Fatalf("confirmar debe iniciar el plan: installing=%v planning=%v", screen.installing, screen.planning)
	}
}

func TestNoInstallQuestionWithoutRecipes(t *testing.T) {
	app := testApp(false, true)
	app.implantSession = &implant.ImplantSession{}
	apps := []appChoice{{ID: "demo", Name: "Demo", Selected: true, Count: 1}}
	screen := &implantAppsScreen{screenBase: screenBase{app: app}, apps: apps}
	app.begin(screen)
	send(t, app, keyMsg("enter"))
	if screen.installing {
		t.Fatal("sin apps instalables no debe aparecer la pregunta")
	}
	if !screen.planning {
		t.Fatal("sin pregunta el plan debe iniciarse")
	}
}

func TestInstallQuestionWarnsWhenReleaseHasNoChecksum(t *testing.T) {
	app, screen, _ := installTestScreen(t)
	send(t, app, keyMsg("enter"))
	screen.installChoices[0].Recipes = []install.Recipe{{Method: install.MethodGithubRelease, Repo: "acme/demo"}}
	view := screen.View(120, 26)
	if !strings.Contains(view, "https://github.com/acme/demo/releases") || !strings.Contains(view, texts.D1InstallNoSum) {
		t.Fatalf("debe mostrar el origen y el aviso de sin suma:\n%s", view)
	}
	screen.installChoices[0].Recipes[0].Checksums = "sums.txt"
	if strings.Contains(screen.View(120, 26), texts.D1InstallNoSum) {
		t.Fatal("con checksums no debe haber aviso")
	}
}

func TestAppsScreenListsMissingAppsWithStatusAndAsksBeforeInstalling(t *testing.T) {
	session, _ := installableSession(t)
	t.Cleanup(session.Close)
	app := testApp(false, true)
	app.installRunner = &noToolRunner{}
	screen := &implantAppsScreen{screenBase: screenBase{app: app}, preparing: true}
	app.begin(screen)
	send(t, app, implantPreparedMsg{session: session})
	if len(screen.apps) != 1 || screen.apps[0].Detected || !screen.apps[0].Installable {
		t.Fatalf("la app ausente debe listarse como instalable: %#v", screen.apps)
	}
	view := screen.View(100, 26)
	if !strings.Contains(view, texts.D1ColumnState) || !strings.Contains(view, texts.D1StateMissing) {
		t.Fatalf("D1 debe mostrar el estado de cada app:\n%s", view)
	}
	send(t, app, keyMsg("enter"))
	if screen.installing || len(screen.messages) == 0 || screen.messages[len(screen.messages)-1].Text != texts.D1NoneSelected {
		t.Fatalf("sin marcar nada debe avisar: %v", screen.messages)
	}
	send(t, app, keyMsg(" "))
	send(t, app, keyMsg("enter"))
	if !screen.installing || len(screen.installChoices) != 1 || !screen.installChoices[0].Selected {
		t.Fatalf("marcar una app ausente debe pedir confirmar su instalación: %+v", screen.installChoices)
	}
}

func TestAppsScreenDoesNotLetMarkMissingAppsWithoutRecipe(t *testing.T) {
	app := testApp(false, true)
	app.implantSession = &implant.ImplantSession{}
	screen := &implantAppsScreen{screenBase: screenBase{app: app}, apps: []appChoice{
		{ID: "ok", Name: "Ok", Detected: true, Count: 1},
		{ID: "nope", Name: "Nope", Count: 1},
	}}
	app.begin(screen)
	send(t, app, keyMsg("a"))
	if !screen.apps[0].Selected || screen.apps[1].Selected {
		t.Fatalf("'a' solo marca lo seleccionable: %+v", screen.apps)
	}
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg(" "))
	if screen.apps[1].Selected {
		t.Fatal("una app ausente sin receta no se puede marcar")
	}
	if view := screen.View(100, 26); !strings.Contains(view, "[-]") {
		t.Fatalf("debe mostrarse deshabilitada:\n%s", view)
	}
}

type brewRunner struct{ noToolRunner }

func (r *brewRunner) Lookup(name string) bool { return name == "brew" }

func TestInstallConfirmationListsEveryWayAndLetsChooseOne(t *testing.T) {
	session, _ := installableSessionWith(t, []install.Recipe{
		{Method: install.MethodBrew, Formula: "demo"},
		{Method: install.MethodGithubRelease, Repo: "acme/demo", Asset: "demo", Format: install.FormatRaw, Dest: ".local/bin/demo"},
	})
	t.Cleanup(session.Close)
	app := testApp(false, true)
	app.installRunner = &brewRunner{}
	app.implantSession = session
	screen := &implantAppsScreen{screenBase: screenBase{app: app}}
	app.begin(screen)
	screen.apps = []appChoice{{ID: "demo", Name: "Demo", Installable: true, Selected: true, Count: 1}}
	send(t, app, keyMsg("enter"))
	view := screen.View(120, 26)
	for _, want := range []string{"brew", "github-release", "https://github.com/acme/demo/releases", texts.D1InstallNoSum} {
		if !strings.Contains(view, want) {
			t.Fatalf("la confirmación debe listar todas las vías, falta %q:\n%s", want, view)
		}
	}
	if screen.installChoices[0].Option != 0 {
		t.Fatal("la vía por defecto es la primera del catálogo")
	}
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg("down"))
	send(t, app, keyMsg(" "))
	if screen.installChoices[0].Option != 1 || !screen.installChoices[0].Selected {
		t.Fatalf("Espacio sobre una vía la elige y marca la app: %+v", screen.installChoices[0])
	}
}
