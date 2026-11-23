package snapshot

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	kerrors "github.com/Gunmer/kamino/internal/errors"
)

func TestLockIsExclusive(t *testing.T) {
	directory := t.TempDir()
	release, err := Lock(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	go func() {
		second, err := Lock(context.Background(), directory)
		if err != nil {
			return
		}
		close(acquired)
		_ = second()
	}()
	select {
	case <-acquired:
		t.Fatal("a second implant must not take the lock while it is held")
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
	release, err := Lock(context.Background(), directory)
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

func TestLockStopsWaitingWhenTheContextIsCancelled(t *testing.T) {
	directory := t.TempDir()
	release, err := Lock(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = release() }()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := Lock(ctx, directory)
		result <- err
	}()
	select {
	case err := <-result:
		t.Fatalf("the lock is held, Lock must wait: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-result:
		var cancelled *kerrors.OperationCancelled
		if !errors.As(err, &cancelled) {
			t.Fatalf("expected a cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Lock must return once the context is cancelled")
	}
}
