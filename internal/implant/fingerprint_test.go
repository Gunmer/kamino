package implant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/paths"
)

func TestFingerprintOfDistinguishesMissingFromUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	home := t.TempDir()
	locked := filepath.Join(home, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	safe, err := paths.NewSafeHome(home, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = safe.Close() }()

	missing, err := fingerprintOf(safe, "absent")
	if err != nil || missing.kind != "missing" {
		t.Fatalf("absent file: got %+v, %v", missing, err)
	}

	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if _, err := fingerprintOf(safe, "locked/file"); err == nil {
		t.Fatal("an unreadable existing file was reported as missing")
	}
}
