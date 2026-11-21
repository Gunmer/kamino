package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectDirReportsPerFileResultsAndDuplicates(t *testing.T) {
	directory := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("good.toml", `id = "good"
name = "Good"
command = "good"
paths = [".config/good"]
`)
	write("bad.toml", `id = "Bad"
name = ""
command = "bad"
paths = [".config/bad"]
`)
	write("one.toml", `id = "same"
name = "One"
command = "one"
paths = [".config/one"]
exclude = ["**/.cache"]
`)
	write("two.toml", `id = "same"
name = "Two"
command = "two"
paths = [".config/two"]
exclude = ["**/.cache"]
`)
	write("ignored.txt", "not toml")
	results, err := InspectDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 {
		t.Fatalf("expected four TOML results, got %d", len(results))
	}
	byFile := map[string]DefinitionResult{}
	for _, result := range results {
		byFile[result.File] = result
	}
	if byFile["good.toml"].Err != nil || byFile["good.toml"].ID != "good" {
		t.Fatalf("valid file not reported: %#v", byFile["good.toml"])
	}
	if byFile["bad.toml"].Err == nil {
		t.Fatal("invalid file must report an error")
	}
	if !strings.Contains(byFile["two.toml"].Err.Error(), "one.toml") || !strings.Contains(byFile["two.toml"].Err.Error(), "two.toml") {
		t.Fatalf("duplicate error must name both files: %v", byFile["two.toml"].Err)
	}
}

func TestInspectDirMissingDirectoryIsEmpty(t *testing.T) {
	results, err := InspectDir(filepath.Join(t.TempDir(), "missing"))
	if err != nil || len(results) != 0 {
		t.Fatalf("missing directory must be an empty catalog: %#v %v", results, err)
	}
}

func TestInspectFileWarnings(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "miapp.toml")
	content := `id = "otra"
name = "Otra"
command = "otra"
paths = [".config/otra"]
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	result := InspectFile(file)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	joined := strings.Join(result.Warnings, " ")
	for _, want := range []string{"no declara exclusiones", "no coincide con el nombre"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing warning %q in %#v", want, result.Warnings)
		}
	}
}

func TestLoadCatalogRemainsFailFast(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "bad.toml"), []byte("not toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCatalog(directory); err == nil {
		t.Fatal("LoadCatalog must keep failing on an invalid user file")
	}
}

func TestRenderTemplateContentAndEscaping(t *testing.T) {
	content, err := RenderTemplate(TemplateOptions{
		ID:            "miapp",
		Name:          "Mi \"App\" \\ test",
		Command:       "miapp",
		Paths:         []string{".config/miapp/config.toml", ".config/miapp/a\"b"},
		OptionalPaths: []string{".config/miapp/themes"},
		SearchPaths:   []string{"/opt/homebrew/bin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"optional_patterns", "env_paths", "xdg_paths", "command_aliases",
		"install_marker", "exclude = [", "paths = [", "optional_paths = [",
		"search_paths = [", "command = \"miapp\"",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("template is missing %q:\n%s", want, content)
		}
	}
	if _, err := Parse("miapp.toml", []byte(content)); err != nil {
		t.Fatalf("generated template must parse and validate: %v\n%s", err, content)
	}
	if _, err := RenderTemplate(TemplateOptions{ID: "miapp", Name: "Miapp", BundleID: "com.example.miapp", Paths: []string{".config/miapp"}}); err != nil {
		t.Fatal(err)
	}
}

func TestRenderTemplateSkeletonIsInvalidUntilEdited(t *testing.T) {
	content, err := RenderTemplate(TemplateOptions{Skeleton: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("borrador", []byte(content)); err == nil {
		t.Fatal("the skeleton must not be a valid definition")
	}
	for _, want := range []string{"# bundle_id", "# optional_paths", "exclude = ["} {
		if !strings.Contains(content, want) {
			t.Fatalf("skeleton is missing %q:\n%s", want, content)
		}
	}
}

func TestCreateDefinitionWritesAtomicallyAndChecksDuplicates(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "nested", "apps")
	content, err := RenderTemplate(TemplateOptions{
		ID: "miapp", Name: "Mi App", Command: "miapp", Paths: []string{".config/miapp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := CreateDefinition(directory, []byte(content), false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != filepath.Join(directory, "miapp.toml") {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if data, err := os.ReadFile(result.Path); err != nil || string(data) != content {
		t.Fatalf("written content mismatch: %v", err)
	}
	if _, err := CreateDefinition(directory, []byte(content), false); err == nil {
		t.Fatal("an existing target must require --force")
	}
	updated := strings.Replace(content, `name = "Mi App"`, `name = "Mi App 2"`, 1)
	if _, err := CreateDefinition(directory, []byte(updated), true); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(result.Path); !strings.Contains(string(data), "Mi App 2") {
		t.Fatal("--force must replace the target")
	}

	if err := os.WriteFile(filepath.Join(directory, "otra.toml"), []byte(`id = "miapp"
name = "Otra"
command = "otra"
paths = [".config/otra"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateDefinition(directory, []byte(content), true); err == nil || !strings.Contains(err.Error(), "otra.toml") {
		t.Fatalf("a duplicate in another file must be rejected: %v", err)
	}
	if _, err := CreateDefinition(directory, []byte("not toml"), false); err == nil {
		t.Fatal("invalid TOML must be rejected")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("temporary file left behind: %s", entry.Name())
		}
	}
}

func TestCreateDefinitionWarnsAboutEmbeddedOverride(t *testing.T) {
	directory := t.TempDir()
	content, err := RenderTemplate(TemplateOptions{
		ID: "git", Name: "Git propio", Command: "git", Paths: []string{".gitconfig"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := CreateDefinition(directory, []byte(content), false)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.Warnings, " ")
	if !strings.Contains(joined, "reemplaza la definición integrada") {
		t.Fatalf("expected an embedded override warning, got %#v", result.Warnings)
	}
}
