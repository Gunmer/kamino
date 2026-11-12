// Package defaults exposes the shared default paths.
package defaults

import (
	"os"
	"path/filepath"
)

// CatalogDir is the default directory for user application definitions.
func CatalogDir(home string) string {
	return filepath.Join(homeOrUser(home), ".config", "kamino", "apps")
}

// VaultDir is the default directory for clones and snapshots.
func VaultDir(home string) string {
	return filepath.Join(homeOrUser(home), ".config", "kamino", "vault")
}

// SnapshotsDir is the default directory for safety snapshots.
func SnapshotsDir(home string) string {
	return filepath.Join(VaultDir(home), "snapshot")
}

func homeOrUser(home string) string {
	if home != "" {
		return home
	}
	value, _ := os.UserHomeDir()
	return value
}
