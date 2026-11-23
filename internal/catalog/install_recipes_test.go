package catalog

import (
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
