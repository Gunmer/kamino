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
	expected := mustDefaultConfig(t, home)
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
	if err := os.MkdirAll(filepath.Dir(mustConfigPath(t, home)), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "[catalog]\ndir = \"" + catalog + "\"\n\n[clone]\noutput_dir = \"" + clones + "\"\n\n[implant]\nsnapshots_dir = \"" + snapshots + "\"\n"
	if err := os.WriteFile(mustConfigPath(t, home), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	result := LoadConfig(home, "")
	if result.Error != nil || result.Config.CatalogDir != catalog || result.Config.CloneOutputDir != clones {
		t.Fatalf("unexpected config: %#v", result)
	}

	invalid := "[ui]\nplain = false\n"
	if err := os.WriteFile(mustConfigPath(t, home), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	result = LoadConfig(home, "")
	if result.Error == nil || !result.Exists {
		t.Fatalf("expected invalid config error: %#v", result)
	}
	if result.Config != mustDefaultConfig(t, home) {
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
		// The lexical validator accepts this path; it only rejects a doubled slash.
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

func TestSnapshotsLimitDefaultAndLoad(t *testing.T) {
	home := t.TempDir()
	if got := LoadConfig(home, "").Config.SnapshotsLimit; got != DefaultSnapshotsLimit {
		t.Fatalf("missing file must use the default limit, got %d", got)
	}
	if DefaultSnapshotsLimit != 5 {
		t.Fatalf("default limit must be 5, got %d", DefaultSnapshotsLimit)
	}
	writeConfig := func(content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(mustConfigPath(t, home)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(mustConfigPath(t, home), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig("[implant]\nsnapshots_limit = 3\n")
	loaded := LoadConfig(home, "")
	if loaded.Error != nil {
		t.Fatalf("unexpected error: %v", loaded.Error)
	}
	if loaded.Config.SnapshotsLimit != 3 {
		t.Fatalf("expected the configured limit, got %d", loaded.Config.SnapshotsLimit)
	}
	if loaded.Raw[SnapshotsLimitKey] != "3" {
		t.Fatalf("expected the raw limit for the settings screen, got %q", loaded.Raw[SnapshotsLimitKey])
	}
	writeConfig("[implant]\nsnapshots_limit = 0\n")
	loaded = LoadConfig(home, "")
	if loaded.Error != nil || loaded.Config.SnapshotsLimit != 0 {
		t.Fatalf("zero must disable the limit: %#v", loaded)
	}
}

func TestSnapshotsLimitRejectsInvalidValues(t *testing.T) {
	cases := map[string]string{
		"negative":     "[implant]\nsnapshots_limit = -1\n",
		"string":       "[implant]\nsnapshots_limit = \"5\"\n",
		"float":        "[implant]\nsnapshots_limit = 2.5\n",
		"pathAsNumber": "[catalog]\ndir = 5\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(mustConfigPath(t, home)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(mustConfigPath(t, home), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			loaded := LoadConfig(home, "")
			if loaded.Error == nil {
				t.Fatalf("expected a configuration error for %s", name)
			}
			if loaded.Config != mustDefaultConfig(t, home) {
				t.Fatalf("invalid configuration must fall back to defaults: %#v", loaded.Config)
			}
		})
	}
}

func TestSaveAndLoadSnapshotsLimit(t *testing.T) {
	home := t.TempDir()
	value := mustDefaultConfig(t, home)
	value.SnapshotsLimit = 2
	path, err := SaveConfig(value, home, "")
	if err != nil {
		t.Fatal(err)
	}
	loaded := LoadConfig(home, path)
	if loaded.Error != nil || loaded.Config != value {
		t.Fatalf("limit roundtrip failed: %#v", loaded)
	}
	value.SnapshotsLimit = -1
	if _, err := SaveConfig(value, home, ""); err == nil {
		t.Fatal("expected negative limit rejection")
	}
}

func TestParseSnapshotsLimit(t *testing.T) {
	valid := map[string]int{"5": 5, " 7 ": 7, "0": 0, "12": 12}
	for input, expected := range valid {
		value, err := ParseSnapshotsLimit(input)
		if err != nil || value != expected {
			t.Fatalf("unexpected parse of %q: %d %v", input, value, err)
		}
	}
	for _, input := range []string{"", "-1", "abc", "2.5", "5x"} {
		if _, err := ParseSnapshotsLimit(input); err == nil {
			t.Fatalf("expected rejection of %q", input)
		}
	}
}

func TestSaveRejectsNonDirectory(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	value := mustDefaultConfig(t, home)
	value.CatalogDir = file
	if _, err := SaveConfig(value, home, ""); err == nil {
		t.Fatal("expected non-directory rejection")
	}
}

func mustDefaultConfig(t *testing.T, home string) AppConfig {
	t.Helper()
	value, err := DefaultConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustConfigPath(t *testing.T, home string) string {
	t.Helper()
	path, err := ConfigPath(home)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
