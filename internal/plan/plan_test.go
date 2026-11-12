package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/catalog"
)

func TestFnmatchPythonSemantics(t *testing.T) {
	cases := []struct {
		name, pattern string
		want          bool
	}{
		{"a/b/c", "a/*", true},
		{".env", "**/.env", true},
		{"a/.env", "**/.env", true},
		{"a/b", "a/?", true},
		{"a/x", "a/[xy]", true},
		{"a/z", "a/[!xy]", true},
		{"abc", "a[", false},
	}
	for _, testCase := range cases {
		if got := matchesPath(testCase.name, testCase.pattern); got != testCase.want {
			t.Fatalf("fnmatch(%q, %q) = %v, want %v", testCase.name, testCase.pattern, got, testCase.want)
		}
	}
}

func TestIsExcludedUsesAncestors(t *testing.T) {
	if !IsExcluded("a/.ssh/config", []string{"**/.ssh"}) {
		t.Fatal("expected ancestor exclusion")
	}
	if !IsExcluded("a/cache/file", []string{"**/cache/**"}) {
		t.Fatal("expected nested exclusion")
	}
	if IsExcluded("a/config", []string{"**/cache"}) {
		t.Fatal("unexpected exclusion")
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPlanExpandsAndSkips(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/config", "one")
	writeFile(t, home, ".config/app/optional", "two")
	writeFile(t, home, ".config/app/dir/a", "a")
	writeFile(t, home, ".config/app/dir/b", "b")
	writeFile(t, home, ".config/app/ignored.log", "log")
	writeFile(t, home, ".config/app/secret/.env", "secret")
	if err := os.Symlink("config", filepath.Join(home, ".config/app/safe-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(home, ".config/app/unsafe-link")); err != nil {
		t.Fatal(err)
	}
	app := &catalog.AppDefinition{
		ID: "app", Name: "App", Command: ptr("app"), Paths: []string{".config/app"},
		OptionalPatterns: []string{".config/app/*.log"},
		Exclude:          []string{"**/*.log", "**/.env"},
	}
	built, err := BuildPlan([]*catalog.AppDefinition{app}, home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, operation := range built.Operations {
		paths[operation.Path] = true
	}
	for _, expected := range []string{
		".config/app/config", ".config/app/optional", ".config/app/dir/a",
		".config/app/dir/b", ".config/app/safe-link",
	} {
		if !paths[expected] {
			t.Fatalf("missing %s: %#v", expected, paths)
		}
	}
	if paths[".config/app/ignored.log"] || paths[".config/app/unsafe-link"] || paths[".config/app/secret/.env"] {
		t.Fatalf("unexpected included paths: %#v", paths)
	}
	skipped := map[string]bool{}
	for _, item := range built.Skipped {
		skipped[item.Path] = true
	}
	if !skipped[".config/app/unsafe-link"] {
		t.Fatalf("expected unsafe link skip: %#v", built.Skipped)
	}
}

func TestRequiredMissingAndOptionalMissing(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/optional", "x")
	app := &catalog.AppDefinition{
		ID: "app", Name: "App", Command: ptr("app"),
		Paths: []string{".config/app/required"}, OptionalPaths: []string{".config/app/optional", ".config/app/absent"},
	}
	built, err := BuildPlan([]*catalog.AppDefinition{app}, home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Operations) != 1 {
		t.Fatalf("unexpected operations: %#v", built.Operations)
	}
	if len(built.Skipped) != 1 || built.Skipped[0].Path != ".config/app/required" {
		t.Fatalf("unexpected skips: %#v", built.Skipped)
	}
	count, err := CountFiles([]*catalog.AppDefinition{app}, home, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count.Counts["app"] != 1 {
		t.Fatalf("unexpected count: %#v", count)
	}
}

func TestEnvAndXDGPaths(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/app/env", "env")
	writeFile(t, home, ".config/xdg/app/xdg", "xdg")
	t.Setenv("APP_CONFIG", filepath.Join(home, ".config/app/env"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config/xdg"))
	app := &catalog.AppDefinition{
		ID: "app", Name: "App", Command: ptr("app"),
		EnvPaths: []string{"APP_CONFIG"}, XDGPaths: []string{"app/xdg"},
	}
	built, err := BuildPlan([]*catalog.AppDefinition{app}, home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Operations) != 2 {
		t.Fatalf("unexpected operations: %#v", built.Operations)
	}
}

func TestGhosttyReferencesAndCycles(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".config/ghostty/config", "config-file = themes/theme\nconfig-file = config\nconfig-file = /etc/passwd\ntheme = dark:mytheme\n")
	writeFile(t, home, ".config/ghostty/themes/theme", "background = black\n")
	writeFile(t, home, ".config/ghostty/themes/mytheme", "foreground = white\n")
	app := &catalog.AppDefinition{
		ID: "ghostty", Name: "Ghostty", Command: ptr("ghostty"),
		Paths: []string{".config/ghostty/config"},
	}
	built, err := BuildPlan([]*catalog.AppDefinition{app}, home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, operation := range built.Operations {
		paths[operation.Path] = true
	}
	if !paths[".config/ghostty/themes/theme"] || !paths[".config/ghostty/themes/mytheme"] {
		t.Fatalf("missing theme references: %#v", paths)
	}
	hasCycleNotice := false
	for _, notice := range built.Notices {
		if notice == "referencia circular de Ghostty en .config/ghostty/config" {
			hasCycleNotice = true
		}
	}
	if !hasCycleNotice {
		t.Fatalf("expected cycle notice: %#v", built.Notices)
	}
	hasOutside := false
	for _, item := range built.Skipped {
		if item.Path == "/etc/passwd" {
			hasOutside = true
		}
	}
	if !hasOutside {
		t.Fatalf("expected outside-home skip: %#v", built.Skipped)
	}
}

func ptr(value string) *string { return &value }
