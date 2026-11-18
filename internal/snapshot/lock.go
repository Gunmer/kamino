package snapshot

import (
	"os"
	"path/filepath"
	"syscall"

	kerrors "github.com/Gunmer/kamino/internal/errors"
)

// lockName is the advisory lock file kept next to the snapshots. It never
// matches snapshotName, so pruning never deletes it.
const lockName = ".snapshot.lock"

// Lock takes an exclusive advisory lock on the snapshots directory so two
// concurrent deploys cannot prune each other's recovery snapshot. It blocks
// until the lock is available and returns a release function.
func Lock(directory string) (func() error, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, lockError(directory, err)
	}
	path := filepath.Join(directory, lockName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, lockError(path, err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, lockError(path, err)
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
