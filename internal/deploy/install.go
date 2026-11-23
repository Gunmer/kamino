package deploy

import (
	"context"
	"errors"
	"fmt"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/install"
	"github.com/Gunmer/kamino/internal/progress"
)

// Install outcome states.
const (
	// InstallPlanned marks an app that a dry run would install. Nothing runs.
	InstallPlanned = "planned"
	// InstallDone marks an app installed and detected during this deploy.
	InstallDone = "installed"
	// InstallFailed marks an app whose installation failed. It stays omitted.
	InstallFailed = "failed"
)

// InstallOutcome describes what happened to one missing app.
type InstallOutcome struct {
	AppID  string
	Method string
	// URL is the exact address shown before an installer runs.
	URL    string
	Status string
	Reason string
}

// InstallSettings selects the missing apps to install during Plan, and the
// installer that runs them. A nil value leaves every missing app alone.
type InstallSettings struct {
	AppIDs    []string
	Installer *install.Installer
	OS        string
	Arch      string
}

// InstallCandidate is a missing app that has a recipe for this platform.
type InstallCandidate struct {
	AppID  string
	Name   string
	Recipe install.Recipe
}

// InstallCandidates lists the missing apps that may be offered for installation
// on the given platform, in manifest order. Clone data never adds a recipe: the
// recipe always comes from the catalog definition.
func (s *DeploySession) InstallCandidates(goos, goarch string) []InstallCandidate {
	candidates := make([]InstallCandidate, 0)
	for _, app := range s.NotInstalledApps {
		definition := s.definitions[app.ID]
		if definition == nil || len(definition.Install) == 0 {
			continue
		}
		recipe, err := install.Applicable(definition.Install, goos, goarch)
		if err != nil {
			continue
		}
		candidates = append(candidates, InstallCandidate{AppID: app.ID, Name: definition.Name, Recipe: recipe})
	}
	return candidates
}

// SetInstall selects which missing apps Plan installs. It must be called before
// Plan; without it Plan installs nothing.
func (s *DeploySession) SetInstall(settings InstallSettings) {
	s.install = &settings
}

// installMissing runs the chosen installations after the archive is verified
// and before classification and snapshot. Installed apps move to the deployable
// set when their detectors confirm them; failures stay omitted.
func (s *DeploySession) installMissing(ctx context.Context, dryRun bool) ([]InstallOutcome, error) {
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
			s.DeployableApps = append(s.DeployableApps, app)
		} else {
			remaining = append(remaining, app)
		}
	}
	s.NotInstalledApps = remaining
	return outcomes, nil
}

func (s *DeploySession) installApp(ctx context.Context, app archive.AppEntry, dryRun bool) (InstallOutcome, bool, error) {
	outcome := InstallOutcome{AppID: app.ID}
	definition := s.definitions[app.ID]
	if definition == nil || len(definition.Install) == 0 {
		outcome.Status, outcome.Reason = InstallFailed, "sin receta de instalación"
		return outcome, false, nil
	}
	if dryRun {
		recipe, err := install.Applicable(definition.Install, s.install.OS, s.install.Arch)
		if err != nil {
			outcome.Status, outcome.Reason = InstallFailed, "ninguna receta aplicable a este Mac"
			return outcome, false, nil
		}
		outcome.Method, outcome.URL, outcome.Status = recipe.Method, recipe.URL, InstallPlanned
		return outcome, false, nil
	}
	if err := progress.EnsureNotCancelled(ctx); err != nil {
		return outcome, false, err
	}
	recipe, err := install.Select(definition.Install, s.install.OS, s.install.Arch, s.install.Installer.Runner)
	if err == nil {
		outcome.Method, outcome.URL = recipe.Method, recipe.URL
		err = s.install.Installer.Install(ctx, definition.Install, s.install.OS, s.install.Arch)
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

func installReason(err error) string {
	if errors.Is(err, install.ErrNoRecipe) {
		return "ninguna receta aplicable: falta la herramienta del método"
	}
	return fmt.Sprintf("la instalación falló: %v", err)
}
