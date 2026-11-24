package catalog

import (
	"github.com/Gunmer/kamino/internal/install"
	"strings"
	"testing"
)

const installBase = `id = "demo"
name = "Demo"
command = "demo"
paths = [".demo"]
`

func TestParseAcceptsInstallRecipes(t *testing.T) {
	data := installBase + `
[[install]]
method = "brew"
formula = "demo"

[[install]]
method = "binary"
url = "https://example.com/demo"
sha256 = "0000000000000000000000000000000000000000000000000000000000000000"
dest = ".local/bin/demo"
`
	definition, err := Parse("demo.toml", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Install) != 2 || definition.Install[0].Method != "brew" {
		t.Fatalf("recetas inesperadas: %+v", definition.Install)
	}
}

func TestParseRejectsInvalidInstallRecipes(t *testing.T) {
	cases := map[string]string{
		"método no permitido": `[[install]]
method = "script"
url = "https://example.com/x"
`,
		"binary sin sha256": `[[install]]
method = "binary"
url = "https://example.com/x"
dest = ".local/bin/demo"
`,
		"installer con http": `[[install]]
method = "installer"
url = "http://example.com/install.sh"
`,
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse("demo.toml", []byte(installBase+block))
			if err == nil {
				t.Fatal("se esperaba un error de validación")
			}
			if !strings.Contains(err.Error(), "[[install]] 1") {
				t.Fatalf("el error debe nombrar la receta: %v", err)
			}
		})
	}
}

func TestDefinitionWithoutRecipesLoadsAsBefore(t *testing.T) {
	definition, err := Parse("demo.toml", []byte(installBase))
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Install) != 0 {
		t.Fatalf("sin [[install]] no debe haber recetas: %+v", definition.Install)
	}
}

func TestEmbeddedRecipesAreValidAndFollowTheMethodOrder(t *testing.T) {
	catalog, err := loadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	rank := map[string]int{
		install.MethodBrew: 0, install.MethodNpm: 1, install.MethodInstaller: 2, install.MethodGithubRelease: 3,
	}
	for _, id := range catalog.SortedIDs() {
		previous := -1
		for index, recipe := range catalog[id].Install {
			if err := recipe.Validate(); err != nil {
				t.Errorf("%s [[install]] %d: %v", id, index+1, err)
			}
			current, known := rank[recipe.Method]
			if !known {
				t.Errorf("%s [[install]] %d: el catálogo integrado no usa %q", id, index+1, recipe.Method)
			}
			if current < previous {
				t.Errorf("%s [[install]] %d: %q rompe el orden brew, npm, installer, github-release", id, index+1, recipe.Method)
			}
			previous = current
		}
	}
}

func TestEmbeddedGithubReleaseRecipesCoverBothArchitectures(t *testing.T) {
	catalog, err := loadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range catalog.SortedIDs() {
		archs := map[string]int{}
		for _, recipe := range catalog[id].Install {
			if recipe.Method == install.MethodGithubRelease {
				archs[recipe.Arch]++
			}
		}
		if len(archs) > 0 && (archs["arm64"] != 1 || archs["amd64"] != 1 || len(archs) != 2) {
			t.Errorf("%s: github-release debe declarar una receta arm64 y otra amd64, got %v", id, archs)
		}
	}
}

func TestEmbeddedCatalogOffersNpmAndReleaseWhereTheyExist(t *testing.T) {
	catalog, err := loadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	methods := func(id string) []string {
		var out []string
		for _, recipe := range catalog[id].Install {
			if len(out) == 0 || out[len(out)-1] != recipe.Method {
				out = append(out, recipe.Method)
			}
		}
		return out
	}
	want := map[string]string{
		"codex":       "brew npm installer github-release",
		"openspec":    "brew npm installer github-release",
		"claude-code": "brew npm github-release",
		"yazi":        "brew github-release",
		"gemini-cli":  "brew npm",
		"pi":          "npm installer",
	}
	for id, expected := range want {
		if got := strings.Join(methods(id), " "); got != expected {
			t.Errorf("%s: métodos %q, se esperaba %q", id, got, expected)
		}
	}
}
