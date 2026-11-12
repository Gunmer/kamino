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
	path, err := CreateSnapshot([]string{target, filepath.Join(home, ".config/app")}, home, directory, "2026-01-01T00:00:00Z")
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
	first, err := CreateSnapshot([]string{target}, home, directory, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateSnapshot([]string{target}, home, directory, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Base(second) != "2026-01-01T00:00:00Z-1.tar.gz" {
		t.Fatalf("unexpected suffix: %s %s", first, second)
	}
	if _, err := CreateSnapshot([]string{target}, home, directory, "../evil"); err == nil {
		t.Fatal("expected timestamp rejection")
	}
	if path, err := CreateSnapshot(nil, home, directory, ""); err != nil || path != "" {
		t.Fatalf("empty snapshot: %q %v", path, err)
	}
}
