package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	kapp "github.com/Gunmer/kamino/internal/app"
)

func TestAuthoringIntegrationCLIAndTUI(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	writeExecutableForTest(t, bin, "cli-app")
	writeExecutableForTest(t, bin, "tui-app")
	for _, appID := range []string{"cli-app", "tui-app"} {
		directory := filepath.Join(home, ".config", appID)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "config"), []byte("value\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cliOutcome, err := kapp.ExecuteAppsNew(home, kapp.AppsNewOptions{
		ID: "cli-app", Name: "CLI App", Command: "cli-app", Paths: []string{".config/cli-app"},
	}, nil, bin)
	if err != nil {
		t.Fatal(err)
	}
	if !cliOutcome.Detected {
		t.Fatal("the CLI-created app should be detected")
	}

	app := NewApp(AppOptions{
		Home: home, Cwd: home, Now: fixedNow, ThemeName: "ansi-dark",
		PathValue: bin, MenuMode: true,
		Editor: writeAuthoringEditor(validAuthoringContent("tui-app")),
	})
	app.startClone()
	if app.catalog == nil || app.catalog["cli-app"] == nil {
		t.Fatal("the TUI did not see the CLI-created app")
	}
	if !hasCloneChoice(app, "cli-app") {
		t.Fatal("the CLI-created app is missing from C1")
	}

	screen := startAuthoringForTest(t, app)
	send(t, app, execute(t, send(t, app, keyMsg("enter"))))
	if screen.phase != authoringPhaseDone {
		t.Fatalf("the TUI did not create the app: %v (%#v)", screen.phase, screen.messages)
	}
	validated, err := kapp.ExecuteAppsValidate(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if !validated.Valid || len(validated.Entries) != 2 {
		t.Fatalf("the TUI-created app did not validate: %#v", validated)
	}

	for _, appID := range []string{"cli-app", "tui-app"} {
		requested := []string{appID}
		outcome, err := kapp.ExecuteClone(context.Background(), &requested, "", true, false, nil, home, nil, bin)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, operation := range outcome.Result.Plan.Operations {
			if operation.AppID == appID {
				found = true
			}
		}
		if !found {
			t.Fatalf("clone did not include %s", appID)
		}
	}
}

func hasCloneChoice(app *App, id string) bool {
	for _, choice := range app.cloneApps {
		if choice.ID == id {
			return true
		}
	}
	return false
}

func writeExecutableForTest(t *testing.T, directory, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
