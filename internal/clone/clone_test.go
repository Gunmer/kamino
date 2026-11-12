package clone

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
)

func fixture(t *testing.T) (string, string, catalog.Catalog) {
	t.Helper()
	home := t.TempDir()
	bin := t.TempDir()
	command := "kamino-test-app"
	if err := os.WriteFile(filepath.Join(bin, command), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "test", "config"), []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	definitions := catalog.Catalog{
		"test": {
			ID: "test", Name: "Test", Command: &command,
			Paths: []string{".config/test/config"},
		},
	}
	return home, bin, definitions
}

func TestPlanAndWriteClone(t *testing.T) {
	home, bin, definitions := fixture(t)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	selected := []string{"test"}
	now := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	planned, err := PlanClone(PlanOptions{
		Home: home, Selected: &selected, Output: output, Catalog: definitions,
		Path: bin, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if planned.FileCount() != 1 || planned.Output != output {
		t.Fatalf("unexpected plan: %#v", planned)
	}
	result, err := WriteClone(planned, home, false, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != output {
		t.Fatalf("unexpected result: %#v", result)
	}
	manifest, err := archive.ReadManifest(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Apps) != 1 || manifest.Apps[0].ID != "test" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if _, err := WriteClone(planned, home, false, nil, nil, nil); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if _, err := WriteClone(planned, home, true, nil, nil, nil); err != nil {
		t.Fatalf("expected forced overwrite: %v", err)
	}
}

func TestRunCloneDryRunDoesNotWrite(t *testing.T) {
	home, bin, definitions := fixture(t)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	selected := []string{"test"}
	result, err := RunClone(PlanOptions{
		Home: home, Selected: &selected, Output: output, Catalog: definitions, Path: bin,
	}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || result.Output != "" {
		t.Fatalf("unexpected dry run: %#v", result)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("dry run must not write")
	}
}

func TestDefaultNameAndValidateDestination(t *testing.T) {
	name := DefaultName(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	if name != "clone-20260102.tar.gz" {
		t.Fatalf("unexpected name: %s", name)
	}
	if message := ValidateDestination(filepath.Join(t.TempDir(), "missing", "x.tar.gz"), false); message == "" {
		t.Fatal("expected missing parent error")
	}
	if message := ValidateDestination(filepath.Join(t.TempDir(), "missing", "x.tar.gz"), true); message != "" {
		t.Fatalf("unexpected error: %s", message)
	}
}
