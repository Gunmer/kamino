package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNextVersion(t *testing.T) {
	commits := func(types ...string) []Commit {
		result := make([]Commit, 0, len(types))
		for _, item := range types {
			breaking := strings.HasSuffix(item, "!")
			result = append(result, Commit{
				SHA:      "0123456789abcdef0123456789abcdef01234567",
				Type:     strings.TrimSuffix(item, "!"),
				Breaking: breaking,
			})
		}
		return result
	}
	cases := []struct {
		name    string
		current string
		commits []Commit
		want    string
		bump    Bump
	}{
		{"fix", "0.1.0", commits("fix"), "0.1.1", BumpPatch},
		{"feat", "0.1.0", commits("feat"), "0.2.0", BumpMinor},
		{"breaking feat on zero", "0.1.0", commits("feat!"), "0.2.0", BumpMinor},
		{"breaking fix on zero", "0.1.0", commits("fix!"), "0.2.0", BumpMinor},
		{"breaking chore on zero", "0.1.0", commits("chore!"), "0.2.0", BumpMinor},
		{"ci only", "0.1.0", commits("ci"), "0.1.0", BumpNone},
		{"docs only", "0.1.0", commits("docs"), "0.1.0", BumpNone},
		{"perf only", "0.1.0", commits("perf"), "0.1.0", BumpNone},
		{"refactor only", "0.1.0", commits("refactor"), "0.1.0", BumpNone},
		{"patch and minor", "0.1.0", commits("fix", "feat"), "0.2.0", BumpMinor},
		{"breaking on stable", "1.0.0", commits("fix!"), "2.0.0", BumpMajor},
		{"no commits", "0.1.0", nil, "0.1.0", BumpNone},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, bump, err := NextVersion(testCase.current, testCase.commits)
			if err != nil {
				t.Fatal(err)
			}
			if got != testCase.want || bump != testCase.bump {
				t.Fatalf("got %s/%v, want %s/%v", got, bump, testCase.want, testCase.bump)
			}
		})
	}
}

func TestParseCommit(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	t.Run("simple", func(t *testing.T) {
		commits := ParseCommit(sha, "feat(cli): añade una opción")
		if len(commits) != 1 {
			t.Fatalf("commits: %#v", commits)
		}
		got := commits[0]
		if got.Type != "feat" || got.Scope != "cli" || got.Description != "añade una opción" || got.Breaking {
			t.Fatalf("unexpected commit: %#v", got)
		}
	})
	t.Run("breaking footer", func(t *testing.T) {
		commits := ParseCommit(sha, "fix: cambia el formato\n\nBREAKING CHANGE: requiere migración")
		if len(commits) != 1 || !commits[0].Breaking {
			t.Fatalf("commits: %#v", commits)
		}
		if len(commits[0].BreakingNotes) != 1 || commits[0].BreakingNotes[0] != "requiere migración" {
			t.Fatalf("notes: %#v", commits[0].BreakingNotes)
		}
	})
	t.Run("squash", func(t *testing.T) {
		commits := ParseCommit(sha, "feat: una\n\nfix: dos\n\nchore: tres")
		if len(commits) != 3 {
			t.Fatalf("commits: %#v", commits)
		}
		if commits[0].Type != "feat" || commits[1].Type != "fix" || commits[2].Type != "chore" {
			t.Fatalf("commits: %#v", commits)
		}
	})
	t.Run("merge", func(t *testing.T) {
		if commits := ParseCommit(sha, "Merge pull request #4 from example/branch"); len(commits) != 0 {
			t.Fatalf("commits: %#v", commits)
		}
	})
	t.Run("non conventional", func(t *testing.T) {
		if commits := ParseCommit(sha, "mensaje sin convención"); len(commits) != 0 {
			t.Fatalf("commits: %#v", commits)
		}
	})
}

func TestValidateTag(t *testing.T) {
	for _, tag := range []string{"v0.1.0", "v1.2.3", "v10.20.30"} {
		if _, err := ValidateTag(tag); err != nil {
			t.Fatalf("%s: %v", tag, err)
		}
	}
	for _, tag := range []string{"--help", "v01.2.3", "v1.2", "v1.2.3\n", "v1.2.3;whoami", "1.2.3"} {
		if _, err := ValidateTag(tag); err == nil {
			t.Fatalf("%q should fail", tag)
		}
	}
}

func TestRenderChangelogEntry(t *testing.T) {
	commits := []Commit{
		{SHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Type: "feat", Description: "añade una aplicación"},
		{SHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Type: "fix", Description: "corrige el despliegue"},
		{SHA: "cccccccccccccccccccccccccccccccccccccccc", Type: "docs", Description: "mejora la guía"},
		{
			SHA:           "dddddddddddddddddddddddddddddddddddddddd",
			Type:          "feat",
			Description:   "cambia el formato",
			Breaking:      true,
			BreakingNotes: []string{"requiere migración"},
		},
	}
	entry, err := RenderChangelogEntry("0.2.0", time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), commits, func(sha string) string {
		return "https://github.com/example/kamino/commit/" + sha
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"## v0.2.0 (2026-11-15)",
		"### Correcciones",
		"### Documentación",
		"### Funcionalidades",
		"- corrige el despliegue ([`bbbbbbb`](https://github.com/example/kamino/commit/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb))",
		"**Cambio incompatible.** requiere migración",
	} {
		if !strings.Contains(entry, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, entry)
		}
	}
	if strings.Index(entry, "### Correcciones") > strings.Index(entry, "### Documentación") {
		t.Fatalf("heading order: %s", entry)
	}
}

func TestInsertChangelogEntry(t *testing.T) {
	previous := "# Historial de cambios\n\n<!-- version list -->\n\n" +
		"## Sin publicar\n\n### Mantenimiento\n\n- retirar la implementación anterior\n\n" +
		"## v0.1.0 (2026-11-01)\n\n### Funcionalidades\n\n- inicial\n"
	entry := "## v0.2.0 (2026-11-15)\n\n### Correcciones\n\n- corrige"
	updated, err := InsertChangelogEntry(previous, entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(updated, "## Sin publicar") {
		t.Fatalf("unreleased section was not removed:\n%s", updated)
	}
	if !strings.Contains(updated, "<!-- version list -->\n\n"+entry+"\n\n## v0.1.0") {
		t.Fatalf("unexpected insertion:\n%s", updated)
	}
	if !strings.HasSuffix(updated, "\n") {
		t.Fatalf("missing trailing newline: %q", updated)
	}
	if _, err := InsertChangelogEntry("# Historial\n", entry); err == nil {
		t.Fatal("missing marker should fail")
	}
}

func TestReleaseNotes(t *testing.T) {
	changelog := "# Historial\n\n## v0.2.0 (2026-11-15)\n\n- dos\n\n## v0.1.0 (2026-11-01)\n\n- uno\n"
	notes, err := ReleaseNotes(changelog, "v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if notes != "## v0.2.0 (2026-11-15)\n\n- dos\n" {
		t.Fatalf("notes: %q", notes)
	}
	if _, err := ReleaseNotes(changelog+"\n## v0.2.0 (2026-11-16)\n\n- duplicado\n", "v0.2.0"); err == nil {
		t.Fatal("duplicate section should fail")
	}
	if _, err := ReleaseNotes(changelog, "v0.3.0"); err == nil {
		t.Fatal("missing section should fail")
	}
}

func TestVersionFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version.go")
	source := "package version\n\n// Version is overridden in release builds with -X.\nvar Version = \"0.1.0\"\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteVersionFile(path, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadVersionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.2.0" {
		t.Fatalf("version: %s", got)
	}
}

func TestWriteRelease(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "version"), 0o755); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(root, "internal", "version", "version.go")
	if err := os.WriteFile(versionPath, []byte("package version\n\nvar Version = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changelogPath := filepath.Join(root, "CHANGELOG.md")
	previous := "# Historial\n\n<!-- version list -->\n\n## v0.1.0 (2026-11-01)\n\n- inicial\n"
	if err := os.WriteFile(changelogPath, []byte(previous), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := "## v0.2.0 (2026-11-15)\n\n### Funcionalidades\n\n- nueva"
	if err := writeRelease(root, "0.2.0", entry); err != nil {
		t.Fatal(err)
	}
	changelog, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changelog), entry) || !strings.Contains(string(changelog), "## v0.1.0 (2026-11-01)") {
		t.Fatalf("unexpected changelog:\n%s", changelog)
	}
	version, err := ReadVersionFile(versionPath)
	if err != nil || version != "0.2.0" {
		t.Fatalf("version: %q %v", version, err)
	}
}

func TestCommitRelease(t *testing.T) {
	root := t.TempDir()
	if err := runGitOK(root, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "add", "--all"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("release\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := commitRelease(root, "v0.2.0", "0.2.0"); err != nil {
		t.Fatal(err)
	}
	subject, err := runGit(root, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "chore(release): 0.2.0 [skip ci]" {
		t.Fatalf("subject: %q", subject)
	}
	tags, err := runGit(root, "tag", "--list")
	if err != nil {
		t.Fatal(err)
	}
	if tags != "v0.2.0" {
		t.Fatalf("tags: %q", tags)
	}
	status, err := runGit(root, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if status != "" {
		t.Fatalf("status: %q", status)
	}
}

func TestValidateRelease(t *testing.T) {
	root := t.TempDir()
	if err := runGitOK(root, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "version"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "version", "version.go"),
		[]byte("package version\n\nvar Version = \"0.2.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"),
		[]byte("# Historial\n\n## v0.2.0 (2026-11-15)\n\n- nueva\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "add", "--all"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "commit", "-m", "chore(release): 0.2.0 [skip ci]"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "tag", "v0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := validateRelease(root, "v0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "tag", "-d", "v0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "commit", "--allow-empty", "-m", "otro cambio"); err != nil {
		t.Fatal(err)
	}
	if err := runGitOK(root, "tag", "v0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := validateRelease(root, "v0.2.0"); err == nil {
		t.Fatal("non-release tag should fail")
	}
}
