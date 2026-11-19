package tui

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Gunmer/kamino/internal/texts"
)

var fieldSGR = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func stripSGR(value string) string {
	return fieldSGR.ReplaceAllString(value, "")
}

func reverseSGR(value string) int {
	count := 0
	for _, match := range fieldSGR.FindAllStringSubmatch(value, -1) {
		for _, param := range strings.Split(match[1], ";") {
			if param == "7" {
				count++
			}
		}
	}
	return count
}

func fieldHasSGRParam(value, wanted string) bool {
	for _, match := range fieldSGR.FindAllStringSubmatch(value, -1) {
		for _, param := range strings.Split(match[1], ";") {
			if param == wanted {
				return true
			}
		}
	}
	return false
}

func fieldColorSGR(value string) bool {
	for _, match := range fieldSGR.FindAllStringSubmatch(value, -1) {
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

func newTestField(label, value, placeholder string) field {
	value2 := field{Label: label, Placeholder: placeholder}
	value2.SetValue(value)
	return value2
}

func TestFieldEditing(t *testing.T) {
	value := field{}
	value.SetValue("ab cd")
	if value.Cursor != 5 {
		t.Fatalf("SetValue must leave the cursor at the end, got %d", value.Cursor)
	}
	value.MoveCursor(-3)
	value.Insert("X")
	if value.Value != "abX cd" || value.Cursor != 3 {
		t.Fatalf("insert in the middle failed: %q %d", value.Value, value.Cursor)
	}
	value.Backspace()
	if value.Value != "ab cd" || value.Cursor != 2 {
		t.Fatalf("backspace in the middle failed: %q %d", value.Value, value.Cursor)
	}
	value.MoveCursor(-100)
	if value.Cursor != 0 {
		t.Fatalf("cursor must stop at the start, got %d", value.Cursor)
	}
	value.MoveCursor(100)
	if value.Cursor != 5 {
		t.Fatalf("cursor must stop at the end, got %d", value.Cursor)
	}
	if value.Value != "ab cd" {
		t.Fatalf("moving the cursor must not edit the value: %q", value.Value)
	}
}

func TestFieldEditingUTF8(t *testing.T) {
	value := field{}
	value.SetValue("áé")
	value.MoveCursor(-1)
	value.Insert("ñ")
	if value.Value != "áñé" || value.Cursor != 2 {
		t.Fatalf("UTF-8 insertion failed: %q %d", value.Value, value.Cursor)
	}
	value.Backspace()
	if value.Value != "áé" || value.Cursor != 1 {
		t.Fatalf("UTF-8 backspace failed: %q %d", value.Value, value.Cursor)
	}
	value.MoveCursor(-1)
	value.Insert("x")
	if value.Value != "xáé" || value.Cursor != 1 {
		t.Fatalf("cursor must be a rune index: %q %d", value.Value, value.Cursor)
	}
}

func TestFieldRenderFocusedAndIdle(t *testing.T) {
	current := testApp(false, true).theme
	value := newTestField(texts.FieldDestination, "~/vault", "")
	focused := value.Render(current, fieldOptions{Width: 40, LabelWidth: 9, Focused: true})
	if strings.Contains(focused, "\n") {
		t.Fatal("a field must render exactly one row")
	}
	if !strings.HasPrefix(stripSGR(focused), "▌ Destino") {
		t.Fatalf("missing marker or label: %q", stripSGR(focused))
	}
	if reverseSGR(focused) != 1 {
		t.Fatalf("the focused field must draw exactly one cursor: %q", focused)
	}
	idle := value.Render(current, fieldOptions{Width: 40, LabelWidth: 9})
	if strings.Contains(stripSGR(idle), "▌") {
		t.Fatalf("an idle field must not draw the marker: %q", stripSGR(idle))
	}
	if reverseSGR(idle) != 0 {
		t.Fatalf("an idle field must not draw the cursor: %q", idle)
	}
	if fieldColorSGR(idle) {
		t.Fatalf("an idle field must use the attenuated style: %q", idle)
	}
}

func TestFieldRenderEmptyPlaceholder(t *testing.T) {
	current := testApp(true, true).theme
	value := newTestField(texts.FieldPath, "", texts.FieldPathPlaceholder)
	focused := value.Render(current, fieldOptions{Width: 60, LabelWidth: 9, Focused: true})
	plain := stripSGR(focused)
	if !strings.Contains(plain, texts.FieldPathPlaceholder) {
		t.Fatalf("missing placeholder: %q", plain)
	}
	cursor := strings.Index(focused, "\x1b[7m")
	placeholder := strings.Index(focused, texts.FieldPathPlaceholder)
	if cursor < 0 || cursor > placeholder {
		t.Fatalf("the cursor must come before the placeholder: %q", focused)
	}
	idle := value.Render(current, fieldOptions{Width: 60, LabelWidth: 9})
	if reverseSGR(idle) != 0 {
		t.Fatalf("an empty idle field must not draw the cursor: %q", idle)
	}
	if !strings.Contains(stripSGR(idle), texts.FieldPathPlaceholder) {
		t.Fatalf("an idle field must keep the placeholder: %q", stripSGR(idle))
	}
}

func TestFieldRenderError(t *testing.T) {
	current := testApp(false, true).theme
	value := newTestField(texts.FieldName, "invalido", "")
	value.Invalid = true
	rendered := value.Render(current, fieldOptions{Width: 40, LabelWidth: 9, Focused: true})
	if !fieldHasSGRParam(rendered, "31") {
		t.Fatalf("the invalid field must use the error color: %q", rendered)
	}
	if strings.Contains(stripSGR(rendered), texts.FieldEditing) {
		t.Fatal("the error style must not add the editing suffix")
	}
	if reverseSGR(rendered) != 1 {
		t.Fatal("an invalid focused field must keep exactly one cursor")
	}
}

func TestFieldHorizontalWindow(t *testing.T) {
	current := testApp(false, true).theme
	long := "/Users/test/" + strings.Repeat("carpeta/", 20) + "final"
	value := newTestField(texts.FieldDestination, long, "")
	width := 30

	focused := value.Render(current, fieldOptions{Width: width, LabelWidth: 9, Focused: true})
	plain := stripSGR(focused)
	if Width(plain) > width {
		t.Fatalf("the row exceeds its width: %d > %d", Width(plain), width)
	}
	if !strings.Contains(plain, "…") || !strings.Contains(plain, "final") {
		t.Fatalf("the cursor at the end must keep the suffix visible: %q", plain)
	}
	if reverseSGR(focused) != 1 {
		t.Fatal("the cursor must stay visible")
	}

	value.Cursor = 0
	focused = value.Render(current, fieldOptions{Width: width, LabelWidth: 9, Focused: true})
	plain = stripSGR(focused)
	if !strings.HasSuffix(plain, "…") {
		t.Fatalf("the hidden end must be marked with an ellipsis: %q", plain)
	}
	if !strings.Contains(plain, "/Users") {
		t.Fatalf("the beginning must stay visible with the cursor at the start: %q", plain)
	}

	value.Cursor = len([]rune(value.Value)) / 2
	focused = value.Render(current, fieldOptions{Width: width, LabelWidth: 9, Focused: true})
	plain = stripSGR(focused)
	if strings.Count(plain, "…") != 2 {
		t.Fatalf("a middle cursor must mark both hidden sides: %q", plain)
	}
	if reverseSGR(focused) != 1 {
		t.Fatal("the middle cursor must stay visible")
	}
}

func TestFieldHorizontalWindowWideRunes(t *testing.T) {
	current := testApp(false, true).theme
	value := newTestField(texts.FieldDestination, "配置/目录/文件/配置/目录", "")
	value.Cursor = 5
	for width := 8; width <= 30; width++ {
		rendered := value.Render(current, fieldOptions{Width: width, LabelWidth: 4, Focused: true})
		plain := stripSGR(rendered)
		if !utf8.ValidString(plain) {
			t.Fatalf("width %d split a rune: %q", width, plain)
		}
		if Width(plain) > width {
			t.Fatalf("width %d produced a %d-column row: %q", width, Width(plain), plain)
		}
	}
}

func TestFieldASCIIEllipsis(t *testing.T) {
	current := testApp(false, false).theme
	value := newTestField(texts.FieldPath, "/Users/test/"+strings.Repeat("carpeta/", 20), "")
	rendered := value.Render(current, fieldOptions{Width: 30, LabelWidth: 5, Focused: true})
	if strings.Contains(rendered, "…") {
		t.Fatalf("ASCII mode must not use the Unicode ellipsis: %q", rendered)
	}
	if !strings.Contains(stripSGR(rendered), "...") {
		t.Fatalf("ASCII mode must use three dots: %q", stripSGR(rendered))
	}
	if reverseSGR(rendered) != 1 {
		t.Fatal("ASCII mode must keep the reverse-video cursor")
	}
}

func TestFieldNoColorFocus(t *testing.T) {
	current := testApp(true, true).theme
	value := newTestField(texts.FieldDestination, "~/vault", "")
	rendered := value.Render(current, fieldOptions{Width: 60, LabelWidth: 20, Focused: true})
	plain := stripSGR(rendered)
	for _, part := range []string{"▌", texts.FieldEditing} {
		if !strings.Contains(plain, part) {
			t.Fatalf("NO_COLOR focus must show %q: %q", part, plain)
		}
	}
	if !fieldHasSGRParam(rendered, "7") {
		t.Fatalf("NO_COLOR focus must keep the reverse-video cursor: %q", rendered)
	}
	if fieldColorSGR(rendered) {
		t.Fatalf("NO_COLOR focus must not emit color or background: %q", rendered)
	}
}

func TestFieldLabelWidthAndAlignment(t *testing.T) {
	for _, noColor := range []bool{false, true} {
		current := testApp(noColor, true).theme
		labelWidth := fieldLabelWidth(current, texts.FieldDestination, "Nombre del archivo")
		first := newTestField(texts.FieldDestination, "uno", "")
		second := newTestField("Nombre del archivo", "dos", "")
		firstLine := first.Render(current, fieldOptions{Width: 70, LabelWidth: labelWidth, Focused: true})
		secondLine := second.Render(current, fieldOptions{Width: 70, LabelWidth: labelWidth})
		firstColumn := strings.Index(firstLine, "uno")
		secondColumn := strings.Index(secondLine, "dos")
		if firstColumn < 0 || secondColumn < 0 {
			t.Fatalf("missing values: %q %q", firstLine, secondLine)
		}
		if Width(firstLine[:firstColumn]) != Width(secondLine[:secondColumn]) {
			t.Fatalf("values are not aligned:\n%q\n%q", stripSGR(firstLine), stripSGR(secondLine))
		}
		hasSuffix := strings.Contains(stripSGR(firstLine), texts.FieldEditing)
		if noColor != hasSuffix {
			t.Fatalf("editing suffix must only appear with NO_COLOR: noColor=%v %q", noColor, stripSGR(firstLine))
		}
		if strings.Contains(stripSGR(secondLine), texts.FieldEditing) {
			t.Fatal("only the focused field may show the editing suffix")
		}
	}
}
