package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const badRecipeDefinition = `id = "bad"
name = "Bad"
command = "bad"
paths = [".bad"]

[[install]]
method = "binary"
url = "https://example.com/bad"
dest = ".local/bin/bad"
`

func TestAppsValidateExitsOneOnInvalidInstallRecipe(t *testing.T) {
	file := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(file, []byte(badRecipeDefinition), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run("apps", "validate", file)
	if code != 1 {
		t.Fatalf("validate debe salir con 1 ante una receta inválida, got %d", code)
	}
	if !strings.Contains(stdout+stderr, "sha256") {
		t.Fatalf("el error debe nombrar sha256:\n%s%s", stdout, stderr)
	}
}

func TestAppsListShowsInvalidInstallRecipeWithoutChangingExitCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	userDir := filepath.Join(home, ".config", "kamino", "apps")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "bad.toml"), []byte(badRecipeDefinition), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := run("apps", "list")
	if code != 0 {
		t.Fatalf("list mantiene el código 0 ante un archivo inválido, got %d", code)
	}
	if !strings.Contains(stdout, "ERROR") || !strings.Contains(stdout, "sha256") {
		t.Fatalf("list debe mostrar el error del archivo:\n%s", stdout)
	}
}
