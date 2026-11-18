// Package defaults exposes the shared default paths.
package defaults

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsRequireAResolvableHome(t *testing.T) {
	t.Setenv("HOME", "")
	cases := map[string]func(string) (string, error){
		"catalog":   CatalogDir,
		"vault":     VaultDir,
		"snapshots": SnapshotsDir,
	}
	for name, resolve := range cases {
		t.Run(name, func(t *testing.T) {
			if resolved, err := resolve(""); err == nil {
				t.Fatalf("expected an error without HOME, got %q", resolved)
			}
			home := t.TempDir()
			resolved, err := resolve(home)
			if err != nil {
				t.Fatalf("explicit home must resolve: %v", err)
			}
			if !filepath.IsAbs(resolved) || filepath.Dir(resolved) == home {
				t.Fatalf("unexpected default path: %q", resolved)
			}
		})
	}
}

func TestDefaultPathsUseTheInjectedHome(t *testing.T) {
	home := t.TempDir()
	catalog, err := CatalogDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "kamino", "apps"); catalog != want {
		t.Fatalf("unexpected catalog dir: %q", catalog)
	}
	vault, err := VaultDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "kamino", "vault"); vault != want {
		t.Fatalf("unexpected vault dir: %q", vault)
	}
	snapshots, err := SnapshotsDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(vault, "snapshot"); snapshots != want {
		t.Fatalf("unexpected snapshots dir: %q", snapshots)
	}
}
