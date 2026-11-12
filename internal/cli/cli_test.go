package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/app"
	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/tui"
	"github.com/Gunmer/kamino/internal/update"
	"github.com/Gunmer/kamino/internal/version"
)

func run(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Run(args, Options{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr, Interactive: func() bool { return false }})
	return code, stdout.String(), stderr.String()
}

func TestVersionAndHelp(t *testing.T) {
	code, stdout, _ := run("--version")
	if code != 0 || stdout != "kamino "+version.Version+"\n" {
		t.Fatalf("unexpected version: %d %q", code, stdout)
	}
	code, stdout, _ = run("--help")
	if code != 0 || !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, "clone") || !strings.Contains(stdout, "deploy") {
		t.Fatalf("unexpected help: %d %q", code, stdout)
	}
	for _, option := range []string{"--version", "-V", "--install-completion", "--show-completion"} {
		if !strings.Contains(stdout, option) {
			t.Fatalf("missing %s in help", option)
		}
	}
}

func TestCloneCommandParsesFlagsAndAliases(t *testing.T) {
	original := executeClone
	var requested *[]string
	var output string
	var dryRun, force bool
	executeClone = func(sel *[]string, out string, dry, f bool, _ catalog.Catalog, _ string, _ []string, _ string) (app.CloneOutcome, error) {
		requested = sel
		output = out
		dryRun = dry
		force = f
		return app.CloneOutcome{Result: clone.CloneResult{Plan: plan.Plan{}, Manifest: &archive.Manifest{}}, Code: 0}, nil
	}
	defer func() { executeClone = original }()
	code, stdout, _ := run("clone", "-a", "git,git, nope", "-o", "out.tar.gz", "-n", "-f")
	if code != 0 || requested == nil || len(*requested) != 2 || output != "out.tar.gz" || !dryRun || !force {
		t.Fatalf("unexpected parse: %d %#v %q %v %v", code, requested, output, dryRun, force)
	}
	if !strings.Contains(stdout, "aviso") {
		t.Fatalf("expected privacy warning, got %q", stdout)
	}
}

func TestCloneExitCodeAndMissingWarning(t *testing.T) {
	original := executeClone
	executeClone = func(*[]string, string, bool, bool, catalog.Catalog, string, []string, string) (app.CloneOutcome, error) {
		return app.CloneOutcome{Result: clone.CloneResult{Plan: plan.Plan{}, Manifest: &archive.Manifest{}}, Code: 3, Missing: []string{"nope"}}, nil
	}
	defer func() { executeClone = original }()
	code, _, stderr := run("clone", "--apps", "git,nope")
	if code != 3 || !strings.Contains(stderr, "nope") {
		t.Fatalf("unexpected: %d %q", code, stderr)
	}
}

func TestCloneErrorExitsOne(t *testing.T) {
	original := executeClone
	executeClone = func(*[]string, string, bool, bool, catalog.Catalog, string, []string, string) (app.CloneOutcome, error) {
		return app.CloneOutcome{}, &testError{"fallo de clon"}
	}
	defer func() { executeClone = original }()
	code, _, stderr := run("clone", "--apps", "git")
	if code != 1 || !strings.Contains(stderr, "fallo de clon") {
		t.Fatalf("unexpected: %d %q", code, stderr)
	}
}

type testError struct{ message string }

func (e *testError) Error() string { return e.message }

func TestDeployRequiresYes(t *testing.T) {
	code, _, stderr := run("deploy", "clone.tar.gz")
	if code != 1 || !strings.Contains(stderr, "--yes") {
		t.Fatalf("unexpected: %d %q", code, stderr)
	}
}

func TestDeployYesAndDryRun(t *testing.T) {
	original := executeDeploy
	calls := 0
	executeDeploy = func(string, *[]string, bool, catalog.Catalog, string, []string, string, string) (app.DeployOutcome, error) {
		calls++
		return app.DeployOutcome{Result: deploy.DeployResult{Plan: &deploy.DeployPlan{}, Manifest: &archive.Manifest{}}, Code: 0}, nil
	}
	defer func() { executeDeploy = original }()
	if code, _, _ := run("deploy", "clone.tar.gz", "--yes"); code != 0 {
		t.Fatalf("unexpected yes code: %d", code)
	}
	if code, _, _ := run("deploy", "clone.tar.gz", "--dry-run"); code != 0 {
		t.Fatalf("unexpected dry-run code: %d", code)
	}
	if calls != 2 {
		t.Fatalf("expected two deploy calls, got %d", calls)
	}
}

func TestInteractiveGating(t *testing.T) {
	originalMenu := runMenu
	originalClone := runCloneFlow
	originalDeploy := runDeployFlow
	originalUpdate := runUpdateFlow
	defer func() {
		runMenu = originalMenu
		runCloneFlow = originalClone
		runDeployFlow = originalDeploy
		runUpdateFlow = originalUpdate
	}()
	runMenu = func() int { return 3 }
	var stdout, stderr bytes.Buffer
	code := Run(nil, Options{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr, Interactive: func() bool { return true }})
	if code != 3 {
		t.Fatalf("expected menu code, got %d", code)
	}
	runCloneFlow = func(options tui.CloneFlowOptions) int {
		if options.Requested == nil || len(*options.Requested) != 1 {
			t.Fatalf("unexpected requested: %#v", options.Requested)
		}
		return 0
	}
	code = Run([]string{"clone", "-a", "git", "-o", "out.tar.gz"}, Options{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr, Interactive: func() bool { return true }})
	if code != 0 {
		t.Fatalf("unexpected clone code: %d", code)
	}
	existing := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runDeployFlow = func(options tui.DeployFlowOptions) int {
		if options.FileError != "" {
			t.Fatalf("unexpected file error: %s", options.FileError)
		}
		return 0
	}
	code = Run([]string{"deploy", existing}, Options{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr, Interactive: func() bool { return true }})
	if code != 0 {
		t.Fatalf("unexpected deploy code: %d", code)
	}
	runUpdateFlow = func() int { return 0 }
	code = Run([]string{"update"}, Options{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr, Interactive: func() bool { return true }})
	if code != 0 {
		t.Fatalf("unexpected update code: %d", code)
	}
}

func TestKaminoPlainForcesTextMode(t *testing.T) {
	t.Setenv("KAMINO_PLAIN", "1")
	if defaultInteractive(strings.NewReader(""), &bytes.Buffer{}) {
		t.Fatal("KAMINO_PLAIN must force text mode")
	}
}

func TestUpdateCommand(t *testing.T) {
	original := executeUpdate
	executeUpdate = func(string, string) (update.Result, error) {
		return update.Result{CurrentVersion: "0.8.0", LatestVersion: "0.9.0", Target: "/tmp/kamino", Updated: true, Outcome: update.OutcomeUpdated}, nil
	}
	defer func() { executeUpdate = original }()
	code, stdout, _ := run("update")
	if code != 0 || !strings.Contains(stdout, "0.8.0 → 0.9.0") || !strings.Contains(stdout, "Reinicia kamino") {
		t.Fatalf("unexpected update: %d %q", code, stdout)
	}
}

func TestHelpAliases(t *testing.T) {
	cases := []struct {
		args     []string
		expected []string
	}{
		{[]string{"clone", "-h"}, []string{"--force", "-f", "--help", "-h"}},
		{[]string{"deploy", "-h"}, []string{"--yes", "-y", "--help", "-h"}},
	}
	for _, testCase := range cases {
		code, stdout, _ := run(testCase.args...)
		if code != 0 {
			t.Fatalf("unexpected help code: %d", code)
		}
		for _, want := range testCase.expected {
			if !strings.Contains(stdout, want) {
				t.Fatalf("missing %s in %q", want, stdout)
			}
		}
	}
}
