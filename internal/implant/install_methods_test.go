package implant

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/install"
)

// methodRunner fakes npm and curl. Installing a package or downloading a
// release creates the app command where detection looks for it.
type methodRunner struct {
	tools   map[string]bool
	bin     string
	payload []byte
	// failDownload makes every curl download fail like a 404.
	failDownload bool
	calls        []string
}

func (r *methodRunner) Lookup(name string) bool { return r.tools[name] }

func (r *methodRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	switch name {
	case "npm":
		command := "kamino-" + args[len(args)-1]
		return nil, os.WriteFile(filepath.Join(r.bin, command), []byte("#!/bin/sh\n"), 0o755)
	case "curl":
		if r.failDownload {
			return []byte("curl: (22) HTTP 404"), errors.New("exit status 22")
		}
		for _, arg := range args {
			if arg == "--write-out" {
				return []byte("https://github.com/acme/alpha/releases/tag/v1.2.3"), nil
			}
		}
		for index, arg := range args {
			if arg == "--output" {
				return nil, os.WriteFile(args[index+1], r.payload, 0o600)
			}
		}
	}
	return nil, errors.New("comando inesperado")
}

func methodRecipes() []install.Recipe {
	return []install.Recipe{
		{Method: install.MethodBrew, Formula: "alpha"},
		{Method: install.MethodNpm, Package: "alpha"},
		{Method: install.MethodGithubRelease, Repo: "acme/alpha", Asset: "alpha", Format: install.FormatRaw, Dest: ".local/bin/kamino-alpha"},
	}
}

func methodSession(t *testing.T, runner *methodRunner, path string, home string, cloneFile string, bin string) *ImplantSession {
	t.Helper()
	_, _, _, definitions := installFixture(t, "alpha")
	definitions["alpha"].Install = methodRecipes()
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: path, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)
	session.SetInstall(installSettings([]string{"alpha"}, runner, home))
	return session
}

func TestInstallFallsBackToNpmWhenBrewIsMissing(t *testing.T) {
	cloneFile, bin, home, _ := installFixture(t, "alpha")
	runner := &methodRunner{tools: map[string]bool{"npm": true}, bin: bin}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Installs) != 1 || plan.Installs[0].Status != InstallDone || plan.Installs[0].Method != install.MethodNpm {
		t.Fatalf("debe instalar con npm: %+v", plan.Installs)
	}
	if len(runner.calls) != 1 || runner.calls[0] != "npm install --global alpha" {
		t.Fatalf("comando inesperado: %v", runner.calls)
	}
}

func TestInstallFallsBackToGithubReleaseWithoutBrewOrNpm(t *testing.T) {
	cloneFile, _, home, _ := installFixture(t, "alpha")
	bin := filepath.Join(home, ".local/bin")
	runner := &methodRunner{tools: map[string]bool{}, payload: []byte("#!/bin/sh\necho alpha\n")}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Installs) != 1 || plan.Installs[0].Status != InstallDone || plan.Installs[0].Method != install.MethodGithubRelease {
		t.Fatalf("debe instalar desde el release: %+v", plan.Installs)
	}
	info, err := os.Stat(filepath.Join(bin, "kamino-alpha"))
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("el binario debe quedar en el home con 0755: %v, %v", info, err)
	}
	result, err := session.Apply(context.Background(), conflicts.ConflictDecisions{}, nil)
	if err != nil || ResultCode(result) != 0 {
		t.Fatalf("la app instalada debe recibir su configuración con código 0: %+v, %v", result, err)
	}
}

func TestInstallDryRunReportsGithubReleaseWithoutWriting(t *testing.T) {
	cloneFile, _, home, _ := installFixture(t, "alpha")
	bin := filepath.Join(home, ".local/bin")
	runner := &methodRunner{tools: map[string]bool{}, payload: []byte("x")}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	// Brew and npm apply on any Mac; the dry run reports the first recipe.
	session.definitions["alpha"].Install = methodRecipes()[2:]
	plan, err := session.Plan(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := plan.Installs[0]
	if got.Status != InstallPlanned || got.Method != install.MethodGithubRelease ||
		got.URL != "https://github.com/acme/alpha/releases" || !got.Unverified {
		t.Fatalf("la simulación debe informar origen y falta de verificación: %+v", got)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("la simulación no debe ejecutar nada: %v", runner.calls)
	}
	if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
		t.Fatal("la simulación no debe escribir nada")
	}
}

func TestInstallReleaseFailureLeavesExitThree(t *testing.T) {
	cloneFile, _, home, _ := installFixture(t, "alpha")
	bin := filepath.Join(home, ".local/bin")
	runner := &methodRunner{tools: map[string]bool{}, failDownload: true}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Installs[0].Status != InstallFailed {
		t.Fatalf("la descarga fallida debe omitir la app: %+v", plan.Installs)
	}
	result, err := session.Apply(context.Background(), conflicts.ConflictDecisions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ResultCode(result) != 3 {
		t.Fatalf("un fallo debe dejar código 3, got %d", ResultCode(result))
	}
	if _, err := os.Stat(filepath.Join(bin, "kamino-alpha")); !os.IsNotExist(err) {
		t.Fatal("un fallo no debe dejar el binario")
	}
}

func TestInstallCandidatesAndDryRunUseTheRecipeThatWouldRun(t *testing.T) {
	cloneFile, _, home, _ := installFixture(t, "alpha")
	bin := filepath.Join(home, ".local/bin")
	runner := &methodRunner{tools: map[string]bool{}}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	candidates := session.InstallCandidates("darwin", "arm64", runner)
	if len(candidates) != 1 || candidates[0].Recipe.Method != install.MethodGithubRelease {
		t.Fatalf("sin brew ni npm se anuncia github-release: %+v", candidates)
	}
	plan, err := session.Plan(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Installs[0].Method != install.MethodGithubRelease {
		t.Fatalf("la simulación debe anunciar el método que se ejecutaría: %+v", plan.Installs[0])
	}
	session.definitions["alpha"].Install = methodRecipes()[:2]
	if got := session.InstallCandidates("darwin", "arm64", runner); len(got) != 0 {
		t.Fatalf("sin herramienta para ninguna receta no se ofrece la app: %+v", got)
	}
}

func TestInstallUsesTheChosenRecipeInsteadOfTheDefault(t *testing.T) {
	cloneFile, _, home, _ := installFixture(t, "alpha")
	bin := filepath.Join(home, ".local/bin")
	runner := &methodRunner{tools: map[string]bool{"npm": true}, bin: bin, payload: []byte("#!/bin/sh\n")}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	candidates := session.InstallCandidates("darwin", "arm64", runner)
	if len(candidates) != 1 || len(candidates[0].Recipes) != 2 || candidates[0].Recipe.Method != install.MethodNpm {
		t.Fatalf("deben listarse todas las vías ejecutables, npm por defecto: %+v", candidates)
	}
	settings := installSettings([]string{"alpha"}, runner, home)
	settings.Chosen = map[string]install.Recipe{"alpha": candidates[0].Recipes[1]}
	session.SetInstall(settings)
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Installs[0].Status != InstallDone || plan.Installs[0].Method != install.MethodGithubRelease {
		t.Fatalf("debe ejecutarse la vía elegida: %+v", plan.Installs[0])
	}
	for _, call := range runner.calls {
		if strings.HasPrefix(call, "npm") {
			t.Fatalf("no debe ejecutarse la receta por defecto: %v", runner.calls)
		}
	}
}

func TestInstallRejectsAChosenRecipeOutsideTheCatalogDefinition(t *testing.T) {
	cloneFile, _, home, _ := installFixture(t, "alpha")
	bin := filepath.Join(home, ".local/bin")
	runner := &methodRunner{tools: map[string]bool{"npm": true}, bin: bin}
	session := methodSession(t, runner, bin, home, cloneFile, bin)
	settings := installSettings([]string{"alpha"}, runner, home)
	settings.Chosen = map[string]install.Recipe{"alpha": {Method: install.MethodInstaller, URL: "https://evil.example/i.sh"}}
	session.SetInstall(settings)
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Installs[0].Status != InstallFailed || len(runner.calls) != 0 {
		t.Fatalf("una receta ajena al catálogo no debe ejecutarse: %+v, %v", plan.Installs[0], runner.calls)
	}
}
