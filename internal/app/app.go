// Package app contains the shared command flows without output.
package app

import (
	"os"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/detect"
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
func ExecuteClone(requested *[]string, output string, dryRun, force bool, definitions catalog.Catalog, home string, applicationDirs []string, pathValue string) (CloneOutcome, error) {
	if definitions == nil {
		root := home
		if root == "" {
			root, _ = userHome()
		}
		loaded, err := catalog.LoadCatalog(defaults.CatalogDir(root))
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
	result, err := clone.RunClone(clone.PlanOptions{
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

// ExecuteDeploy runs the text deploy flow.
func ExecuteDeploy(cloneFile string, requested *[]string, dryRun bool, definitions catalog.Catalog, home string, applicationDirs []string, pathValue, snapshotsDir string) (DeployOutcome, error) {
	session, err := deploy.PrepareDeploy(cloneFile, deploy.Options{
		Home:            home,
		Catalog:         definitions,
		ApplicationDirs: applicationDirs,
		Path:            pathValue,
		SnapshotsDir:    snapshotsDir,
	})
	if err != nil {
		return DeployOutcome{}, err
	}
	defer session.Close()
	deployable := map[string]bool{}
	for _, item := range session.DeployableApps {
		deployable[item.ID] = true
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
	plan, err := session.Plan(selected, dryRun, ptrValue(requested), nil, nil)
	if err != nil {
		return DeployOutcome{}, err
	}
	var result deploy.DeployResult
	if dryRun {
		result = session.DryRunResult(plan)
	} else {
		result, err = session.Apply(conflicts.ApplyAll(plan.Conflicts), nil, nil)
		if err != nil {
			return DeployOutcome{}, err
		}
	}
	return DeployOutcome{Result: result, Code: deploy.ResultCode(result), Missing: missing}, nil
}

// ExecuteUpdate runs the text update flow.
func ExecuteUpdate(currentVersion string, target string) (update.Result, error) {
	return update.RunUpdate(currentVersion, target, nil, nil, nil)
}

// DeployableIDs returns the ids the manifest allows deploying here.
func DeployableIDs(session *deploy.DeploySession) map[string]bool {
	result := map[string]bool{}
	for _, item := range session.DeployableApps {
		result[item.ID] = true
	}
	return result
}

func ptrValue(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}

func userHome() (string, error) {
	return os.UserHomeDir()
}
