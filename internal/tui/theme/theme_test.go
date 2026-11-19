package theme

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestTokens(t *testing.T) {
	tokens := DefaultTokens()
	if tokens.Gutter != 2 || tokens.PadX != 2 || tokens.PadY != 1 || tokens.Gap != 1 || tokens.ColGap != 4 {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestOnlyANSIColors(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		for _, forbidden := range []string{"AdaptiveColor", "RGBColor", `Color("#`, "Color(\"rgb"} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s contains forbidden color construction %q", name, forbidden)
			}
		}
		if regexp.MustCompile(`#[0-9a-fA-F]{3,8}`).MatchString(source) {
			t.Fatalf("%s contains a hex color", name)
		}
	}
}

func TestHierarchy(t *testing.T) {
	dark := New(Options{Name: "ansi-dark", Unicode: true})
	if !strings.Contains(dark.Styles.Primary.Render("x"), "\x1b[1m") {
		t.Fatal("primary must be bold")
	}
	if !strings.Contains(dark.Styles.Accent.Render("x"), "\x1b[36m") {
		t.Fatal("accent must be ANSI 6")
	}
	if !strings.Contains(dark.Styles.Info.Render("x"), "\x1b[32m") {
		t.Fatal("info must be ANSI 2")
	}
	if !strings.Contains(dark.Styles.Warning.Render("x"), "\x1b[33m") {
		t.Fatal("warning must be ANSI 3")
	}
	if !strings.Contains(dark.Styles.Error.Render("x"), "\x1b[31m") {
		t.Fatal("error must be ANSI 1")
	}
	if !strings.Contains(dark.Styles.Tertiary.Render("x"), "\x1b[2m") {
		t.Fatal("dark tertiary must use faint")
	}
	light := New(Options{Name: "ansi-light", Unicode: true})
	if !strings.Contains(light.Styles.Tertiary.Render("x"), "\x1b[90m") {
		t.Fatal("light tertiary must use ANSI 8")
	}
	if strings.Contains(light.Styles.Tertiary.Render("x"), "\x1b[2m") {
		t.Fatal("light tertiary must not use faint")
	}
	if strings.Contains(light.Styles.Discarded.Render("x"), "\x1b[2m") {
		t.Fatal("light discarded text must not use faint")
	}
}

func TestNoColorKeepsAttributes(t *testing.T) {
	value := New(Options{Name: "ansi-dark", NoColor: true, Unicode: true})
	rendered := value.Styles.Warning.Render("aviso") + value.Styles.Primary.Render("primary")
	if strings.Contains(rendered, "\x1b[33m") || strings.Contains(rendered, "\x1b[32m") || strings.Contains(rendered, "\x1b[31m") {
		t.Fatalf("NO_COLOR must remove colors: %q", rendered)
	}
	if !strings.Contains(rendered, "\x1b[1m") {
		t.Fatal("NO_COLOR must keep bold")
	}
}

func TestSymbols(t *testing.T) {
	unicode := New(Options{Name: "ansi-dark", Unicode: true})
	if unicode.Symbols.Cursor != "▌" || unicode.Symbols.Zero != "·" || unicode.Symbols.NoLine != "— sin línea —" {
		t.Fatalf("unexpected unicode symbols: %#v", unicode.Symbols)
	}
	ascii := New(Options{Name: "ansi-dark", Unicode: false})
	if ascii.Symbols.Cursor != ">" || ascii.Symbols.Zero != "." || ascii.Symbols.NoLine != "-- sin línea --" {
		t.Fatalf("unexpected ascii symbols: %#v", ascii.Symbols)
	}
}

var sgrPattern = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func hasSGRParam(rendered, wanted string) bool {
	for _, match := range sgrPattern.FindAllStringSubmatch(rendered, -1) {
		for _, param := range strings.Split(match[1], ";") {
			if param == wanted {
				return true
			}
		}
	}
	return false
}

func colorSGR(rendered string) bool {
	for _, match := range sgrPattern.FindAllStringSubmatch(rendered, -1) {
		for _, param := range strings.Split(match[1], ";") {
			switch {
			case param >= "30" && param <= "37":
				return true
			case param >= "40" && param <= "47":
				return true
			case param >= "90" && param <= "97":
				return true
			case param >= "100" && param <= "107":
				return true
			}
		}
	}
	return false
}

func TestFieldStyles(t *testing.T) {
	dark := New(Options{Name: "ansi-dark", Unicode: true})
	focus := dark.Styles.FieldFocus.Render("x")
	if !hasSGRParam(focus, "36") || !hasSGRParam(focus, "100") {
		t.Fatalf("dark focus must use accent over ANSI 8: %q", focus)
	}
	if !hasSGRParam(dark.Styles.FieldLabelFocus.Render("x"), "100") {
		t.Fatal("dark focus label must share the field background")
	}
	if !hasSGRParam(dark.Styles.FieldCaret.Render("x"), "7") {
		t.Fatal("the cursor must use reverse video")
	}
	if !hasSGRParam(dark.Styles.FieldIdle.Render("x"), "2") {
		t.Fatal("dark idle fields must use faint text")
	}

	light := New(Options{Name: "ansi-light", Unicode: true})
	lightFocus := light.Styles.FieldFocus.Render("x")
	if !hasSGRParam(lightFocus, "36") || !hasSGRParam(lightFocus, "47") {
		t.Fatalf("light focus must use accent over ANSI 7: %q", lightFocus)
	}
	if !hasSGRParam(light.Styles.FieldLabelFocus.Render("x"), "47") {
		t.Fatal("light focus label must share the field background")
	}
}

func TestFieldStylesRespectNoColor(t *testing.T) {
	value := New(Options{Name: "ansi-dark", NoColor: true, Unicode: true})
	rendered := value.Styles.FieldFocus.Render("x") +
		value.Styles.FieldIdle.Render("x") +
		value.Styles.FieldLabelFocus.Render("x") +
		value.Styles.FieldLabelIdle.Render("x") +
		value.Styles.FieldError.Render("x") +
		value.Styles.FieldCaret.Render("x")
	if colorSGR(rendered) {
		t.Fatalf("NO_COLOR fields must not emit color or background: %q", rendered)
	}
	if hasSGRParam(rendered, "38") || hasSGRParam(rendered, "48") {
		t.Fatalf("NO_COLOR fields must not use extended colors: %q", rendered)
	}
	if !hasSGRParam(value.Styles.FieldCaret.Render("x"), "7") {
		t.Fatal("NO_COLOR must keep the reverse-video cursor")
	}
	if !hasSGRParam(value.Styles.FieldLabelFocus.Render("x"), "1") {
		t.Fatal("NO_COLOR must keep the bold focus label")
	}
}

func TestFieldStylesEmitOnlySGR(t *testing.T) {
	value := New(Options{Name: "ansi-dark", Unicode: true})
	rendered := value.Styles.FieldFocus.Render("x") +
		value.Styles.FieldIdle.Render("x") +
		value.Styles.FieldLabelFocus.Render("x") +
		value.Styles.FieldLabelIdle.Render("x") +
		value.Styles.FieldError.Render("x") +
		value.Styles.FieldCaret.Render("x")
	if strings.Contains(sgrPattern.ReplaceAllString(rendered, ""), "\x1b") {
		t.Fatalf("field styles must only emit SGR escapes: %q", rendered)
	}
	if strings.Contains(rendered, "\x1b[38;2;") || strings.Contains(rendered, "\x1b[48;2;") {
		t.Fatalf("field styles must not use truecolor: %q", rendered)
	}
}
