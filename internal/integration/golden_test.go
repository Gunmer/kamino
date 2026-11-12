package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/texts"
)

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func TestGoldenManifestAndArchive(t *testing.T) {
	root := repoRoot(t)
	cloneFile := filepath.Join(root, "testdata", "golden", "clone.tar.gz")
	manifest, err := archive.ReadManifest(cloneFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.VerifyArchive(cloneFile, manifest, nil, nil); err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]any)
	readJSON(t, filepath.Join(root, "testdata", "golden", "manifest.json"), &expected)
	if expected["format_version"].(float64) != float64(manifest.FormatVersion) {
		t.Fatalf("format_version mismatch")
	}
	if expected["hostname"] != manifest.Hostname || expected["created_at"] != manifest.CreatedAt {
		t.Fatalf("manifest metadata mismatch: %#v", manifest)
	}
	if len(manifest.Apps) != 2 {
		t.Fatalf("unexpected apps: %#v", manifest.Apps)
	}
}

func TestGoldenPlan(t *testing.T) {
	root := repoRoot(t)
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]\n\tname = Golden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "git", "ignore"), []byte("ignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := "git"
	definition := &catalog.AppDefinition{ID: "git", Name: "Git", Command: &command, Paths: []string{".gitconfig"}}
	built, err := plan.BuildPlan([]*catalog.AppDefinition{definition}, home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Operations []struct {
			AppID  string  `json:"app_id"`
			Path   string  `json:"path"`
			Size   int64   `json:"size"`
			SHA256 *string `json:"sha256"`
		} `json:"operations"`
	}
	readJSON(t, filepath.Join(root, "testdata", "golden", "plan.json"), &expected)
	if len(built.Operations) != len(expected.Operations) {
		t.Fatalf("operation count mismatch: %d != %d", len(built.Operations), len(expected.Operations))
	}
	for index, operation := range built.Operations {
		want := expected.Operations[index]
		if operation.Path != want.Path || operation.Size != want.Size {
			t.Fatalf("operation %d mismatch: %#v != %#v", index, operation, want)
		}
		if (operation.SHA256 == nil) != (want.SHA256 == nil) {
			t.Fatalf("sha mismatch at %d", index)
		}
		if operation.SHA256 != nil && *operation.SHA256 != *want.SHA256 {
			t.Fatalf("sha mismatch at %d", index)
		}
	}
}

func TestGoldenConflictsAndTexts(t *testing.T) {
	root := repoRoot(t)
	var expected struct {
		Hunks []struct {
			Index int `json:"index"`
			Rows  []struct {
				Kind  string `json:"kind"`
				Local string `json:"local"`
				Clone string `json:"clone"`
			} `json:"rows"`
		} `json:"hunks"`
		ComposeClone string `json:"compose_clone"`
		ComposeLocal string `json:"compose_local"`
	}
	readJSON(t, filepath.Join(root, "testdata", "golden", "conflicts.json"), &expected)
	local := ""
	for index := 0; index < 10; index++ {
		local += "line " + itoa(index) + "\n"
	}
	clone := strings.Replace(local, "line 4\n", "LINE FOUR\n", 1)
	hunks := conflicts.BuildHunks(local, clone)
	if len(hunks) != len(expected.Hunks) {
		t.Fatalf("hunk count mismatch")
	}
	for index, hunk := range hunks {
		if hunk.Index != expected.Hunks[index].Index || len(hunk.Rows) != len(expected.Hunks[index].Rows) {
			t.Fatalf("hunk %d mismatch", index)
		}
		for rowIndex, row := range hunk.Rows {
			want := expected.Hunks[index].Rows[rowIndex]
			if string(row.Kind) != want.Kind || row.Local != want.Local || row.Clone != want.Clone {
				t.Fatalf("row %d/%d mismatch: %#v != %#v", index, rowIndex, row, want)
			}
		}
	}
	decisions := make([]conflicts.RowDecision, 0)
	for _, hunk := range hunks {
		for _, row := range hunk.Rows {
			if row.Kind != conflicts.RowEqual {
				decisions = append(decisions, conflicts.RowClone)
			}
		}
	}
	composed, err := conflicts.Compose(local, clone, decisions)
	if err != nil || composed != expected.ComposeClone {
		t.Fatalf("compose mismatch: %q %v != %q", composed, err, expected.ComposeClone)
	}

	var expectedTexts struct {
		HelpC1 string `json:"help_c1"`
		HelpD2 string `json:"help_d2"`
	}
	readJSON(t, filepath.Join(root, "testdata", "golden", "texts.json"), &expectedTexts)
	if got := texts.HelpTexts(texts.ScreenC1, texts.HelpC1, texts.CloneSteps, "apps", false, true); got != expectedTexts.HelpC1 {
		t.Fatalf("help C1 mismatch:\n%s\n---\n%s", got, expectedTexts.HelpC1)
	}
	if got := texts.HelpTexts(texts.ScreenD2, texts.HelpD2, texts.DeploySteps, "conflictos", true, true); got != expectedTexts.HelpD2 {
		t.Fatalf("help D2 mismatch:\n%s\n---\n%s", got, expectedTexts.HelpD2)
	}
}

func TestGoldenCodes(t *testing.T) {
	root := repoRoot(t)
	binary := buildGo(t)
	var cases []struct {
		Args []string `json:"args"`
		Code int      `json:"code"`
	}
	readJSON(t, filepath.Join(root, "testdata", "golden", "codes.json"), &cases)
	home := seedHome(t)
	for _, testCase := range cases {
		args := append([]string(nil), testCase.Args...)
		for index, argument := range args {
			if strings.HasSuffix(argument, "clone.tar.gz") {
				args[index] = filepath.Join(root, "testdata", "golden", "clone.tar.gz")
			}
		}
		command := exec.Command(binary, args...)
		command.Dir = root
		command.Env = append(os.Environ(), "HOME="+home, "KAMINO_PLAIN=1")
		err := command.Run()
		code := 0
		if exitError, ok := err.(*exec.ExitError); ok {
			code = exitError.ExitCode()
		} else if err != nil {
			t.Fatalf("running %v: %v", args, err)
		}
		if code != testCase.Code {
			t.Fatalf("%v: code %d, want %d", args, code, testCase.Code)
		}
	}
}

func itoa(value int) string { return strconv.Itoa(value) }
