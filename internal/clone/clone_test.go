package clone

import (
	"context"
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
	planned, err := PlanClone(context.Background(), PlanOptions{
		Home: home, Selected: &selected, Output: output, Catalog: definitions,
		Path: bin, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if planned.FileCount() != 1 || planned.Output != output {
		t.Fatalf("unexpected plan: %#v", planned)
	}
	result, err := WriteClone(context.Background(), planned, home, false, nil, nil)
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
	if _, err := WriteClone(context.Background(), planned, home, false, nil, nil); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if _, err := WriteClone(context.Background(), planned, home, true, nil, nil); err != nil {
		t.Fatalf("expected forced overwrite: %v", err)
	}
}

func TestRunCloneDryRunDoesNotWrite(t *testing.T) {
	home, bin, definitions := fixture(t)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	selected := []string{"test"}
	result, err := RunClone(context.Background(), PlanOptions{
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

func partialFixture(t *testing.T) (string, string, catalog.Catalog) {
	t.Helper()
	home, bin, definitions := fixture(t)
	for _, name := range []string{"one", "two", "three"} {
		if err := os.WriteFile(filepath.Join(home, ".config", "test", name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	definitions["test"].Paths = []string{".config/test"}
	other := "kamino-other-app"
	if err := os.WriteFile(filepath.Join(bin, other), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".other"), []byte("o"), 0o644); err != nil {
		t.Fatal(err)
	}
	definitions["other"] = &catalog.AppDefinition{ID: "other", Name: "Other", Command: &other, Paths: []string{".other"}}
	return home, bin, definitions
}

func manifestPaths(manifest *archive.Manifest, id string) []string {
	for _, app := range manifest.Apps {
		if app.ID == id {
			result := []string{}
			for _, file := range app.Files {
				result = append(result, file.Path)
			}
			return result
		}
	}
	return nil
}

func TestPlanCloneFilesFilter(t *testing.T) {
	home, bin, definitions := partialFixture(t)
	selected := []string{"test", "other"}
	options := PlanOptions{
		Home: home, Selected: &selected, Output: filepath.Join(t.TempDir(), "c.tar.gz"), Catalog: definitions, Path: bin,
		Files: map[string][]string{"test": {".config/test/one", ".config/test/three", ".config/test/unknown"}},
	}
	planned, err := PlanClone(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	got := manifestPaths(planned.Manifest, "test")
	if len(got) != 2 || got[0] != ".config/test/one" || got[1] != ".config/test/three" {
		t.Fatalf("unexpected filtered files: %v", got)
	}
	if other := manifestPaths(planned.Manifest, "other"); len(other) != 1 {
		t.Fatalf("absent app must keep every file: %v", other)
	}
	if planned.FileCount() != 3 {
		t.Fatalf("plan must match manifest: %d", planned.FileCount())
	}
}

func TestPartialCloneRoundTrip(t *testing.T) {
	home, bin, definitions := partialFixture(t)
	selected := []string{"test"}
	output := filepath.Join(t.TempDir(), "c.tar.gz")
	options := PlanOptions{
		Home: home, Selected: &selected, Output: output, Catalog: definitions, Path: bin,
		Files: map[string][]string{"test": {".config/test/two"}},
	}
	dry, err := RunClone(context.Background(), options, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(dry.Plan.Operations) != 1 || dry.Plan.Operations[0].Path != ".config/test/two" {
		t.Fatalf("unexpected dry run plan: %#v", dry.Plan.Operations)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("dry run must not write")
	}
	if _, err := RunClone(context.Background(), options, false, false); err != nil {
		t.Fatal(err)
	}
	manifest, err := archive.ReadManifest(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := manifestPaths(manifest, "test"); len(got) != 1 || got[0] != ".config/test/two" {
		t.Fatalf("manifest must list only included files: %v", got)
	}
	if archive.ReadFileFromArchive(output, "test", ".config/test/two", 1024) == nil {
		t.Fatal("included file missing from archive")
	}
	if archive.ReadFileFromArchive(output, "test", ".config/test/one", 1024) != nil {
		t.Fatal("excluded file present in archive")
	}
}
