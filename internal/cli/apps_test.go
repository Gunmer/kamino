package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/app"
)

func TestAppsHelpAndDiscovery(t *testing.T) {
	code, stdout, _ := run("apps")
	if code != 0 || !strings.Contains(stdout, "list") || !strings.Contains(stdout, "validate") || !strings.Contains(stdout, "new") {
		t.Fatalf("unexpected apps help: %d %q", code, stdout)
	}
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"apps", "list", "-h"}, []string{"kamino apps list"}},
		{[]string{"apps", "validate", "-h"}, []string{"[RUTA]"}},
		{[]string{"apps", "new", "-h"}, []string{"--command", "--bundle-id", "--path", "--optional-path", "--name", "--force", "--print"}},
	}
	for _, testCase := range cases {
		code, stdout, _ := run(testCase.args...)
		if code != 0 {
			t.Fatalf("unexpected help code for %v: %d", testCase.args, code)
		}
		for _, want := range testCase.want {
			if !strings.Contains(stdout, want) {
				t.Fatalf("missing %q in %q", want, stdout)
			}
		}
	}
	code, _, stderr := run("apps", "nope")
	if code != 2 || !strings.Contains(stderr, "comando desconocido") {
		t.Fatalf("unexpected unknown subcommand: %d %q", code, stderr)
	}
	code, stdout, _ = run("--help")
	if code != 0 || !strings.Contains(stdout, "apps") {
		t.Fatalf("main help must list apps: %d %q", code, stdout)
	}
}

func TestAppsNewParsesRepeatableFlags(t *testing.T) {
	original := executeAppsNew
	defer func() { executeAppsNew = original }()
	var captured app.AppsNewOptions
	var capturedHome string
	executeAppsNew = func(home string, options app.AppsNewOptions, _ []string, _ string) (app.AppsNewOutcome, error) {
		captured = options
		capturedHome = home
		return app.AppsNewOutcome{Path: "/tmp/miapp.toml"}, nil
	}
	code, stdout, _ := run("apps", "new", "miapp",
		"--command", "miapp",
		"--path", ".config/a",
		"--path", ".config/b",
		"--optional-path", ".config/c",
		"--search-path", "/opt/homebrew/bin",
		"--name", "Mi App",
		"--force",
	)
	if code != 0 {
		t.Fatalf("unexpected code: %d %q", code, stdout)
	}
	if capturedHome != "" {
		t.Fatalf("unexpected home: %q", capturedHome)
	}
	if captured.ID != "miapp" || captured.Command != "miapp" || captured.Name != "Mi App" || !captured.Force || captured.Print {
		t.Fatalf("unexpected options: %#v", captured)
	}
	if len(captured.Paths) != 2 || len(captured.OptionalPaths) != 1 || len(captured.SearchPaths) != 1 {
		t.Fatalf("repeatable flags lost values: %#v", captured)
	}
	if !strings.Contains(stdout, "creado /tmp/miapp.toml") {
		t.Fatalf("missing creation output: %q", stdout)
	}

	executeAppsNew = func(_ string, options app.AppsNewOptions, _ []string, _ string) (app.AppsNewOutcome, error) {
		captured = options
		return app.AppsNewOutcome{Content: "id = \"miapp\"\n"}, nil
	}
	code, stdout, _ = run("apps", "new", "miapp", "--command", "miapp", "--path", ".config/miapp", "--print")
	if code != 0 || stdout != "id = \"miapp\"\n" || !captured.Print {
		t.Fatalf("unexpected print result: %d %q %#v", code, stdout, captured)
	}
}

func TestAppsNewUsageErrors(t *testing.T) {
	original := executeAppsNew
	defer func() { executeAppsNew = original }()
	called := false
	executeAppsNew = func(string, app.AppsNewOptions, []string, string) (app.AppsNewOutcome, error) {
		called = true
		return app.AppsNewOutcome{Path: "/tmp/x.toml"}, nil
	}
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"apps", "new"}, "ID"},
		{[]string{"apps", "new", "miapp"}, "--command"},
		{[]string{"apps", "new", "miapp", "--command", "x"}, "--path"},
		{[]string{"apps", "new", "miapp", "--command", "x", "--path", ".x", "--unknown"}, "desconocida"},
	}
	for _, testCase := range cases {
		code, _, stderr := run(testCase.args...)
		if code != 2 || !strings.Contains(stderr, testCase.want) {
			t.Fatalf("unexpected usage result for %v: %d %q", testCase.args, code, stderr)
		}
	}
	if called {
		t.Fatal("usage errors must not create definitions")
	}
	if code, _, _ := run("apps", "new", "miapp", "--bundle-id", "com.example", "--path", ".x"); code != 0 {
		t.Fatalf("bundle-id without command must be accepted: %d", code)
	}
}

func TestAppsListOutput(t *testing.T) {
	original := executeAppsList
	defer func() { executeAppsList = original }()
	executeAppsList = func(string) (app.AppsListOutcome, error) {
		return app.AppsListOutcome{
			Directory:   "/Users/test/.config/kamino/apps",
			ConfigError: errors.New("configuración inválida en config.toml"),
			Entries: []app.AppListEntry{
				{ID: "opencode", Name: "OpenCode", Source: "integrado"},
				{ID: "miapp", Name: "Mi App", Source: "usuario", Warnings: []string{"no declara exclusiones"}},
				{File: "roto.toml", Source: "usuario", Err: errors.New("TOML inválido")},
			},
		}, nil
	}
	code, stdout, stderr := run("apps", "list")
	if code != 0 {
		t.Fatalf("list must not fail on invalid definitions: %d", code)
	}
	for _, want := range []string{"Catálogo: /Users/test/.config/kamino/apps", "integrado", "opencode", "usuario", "miapp", "ERROR", "roto.toml", "TOML inválido"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %q in list output:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stderr, "aviso:") || !strings.Contains(stderr, "configuración inválida") {
		t.Fatalf("missing config warning: %q", stderr)
	}
}

func TestAppsValidateOutputAndCode(t *testing.T) {
	original := executeAppsValidate
	defer func() { executeAppsValidate = original }()
	executeAppsValidate = func(_ string, path string) (app.AppsValidateOutcome, error) {
		if path != "" {
			return app.AppsValidateOutcome{Valid: true, ConfigError: errors.New("configuración inválida")}, nil
		}
		return app.AppsValidateOutcome{
			Directory:   "/Users/test/.config/kamino/apps",
			ConfigError: errors.New("configuración inválida"),
			Entries: []app.AppValidateEntry{
				{File: "ok.toml", ID: "ok", Name: "Ok"},
				{File: "warn.toml", ID: "warn", Name: "Warn", Warnings: []string{"no declara exclusiones"}},
				{File: "roto.toml", Err: errors.New("TOML inválido")},
			},
			Valid: false,
		}, nil
	}
	code, stdout, stderr := run("apps", "validate")
	if code != 1 || !strings.Contains(stdout, "OK") || !strings.Contains(stdout, "AVISO") || !strings.Contains(stdout, "ERROR") {
		t.Fatalf("unexpected validate result: %d\n%s", code, stdout)
	}
	if !strings.Contains(stderr, "catálogo integrado") {
		t.Fatalf("missing effective catalog warning: %q", stderr)
	}
	code, stdout, stderr = run("apps", "validate", "/tmp/file.toml")
	if code != 0 || !strings.Contains(stdout, "Ruta: /tmp/file.toml") || strings.Contains(stderr, "aviso:") {
		t.Fatalf("explicit path must not depend on config: %d %q %q", code, stdout, stderr)
	}
	executeAppsValidate = func(string, string) (app.AppsValidateOutcome, error) {
		return app.AppsValidateOutcome{Entries: []app.AppValidateEntry{{File: "ok.toml", ID: "ok", Name: "Ok"}}, Valid: true}, nil
	}
	if code, _, _ := run("apps", "validate"); code != 0 {
		t.Fatalf("a valid catalog must exit 0: %d", code)
	}
}

func TestAppsRejectsExtraPositionals(t *testing.T) {
	if code, _, _ := run("apps", "list", "extra"); code != 2 {
		t.Fatalf("apps list must reject positionals: %d", code)
	}
	if code, _, _ := run("apps", "validate", "a", "b"); code != 2 {
		t.Fatalf("apps validate must reject two paths: %d", code)
	}
}

func TestCompletionIncludesApps(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		code, stdout, _ := run("--show-completion", shell)
		if code != 0 || !strings.Contains(stdout, "apps") {
			t.Fatalf("completion for %s is missing apps: %d %q", shell, code, stdout)
		}
	}
}
