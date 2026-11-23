// Package app contains the shared command flows without output.
package app

import (
	"context"
	"runtime"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/detect"
	"github.com/Gunmer/kamino/internal/install"
	"github.com/Gunmer/kamino/internal/update"
)

// CloneOutcome is the result of a text clone.
type CloneOutcome struct {
	Result  clone.CloneResult
	Code    int
	Missing []string
}

// DeployOutcome is the result of a text deploy.
type DeployOutcome struct {
	Result  deploy.DeployResult
	Code    int
	Missing []string
}

// InstalledApps returns the installed catalog apps sorted by id.
func InstalledApps(definitions catalog.Catalog, home string, applicationDirs []string, pathValue string) []*catalog.AppDefinition {
	result := make([]*catalog.AppDefinition, 0)
	for _, id := range definitions.SortedIDs() {
		definition := definitions[id]
		if detect.IsInstalled(definition, detect.Options{Home: home, ApplicationDirs: applicationDirs, Path: pathValue}) {
			result = append(result, definition)
		}
	}
	return result
}

// ExecuteClone runs the text clone flow.
func ExecuteClone(ctx context.Context, requested *[]string, output string, dryRun, force bool, definitions catalog.Catalog, home string, applicationDirs []string, pathValue string) (CloneOutcome, error) {
	if definitions == nil {
		directory, err := CatalogDir(home)
		if err != nil {
			return CloneOutcome{}, err
		}
		loaded, err := catalog.LoadCatalog(directory)
		if err != nil {
			return CloneOutcome{}, err
		}
		definitions = loaded
	}
	installed := InstalledApps(definitions, home, applicationDirs, pathValue)
	selected := make([]string, 0)
	missing := make([]string, 0)
	if requested == nil {
		for _, item := range installed {
			selected = append(selected, item.ID)
		}
	} else {
		available := map[string]bool{}
		for _, item := range installed {
			available[item.ID] = true
		}
		for _, id := range *requested {
			if available[id] {
				selected = append(selected, id)
			} else {
				missing = append(missing, id)
			}
		}
	}
	result, err := clone.RunClone(ctx, clone.PlanOptions{
		Home:            home,
		Selected:        &selected,
		Output:          output,
		Catalog:         definitions,
		ApplicationDirs: applicationDirs,
		Path:            pathValue,
	}, dryRun, force)
	if err != nil {
		return CloneOutcome{}, err
	}
	code := 0
	if len(result.Plan.Skipped) > 0 || len(missing) > 0 {
		code = 3
	}
	return CloneOutcome{Result: result, Code: code, Missing: missing}, nil
}

// ExecuteDeploy runs the text deploy flow. With installMissing, the missing
// apps that have a recipe for this platform are installed before classification,
// restricted to the requested ids when any were given.
func ExecuteDeploy(ctx context.Context, cloneFile string, requested *[]string, dryRun bool, definitions catalog.Catalog, home string, applicationDirs []string, pathValue, snapshotsDir string, installMissing bool) (DeployOutcome, error) {
	directory, err := CatalogDir(home)
	if err != nil {
		return DeployOutcome{}, err
	}
	session, err := deploy.PrepareDeploy(cloneFile, deploy.Options{
		Home:            home,
		Catalog:         definitions,
		CatalogDir:      directory,
		ApplicationDirs: applicationDirs,
		Path:            pathValue,
		SnapshotsDir:    snapshotsDir,
		SnapshotsLimit:  snapshotsLimit(home),
	})
	if err != nil {
		return DeployOutcome{}, err
	}
	defer session.Close()
	deployable := map[string]bool{}
	for _, item := range session.DeployableApps {
		deployable[item.ID] = true
	}
	if installMissing {
		installIDs := make([]string, 0)
		for _, candidate := range session.InstallCandidates(runtime.GOOS, runtime.GOARCH) {
			if requested != nil && !containsID(*requested, candidate.AppID) {
				continue
			}
			installIDs = append(installIDs, candidate.AppID)
			deployable[candidate.AppID] = true
		}
		session.SetInstall(deploy.InstallSettings{
			AppIDs:    installIDs,
			Installer: &install.Installer{Runner: install.ExecRunner{}, Home: session.Home},
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		})
	}
	var selected *[]string
	missing := make([]string, 0)
	if requested != nil {
		chosen := make([]string, 0)
		for _, id := range *requested {
			if deployable[id] {
				chosen = append(chosen, id)
			} else {
				missing = append(missing, id)
			}
		}
		selected = &chosen
	}
	plan, err := session.Plan(ctx, selected, dryRun, ptrValue(requested), nil)
	if err != nil {
		return DeployOutcome{}, err
	}
	var result deploy.DeployResult
	if dryRun {
		result = session.DryRunResult(plan)
	} else {
		result, err = session.Apply(ctx, conflicts.ApplyAll(plan.Conflicts), nil)
		if err != nil {
			return DeployOutcome{}, err
		}
	}
	return DeployOutcome{Result: result, Code: deploy.ResultCode(result), Missing: missing}, nil
}

func containsID(ids []string, id string) bool {
	for _, value := range ids {
		if value == id {
			return true
		}
	}
	return false
}

// ExecuteUpdate runs the text update flow.
func ExecuteUpdate(ctx context.Context, currentVersion string, target string) (update.Result, error) {
	return update.RunUpdate(ctx, currentVersion, target, nil, nil)
}

// CatalogDir resolves the effective user catalog directory. A missing or
// invalid configuration keeps the built-in directory, so the CLI never fails
// because of an unrelated configuration error.
func CatalogDir(home string) (string, error) {
	directory, _, err := catalogResolution(home)
	return directory, err
}

func catalogResolution(home string) (string, config.LoadResult, error) {
	loaded := config.LoadConfig(home, "")
	if loaded.Error != nil {
		directory, err := defaults.CatalogDir(home)
		return directory, loaded, err
	}
	return loaded.Config.CatalogDir, loaded, nil
}

// snapshotsLimit resolves the configured snapshot limit for the text flows.
// A missing or invalid configuration keeps the default, so the CLI never fails
// because of an unrelated configuration error.
func snapshotsLimit(home string) int {
	loaded := config.LoadConfig(home, "")
	if loaded.Error != nil {
		return config.DefaultSnapshotsLimit
	}
	return loaded.Config.SnapshotsLimit
}

func ptrValue(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}
