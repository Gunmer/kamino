package release

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildAndSmoke(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("el binario de release solo se construye en macOS ARM64")
	}
	if testing.Short() {
		t.Skip("skipping release build in short mode")
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Clean(filepath.Join(root, "..", ".."))
	binary := filepath.Join(t.TempDir(), AssetName)
	if err := Build(root, "9.9.9", binary); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(binary); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("binary not executable: %v %v", info, err)
	}
	if err := Smoke(binary, "9.9.9"); err != nil {
		t.Fatal(err)
	}
}
