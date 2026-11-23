package snapshot

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"

	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/progress"
)

// lockName is the advisory lock file kept next to the snapshots. It never
// matches snapshotName, so pruning never deletes it.
const lockName = ".snapshot.lock"

// lockPoll is how often a waiting Lock retries and checks for cancellation.
const lockPoll = 50 * time.Millisecond

// Lock takes an exclusive advisory lock on the snapshots directory so two
// concurrent implants cannot prune each other's recovery snapshot. It waits
// until the lock is available or ctx is cancelled, and returns a release
// function.
func Lock(ctx context.Context, directory string) (func() error, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, lockError(directory, err)
	}
	path := filepath.Join(directory, lockName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, lockError(path, err)
	}
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			_ = file.Close()
			return nil, lockError(path, err)
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, progress.EnsureNotCancelled(ctx)
		case <-time.After(lockPoll):
		}
	}
	return func() error {
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		closeErr := file.Close()
		if unlockErr != nil {
			return lockError(path, unlockErr)
		}
		return closeErr
	}, nil
}

func lockError(path string, err error) error {
	return &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
		Msg: "no se pudo bloquear el directorio de holocrones " + path + ": " + err.Error(),
	}}
}
