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
		Install: []install.Recipe{
			{Method: install.MethodInstaller, URL: "https://example.com/demo-install.sh"},
		},
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
	apps := []appChoice{{ID: "demo", Name: "Demo", Selected: true, Count: 1}}
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
	send(t, app, keyMsg("enter"))
	send(t, app, keyMsg("enter"))
	if screen.installing || !screen.planning {
		t.Fatalf("rechazar todas las instalaciones debe seguir con el plan: installing=%v planning=%v", screen.installing, screen.planning)
	}
	if runner.runs != 0 {
		t.Fatal("rechazar no debe ejecutar nada")
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
	send(t, app, keyMsg(" "))
	if !screen.installChoices[0].Selected {
		t.Fatal("espacio debe marcar la app para instalarla")
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
