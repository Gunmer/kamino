package plan

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/catalog"
)

func writeHomeFile(t *testing.T, home, rel, content string, mode os.FileMode) {
	t.Helper()
	full := filepath.Join(home, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func skippedReason(skipped []SkippedPath, rel string) string {
	for _, item := range skipped {
		if item.Path == rel {
			return item.Reason
		}
	}
	return ""
}

func operationPaths(operations []Operation) map[string]bool {
	result := map[string]bool{}
	for _, operation := range operations {
		result[operation.Path] = true
	}
	return result
}

func TestCloneNeverStoresBinariesOrBundles(t *testing.T) {
	home := t.TempDir()
	writeHomeFile(t, home, ".local/bin/tool", "#!/bin/sh\necho tool\n", 0o755)
	writeHomeFile(t, home, ".demo/Demo.app/Contents/MacOS/demo", "app", 0o755)
	writeHomeFile(t, home, ".config/demo/settings.toml", "theme = \"dark\"\n", 0o644)
	writeHomeFile(t, home, ".config/demo/helper", "\xCF\xFA\xED\xFE rest of macho", 0o644)
	writeHomeFile(t, home, ".config/demo/Helper.app/Contents/Info.plist", "plist", 0o644)

	definition := &catalog.AppDefinition{
		ID:    "demo",
		Name:  "Demo",
		Paths: []string{".local/bin/tool", ".demo/Demo.app", ".config/demo"},
	}
	result, err := BuildPlan(context.Background(), []*catalog.AppDefinition{definition}, home, nil)
	if err != nil {
		t.Fatal(err)
	}
	included := operationPaths(result.Operations)
	if !included[".config/demo/settings.toml"] {
		t.Fatalf("la configuración debe clonarse: %v", included)
	}
	for _, rel := range []string{".local/bin/tool", ".demo/Demo.app", ".config/demo/helper", ".config/demo/Helper.app"} {
		if included[rel] {
			t.Fatalf("%s no debe incluirse en el clon", rel)
		}
	}
	if reason := skippedReason(result.Skipped, ".local/bin/tool"); reason != reasonExecutable {
		t.Fatalf("el ejecutable debe informarse como no clonado, got %q", reason)
	}
	if reason := skippedReason(result.Skipped, ".demo/Demo.app"); reason != reasonBundle {
		t.Fatalf("el bundle debe informarse como no clonado, got %q", reason)
	}
	if reason := skippedReason(result.Skipped, ".config/demo/helper"); reason != reasonExecutable {
		t.Fatalf("un Mach-O sin bit de ejecución debe rechazarse, got %q", reason)
	}
	if !strings.Contains(strings.Join(skippedPaths(result.Skipped), " "), ".config/demo/Helper.app") {
		t.Fatalf("un bundle anidado debe informarse: %v", result.Skipped)
	}
}

func skippedPaths(skipped []SkippedPath) []string {
	result := make([]string, 0, len(skipped))
	for _, item := range skipped {
		result = append(result, item.Path)
	}
	return result
}
