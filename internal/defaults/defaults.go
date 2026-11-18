// Package defaults exposes the shared default paths.
package defaults

import (
	"path/filepath"

	"github.com/Gunmer/kamino/internal/paths"
)

// CatalogDir is the default directory for user application definitions.
func CatalogDir(home string) (string, error) {
	root, err := paths.ResolveHome(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".config", "kamino", "apps"), nil
}

// VaultDir is the default directory for clones and snapshots.
func VaultDir(home string) (string, error) {
	root, err := paths.ResolveHome(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".config", "kamino", "vault"), nil
}

// SnapshotsDir is the default directory for safety snapshots.
func SnapshotsDir(home string) (string, error) {
	vault, err := VaultDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(vault, "snapshot"), nil
}
