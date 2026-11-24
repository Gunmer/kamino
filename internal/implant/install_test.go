package implant

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/install"
)

// fakeBrew stands in for brew: a successful install creates the app command in
// the detection PATH, as a real formula would.
type fakeBrew struct {
	bin      string
	failFor  map[string]bool
	installs []string
	calls    int
	// onInstall runs after a successful fake install, as an app writing its
	// default configuration would.
	onInstall func(formula string)
}

func (f *fakeBrew) Lookup(name string) bool { return name == "brew" }

func (f *fakeBrew) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls++
	if name != "brew" || len(args) != 2 || args[0] != "install" {
		return nil, errors.New("comando inesperado")
	}
	formula := args[1]
	if f.failFor[formula] {
		return []byte("Error: formula fallida"), errors.New("exit status 1")
	}
	f.installs = append(f.installs, formula)
	if err := os.WriteFile(filepath.Join(f.bin, "kamino-"+formula), []byte("#!/bin/sh\n"), 0o755); err != nil {
		return nil, err
	}
	if f.onInstall != nil {
		f.onInstall(formula)
	}
	return nil, nil
}

// installFixture builds a clone holding a config file for each app id. Every
// app is missing on this Mac until the fake brew installs it.
func installFixture(t *testing.T, ids ...string) (cloneFile, bin, home string, definitions catalog.Catalog) {
	t.Helper()
	sourceHome := t.TempDir()
	bin = t.TempDir()
	home = t.TempDir()
	definitions = catalog.Catalog{}
	manifest := &archive.Manifest{
		FormatVersion: 1, ToolVersion: "0.15.0", CreatedAt: "2026-01-01T00:00:00Z",
		Arch: "arm64", Hostname: "h",
	}
	digest, err := checksum([]byte("value=2\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		rel := filepath.Join(".config", id, "config")
		if err := os.MkdirAll(filepath.Join(sourceHome, ".config", id), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceHome, rel), []byte("value=2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		command := "kamino-" + id
		definitions[id] = &catalog.AppDefinition{
			ID: id, Name: id, Command: &command,
			Paths:   []string{filepath.ToSlash(rel)},
			Install: []install.Recipe{{Method: install.MethodBrew, Formula: id}},
		}
		manifest.Apps = append(manifest.Apps, archive.AppEntry{
			ID: id, Name: id,
			Files: []archive.FileEntry{{Path: filepath.ToSlash(rel), SHA256: &digest, Size: 8, Mode: "0644"}},
		})
	}
	cloneFile = filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := archive.WriteArchive(context.Background(), cloneFile, manifest, archive.WriteOptions{SourceRoot: sourceHome}); err != nil {
		t.Fatal(err)
	}
	return cloneFile, bin, home, definitions
}

func installSettings(ids []string, runner install.Runner, home string) InstallSettings {
	return InstallSettings{
		AppIDs:    ids,
		Installer: &install.Installer{Runner: runner, Home: home},
		OS:        "darwin",
		Arch:      "arm64",
	}
}

func TestInstallRunsBeforeClassificationAndReclassifiesInstalledApp(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha")
	runner := &fakeBrew{bin: bin}
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if len(session.ImplantableApps) != 0 || len(session.NotInstalledApps) != 1 {
		t.Fatalf("alpha debe empezar ausente: %d desplegables, %d ausentes", len(session.ImplantableApps), len(session.NotInstalledApps))
	}
	session.SetInstall(installSettings([]string{"alpha"}, runner, home))
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Installs) != 1 || plan.Installs[0].Status != InstallDone {
		t.Fatalf("la instalación debe quedar hecha: %+v", plan.Installs)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].AppID != "alpha" {
		t.Fatalf("la app instalada debe clasificarse en el plan: %+v", plan.Entries)
	}
	result, err := session.Apply(context.Background(), conflicts.ConflictDecisions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Implanted) != 1 || ResultCode(result) != 0 {
		t.Fatalf("la app instalada debe recibir su configuración con código 0: %+v", result)
	}
	if data, _ := os.ReadFile(filepath.Join(home, ".config/alpha/config")); string(data) != "value=2\n" {
		t.Fatalf("configuración no desplegada: %q", data)
	}
}

func TestInstallHappensBeforeClassificationSoDefaultConfigIsSeen(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha")
	// The app writes its own default configuration while being installed.
	runner := &fakeBrew{bin: bin, onInstall: func(formula string) {
		dir := filepath.Join(home, ".config", formula)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "config"), []byte("default=1\n"), 0o644)
	}}
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	session.SetInstall(installSettings([]string{"alpha"}, runner, home))
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Status != StatusDifferent || len(plan.Conflicts) != 1 {
		t.Fatalf("la configuración creada por la instalación debe clasificarse como conflicto: %+v", plan.Entries)
	}
}

func TestInstallFailureOmitsOnlyThatAppAndExitsWithThree(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha", "beta")
	runner := &fakeBrew{bin: bin, failFor: map[string]bool{"beta": true}}
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	session.SetInstall(installSettings([]string{"alpha", "beta"}, runner, home))
	plan, err := session.Plan(context.Background(), nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Apply(context.Background(), conflicts.ConflictDecisions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Installs) != 2 || plan.Installs[1].Status != InstallFailed {
		t.Fatalf("beta debe quedar como instalación fallida: %+v", plan.Installs)
	}
	if len(result.Implanted) != 1 {
		t.Fatalf("alpha debe desplegarse aunque beta falle: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(home, ".config/beta/config")); !os.IsNotExist(err) {
		t.Fatal("la app fallida no debe recibir configuración")
	}
	if ResultCode(result) != 3 {
		t.Fatalf("una instalación fallida debe dar código 3, got %d", ResultCode(result))
	}
}

func TestInstallNeverRunsWhenArchiveIsInvalid(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha")
	runner := &fakeBrew{bin: bin}
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	session.Manifest.Apps[0].Files[0].SHA256 = ptr("0000000000000000000000000000000000000000000000000000000000000000")
	session.SetInstall(installSettings([]string{"alpha"}, runner, home))
	if _, err := session.Plan(context.Background(), nil, false, nil, nil); err == nil {
		t.Fatal("un archivo que no supera la verificación debe fallar")
	}
	if runner.calls != 0 {
		t.Fatalf("archivo inválido: no debe instalarse nada, got %d llamadas", runner.calls)
	}
}

func TestInstallDryRunOnlyPlans(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha")
	runner := &fakeBrew{bin: bin}
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	session.SetInstall(installSettings([]string{"alpha"}, runner, home))
	plan, err := session.Plan(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Installs) != 1 || plan.Installs[0].Status != InstallPlanned || plan.Installs[0].Method != install.MethodBrew {
		t.Fatalf("la simulación debe informar qué se instalaría: %+v", plan.Installs)
	}
	if runner.calls != 0 {
		t.Fatalf("la simulación no debe llamar al ejecutor, got %d", runner.calls)
	}
	if _, err := os.Stat(filepath.Join(bin, "kamino-alpha")); !os.IsNotExist(err) {
		t.Fatal("la simulación no debe instalar nada")
	}
}

func TestInstallIgnoresRecipesFromTheClone(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha")
	runner := &fakeBrew{bin: bin}
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	// The clone claims a different source: only the catalog recipe may be used.
	session.Manifest.Apps[0].Name = "https://evil.example/install.sh"
	session.SetInstall(installSettings([]string{"alpha"}, runner, home))
	plan, err := session.Plan(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Installs) != 1 || plan.Installs[0].URL != "" || plan.Installs[0].Method != install.MethodBrew {
		t.Fatalf("la receta debe venir solo del catálogo: %+v", plan.Installs)
	}
}

func TestInstallCandidatesOnlyListMissingAppsWithRecipes(t *testing.T) {
	cloneFile, bin, home, definitions := installFixture(t, "alpha")
	session, err := PrepareImplant(cloneFile, Options{Home: home, Catalog: definitions, Path: bin, SnapshotsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	candidates := session.InstallCandidates("darwin", "arm64", &fakeBrew{})
	if len(candidates) != 1 || candidates[0].AppID != "alpha" {
		t.Fatalf("alpha debe ofrecerse: %+v", candidates)
	}
	if len(session.InstallCandidates("darwin", "amd64", &fakeBrew{})) != 1 {
		t.Fatal("la receta sin os/arch aplica en cualquier plataforma")
	}
}
