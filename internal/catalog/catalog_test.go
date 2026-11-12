package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedCatalogLoads(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) < 20 {
		t.Fatalf("expected the embedded catalog, got %d entries", len(catalog))
	}
	if catalog["ghostty"] == nil || catalog["git"] == nil {
		t.Fatal("expected ghostty and git definitions")
	}
	if catalog["ghostty"].BundleID == nil || *catalog["ghostty"].BundleID != "com.mitchellh.ghostty" {
		t.Fatalf("unexpected ghostty: %#v", catalog["ghostty"])
	}
}

func TestUserCatalogOverridesAndAdds(t *testing.T) {
	userDir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(userDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("git.toml", `id = "git"
name = "Git custom"
command = "git"
paths = [".gitconfig"]
`)
	write("custom.toml", `id = "custom"
name = "Custom"
command = "custom"
optional_paths = [".custom"]
`)
	catalog, err := LoadCatalog(userDir)
	if err != nil {
		t.Fatal(err)
	}
	if catalog["git"].Name != "Git custom" {
		t.Fatalf("user definition did not override: %#v", catalog["git"])
	}
	if catalog["custom"] == nil {
		t.Fatal("user definition not added")
	}
}

func TestInvalidDefinitionsAreRejected(t *testing.T) {
	cases := []string{
		`id = "Bad" name = "Bad" command = "x" paths = [".x"]`,
		`id = "ok" name = " " command = "x" paths = [".x"]`,
		`id = "ok" name = "Ok" paths = [".x"]`,
		`id = "ok" name = "Ok" command = "x" paths = ["../x"]`,
		`id = "ok" name = "Ok" command = "x" paths = ["~/.x"]`,
		`id = "ok" name = "Ok" command = "x" optional_patterns = ["**/*.conf"]`,
		`id = "ok" name = "Ok" command = "x" optional_patterns = ["*.conf"]`,
		`id = "ok" name = "Ok" command = "x" paths = [".x"] unknown = true`,
		`id = "ok" name = "Ok" command = "x" paths = [".x"] env_paths = ["lower"]`,
	}
	for _, content := range cases {
		userDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(userDir, "bad.toml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadCatalog(userDir); err == nil {
			t.Fatalf("expected error for %s", content)
		}
	}
}

func TestDuplicateUserIDIsRejected(t *testing.T) {
	userDir := t.TempDir()
	content := `id = "same" name = "Same" command = "x" paths = [".x"]`
	for _, name := range []string{"a.toml", "b.toml"} {
		if err := os.WriteFile(filepath.Join(userDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadCatalog(userDir); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
