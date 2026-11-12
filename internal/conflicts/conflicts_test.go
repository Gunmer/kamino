package conflicts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
)

func entryFor(path string, data []byte, key string) archive.FileEntry {
	if key != "" {
		return archive.FileEntry{Path: path, Link: &key, Mode: "0644"}
	}
	digest := sha256.Sum256(data)
	value := hex.EncodeToString(digest[:])
	return archive.FileEntry{Path: path, SHA256: &value, Size: int64(len(data)), Mode: "0644"}
}

func makeConflict(local, clone []byte, path, localLink, cloneLink string) *Conflict {
	localFile := LocalFile{Kind: "file", Data: local, Mode: "0600"}
	if localLink != "" {
		localFile = LocalFile{Kind: "symlink", Link: localLink, Mode: "0644"}
	}
	entry := entryFor(path, clone, cloneLink)
	var cloneContent []byte
	if cloneLink == "" {
		cloneContent = clone
	}
	return BuildConflict("foo", path, "/home/user/"+path, entry, localFile, cloneContent)
}

func TestTextConflictAndIdentical(t *testing.T) {
	conflict := makeConflict([]byte("a = 1\n"), []byte("a = 2\n"), "cfg.toml", "", "")
	if conflict == nil || conflict.Kind != KindText {
		t.Fatalf("expected text conflict: %#v", conflict)
	}
	if len(conflict.Hunks) != 1 || conflict.Hunks[0].Rows[0].Kind != RowReplace {
		t.Fatalf("unexpected hunk: %#v", conflict.Hunks)
	}
	if makeConflict([]byte("same\n"), []byte("same\n"), "cfg.toml", "", "") != nil {
		t.Fatal("identical file must not conflict")
	}
	if makeConflict([]byte("same\r\n"), []byte("same\n"), "cfg.toml", "", "") != nil {
		t.Fatal("CRLF-only difference must not conflict")
	}
}

func TestTypeChangesAreNotConflicts(t *testing.T) {
	if makeConflict([]byte("regular\n"), []byte(""), "cfg.toml", "target", "") != nil {
		t.Fatal("regular to symlink is not a conflict")
	}
	entry := entryFor("cfg.toml", []byte("regular\n"), "")
	local := LocalFile{Kind: "symlink", Link: "target", Mode: "0644"}
	if BuildConflict("foo", "cfg.toml", "/home/user/cfg.toml", entry, local, []byte("regular\n")) != nil {
		t.Fatal("symlink to regular is not a conflict")
	}
}

func TestBinaryAndLargeConflicts(t *testing.T) {
	binary := makeConflict([]byte{0, 1}, []byte{0, 2}, "blob", "", "")
	if binary == nil || binary.Kind != KindBinary || len(binary.Hunks) != 0 {
		t.Fatalf("expected binary conflict: %#v", binary)
	}
	large := makeConflict(bytes.Repeat([]byte("a"), MaxTextBytes+1), bytes.Repeat([]byte("b"), MaxTextBytes+1), "big", "", "")
	if large == nil || large.Kind != KindLarge {
		t.Fatalf("expected large conflict: %#v", large)
	}
}

func TestSymlinkConflict(t *testing.T) {
	conflict := makeConflict(nil, nil, "link", "a", "b")
	if conflict == nil || conflict.Kind != KindLink || conflict.LocalLink != "a" || conflict.CloneLink != "b" {
		t.Fatalf("unexpected link conflict: %#v", conflict)
	}
	if makeConflict(nil, nil, "link", "a", "a") != nil {
		t.Fatal("identical symlink must not conflict")
	}
}

func TestHunksContextAndCompose(t *testing.T) {
	local := ""
	for index := 0; index < 20; index++ {
		local += "line " + itoaTest(index) + "\n"
	}
	clone := strings.Replace(local, "line 1\n", "LINE ONE\n", 1)
	clone = strings.Replace(clone, "line 19\n", "LINE NINETEEN\n", 1)
	conflict := makeConflict([]byte(local), []byte(clone), "cfg.toml", "", "")
	if conflict == nil || len(conflict.Hunks) != 2 {
		t.Fatalf("expected two hunks: %#v", conflict)
	}
	if RowCount(*conflict) != 2 {
		t.Fatalf("expected two rows, got %d", RowCount(*conflict))
	}
	merged, err := Compose(*conflict.LocalText, *conflict.CloneText, []RowDecision{RowClone, RowLocal})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(merged, "LINE ONE") || strings.Contains(merged, "LINE NINETEEN") {
		t.Fatalf("unexpected merge: %q", merged)
	}
}

func TestComposeInsideBlockCRLFAndFinalNewline(t *testing.T) {
	local := "a\nb\nc\n"
	clone := "a\nB\nC\n"
	conflict := makeConflict([]byte(local), []byte(clone), "cfg.toml", "", "")
	merged, err := Compose(*conflict.LocalText, *conflict.CloneText, []RowDecision{RowClone, RowLocal})
	if err != nil || merged != "a\nB\nc\n" {
		t.Fatalf("unexpected merge: %q %v", merged, err)
	}

	crlfLocal := "a\r\nb\r\nc\r\n"
	crlfConflict := makeConflict([]byte(crlfLocal), []byte(clone), "cfg.toml", "", "")
	merged, _ = Compose(*crlfConflict.LocalText, *crlfConflict.CloneText, []RowDecision{RowClone, RowClone})
	if merged != "a\r\nB\r\nC\r\n" {
		t.Fatalf("expected CRLF merge, got %q", merged)
	}

	noNewline := makeConflict([]byte("a\nb\n"), []byte("a\nB"), "cfg.toml", "", "")
	merged, _ = Compose(*noNewline.LocalText, *noNewline.CloneText, []RowDecision{RowClone})
	if merged != "a\nB" {
		t.Fatalf("expected missing final newline, got %q", merged)
	}
	if _, err := Compose("a\nb\nc\n", "a\nB\nC\n", []RowDecision{RowClone}); err == nil {
		t.Fatal("expected decision count error")
	}
}

func TestSplitLinesPythonSemantics(t *testing.T) {
	lines := splitLinesKeepEnds("a\nb\r\nc\vd")
	expected := []string{"a\n", "b\r\n", "c\v", "d"}
	if len(lines) != len(expected) {
		t.Fatalf("unexpected lines: %#v", lines)
	}
	for index := range expected {
		if lines[index] != expected[index] {
			t.Fatalf("line %d: %q != %q", index, lines[index], expected[index])
		}
	}
	if splitLines("a\n")[0] != "a" || len(splitLines("a\n")) != 1 {
		t.Fatal("unexpected splitLines")
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		path, local, clone, merged, status string
	}{
		{"cfg.toml", "a = 1\n", "a = 2\n", "a = 2\n", "valid"},
		{"cfg.toml", "a = 1\n", "a = 2\n", "a = \n", "invalid"},
		{"cfg.toml", "a = \n", "a = 2\n", "a = \n", "warning"},
		{"cfg.json", `{"a": 1}`, `{"a": 2}`, `{"a": 1, "a": 2}`, "invalid"},
		{"config.ini", "[a]\nx = 1\n", "[a]\nx = 2\n", "[a]\nx = 1\nx = 2\n", "invalid"},
		{"config.yaml", "a: 1\n", "a: 2\n", "a: 2\n", "unvalidated"},
	}
	for _, testCase := range cases {
		got := ValidateMerge(testCase.path, testCase.local, testCase.clone, testCase.merged)
		if got.Status != testCase.status {
			t.Fatalf("%s: got %s (%s), want %s", testCase.path, got.Status, got.Message, testCase.status)
		}
	}
	plistMerged := `<plist><dict><key>a</key><string>1</string><key>a</key><string>2</string></dict></plist>`
	if got := ValidateMerge("cfg.plist", "<plist><dict/></plist>", "<plist><dict/></plist>", plistMerged); got.Status != "invalid" {
		t.Fatalf("expected invalid plist, got %s", got.Status)
	}
}

func TestDecisions(t *testing.T) {
	first := makeConflict([]byte("a\n"), []byte("b\n"), "a.toml", "", "")
	second := makeConflict([]byte("c\n"), []byte("d\n"), "b.toml", "", "")
	conflicts := []Conflict{*first, *second}
	applied := ApplyAll(conflicts)
	if len(applied.Missing(conflicts)) != 0 {
		t.Fatal("apply all must cover conflicts")
	}
	if decision, _ := applied.Get("a.toml"); decision.Action != ActionApply {
		t.Fatal("unexpected action")
	}
	review := ReviewAll(*first, RowLocal)
	if len(review.Rows) != 1 || review.Rows[0] != RowLocal {
		t.Fatalf("unexpected review: %#v", review)
	}
}

func TestUnifiedDiffShape(t *testing.T) {
	diff := UnifiedDiff([]string{"a = 1"}, []string{"a = 2"}, "original/cfg.toml", "clon/cfg.toml")
	joined := strings.Join(diff, "\n")
	for _, expected := range []string{"--- original/cfg.toml", "+++ clon/cfg.toml", "@@", "-a = 1", "+a = 2"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}
}

func itoaTest(value int) string { return strings.TrimSpace(strings.Repeat(" ", 0) + itoa(value)) }
