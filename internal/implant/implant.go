// Package implant plans and applies a clone implantation with snapshots.
package implant

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

// ImplantStatus classifies a manifest file against its destination.
type ImplantStatus string

const (
	StatusNew         ImplantStatus = "new"
	StatusIdentical   ImplantStatus = "identical"
	StatusDifferent   ImplantStatus = "different"
	StatusUnsupported ImplantStatus = "unsupported"
)

// ResultCategory is the final outcome of an active manifest file.
type ResultCategory string

const (
	CategoryImplanted ResultCategory = "implanted"
	CategoryMerged    ResultCategory = "merged"
	CategoryOmitted   ResultCategory = "omitted"
	CategoryUnchanged ResultCategory = "unchanged"
)

// ImplantEntry is a manifest file classified for implantation.
type ImplantEntry struct {
	AppID       string
	Path        string
	Destination string
	Size        int64
	SHA256      *string
	Mode        string
	Link        *string
	Status      ImplantStatus
}

// ImplantPlan is the implant plan.
type ImplantPlan struct {
	Entries          []ImplantEntry
	Notices          []string
	Conflicts        []conflicts.Conflict
	Omitted          []string
	Unchanged        []string
	NotInstalled     []string
	MissingRequested []string
	Installs         []InstallOutcome
}

// ImplantResult is the result of a implant.
type ImplantResult struct {
	Plan             *ImplantPlan
	Manifest         *archive.Manifest
	Output           string
	DryRun           bool
	Snapshot         string
	Implanted        []string
	Merged           []string
	Omitted          []string
	Unchanged        []string
	NotInstalled     []string
	MissingRequested []string
	Installs         []InstallOutcome
	SnapshotFiles    int
}

// ImplantClassification is the prospective result of applying decisions.
type ImplantClassification struct {
	Implanted        []string
	Merged           []string
	Omitted          []string
	Unchanged        []string
	NotInstalled     []string
	MissingRequested []string
}

// ResultCode returns the exit code of a finished implant.
func ResultCode(result ImplantResult) int {
	if len(result.Omitted) > 0 || len(result.MissingRequested) > 0 || hasFailedInstall(result.Installs) {
		return 3
	}
	return 0
}

func hasFailedInstall(outcomes []InstallOutcome) bool {
	for _, outcome := range outcomes {
		if outcome.Status == InstallFailed {
			return true
		}
	}
	return false
}

// Options controls PrepareImplant.
type Options struct {
	Home            string
	Catalog         catalog.Catalog
	CatalogDir      string
	ApplicationDirs []string
	Path            string
	SnapshotsDir    string
	// SnapshotsLimit is how many snapshots survive the next implant; zero keeps all.
	SnapshotsLimit int
}

// ImplantSession is a prepared implant with plan, preview and apply steps.
type ImplantSession struct {
	CloneFile        string
	Home             string
	Manifest         *archive.Manifest
	ImplantableApps  []archive.AppEntry
	NotInstalledApps []archive.AppEntry
	SnapshotsDir     string
	SnapshotsLimit   int
	Notices          []string
	Warnings         []string
	staging          string
	plan             *ImplantPlan
	fingerprints     map[string]fingerprint
	closed           bool
	definitions      catalog.Catalog
	applicationDirs  []string
	pathValue        string
	install          *InstallSettings
}

// SnapshotDirectory is where the previous-state snapshot will be written.
func (s *ImplantSession) SnapshotDirectory() (string, error) {
	if s.SnapshotsDir != "" {
		return s.SnapshotsDir, nil
	}
	return defaults.SnapshotsDir(s.Home)
}

// PrepareImplant reads the manifest and splits implantable and missing apps.
func PrepareImplant(cloneFile string, options Options) (*ImplantSession, error) {
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
	implantable := make([]archive.AppEntry, 0)
	missing := make([]archive.AppEntry, 0)
	for _, app := range manifest.Apps {
		if isAppInstalled(app, definitions, options.ApplicationDirs, options.Path, root) {
			implantable = append(implantable, app)
		} else {
			missing = append(missing, app)
		}
	}
	session := &ImplantSession{
		CloneFile:        cloneFile,
		Home:             root,
		Manifest:         manifest,
		ImplantableApps:  implantable,
		NotInstalledApps: missing,
		SnapshotsDir:     options.SnapshotsDir,
		SnapshotsLimit:   options.SnapshotsLimit,
		fingerprints:     map[string]fingerprint{},
		definitions:      definitions,
		applicationDirs:  options.ApplicationDirs,
		pathValue:        options.Path,
	}
	session.Notices = ghosttyPortabilityNotices(manifest, cloneFile, root)
	session.Warnings = platformWarnings(manifest)
	return session, nil
}

// Close removes the staging directory.
func (s *ImplantSession) Close() {
	s.clearStaging()
	s.closed = true
}

func (s *ImplantSession) clearStaging() {
	if s.staging != "" {
		_ = os.RemoveAll(s.staging)
		s.staging = ""
	}
}

func (s *ImplantSession) resetStaging(ctx context.Context, progressCallback progress.Callback) error {
	if s.staging != "" {
		_ = os.RemoveAll(s.staging)
	}
	directory, err := os.MkdirTemp("", "kamino-implant-")
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
