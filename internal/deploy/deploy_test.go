package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/conflicts"
	kerrors "github.com/Gunmer/kamino/internal/errors"
)

func deployFixture(t *testing.T) (sourceHome, appBin, cloneFile string, definitions catalog.Catalog) {
	t.Helper()
	sourceHome = t.TempDir()
	appBin = t.TempDir()
	command := "kamino-deploy-test"
	if err := os.WriteFile(filepath.Join(appBin, command), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceHome, ".config", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, ".config", "test", "config"), []byte("value=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	definitions = catalog.Catalog{
		"test": {
			ID: "test", Name: "Test", Command: &command,
			Paths: []string{".config/test/config"},
		},
	}
	manifest := &archive.Manifest{
		FormatVersion: 1, ToolVersion: "0.8.0", CreatedAt: "2026-01-01T00:00:00Z",
		MacOSVersion: "", Arch: "arm64", Hostname: "h",
		Apps: []archive.AppEntry{{
			ID: "test", Name: "Test",
			Files: []archive.FileEntry{{
				Path:   ".config/test/config",
				SHA256: ptr("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
				Size:   0,
				Mode:   "0644",
			}},
		}},
	}
	// Compute the real checksum.
	digest, err := checksum([]byte("value=2\n"))
	if err != nil {
		t.Fatal(err)
	}
	manifest.Apps[0].Files[0].SHA256 = &digest
	manifest.Apps[0].Files[0].Size = 8
	cloneFile = filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := archive.WriteArchive(cloneFile, manifest, archive.WriteOptions{SourceRoot: sourceHome}); err != nil {
		t.Fatal(err)
	}
	return sourceHome, appBin, cloneFile, definitions
}

func ptr(value string) *string { return &value }

func checksum(data []byte) (string, error) {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func TestDeployNewAndIdentical(t *testing.T) {
	_, bin, cloneFile, definitions := deployFixture(t)
	destination := t.TempDir()
	session, err := PrepareDeploy(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := session.Plan(nil, false, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Status != StatusNew {
		t.Fatalf("expected new entry: %#v", plan.Entries)
	}
	result, err := session.Apply(conflicts.ApplyAll(plan.Conflicts), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Deployed) != 1 {
		t.Fatalf("expected deployed file: %#v", result)
	}
	if data, _ := os.ReadFile(filepath.Join(destination, ".config/test/config")); string(data) != "value=2\n" {
		t.Fatalf("unexpected destination content: %q", data)
	}
	session.Close()

	session, _ = PrepareDeploy(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin})
	plan, err = session.Plan(nil, false, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Entries[0].Status != StatusIdentical {
		t.Fatalf("expected identical: %s", plan.Entries[0].Status)
	}
	session.Close()
}

func conflictExecution(t *testing.T) (string, string, string, catalog.Catalog, *DeploySession, *DeployPlan) {
	t.Helper()
	_, bin, cloneFile, definitions := deployFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".config/test/config"), []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := PrepareDeploy(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := session.Plan(nil, false, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("expected conflict: %#v", plan)
	}
	return destination, bin, cloneFile, definitions, session, plan
}

func TestDeployConflictApply(t *testing.T) {
	destination, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	result, err := session.Apply(conflicts.ApplyAll(plan.Conflicts), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Snapshot == "" || result.SnapshotFiles != 1 {
		t.Fatalf("expected snapshot: %#v", result)
	}
	if data, _ := os.ReadFile(filepath.Join(destination, ".config/test/config")); string(data) != "value=2\n" {
		t.Fatalf("unexpected content: %q", data)
	}
}

func TestDeployConflictOmit(t *testing.T) {
	destination, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	omitted, err := session.Apply(conflicts.OmitAll(plan.Conflicts), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(omitted.Omitted) != 1 {
		t.Fatalf("expected omitted: %#v", omitted)
	}
	if data, _ := os.ReadFile(filepath.Join(destination, ".config/test/config")); string(data) != "value=1\n" {
		t.Fatalf("omit must keep local: %q", data)
	}
}

func TestDeployConflictReview(t *testing.T) {
	_, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	review := conflicts.ReviewAll(plan.Conflicts[0], conflicts.RowClone)
	merged, err := session.Apply(conflicts.ConflictDecisions{Decisions: map[string]conflicts.FileDecision{
		plan.Conflicts[0].Path: review,
	}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Deployed) != 1 {
		t.Fatalf("expected reviewed write: %#v", merged)
	}
}

func TestDeployDryRunDoesNotWrite(t *testing.T) {
	_, bin, cloneFile, definitions := deployFixture(t)
	destination := t.TempDir()
	session, _ := PrepareDeploy(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin})
	defer session.Close()
	plan, err := session.Plan(nil, true, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := session.DryRunResult(plan)
	if !result.DryRun || len(result.Deployed) != 0 {
		t.Fatalf("unexpected dry run: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(destination, ".config/test/config")); !os.IsNotExist(err) {
		t.Fatal("dry run must not write")
	}
}

func TestDeployConcurrentChange(t *testing.T) {
	_, bin, cloneFile, definitions := deployFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, ".config/test/config")
	if err := os.WriteFile(target, []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, _ := PrepareDeploy(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	defer session.Close()
	plan, err := session.Plan(nil, false, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("changed=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = session.Apply(conflicts.ApplyAll(plan.Conflicts), nil, nil)
	var concurrent *kerrors.ConcurrentChangeError
	if !errors.As(err, &concurrent) {
		t.Fatalf("expected concurrent change, got %v", err)
	}
}
