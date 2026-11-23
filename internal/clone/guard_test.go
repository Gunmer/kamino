package clone

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
)

func TestWrittenCloneNeverContainsBinariesOrBundles(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	command := "demo"
	write := func(rel, content string, mode os.FileMode) {
		t.Helper()
		full := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, command), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(".local/bin/demo", "#!/bin/sh\necho demo\n", 0o755)
	write(".demo/Demo.app/Contents/MacOS/demo", "app", 0o755)
	write(".config/demo/settings.toml", "theme = \"dark\"\n", 0o644)

	// A user definition declares the binary and the bundle on purpose.
	definitions := catalog.Catalog{"demo": {
		ID: "demo", Name: "Demo", Command: &command,
		Paths: []string{".local/bin/demo", ".demo/Demo.app", ".config/demo/settings.toml"},
	}}
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	selected := []string{"demo"}
	planned, err := PlanClone(context.Background(), PlanOptions{
		Home: home, Selected: &selected, Output: output, Catalog: definitions,
		Path: bin, Now: time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteClone(context.Background(), planned, home, false, nil, nil); err != nil {
		t.Fatal(err)
	}
	manifest, err := archive.ReadManifest(output)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, app := range manifest.Apps {
		for _, file := range app.Files {
			files = append(files, file.Path)
		}
	}
	if len(files) != 1 || files[0] != ".config/demo/settings.toml" {
		t.Fatalf("el clon solo debe guardar la configuración, got %v", files)
	}
}
