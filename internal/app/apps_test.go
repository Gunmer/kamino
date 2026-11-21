package app

import (
	"context"
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
)

func TestCatalogDirResolution(t *testing.T) {
	home := t.TempDir()
	defaultDir, err := config.DefaultConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := CatalogDir(home); err != nil || got != defaultDir.CatalogDir {
		t.Fatalf("missing config must use the default: %q %v", got, err)
	}
	writeTestConfig(t, home, "[catalog]\ndir = \"~/kamino-apps\"\n")
	custom := filepath.Join(home, "kamino-apps")
	if got, err := CatalogDir(home); err != nil || got != custom {
		t.Fatalf("valid catalog.dir not used: %q %v", got, err)
	}
	writeTestConfig(t, home, "[catalog]\ndir = \"~/kamino-apps\"\n\n[clone]\noutput_dir = \"Downloads\"\n")
	if got, err := CatalogDir(home); err != nil || got != defaultDir.CatalogDir {
		t.Fatalf("any config error must use the default: %q %v", got, err)
	}
	writeTestConfig(t, home, "[catalog]\ndir = \"~/kamino-apps\"\n\n[ui]\nplain = false\n")
	if got, err := CatalogDir(home); err != nil || got != defaultDir.CatalogDir {
		t.Fatalf("an invalid section must use the default: %q %v", got, err)
	}
}

func TestCloneUsesCustomCatalogDir(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(home, "kamino-apps")
	writeTestConfig(t, home, "[catalog]\ndir = \""+custom+"\"\n")
	definition, err := catalog.RenderTemplate(catalog.TemplateOptions{
		ID: "miapp", Name: "Mi App", Command: "miapp-test", Paths: []string{".config/miapp/config"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, "miapp.toml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	writeTestExecutable(t, bin, "miapp-test")
	if err := os.MkdirAll(filepath.Join(home, ".config", "miapp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "miapp", "config"), []byte("value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requested := []string{"miapp"}
	outcome, err := ExecuteClone(context.Background(), &requested, "", true, false, nil, home, nil, bin)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, operation := range outcome.Result.Plan.Operations {
		if operation.AppID == "miapp" {
			found = true
		}
	}
	if !found {
		t.Fatalf("custom catalog app was not cloned: %#v", outcome.Result.Plan.Operations)
	}
	if outcome.Code != 0 {
		t.Fatalf("unexpected clone code: %d", outcome.Code)
	}
}

func TestDeployUsesCustomCatalogDir(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(home, "kamino-apps")
	writeTestConfig(t, home, "[catalog]\ndir = \""+custom+"\"\n")
	definition, err := catalog.RenderTemplate(catalog.TemplateOptions{
		ID: "miapp", Name: "Mi App", Command: "miapp-test", Paths: []string{".config/miapp/config"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, "miapp.toml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	writeTestExecutable(t, bin, "miapp-test")
	cloneFile := filepath.Join(t.TempDir(), "clone.tar.gz")
	writeTestClone(t, cloneFile, "miapp")
	outcome, err := ExecuteDeploy(context.Background(), cloneFile, nil, true, nil, home, nil, bin, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Result.NotInstalled) != 0 {
		t.Fatalf("custom app must be recognized as installed: %#v", outcome.Result.NotInstalled)
	}
	found := false
	for _, entry := range outcome.Result.Plan.Entries {
		if entry.AppID == "miapp" {
			found = true
		}
	}
	if !found {
		t.Fatalf("custom app missing from deploy plan: %#v", outcome.Result.Plan.Entries)
	}
}

func TestAppsNewListAndValidate(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	writeTestExecutable(t, bin, "miapp")
	outcome, err := ExecuteAppsNew(home, AppsNewOptions{
		ID: "miapp", Name: "Mi App", Command: "miapp", Paths: []string{".config/miapp"},
	}, nil, bin)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Detected {
		t.Fatal("the app should be detected with the injected PATH")
	}
	if outcome.Path != filepath.Join(home, ".config", "kamino", "apps", "miapp.toml") {
		t.Fatalf("unexpected path: %s", outcome.Path)
	}
	if len(outcome.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", outcome.Warnings)
	}
	validated, err := ExecuteAppsValidate(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if !validated.Valid || len(validated.Entries) != 1 || validated.Entries[0].ID != "miapp" {
		t.Fatalf("new definition must validate: %#v", validated)
	}
	listed, err := ExecuteAppsList(home)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range listed.Entries {
		if entry.ID == "miapp" && entry.Source == "usuario" {
			found = true
		}
	}
	if !found {
		t.Fatalf("new definition missing from list: %#v", listed.Entries)
	}
	if _, err := ExecuteAppsNew(home, AppsNewOptions{
		ID: "miapp", Name: "Mi App", Command: "miapp", Paths: []string{".config/miapp"},
	}, nil, bin); err == nil {
		t.Fatal("an existing target must require force")
	}
	if _, err := ExecuteAppsNew(home, AppsNewOptions{
		ID: "miapp", Name: "Mi App 2", Command: "miapp", Paths: []string{".config/miapp"}, Force: true,
	}, nil, bin); err != nil {
		t.Fatal(err)
	}
}

func TestAppsNewNotDetectedAndPrint(t *testing.T) {
	home := t.TempDir()
	outcome, err := ExecuteAppsNew(home, AppsNewOptions{
		ID: "miapp", Name: "Mi App", Command: "no-existe", Paths: []string{".config/miapp"},
	}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Detected {
		t.Fatal("a missing command must not be detected")
	}
	if !strings.Contains(strings.Join(outcome.Warnings, " "), "no está instalada") {
		t.Fatalf("missing detection warning: %#v", outcome.Warnings)
	}
	printed, err := ExecuteAppsNew(home, AppsNewOptions{
		ID: "otra", Name: "Otra", Command: "otra", Paths: []string{".config/otra"}, Print: true,
	}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if printed.Path != "" || !strings.Contains(printed.Content, `id = "otra"`) {
		t.Fatalf("--print must not write: %#v", printed)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "kamino", "apps", "otra.toml")); !os.IsNotExist(err) {
		t.Fatal("--print must not create a file")
	}
}

func TestAppsValidateReportsDuplicatesAndWarnings(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".config", "kamino", "apps")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `id = "miapp"
name = "Miapp"
command = "miapp"
paths = [".config/miapp"]
`
	if err := os.WriteFile(filepath.Join(directory, "miapp.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "otra.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outcome, err := ExecuteAppsValidate(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Valid {
		t.Fatal("duplicate ids must make the catalog invalid")
	}
	if len(outcome.Entries) != 2 {
		t.Fatalf("expected two entries, got %#v", outcome.Entries)
	}
	if outcome.Entries[0].Warnings == nil {
		t.Fatal("a definition without excludes must warn")
	}
	var duplicate bool
	for _, entry := range outcome.Entries {
		if entry.Err != nil && strings.Contains(entry.Err.Error(), "miapp.toml") && strings.Contains(entry.Err.Error(), "otra.toml") {
			duplicate = true
		}
	}
	if !duplicate {
		t.Fatalf("duplicate entry missing: %#v", outcome.Entries)
	}
}

func writeTestConfig(t *testing.T, home, content string) {
	t.Helper()
	path := filepath.Join(home, ".config", "kamino", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTestExecutable(t *testing.T, directory, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeTestClone(t *testing.T, cloneFile, appID string) {
	t.Helper()
	source := t.TempDir()
	path := filepath.Join(".config", appID, "config")
	if err := os.MkdirAll(filepath.Join(source, ".config", appID), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("value\n")
	if err := os.WriteFile(filepath.Join(source, path), data, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	value := hex.EncodeToString(digest[:])
	manifest := &archive.Manifest{
		FormatVersion: 1,
		ToolVersion:   "0.9.0",
		CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Arch:          "arm64",
		Hostname:      "host",
		Apps: []archive.AppEntry{{
			ID: appID, Name: appID, Files: []archive.FileEntry{{
				Path: path, SHA256: &value, Size: int64(len(data)), Mode: "0644",
			}},
		}},
	}
	if err := archive.WriteArchive(context.Background(), cloneFile, manifest, archive.WriteOptions{SourceRoot: source}); err != nil {
		t.Fatal(err)
	}
}
