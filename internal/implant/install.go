package implant

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/install"
	"github.com/Gunmer/kamino/internal/progress"
)

// Install outcome states.
const (
	// InstallPlanned marks an app that a dry run would install. Nothing runs.
	InstallPlanned = "planned"
	// InstallDone marks an app installed and detected during this implant.
	InstallDone = "installed"
	// InstallFailed marks an app whose installation failed. It stays omitted.
	InstallFailed = "failed"
)

// InstallOutcome describes what happened to one missing app.
type InstallOutcome struct {
	AppID  string
	Method string
	// URL is the address shown before the recipe downloads anything.
	URL string
	// Unverified is true when the download has no checksum to verify it.
	Unverified bool
	Status     string
	Reason     string
}

// InstallSettings selects the missing apps to install during Plan, and the
// installer that runs them. A nil value leaves every missing app alone.
type InstallSettings struct {
	AppIDs    []string
	Installer *install.Installer
	OS        string
	Arch      string
	// Chosen overrides the default recipe of an app with another runnable
	// recipe of its catalog definition. Anything else is rejected.
	Chosen map[string]install.Recipe
}

// InstallCandidate is a missing app that has a recipe for this platform.
type InstallCandidate struct {
	AppID string
	Name  string
	// Recipe is the default one: the first runnable recipe of the definition.
	Recipe install.Recipe
	// Recipes lists every runnable recipe in catalog order, the default first.
	Recipes []install.Recipe
}

// InstallCandidates lists the missing apps that may be offered for installation
// on the given platform, in manifest order. Clone data never adds a recipe: the
// recipe always comes from the catalog definition. The recipe is the one that
// would run: the first whose tool (brew, npm) is on the PATH, so an app with no
// runnable recipe is not offered.
func (s *ImplantSession) InstallCandidates(goos, goarch string, runner install.Runner) []InstallCandidate {
	candidates := make([]InstallCandidate, 0)
	for _, app := range s.NotInstalledApps {
		definition := s.definitions[app.ID]
		if definition == nil || len(definition.Install) == 0 {
			continue
		}
		recipes := install.Runnable(definition.Install, goos, goarch, runner)
		if len(recipes) == 0 {
			continue
		}
		candidates = append(candidates, InstallCandidate{AppID: app.ID, Name: definition.Name, Recipe: recipes[0], Recipes: recipes})
	}
	return candidates
}

// SetInstall selects which missing apps Plan installs. It must be called before
// Plan; without it Plan installs nothing.
func (s *ImplantSession) SetInstall(settings InstallSettings) {
	s.install = &settings
}

// installMissing runs the chosen installations after the archive is verified
// and before classification and snapshot. Installed apps move to the implantable
// set when their detectors confirm them; failures stay omitted.
func (s *ImplantSession) installMissing(ctx context.Context, dryRun bool) ([]InstallOutcome, error) {
	if s.install == nil || len(s.install.AppIDs) == 0 {
		return nil, nil
	}
	if s.install.Installer == nil && !dryRun {
		return nil, fmt.Errorf("falta el instalador para las apps seleccionadas")
	}
	chosen := map[string]bool{}
	for _, id := range s.install.AppIDs {
		chosen[id] = true
	}
	outcomes := make([]InstallOutcome, 0)
	remaining := make([]archive.AppEntry, 0, len(s.NotInstalledApps))
	for _, app := range s.NotInstalledApps {
		if !chosen[app.ID] {
			remaining = append(remaining, app)
			continue
		}
		outcome, installed, err := s.installApp(ctx, app, dryRun)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, outcome)
		if installed {
			s.ImplantableApps = append(s.ImplantableApps, app)
		} else {
			remaining = append(remaining, app)
		}
	}
	s.NotInstalledApps = remaining
	return outcomes, nil
}

func (s *ImplantSession) installApp(ctx context.Context, app archive.AppEntry, dryRun bool) (InstallOutcome, bool, error) {
	outcome := InstallOutcome{AppID: app.ID}
	definition := s.definitions[app.ID]
	if definition == nil || len(definition.Install) == 0 {
		outcome.Status, outcome.Reason = InstallFailed, "sin receta de instalación"
		return outcome, false, nil
	}
	var runner install.Runner = install.ExecRunner{}
	if s.install.Installer != nil {
		runner = s.install.Installer.Runner
	}
	recipe, err := s.recipeFor(app.ID, definition, runner)
	if dryRun {
		if err != nil {
			outcome.Status, outcome.Reason = InstallFailed, "ninguna receta aplicable a este Mac"
			return outcome, false, nil
		}
		outcome.Method, outcome.URL, outcome.Unverified, outcome.Status = recipe.Method, recipe.Source(), recipe.Unverified(), InstallPlanned
		return outcome, false, nil
	}
	if ctxErr := progress.EnsureNotCancelled(ctx); ctxErr != nil {
		return outcome, false, ctxErr
	}
	if err == nil {
		outcome.Method, outcome.URL, outcome.Unverified = recipe.Method, recipe.Source(), recipe.Unverified()
		err = s.install.Installer.InstallRecipe(ctx, recipe)
	}
	if err != nil {
		if ctxErr := progress.EnsureNotCancelled(ctx); ctxErr != nil {
			return outcome, false, ctxErr
		}
		outcome.Status, outcome.Reason = InstallFailed, installReason(err)
		return outcome, false, nil
	}
	if !isAppInstalled(app, s.definitions, s.applicationDirs, s.pathValue, s.Home) {
		outcome.Status, outcome.Reason = InstallFailed, "instalada pero no detectada tras la instalación"
		return outcome, false, nil
	}
	outcome.Status = InstallDone
	return outcome, true, nil
}

// recipeFor returns the recipe to run for an app: the user's choice when it is
// one of the definition's runnable recipes, otherwise the default. A choice that
// is not in the catalog definition is an error, never a recipe to run.
func (s *ImplantSession) recipeFor(id string, definition *catalog.AppDefinition, runner install.Runner) (install.Recipe, error) {
	runnable := install.Runnable(definition.Install, s.install.OS, s.install.Arch, runner)
	if len(runnable) == 0 {
		return install.Recipe{}, install.ErrNoRecipe
	}
	chosen, ok := s.install.Chosen[id]
	if !ok {
		return runnable[0], nil
	}
	for _, recipe := range runnable {
		if reflect.DeepEqual(recipe, chosen) {
			return recipe, nil
		}
	}
	return install.Recipe{}, fmt.Errorf("la receta elegida no pertenece a la definición del catálogo")
}

func installReason(err error) string {
	if errors.Is(err, install.ErrNoRecipe) {
		return "ninguna receta aplicable: falta la herramienta del método"
	}
	return fmt.Sprintf("la instalación falló: %v", err)
}
