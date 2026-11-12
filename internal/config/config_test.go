package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAndMissingFile(t *testing.T) {
	home := t.TempDir()
	result := LoadConfig(home, "")
	if result.Error != nil || result.Exists {
		t.Fatalf("unexpected missing file result: %#v", result)
	}
	expected := DefaultConfig(home)
	if result.Config != expected {
		t.Fatalf("unexpected defaults: %#v", result.Config)
	}
}

func TestLoadValidAndInvalidConfig(t *testing.T) {
	home := t.TempDir()
	catalog := filepath.Join(home, "catalog")
	clones := filepath.Join(home, "clones")
	snapshots := filepath.Join(home, "snapshots")
	for _, directory := range []string{catalog, clones, snapshots} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(ConfigPath(home)), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "[catalog]\ndir = \"" + catalog + "\"\n\n[clone]\noutput_dir = \"" + clones + "\"\n\n[deploy]\nsnapshots_dir = \"" + snapshots + "\"\n"
	if err := os.WriteFile(ConfigPath(home), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	result := LoadConfig(home, "")
	if result.Error != nil || result.Config.CatalogDir != catalog || result.Config.CloneOutputDir != clones {
		t.Fatalf("unexpected config: %#v", result)
	}

	invalid := "[ui]\nplain = false\n"
	if err := os.WriteFile(ConfigPath(home), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	result = LoadConfig(home, "")
	if result.Error == nil || !result.Exists {
		t.Fatalf("expected invalid config error: %#v", result)
	}
	if result.Config != DefaultConfig(home) {
		t.Fatalf("invalid config must fall back: %#v", result.Config)
	}
}

func TestResolveAndSave(t *testing.T) {
	home := t.TempDir()
	if path, err := ResolvePath("~/catalog", home, "catalog.dir"); err != nil || path != filepath.Join(home, "catalog") {
		t.Fatalf("unexpected resolve: %q %v", path, err)
	}
	if _, err := ResolvePath("relative", home, "catalog.dir"); err == nil {
		t.Fatal("expected relative rejection")
	}
	if _, err := ResolvePath("~/../escape", home, "catalog.dir"); err != nil {
		// Python accepts this path lexically; it only rejects a doubled slash.
		t.Fatalf("unexpected rejection: %v", err)
	}
	configDir := filepath.Join(home, "catalog")
	clones := filepath.Join(home, "clones")
	snapshots := filepath.Join(home, "snapshots")
	value := AppConfig{CatalogDir: configDir, CloneOutputDir: clones, SnapshotsDir: snapshots}
	path, err := SaveConfig(value, home, "")
	if err != nil {
		t.Fatal(err)
	}
	loaded := LoadConfig(home, path)
	if loaded.Error != nil || loaded.Config != value {
		t.Fatalf("roundtrip failed: %#v", loaded)
	}
}

func TestSaveRejectsNonDirectory(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	value := DefaultConfig(home)
	value.CatalogDir = file
	if _, err := SaveConfig(value, home, ""); err == nil {
		t.Fatal("expected non-directory rejection")
	}
}
