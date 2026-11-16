package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/config"
)

func TestSnapshotsLimitFromConfig(t *testing.T) {
	home := t.TempDir()
	if got := snapshotsLimit(home); got != config.DefaultSnapshotsLimit {
		t.Fatalf("missing config must use the default limit, got %d", got)
	}
	path := filepath.Join(home, ".config", "kamino", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[deploy]\nsnapshots_limit = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := snapshotsLimit(home); got != 3 {
		t.Fatalf("expected the configured limit, got %d", got)
	}
	if err := os.WriteFile(path, []byte("[deploy]\nsnapshots_limit = -1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := snapshotsLimit(home); got != config.DefaultSnapshotsLimit {
		t.Fatalf("an invalid config must fall back to the default limit, got %d", got)
	}
	// A readable limit next to an unrelated invalid key leaves the file invalid,
	// so the CLI must still use the default instead of the parsed value.
	mixed := "[deploy]\nsnapshots_limit = 3\n\n[ui]\nplain = false\n"
	if err := os.WriteFile(path, []byte(mixed), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := snapshotsLimit(home); got != config.DefaultSnapshotsLimit {
		t.Fatalf("a mixed invalid config must use the default limit, got %d", got)
	}
}
