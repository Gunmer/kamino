package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockIsExclusive(t *testing.T) {
	directory := t.TempDir()
	release, err := Lock(directory)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	go func() {
		second, err := Lock(directory)
		if err != nil {
			return
		}
		close(acquired)
		_ = second()
	}()
	select {
	case <-acquired:
		t.Fatal("a second deploy must not take the lock while it is held")
	case <-time.After(150 * time.Millisecond):
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("the lock must be released")
	}
}

func TestPruneKeepsTheLockFile(t *testing.T) {
	directory := t.TempDir()
	release, err := Lock(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = release() }()
	for _, name := range []string{"2026-01-01T00:00:00Z.tar.gz", "2026-01-02T00:00:00Z.tar.gz"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := PruneSnapshots(directory, 1, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(directory, lockName)); err != nil {
		t.Fatalf("the lock file must survive pruning: %v", err)
	}
}
