package tui

import (
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/texts"
)

func TestBodyWidths(t *testing.T) {
	cases := []struct {
		width int
		want  int
	}{
		{80, 76}, {99, 95}, {100, 96}, {139, 96}, {140, 120}, {200, 120},
	}
	for _, testCase := range cases {
		if got := BodyWidth(testCase.width); got != testCase.want {
			t.Fatalf("BodyWidth(%d) = %d, want %d", testCase.width, got, testCase.want)
		}
	}
}

func TestBodyHeightAndAir(t *testing.T) {
	cases := []struct {
		height   int
		messages int
		want     int
	}{
		{24, 0, 20}, {26, 0, 21}, {30, 0, 25}, {26, 2, 19}, {24, 3, 17},
	}
	for _, testCase := range cases {
		if got := BodyHeight(testCase.height, testCase.messages); got != testCase.want {
			t.Fatalf("BodyHeight(%d, %d) = %d, want %d", testCase.height, testCase.messages, got, testCase.want)
		}
	}
}

func TestFrameHeightAndAirRow(t *testing.T) {
	for _, height := range []int{24, 26, 30} {
		app := testApp(false, true)
		app.current = newMenuScreen(app)
		app.width, app.height = 80, height
		view := app.View()
		lines := strings.Split(view, "\n")
		if len(lines) != height {
			t.Fatalf("height %d: got %d lines", height, len(lines))
		}
		if height == 24 && strings.TrimSpace(lines[2]) == "" {
			t.Fatal("24 rows must not have an air row")
		}
		if height >= 26 && strings.TrimSpace(lines[2]) != "" {
			t.Fatalf("height %d must have an air row", height)
		}
	}
}

func TestBodyContentCenteredOnLargeScreens(t *testing.T) {
	app := testApp(false, true)
	screen := &messageScreen{screenBase: screenBase{app: app}, body: "body"}

	compact := strings.Split(renderCase(app, screen, 80, 26), "\n")
	if compact[3] != "  body" {
		t.Fatalf("compact body = %q, want left aligned with gutter", compact[3])
	}

	wide := strings.Split(renderCase(app, screen, 100, 26), "\n")
	expected := strings.Repeat(" ", 48) + "body"
	if wide[3] != expected {
		t.Fatalf("wide body = %q, want %q", wide[3], expected)
	}
}

func TestBodyContentKeepsFullWidthBlocks(t *testing.T) {
	app := testApp(false, true)
	width := BodyWidth(100)
	screen := &messageScreen{screenBase: screenBase{app: app}, body: strings.Repeat("x", width)}

	lines := strings.Split(renderCase(app, screen, 100, 26), "\n")
	expected := "  " + strings.Repeat("x", width)
	if lines[3] != expected {
		t.Fatalf("full-width body = %q, want %q", lines[3], expected)
	}
}

func TestDetectUnicode(t *testing.T) {
	t.Setenv("LANG", "es_ES.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("KAMINO_NO_LOGO", "")
	if !detectUnicode() {
		t.Fatal("UTF-8 locale must enable Unicode")
	}
	t.Setenv("KAMINO_NO_LOGO", "1")
	if detectUnicode() {
		t.Fatal("KAMINO_NO_LOGO must force ASCII")
	}
	t.Setenv("KAMINO_NO_LOGO", "")
	t.Setenv("LANG", "C")
	if detectUnicode() {
		t.Fatal("C locale must force ASCII")
	}
}

func TestMessageZoneDoesNotReserveRows(t *testing.T) {
	app := testApp(false, true)
	app.current = newMenuScreen(app)
	app.width, app.height = 100, 26
	without := app.View()
	screen := &messageScreen{
		screenBase: screenBase{app: app}, body: "body",
		messages: []Message{{Kind: MessageInfo, Text: "info"}},
	}
	app.current = screen
	with := app.View()
	if strings.Count(without, "\n") != strings.Count(with, "\n") {
		t.Fatal("both views must keep exactly the terminal height")
	}
	if !strings.Contains(with, texts.MessageInfoLabel) {
		t.Fatal("message zone must show the INFO label")
	}
}
