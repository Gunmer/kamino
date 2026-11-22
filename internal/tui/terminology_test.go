package tui

import (
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/texts"
)

func TestMenuAndSummaryUseImplantTerminology(t *testing.T) {
	app := testApp(false, true)
	menu := renderCase(app, newMenuScreen(app), 80, 30)
	if !strings.Contains(menu, "Implantar") {
		t.Fatalf("menu must show Implantar:\n%s", menu)
	}
	if strings.Contains(menu, "Desplegar") {
		t.Fatalf("menu must not show Desplegar:\n%s", menu)
	}
	if texts.D4Deploy != "Implantar" {
		t.Fatalf("confirmation action must be Implantar, got %q", texts.D4Deploy)
	}
	if texts.D5Title != "Implantación completada" {
		t.Fatalf("summary title must be Implantación completada, got %q", texts.D5Title)
	}
	if !strings.Contains(texts.D5Undo, "kamino implant") || strings.Contains(texts.D5Undo, "kamino deploy") {
		t.Fatalf("undo command must use implant, got %q", texts.D5Undo)
	}
}
