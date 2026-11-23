package implant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/conflicts"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/snapshot"
)

func implantFixture(t *testing.T) (sourceHome, appBin, cloneFile string, definitions catalog.Catalog) {
	t.Helper()
	sourceHome = t.TempDir()
	appBin = t.TempDir()
	command := "kamino-implant-test"
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
	if err := archive.WriteArchive(context.Background(), cloneFile, manifest, archive.WriteOptions{SourceRoot: sourceHome}); err != nil {
		t.Fatal(err)
	}
	return sourceHome, appBin, cloneFile, definitions
}

func ptr(value string) *string { return &value }

func checksum(data []byte) (string, error) {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func TestImplantNewAndIdentical(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Status != StatusNew {
		t.Fatalf("expected new entry: %#v", plan.Entries)
	}
	result, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Implanted) != 1 {
		t.Fatalf("expected implanted file: %#v", result)
	}
	if data, _ := os.ReadFile(filepath.Join(destination, ".config/test/config")); string(data) != "value=2\n" {
		t.Fatalf("unexpected destination content: %q", data)
	}
	session.Close()

	session, _ = PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin})
	plan, err = session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Entries[0].Status != StatusIdentical {
		t.Fatalf("expected identical: %s", plan.Entries[0].Status)
	}
	session.Close()
}

func conflictExecution(t *testing.T) (string, string, string, catalog.Catalog, *ImplantSession, *ImplantPlan) {
	t.Helper()
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".config/test/config"), []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("expected conflict: %#v", plan)
	}
	return destination, bin, cloneFile, definitions, session, plan
}

func TestImplantConflictApply(t *testing.T) {
	destination, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	result, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
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

func TestImplantConflictOmit(t *testing.T) {
	destination, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	omitted, err := session.Apply(context.Background(), conflicts.OmitAll(plan.Conflicts), nil)
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

func TestImplantConflictReview(t *testing.T) {
	_, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	review := conflicts.ReviewAll(plan.Conflicts[0], conflicts.RowClone)
	merged, err := session.Apply(context.Background(), conflicts.ConflictDecisions{Decisions: map[string]conflicts.FileDecision{
		plan.Conflicts[0].Path: review,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Implanted) != 1 {
		t.Fatalf("expected reviewed write: %#v", merged)
	}
}

func TestImplantDryRunDoesNotWrite(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	session, _ := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin})
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := session.DryRunResult(plan)
	if !result.DryRun || len(result.Implanted) != 0 {
		t.Fatalf("unexpected dry run: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(destination, ".config/test/config")); !os.IsNotExist(err) {
		t.Fatal("dry run must not write")
	}
}

func TestImplantConcurrentChange(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, ".config/test/config")
	if err := os.WriteFile(target, []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, _ := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("changed=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
	var concurrent *kerrors.ConcurrentChangeError
	if !errors.As(err, &concurrent) {
		t.Fatalf("expected concurrent change, got %v", err)
	}
}

func TestImplantPrunesSnapshotsByLimit(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	snapshots := t.TempDir()
	var last string
	for index := 0; index < 3; index++ {
		destination := t.TempDir()
		if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, ".config/test/config"), []byte("value=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: snapshots, SnapshotsLimit: 2})
		if err != nil {
			t.Fatal(err)
		}
		plan, err := session.Plan(context.Background(), nil, false, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		result, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
		if err != nil {
			t.Fatal(err)
		}
		if result.Snapshot == "" {
			t.Fatalf("iteration %d expected a snapshot: %#v", index, result)
		}
		last = result.Snapshot
		session.Close()
	}
	entries, err := os.ReadDir(snapshots)
	if err != nil {
		t.Fatal(err)
	}
	snapshotsKept := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".gz" {
			snapshotsKept++
		}
	}
	if snapshotsKept != 2 {
		t.Fatalf("expected the limit to keep two snapshots, got %d", snapshotsKept)
	}
	if _, err := os.Lstat(last); err != nil {
		t.Fatalf("the newest snapshot must survive: %v", err)
	}
}

func TestImplantDryRunDoesNotPrune(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	snapshots := t.TempDir()
	for _, name := range []string{"2026-01-01T00:00:00Z.tar.gz", "2026-01-02T00:00:00Z.tar.gz", "2026-01-03T00:00:00Z.tar.gz"} {
		if err := os.WriteFile(filepath.Join(snapshots, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".config/test/config"), []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: snapshots, SnapshotsLimit: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := session.DryRunResult(plan)
	if !result.DryRun || len(result.Implanted) != 0 {
		t.Fatalf("unexpected dry run: %#v", result)
	}
	entries, err := os.ReadDir(snapshots)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("dry run must not prune snapshots, got %d", len(entries))
	}
}

func TestImplantSnapshotFailureAbortsBeforeWriting(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, ".config/test/config")
	if err := os.WriteFile(target, []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshots := t.TempDir()
	if err := os.Chmod(snapshots, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(snapshots, 0o755) })
	session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: snapshots, SnapshotsLimit: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil); err == nil {
		t.Fatal("a failing snapshot must abort the implant")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "value=1\n" {
		t.Fatalf("destinations must not be written when the snapshot fails: %q %v", data, err)
	}
}

func TestImplantWaitsForTheSnapshotLock(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, ".config/test/config")
	if err := os.WriteFile(target, []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshots := t.TempDir()
	release, err := snapshot.Lock(context.Background(), snapshots)
	if err != nil {
		t.Fatal(err)
	}
	session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: snapshots, SnapshotsLimit: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		result ImplantResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
		done <- outcome{result: result, err: err}
	}()
	select {
	case got := <-done:
		t.Fatalf("the implant must wait for the snapshots lock: %#v %v", got.result, got.err)
	case <-time.After(150 * time.Millisecond):
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "value=1\n" {
		t.Fatalf("destinations must not be written while the lock is held: %q %v", data, err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.result.Snapshot == "" {
			t.Fatal("the implant must create its snapshot once the lock is free")
		}
		if data, _ := os.ReadFile(target); string(data) != "value=2\n" {
			t.Fatalf("unexpected destination content: %q", data)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the implant must continue once the lock is released")
	}
}

func TestCancelledContextStopsPlanAndApply(t *testing.T) {
	destination, _, _, _, session, plan := conflictExecution(t)
	defer session.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := session.Plan(ctx, nil, false, nil, nil); !cancelledError(err) {
		t.Fatalf("Plan must report the cancellation: %v", err)
	}
	if _, err := session.Apply(ctx, conflicts.ApplyAll(plan.Conflicts), nil); !cancelledError(err) {
		t.Fatalf("Apply must report the cancellation: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(destination, ".config/test/config")); err != nil || string(data) != "value=1\n" {
		t.Fatalf("a cancelled apply must not write: %q %v", data, err)
	}
}

func cancelledError(err error) bool {
	var target *kerrors.OperationCancelled
	return errors.As(err, &target)
}

func TestImplantDetectsChangeWhileWaitingForTheSnapshotLock(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, ".config/test/config")
	if err := os.WriteFile(target, []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshots := t.TempDir()
	release, err := snapshot.Lock(context.Background(), snapshots)
	if err != nil {
		t.Fatal(err)
	}
	session, err := PrepareImplant(cloneFile, Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: snapshots})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("the implant must wait for the snapshots lock: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := os.WriteFile(target, []byte("changed=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		var concurrent *kerrors.ConcurrentChangeError
		if !errors.As(err, &concurrent) {
			t.Fatalf("expected concurrent change, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the implant must finish once the lock is released")
	}
	if data, _ := os.ReadFile(target); string(data) != "changed=1\n" {
		t.Fatalf("the concurrent edit must survive: %q", data)
	}
}

func TestImplantingASnapshotRestoresThePreviousState(t *testing.T) {
	_, bin, cloneFile, definitions := implantFixture(t)
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, ".config/test"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, ".config/test/config")
	if err := os.WriteFile(target, []byte("value=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := Options{Home: destination, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()}
	session, err := PrepareImplant(cloneFile, options)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Apply(context.Background(), conflicts.ApplyAll(plan.Conflicts), nil)
	if err != nil || result.Snapshot == "" {
		t.Fatalf("implant failed: %+v %v", result, err)
	}
	if data, _ := os.ReadFile(target); string(data) != "value=2\n" {
		t.Fatalf("the implant must change the file: %q", data)
	}

	options.SnapshotsDir = t.TempDir()
	restore, err := PrepareImplant(result.Snapshot, options)
	if err != nil {
		t.Fatal(err)
	}
	defer restore.Close()
	restorePlan, err := restore.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restore.Apply(context.Background(), conflicts.ApplyAll(restorePlan.Conflicts), nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(target); string(data) != "value=1\n" {
		t.Fatalf("the snapshot must restore the previous content: %q", data)
	}
}
