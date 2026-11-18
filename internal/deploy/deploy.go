// Package deploy plans and applies a clone deployment with snapshots.
package deploy

import (
	"context"
	"os"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/progress"
)

// DeployStatus classifies a manifest file against its destination.
type DeployStatus string

const (
	StatusNew         DeployStatus = "new"
	StatusIdentical   DeployStatus = "identical"
	StatusDifferent   DeployStatus = "different"
	StatusUnsupported DeployStatus = "unsupported"
)

// ResultCategory is the final outcome of an active manifest file.
type ResultCategory string

const (
	CategoryDeployed  ResultCategory = "deployed"
	CategoryMerged    ResultCategory = "merged"
	CategoryOmitted   ResultCategory = "omitted"
	CategoryUnchanged ResultCategory = "unchanged"
)

// DeployEntry is a manifest file classified for deployment.
type DeployEntry struct {
	AppID       string
	Path        string
	Destination string
	Size        int64
	SHA256      *string
	Mode        string
	Link        *string
	Status      DeployStatus
}

// DeployPlan is the deploy plan.
type DeployPlan struct {
	Entries          []DeployEntry
	Notices          []string
	Conflicts        []conflicts.Conflict
	Omitted          []string
	Unchanged        []string
	NotInstalled     []string
	MissingRequested []string
}

// DeployResult is the result of a deploy.
type DeployResult struct {
	Plan             *DeployPlan
	Manifest         *archive.Manifest
	Output           string
	DryRun           bool
	Snapshot         string
	Deployed         []string
	Merged           []string
	Omitted          []string
	Unchanged        []string
	NotInstalled     []string
	MissingRequested []string
	SnapshotFiles    int
}

// DeployClassification is the prospective result of applying decisions.
type DeployClassification struct {
	Deployed         []string
	Merged           []string
	Omitted          []string
	Unchanged        []string
	NotInstalled     []string
	MissingRequested []string
}

// ClassificationCode returns 3 only for omitted files or unmet requests.
func ClassificationCode(classification DeployClassification) int {
	if len(classification.Omitted) > 0 || len(classification.MissingRequested) > 0 {
		return 3
	}
	return 0
}

// ResultCode returns the exit code of a finished deploy.
func ResultCode(result DeployResult) int {
	if len(result.Omitted) > 0 || len(result.MissingRequested) > 0 {
		return 3
	}
	return 0
}

// Options controls PrepareDeploy.
type Options struct {
	Home            string
	Catalog         catalog.Catalog
	CatalogDir      string
	ApplicationDirs []string
	Path            string
	SnapshotsDir    string
	// SnapshotsLimit is how many snapshots survive the next deploy; zero keeps all.
	SnapshotsLimit int
}

// DeploySession is a prepared deploy with plan, preview and apply steps.
type DeploySession struct {
	CloneFile        string
	Home             string
	Manifest         *archive.Manifest
	DeployableApps   []archive.AppEntry
	NotInstalledApps []archive.AppEntry
	SnapshotsDir     string
	SnapshotsLimit   int
	Notices          []string
	Warnings         []string
	staging          string
	plan             *DeployPlan
	fingerprints     map[string]fingerprint
	closed           bool
}

// SnapshotDirectory is where the previous-state snapshot will be written.
func (s *DeploySession) SnapshotDirectory() (string, error) {
	if s.SnapshotsDir != "" {
		return s.SnapshotsDir, nil
	}
	return defaults.SnapshotsDir(s.Home)
}

// PrepareDeploy reads the manifest and splits deployable and missing apps.
func PrepareDeploy(cloneFile string, options Options) (*DeploySession, error) {
	root, err := paths.ResolveHome(options.Home)
	if err != nil {
		return nil, err
	}
	definitions := options.Catalog
	if definitions == nil {
		directory := options.CatalogDir
		if directory == "" {
			directory, err = defaults.CatalogDir(root)
			if err != nil {
				return nil, err
			}
		}
		loaded, err := catalog.LoadCatalog(directory)
		if err != nil {
			return nil, err
		}
		definitions = loaded
	}
	manifest, err := archive.ReadManifest(cloneFile)
	if err != nil {
		return nil, err
	}
	deployable := make([]archive.AppEntry, 0)
	missing := make([]archive.AppEntry, 0)
	for _, app := range manifest.Apps {
		if isAppInstalled(app, definitions, options.ApplicationDirs, options.Path, root) {
			deployable = append(deployable, app)
		} else {
			missing = append(missing, app)
		}
	}
	session := &DeploySession{
		CloneFile:        cloneFile,
		Home:             root,
		Manifest:         manifest,
		DeployableApps:   deployable,
		NotInstalledApps: missing,
		SnapshotsDir:     options.SnapshotsDir,
		SnapshotsLimit:   options.SnapshotsLimit,
		fingerprints:     map[string]fingerprint{},
	}
	session.Notices = ghosttyPortabilityNotices(manifest, cloneFile, root)
	session.Warnings = platformWarnings(manifest)
	return session, nil
}

// Close removes the staging directory.
func (s *DeploySession) Close() {
	s.clearStaging()
	s.closed = true
}

func (s *DeploySession) clearStaging() {
	if s.staging != "" {
		_ = os.RemoveAll(s.staging)
		s.staging = ""
	}
}

func (s *DeploySession) resetStaging(ctx context.Context, progressCallback progress.Callback) error {
	if s.staging != "" {
		_ = os.RemoveAll(s.staging)
	}
	directory, err := os.MkdirTemp("", "kamino-deploy-")
	if err != nil {
		return err
	}
	s.staging = directory
	if err := archive.ExtractArchive(ctx, s.CloneFile, s.staging, progressCallback); err != nil {
		_ = os.RemoveAll(s.staging)
		s.staging = ""
		return err
	}
	return nil
}

// RunDeploy is the non-interactive deploy facade.
func RunDeploy(ctx context.Context, cloneFile string, options Options, selected *[]string, dryRun bool) (DeployResult, error) {
	session, err := PrepareDeploy(cloneFile, options)
	if err != nil {
		return DeployResult{}, err
	}
	defer session.Close()
	plan, err := session.Plan(ctx, selected, dryRun, nil, nil)
	if err != nil {
		return DeployResult{}, err
	}
	if dryRun {
		return session.DryRunResult(plan), nil
	}
	return session.Apply(ctx, conflicts.ApplyAll(plan.Conflicts), nil)
}
