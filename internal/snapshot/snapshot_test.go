package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSnapshot(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "value")
	writeFile(t, home, ".config/app/dir/a", "a")
	directory := t.TempDir()
	target := filepath.Join(home, ".config/app/config")
	path, err := CreateSnapshot([]string{target, filepath.Join(home, ".config/app")}, home, directory, "2026-01-01T00:00:00Z", 0)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(directory, "2026-01-01T00:00:00Z.tar.gz") {
		t.Fatalf("unexpected path: %s", path)
	}
	manifest, err := archive.ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Apps) != 1 || manifest.Apps[0].ID != SnapshotAppID {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if len(manifest.Apps[0].Files) != 2 {
		t.Fatalf("expected both files with overlap removed: %#v", manifest.Apps[0].Files)
	}
	if err := archive.VerifyArchive(path, manifest, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotSuffixAndTimestampValidation(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "value")
	directory := t.TempDir()
	target := filepath.Join(home, ".config/app/config")
	first, err := CreateSnapshot([]string{target}, home, directory, "2026-01-01T00:00:00Z", 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateSnapshot([]string{target}, home, directory, "2026-01-01T00:00:00Z", 0)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Base(second) != "2026-01-01T00:00:00Z-1.tar.gz" {
		t.Fatalf("unexpected suffix: %s %s", first, second)
	}
	if _, err := CreateSnapshot([]string{target}, home, directory, "../evil", 0); err == nil {
		t.Fatal("expected timestamp rejection")
	}
	if path, err := CreateSnapshot(nil, home, directory, "", 0); err != nil || path != "" {
		t.Fatalf("empty snapshot: %q %v", path, err)
	}
}

func TestCreateSnapshotPrunesOldest(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "value")
	directory := t.TempDir()
	target := filepath.Join(home, ".config/app/config")
	stamps := []string{"2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", "2026-01-03T00:00:00Z"}
	for _, stamp := range stamps {
		if _, err := CreateSnapshot([]string{target}, home, directory, stamp, 2); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 2 {
		t.Fatalf("expected two kept snapshots, got %v", names)
	}
	for _, kept := range []string{"2026-01-02T00:00:00Z.tar.gz", "2026-01-03T00:00:00Z.tar.gz"} {
		if _, err := os.Lstat(filepath.Join(directory, kept)); err != nil {
			t.Fatalf("expected %s to survive: %v", kept, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(directory, "2026-01-01T00:00:00Z.tar.gz")); !os.IsNotExist(err) {
		t.Fatalf("expected the oldest snapshot to be pruned: %v", err)
	}
}

func TestCreateSnapshotWithoutLimitKeepsEverySnapshot(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "value")
	directory := t.TempDir()
	target := filepath.Join(home, ".config/app/config")
	for _, stamp := range []string{"2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", "2026-01-03T00:00:00Z"} {
		if _, err := CreateSnapshot([]string{target}, home, directory, stamp, 0); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("zero must keep every snapshot, got %d", len(entries))
	}
}

func TestPruneSnapshotsOrderAndForeignFiles(t *testing.T) {
	directory := t.TempDir()
	same := "2026-01-03T00:00:00Z"
	names := []string{same + "-1.tar.gz", same + ".tar.gz", "2026-01-02T00:00:00Z.tar.gz", "notes.txt", "2026-01-01T00:00:00Z.tar.gz.bak", ".hidden"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(directory, "2026-01-04T00:00:00Z.tar.gz")
	if err := os.Symlink(filepath.Join(directory, "nowhere"), link); err != nil {
		t.Fatal(err)
	}
	removed, err := PruneSnapshots(directory, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || filepath.Base(removed[0]) != "2026-01-02T00:00:00Z.tar.gz" {
		t.Fatalf("unexpected pruning: %v", removed)
	}
	for _, name := range []string{names[0], names[1], names[3], names[4], names[5]} {
		if _, err := os.Lstat(filepath.Join(directory, name)); err != nil {
			t.Fatalf("expected %s to survive: %v", name, err)
		}
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("expected the symbolic link to survive: %v", err)
	}
	if removed, err := PruneSnapshots(directory, 0, ""); err != nil || len(removed) != 0 {
		t.Fatalf("zero limit must not prune: %v %v", removed, err)
	}
	if removed, err := PruneSnapshots(filepath.Join(directory, "missing"), 1, ""); err != nil || len(removed) != 0 {
		t.Fatalf("missing directory must not fail: %v %v", removed, err)
	}
}

func TestCreateSnapshotKeepsTheNewSnapshotWhenClockMovedBack(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "value")
	directory := t.TempDir()
	target := filepath.Join(home, ".config/app/config")
	future := filepath.Join(directory, "2026-12-31T23:59:59Z.tar.gz")
	if err := os.WriteFile(future, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := CreateSnapshot([]string{target}, home, directory, "2026-01-01T00:00:00Z", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("the snapshot just written must survive the prune: %v", err)
	}
	if _, err := os.Lstat(future); !os.IsNotExist(err) {
		t.Fatalf("the limit must still prune one snapshot: %v", err)
	}
}

func TestPruneSnapshotsNeverDeletesTheKeptSnapshot(t *testing.T) {
	directory := t.TempDir()
	oldest := filepath.Join(directory, "2026-01-01T00:00:00Z.tar.gz")
	for _, path := range []string{oldest, filepath.Join(directory, "2026-01-02T00:00:00Z.tar.gz")} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := PruneSnapshots(directory, 1, oldest)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != filepath.Join(directory, "2026-01-02T00:00:00Z.tar.gz") {
		t.Fatalf("the kept snapshot must never be pruned: %v", removed)
	}
	if _, err := os.Lstat(oldest); err != nil {
		t.Fatalf("the kept snapshot must survive: %v", err)
	}
}

func TestPruneSnapshotsTreatsAVanishedSnapshotAsPruned(t *testing.T) {
	directory := t.TempDir()
	oldest := filepath.Join(directory, "2026-01-01T00:00:00Z.tar.gz")
	newest := filepath.Join(directory, "2026-01-02T00:00:00Z.tar.gz")
	for _, path := range []string{oldest, newest} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	original := removeFile
	removeFile = func(path string) error {
		if path == oldest {
			// Another deployment pruned this snapshot first.
			return &os.PathError{Op: "remove", Path: path, Err: os.ErrNotExist}
		}
		return original(path)
	}
	t.Cleanup(func() { removeFile = original })
	removed, err := PruneSnapshots(directory, 1, "")
	if err != nil {
		t.Fatalf("a vanished snapshot must not fail the prune: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("a vanished snapshot was not removed by us: %v", removed)
	}
	if _, err := os.Lstat(newest); err != nil {
		t.Fatalf("the newer snapshot must survive: %v", err)
	}
}

func TestPruneSnapshotsRejectsNegativeLimit(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "2026-01-01T00:00:00Z.tar.gz")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PruneSnapshots(directory, -1, ""); err == nil {
		t.Fatal("a negative limit must fail instead of disabling retention")
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("a rejected prune must not delete anything: %v", err)
	}
}

func TestCreateSnapshotRejectsNegativeLimit(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "value")
	directory := t.TempDir()
	target := filepath.Join(home, ".config/app/config")
	if _, err := CreateSnapshot([]string{target}, home, directory, "2026-01-01T00:00:00Z", -1); err == nil {
		t.Fatal("a negative limit must fail")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("a rejected limit must not write a snapshot: %v", entries)
	}
}

func TestPruneSnapshotsAbortsOnRemovalFailure(t *testing.T) {
	directory := t.TempDir()
	oldest := filepath.Join(directory, "2026-01-01T00:00:00Z.tar.gz")
	newest := filepath.Join(directory, "2026-01-02T00:00:00Z.tar.gz")
	for _, path := range []string{oldest, newest} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	original := removeFile
	removeFile = func(path string) error {
		if path == oldest {
			return &os.PathError{Op: "remove", Path: path, Err: os.ErrPermission}
		}
		return original(path)
	}
	t.Cleanup(func() { removeFile = original })
	removed, err := PruneSnapshots(directory, 1, "")
	if err == nil {
		t.Fatal("a removal failure must be reported")
	}
	if len(removed) != 0 {
		t.Fatalf("a failed removal must not be reported as pruned: %v", removed)
	}
	if _, err := os.Lstat(oldest); err != nil {
		t.Fatalf("the snapshot must survive a failed removal: %v", err)
	}
}
