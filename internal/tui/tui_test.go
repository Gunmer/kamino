package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
)

func ptr(value string) *string { return &value }

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

func modelFor(t *testing.T, home, bin string, definitions catalog.Catalog) *Model {
	t.Helper()
	model := NewModel(CloneFlowOptions{})
	model.Home = home
	model.Catalog = definitions
	model.PathValue = bin
	model.Width, model.Height = 80, 24
	model.Now = func() time.Time { return time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC) }
	return model
}

func keyRune(value rune) Key { return Key{Kind: KeyRune, Rune: value} }

func TestSizeGuardAndMenuNavigation(t *testing.T) {
	model := NewModel(CloneFlowOptions{})
	model.Width, model.Height = 79, 23
	if !strings.Contains(model.View(), texts.ErrorSmallTitle) {
		t.Fatal("expected small terminal guard")
	}
	model.Width, model.Height = 80, 24
	view := model.View()
	if !strings.Contains(view, texts.MenuClone) || !strings.Contains(view, texts.MenuExit) {
		t.Fatalf("unexpected menu: %s", view)
	}
	model.Update(Key{Kind: KeyDown})
	if model.MenuIndex != 1 {
		t.Fatalf("unexpected menu index: %d", model.MenuIndex)
	}
	model.Update(Key{Kind: KeyUp})
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenC1 {
		t.Fatalf("expected clone screen, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEsc})
	if model.Screen != ScreenMenu {
		t.Fatalf("expected menu after escape, got %s", model.Screen)
	}
}

func TestHelpAndNoColor(t *testing.T) {
	model := NewModel(CloneFlowOptions{})
	model.NoColor = true
	view := model.View()
	if strings.Contains(view, "\x1b[") {
		t.Fatal("NO_COLOR must remove ANSI codes")
	}
	model.Update(keyRune('?'))
	if model.Screen != ScreenHelp || !strings.Contains(model.View(), texts.HelpTitle) {
		t.Fatalf("expected help: %s", model.View())
	}
	model.Update(Key{Kind: KeyEsc})
	if model.Screen != ScreenMenu {
		t.Fatalf("expected menu after help, got %s", model.Screen)
	}
}

func TestCloneFlowDryRun(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "test", "config"), []byte("value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, _, definitions := fakeCatalog(t)
	model := modelFor(t, home, bin, definitions)
	model.startClone()
	if model.Screen != ScreenC1 {
		t.Fatalf("expected C1, got %s", model.Screen)
	}
	if model.CloneApps[0].Selected {
		t.Fatal("apps must start unselected")
	}
	model.Update(keyRune(' '))
	if !model.CloneApps[0].Selected {
		t.Fatal("space must select")
	}
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenC2 {
		t.Fatalf("expected C2, got %s", model.Screen)
	}
	model.CloneDryRun = true
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenC3 || model.ClonePlan == nil {
		t.Fatalf("expected C3 with plan, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenC4 || model.CloneResult == nil || !model.CloneResult.DryRun {
		t.Fatalf("expected C4 dry run, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEnter})
	if !model.Quit {
		t.Fatal("summary must finish")
	}
}

func TestCloneCancelBeforeWrite(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "test", "config"), []byte("value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, _, definitions := fakeCatalog(t)
	model := modelFor(t, home, bin, definitions)
	model.startClone()
	model.CloneDryRun = true
	model.Update(keyRune(' '))
	model.Update(Key{Kind: KeyEnter})
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenC3 {
		t.Fatalf("expected C3, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyCtrlC})
	if model.Screen != ScreenC1 {
		t.Fatalf("expected C1 after cancel, got %s", model.Screen)
	}
	model.CloneWriting = true
	model.Screen = ScreenC3
	model.Update(Key{Kind: KeyCtrlC})
	if model.Screen != ScreenC3 {
		t.Fatal("Ctrl+C must not cancel after writing started")
	}
}

func buildClone(t *testing.T, home, cloneFile string) {
	t.Helper()
	data := []byte("value=2\n")
	digest := sha256.Sum256(data)
	value := hex.EncodeToString(digest[:])
	manifest := &archive.Manifest{
		FormatVersion: 1, ToolVersion: "0.8.0", CreatedAt: "2026-01-01T00:00:00Z",
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

func TestDeployFlow(t *testing.T) {
	source := t.TempDir()
	cloneFile := filepath.Join(t.TempDir(), "clone.tar.gz")
	buildClone(t, source, cloneFile)
	destination := t.TempDir()
	bin, _, definitions := fakeCatalog(t)
	model := modelFor(t, destination, bin, definitions)
	model.DeployFile = cloneFile
	model.startDeploy()
	if model.Screen != ScreenD1 {
		t.Fatalf("expected D1, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenD2 {
		t.Fatalf("expected D2, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenD4 {
		t.Fatalf("expected D4, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyDown})
	model.Update(Key{Kind: KeyDown})
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenD5 || model.DeployResult == nil {
		t.Fatalf("expected D5 after deploy, got %s", model.Screen)
	}
	if data, _ := os.ReadFile(filepath.Join(destination, ".config/test/config")); string(data) != "value=2\n" {
		t.Fatalf("unexpected deployed content: %q", data)
	}
	model.Update(Key{Kind: KeyEnter})
	if !model.Quit {
		t.Fatal("summary must finish")
	}
}

func TestDeployConflictDetailAndError(t *testing.T) {
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
	model := modelFor(t, destination, bin, definitions)
	model.DeployFile = cloneFile
	model.startDeploy()
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenD2 || len(model.Conflicts) != 1 {
		t.Fatalf("expected one conflict: %s %#v", model.Screen, model.Conflicts)
	}
	model.Update(keyRune('r'))
	if model.Screen != ScreenD3 {
		t.Fatalf("expected D3, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEnter})
	model.Update(Key{Kind: KeyEsc})
	if model.Screen != ScreenD2 {
		t.Fatalf("expected D2, got %s", model.Screen)
	}
	model.Update(Key{Kind: KeyEnter})
	model.Update(Key{Kind: KeyDown})
	model.Update(Key{Kind: KeyDown})
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenD5 {
		t.Fatalf("expected D5, got %s", model.Screen)
	}

	bad := modelFor(t, destination, bin, definitions)
	bad.openDeployFile(filepath.Join(t.TempDir(), "missing.tar.gz"))
	if bad.Error == nil || bad.Screen != ScreenError {
		t.Fatalf("expected four-part error, got %s", bad.Screen)
	}
	view := bad.View()
	for _, section := range []string{texts.ErrorSafety, texts.ErrorAction} {
		if !strings.Contains(view, section) {
			t.Fatalf("missing %s in %s", section, view)
		}
	}
}

func TestUpdateFlowInjected(t *testing.T) {
	model := NewModel(CloneFlowOptions{})
	model.MenuMode = true
	model.UpdateCheckFunc = func() (update.Result, error) {
		return update.Result{CurrentVersion: "0.8.0", LatestVersion: "0.9.0", Target: "/tmp/kamino", Updated: true, Outcome: update.OutcomeUpdated}, nil
	}
	model.startUpdate()
	if model.Screen != ScreenU1 || model.UpdateResult == nil {
		t.Fatalf("expected U1 result: %s %#v", model.Screen, model.UpdateResult)
	}
	model.Update(Key{Kind: KeyEnter})
	if model.Screen != ScreenMenu {
		t.Fatalf("expected menu, got %s", model.Screen)
	}
}

func TestSettingsSave(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "kamino"), 0o755); err != nil {
		t.Fatal(err)
	}
	model := NewModel(CloneFlowOptions{})
	model.Home = home
	model.startSettings()
	if model.Screen != ScreenSettings || len(model.SettingsFields) != 3 {
		t.Fatalf("unexpected settings: %s %#v", model.Screen, model.SettingsFields)
	}
	// Replace the first field with an absolute temp directory.
	target := filepath.Join(home, "catalog")
	model.SettingsFields[0] = target
	model.Update(Key{Kind: KeyCtrlS})
	if model.SettingsError != "" || model.Settings.CatalogDir != target {
		t.Fatalf("unexpected save result: %q %#v", model.SettingsError, model.Settings)
	}
	loaded := config.LoadConfig(home, "")
	if loaded.Error != nil || loaded.Config.CatalogDir != target {
		t.Fatalf("config not saved: %#v", loaded)
	}
}
