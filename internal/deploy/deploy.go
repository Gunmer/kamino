// Package deploy plans and applies a clone deployment with snapshots.
package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/detect"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/ghostty"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/snapshot"
	"github.com/Gunmer/kamino/internal/sysinfo"
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

type fingerprint struct {
	kind   string
	sha256 string
	link   string
}

type resolvedFile struct {
	path        string
	destination string
	content     []byte
	link        string
	mode        string
	category    ResultCategory
	write       bool
	localText   *string
	cloneText   *string
	mergedText  *string
}

// Options controls PrepareDeploy.
type Options struct {
	Home            string
	Catalog         catalog.Catalog
	CatalogDir      string
	ApplicationDirs []string
	Path            string
	SnapshotsDir    string
}

// DeploySession is a prepared deploy with plan, preview and apply steps.
type DeploySession struct {
	CloneFile        string
	Home             string
	Manifest         *archive.Manifest
	DeployableApps   []archive.AppEntry
	NotInstalledApps []archive.AppEntry
	SnapshotsDir     string
	Notices          []string
	Warnings         []string
	staging          string
	plan             *DeployPlan
	fingerprints     map[string]fingerprint
	closed           bool
}

// SnapshotDirectory is where the previous-state snapshot will be written.
func (s *DeploySession) SnapshotDirectory() string {
	if s.SnapshotsDir != "" {
		return s.SnapshotsDir
	}
	return defaults.SnapshotsDir(s.Home)
}

// PrepareDeploy reads the manifest and splits deployable and missing apps.
func PrepareDeploy(cloneFile string, options Options) (*DeploySession, error) {
	root := options.Home
	if root == "" {
		root, _ = os.UserHomeDir()
	}
	definitions := options.Catalog
	if definitions == nil {
		directory := options.CatalogDir
		if directory == "" {
			directory = defaults.CatalogDir(root)
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
		fingerprints:     map[string]fingerprint{},
	}
	session.Notices = ghosttyPortabilityNotices(manifest, cloneFile, root)
	session.Warnings = platformWarnings(manifest)
	return session, nil
}

// Plan classifies the chosen apps and builds every content conflict.
func (s *DeploySession) Plan(selected *[]string, dryRun bool, requested []string, progressCallback progress.Callback, cancelled progress.Cancel) (*DeployPlan, error) {
	if s.closed {
		return nil, fmt.Errorf("deploy session is closed")
	}
	if !dryRun {
		if err := archive.VerifyArchive(s.CloneFile, s.Manifest, progressCallback, cancelled); err != nil {
			return nil, err
		}
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(s.Home, 0o755); err != nil {
			return nil, err
		}
		if err := s.resetStaging(progressCallback, cancelled); err != nil {
			return nil, err
		}
	}
	entries := make([]DeployEntry, 0)
	conflictList := make([]conflicts.Conflict, 0)
	unchanged := make([]string, 0)
	omitted := make([]string, 0)
	notInstalled := make([]string, 0)
	for _, app := range s.NotInstalledApps {
		for _, file := range app.Files {
			notInstalled = append(notInstalled, file.Path)
		}
	}
	deployableIDs := map[string]bool{}
	for _, app := range s.DeployableApps {
		deployableIDs[app.ID] = true
	}
	missingRequested := make([]string, 0)
	for _, appID := range requested {
		if !deployableIDs[appID] {
			missingRequested = append(missingRequested, appID)
		}
	}
	active := s.activeFiles(selected)
	total := len(active)
	seenConflicts := map[string]bool{}
	fingerprints := map[string]fingerprint{}
	var safe *paths.SafeHome
	if info, err := os.Stat(s.Home); err == nil && info.IsDir() {
		safe, _ = paths.NewSafeHome(s.Home, false)
	}
	if safe != nil {
		defer safe.Close()
	}
	for index, current := range active {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			s.clearStaging()
			return nil, err
		}
		app, file := current.app, current.file
		progress.Emit(progressCallback, progress.Classifying, index+1, total, file.Path)
		destination, err := paths.ToAbsolute(file.Path, s.Home)
		if err != nil {
			s.clearStaging()
			return nil, err
		}
		var local *conflicts.LocalFile
		if safe != nil {
			read, err := readLocal(safe, file.Path)
			if errors.Is(err, fs.ErrNotExist) {
				local = nil
			} else if err != nil {
				s.clearStaging()
				return nil, err
			} else {
				local = &read
			}
		}
		status := StatusNew
		if local != nil {
			cloneForClassification := s.cloneContent(app.ID, file, dryRun)
			status = classifyLocal(*local, file, cloneForClassification)
		}
		entries = append(entries, DeployEntry{
			AppID: app.ID, Path: file.Path, Destination: destination,
			Size: file.Size, SHA256: file.SHA256, Mode: file.Mode, Link: file.Link, Status: status,
		})
		if status == StatusIdentical {
			unchanged = append(unchanged, file.Path)
		} else if status == StatusUnsupported {
			omitted = append(omitted, file.Path)
		}
		if status == StatusDifferent && !seenConflicts[file.Path] {
			conflict := s.buildConflict(app.ID, file, destination, local, dryRun)
			if conflict != nil {
				conflictList = append(conflictList, *conflict)
				seenConflicts[file.Path] = true
			}
		}
		if (status == StatusNew || status == StatusDifferent) && safe != nil {
			if _, ok := fingerprints[file.Path]; !ok {
				fingerprints[file.Path] = fingerprintOf(safe, file.Path)
			}
		} else if status == StatusNew {
			if _, ok := fingerprints[file.Path]; !ok {
				fingerprints[file.Path] = fingerprint{kind: "missing"}
			}
		}
	}
	if err := progress.EnsureNotCancelled(cancelled); err != nil {
		s.clearStaging()
		return nil, err
	}
	changed := make([]DeployEntry, 0)
	for _, entry := range entries {
		if entry.Status != StatusUnsupported {
			changed = append(changed, entry)
		}
	}
	if len(changed) > 0 {
		if _, err := os.Stat(s.Home); err == nil {
			toValidate := make([]DeployEntry, 0)
			for _, entry := range changed {
				if entry.Status != StatusIdentical {
					toValidate = append(toValidate, entry)
				}
			}
			if err := validateDestinations(toValidate, s.Home); err != nil {
				s.clearStaging()
				return nil, err
			}
		}
	}
	plan := &DeployPlan{
		Entries:          entries,
		Notices:          s.Notices,
		Conflicts:        conflictList,
		Omitted:          dedupe(omitted),
		Unchanged:        dedupe(unchanged),
		NotInstalled:     dedupe(notInstalled),
		MissingRequested: missingRequested,
	}
	s.plan = plan
	s.fingerprints = fingerprints
	return plan, nil
}

// Preview returns the categories that applying decisions would produce.
func (s *DeploySession) Preview(decisions conflicts.ConflictDecisions) (DeployClassification, error) {
	plan, resolved, err := s.resolve(decisions)
	if err != nil {
		return DeployClassification{}, err
	}
	return classification(plan, resolved), nil
}

// DryRunResult builds the result of a simulation.
func (s *DeploySession) DryRunResult(plan *DeployPlan) DeployResult {
	return DeployResult{
		Plan:             plan,
		Manifest:         s.Manifest,
		DryRun:           true,
		Omitted:          plan.Omitted,
		Unchanged:        plan.Unchanged,
		NotInstalled:     plan.NotInstalled,
		MissingRequested: plan.MissingRequested,
	}
}

// Apply applies the decisions with a snapshot and atomic per-file writes.
func (s *DeploySession) Apply(decisions conflicts.ConflictDecisions, progressCallback progress.Callback, cancelled progress.Cancel) (DeployResult, error) {
	if s.closed {
		return DeployResult{}, fmt.Errorf("deploy session is closed")
	}
	plan, resolved, err := s.resolve(decisions)
	if err != nil {
		return DeployResult{}, err
	}
	if err := validateMerges(resolved); err != nil {
		return DeployResult{}, err
	}
	writes := make([]resolvedFile, 0)
	for _, item := range resolved {
		if item.write {
			writes = append(writes, item)
		}
	}
	if len(writes) == 0 {
		return DeployResult{
			Plan:             plan,
			Manifest:         s.Manifest,
			Output:           s.CloneFile,
			Omitted:          classification(plan, resolved).Omitted,
			Unchanged:        plan.Unchanged,
			NotInstalled:     plan.NotInstalled,
			MissingRequested: plan.MissingRequested,
		}, nil
	}
	safe, err := paths.NewSafeHome(s.Home, false)
	if err != nil {
		return DeployResult{}, err
	}
	for _, item := range writes {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			_ = safe.Close()
			return DeployResult{}, err
		}
		current := fingerprintOf(safe, item.path)
		expected, ok := s.fingerprints[item.path]
		if !ok || current != expected {
			_ = safe.Close()
			return DeployResult{}, &kerrors.ConcurrentChangeError{KaminoError: kerrors.KaminoError{
				Msg: fmt.Sprintf("el destino cambió durante la revisión: %s", item.path),
			}}
		}
	}
	_ = safe.Close()

	destinations := make([]string, 0, len(writes))
	for _, item := range writes {
		destinations = append(destinations, item.destination)
	}
	protected := snapshot.SnapshotFiles(destinations, s.Home)
	if err := progress.EnsureNotCancelled(cancelled); err != nil {
		return DeployResult{}, err
	}
	snapshotPath, err := snapshot.CreateSnapshot(destinations, s.Home, s.SnapshotsDir, "")
	if err != nil {
		return DeployResult{}, err
	}
	if err := writeResolved(writes, s.Home, snapshotPath, progressCallback); err != nil {
		return DeployResult{}, err
	}
	result := classification(plan, resolved)
	return DeployResult{
		Plan:             plan,
		Manifest:         s.Manifest,
		Output:           s.CloneFile,
		Snapshot:         snapshotPath,
		Deployed:         result.Deployed,
		Merged:           result.Merged,
		Omitted:          result.Omitted,
		Unchanged:        result.Unchanged,
		NotInstalled:     result.NotInstalled,
		MissingRequested: result.MissingRequested,
		SnapshotFiles:    protected,
	}, nil
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

func (s *DeploySession) resetStaging(progressCallback progress.Callback, cancelled progress.Cancel) error {
	if s.staging != "" {
		_ = os.RemoveAll(s.staging)
	}
	directory, err := os.MkdirTemp("", "kamino-deploy-")
	if err != nil {
		return err
	}
	s.staging = directory
	if err := archive.ExtractArchive(s.CloneFile, s.staging, progressCallback, cancelled); err != nil {
		_ = os.RemoveAll(s.staging)
		s.staging = ""
		return err
	}
	return nil
}

type activeFile struct {
	app  archive.AppEntry
	file archive.FileEntry
}

func (s *DeploySession) activeFiles(selected *[]string) []activeFile {
	active := make([]activeFile, 0)
	for _, app := range s.DeployableApps {
		if selected != nil {
			found := false
			for _, id := range *selected {
				if id == app.ID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		for _, file := range app.Files {
			active = append(active, activeFile{app: app, file: file})
		}
	}
	return active
}

func (s *DeploySession) cloneContent(appID string, file archive.FileEntry, dryRun bool) []byte {
	if file.Link != nil || file.Size > conflicts.MaxTextBytes {
		return nil
	}
	if s.staging != "" {
		source := filepath.Join(s.staging, archive.AppsDir, appID, filepath.FromSlash(file.Path))
		if info, err := os.Stat(source); err != nil || !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return nil
		}
		return data
	}
	return archive.ReadFileFromArchive(s.CloneFile, appID, file.Path, conflicts.MaxTextBytes+1)
}

func (s *DeploySession) buildConflict(appID string, file archive.FileEntry, destination string, local *conflicts.LocalFile, dryRun bool) *conflicts.Conflict {
	if local == nil {
		return nil
	}
	var cloneContent []byte
	if file.Link == nil {
		cloneContent = s.cloneContent(appID, file, dryRun)
	}
	return conflicts.BuildConflict(appID, file.Path, destination, file, *local, cloneContent)
}

func (s *DeploySession) resolve(decisions conflicts.ConflictDecisions) (*DeployPlan, []resolvedFile, error) {
	if s.plan == nil {
		return nil, nil, fmt.Errorf("call Plan() before Preview() or Apply()")
	}
	if missing := decisions.Missing(s.plan.Conflicts); len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for _, conflict := range missing {
			names = append(names, conflict.Path)
		}
		return nil, nil, &kerrors.DecisionRequiredError{KaminoError: kerrors.KaminoError{
			Msg: "faltan decisiones de conflicto: " + strings.Join(names, ", "),
		}}
	}
	byPath := map[string]conflicts.Conflict{}
	for _, conflict := range s.plan.Conflicts {
		byPath[conflict.Path] = conflict
	}
	resolved := make([]resolvedFile, 0)
	done := map[string]bool{}
	for _, entry := range s.plan.Entries {
		if done[entry.Path] {
			continue
		}
		done[entry.Path] = true
		if entry.Status == StatusIdentical || entry.Status == StatusUnsupported {
			category := CategoryUnchanged
			if entry.Status == StatusUnsupported {
				category = CategoryOmitted
			}
			resolved = append(resolved, resolvedFile{
				path: entry.Path, destination: entry.Destination, mode: entry.Mode, category: category,
			})
			continue
		}
		conflict, hasConflict := byPath[entry.Path]
		if !hasConflict {
			content, link, err := s.cloneOutput(entry)
			if err != nil {
				return nil, nil, err
			}
			resolved = append(resolved, resolvedFile{
				path: entry.Path, destination: entry.Destination, content: content, link: link,
				mode: entry.Mode, category: CategoryDeployed, write: true,
			})
			continue
		}
		decision, ok := decisions.Get(entry.Path)
		if !ok {
			return nil, nil, &kerrors.DecisionRequiredError{KaminoError: kerrors.KaminoError{
				Msg: "falta la decisión de " + entry.Path,
			}}
		}
		item, err := s.resolveConflict(entry, conflict, decision)
		if err != nil {
			return nil, nil, err
		}
		resolved = append(resolved, item)
	}
	return s.plan, resolved, nil
}

func (s *DeploySession) resolveConflict(entry DeployEntry, conflict conflicts.Conflict, decision conflicts.FileDecision) (resolvedFile, error) {
	if decision.Action == conflicts.ActionOmit {
		return resolvedFile{path: entry.Path, destination: entry.Destination, mode: entry.Mode, category: CategoryOmitted}, nil
	}
	if decision.Action == conflicts.ActionApply {
		content, link, err := s.cloneOutput(entry)
		if err != nil {
			return resolvedFile{}, err
		}
		return resolvedFile{path: entry.Path, destination: entry.Destination, content: content, link: link, mode: entry.Mode, category: CategoryDeployed, write: true}, nil
	}
	if conflict.LocalText == nil || conflict.CloneText == nil {
		return resolvedFile{}, &kerrors.ConflictError{KaminoError: kerrors.KaminoError{
			Msg: entry.Path + " no admite revisión por líneas",
		}}
	}
	if len(decision.Rows) != conflicts.RowCount(conflict) {
		return resolvedFile{}, &kerrors.ConflictError{KaminoError: kerrors.KaminoError{
			Msg: "la decisión de " + entry.Path + " no cubre todas las líneas",
		}}
	}
	merged, err := conflicts.Compose(*conflict.LocalText, *conflict.CloneText, decision.Rows)
	if err != nil {
		return resolvedFile{}, err
	}
	if merged == *conflict.LocalText {
		return resolvedFile{
			path: entry.Path, destination: entry.Destination, mode: entry.Mode, category: CategoryOmitted,
			localText: conflict.LocalText, cloneText: conflict.CloneText, mergedText: &merged,
		}, nil
	}
	allClone := true
	for _, row := range decision.Rows {
		if row != conflicts.RowClone {
			allClone = false
			break
		}
	}
	category := CategoryMerged
	if allClone {
		category = CategoryDeployed
	}
	content := []byte(merged)
	return resolvedFile{
		path: entry.Path, destination: entry.Destination, content: content, mode: entry.Mode,
		category: category, write: true, localText: conflict.LocalText, cloneText: conflict.CloneText, mergedText: &merged,
	}, nil
}

func (s *DeploySession) cloneOutput(entry DeployEntry) ([]byte, string, error) {
	if entry.Link != nil {
		return nil, *entry.Link, nil
	}
	if s.staging == "" {
		return nil, "", fmt.Errorf("the archive is not extracted")
	}
	source := filepath.Join(s.staging, archive.AppsDir, entry.AppID, filepath.FromSlash(entry.Path))
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, "", err
	}
	return data, "", nil
}

func readLocal(safe *paths.SafeHome, rel string) (conflicts.LocalFile, error) {
	info, err := safe.Lstat(rel)
	if err != nil {
		return conflicts.LocalFile{}, err
	}
	mode := info.ModeOctal()
	switch info.Kind {
	case paths.KindLink:
		target, err := safe.Readlink(rel)
		if err != nil {
			return conflicts.LocalFile{}, err
		}
		return conflicts.LocalFile{Kind: "symlink", Link: target, Mode: mode}, nil
	case paths.KindFile:
		data, err := safe.ReadBytes(rel)
		if err != nil {
			return conflicts.LocalFile{}, err
		}
		return conflicts.LocalFile{Kind: "file", Data: data, Mode: mode}, nil
	default:
		return conflicts.LocalFile{Kind: "special", Mode: mode}, nil
	}
}

func classifyLocal(local conflicts.LocalFile, entry archive.FileEntry, cloneContent []byte) DeployStatus {
	if local.Kind == "special" {
		return StatusUnsupported
	}
	if local.Kind == "symlink" {
		if entry.Link == nil {
			return StatusDifferent
		}
		if local.Link == *entry.Link {
			return StatusIdentical
		}
		return StatusDifferent
	}
	if entry.Link != nil {
		return StatusDifferent
	}
	if entry.SHA256 != nil {
		digest := sha256.Sum256(local.Data)
		if hex.EncodeToString(digest[:]) == *entry.SHA256 {
			return StatusIdentical
		}
	}
	if cloneContent != nil && textEqual(local.Data, cloneContent) {
		return StatusIdentical
	}
	return StatusDifferent
}

func textEqual(local, clone []byte) bool {
	if strings.ContainsRune(string(local), 0) || strings.ContainsRune(string(clone), 0) {
		return false
	}
	if !validUTF8(local) || !validUTF8(clone) {
		return false
	}
	return strings.ReplaceAll(string(local), "\r\n", "\n") == strings.ReplaceAll(string(clone), "\r\n", "\n")
}

func validUTF8(data []byte) bool {
	return strings.ToValidUTF8(string(data), "") == string(data)
}

func fingerprintOf(safe *paths.SafeHome, rel string) fingerprint {
	info, err := safe.Lstat(rel)
	if err != nil {
		return fingerprint{kind: "missing"}
	}
	switch info.Kind {
	case paths.KindLink:
		target, err := safe.Readlink(rel)
		if err != nil {
			return fingerprint{kind: "special"}
		}
		return fingerprint{kind: "symlink", link: target}
	case paths.KindFile:
		data, err := safe.ReadBytes(rel)
		if err != nil {
			return fingerprint{kind: "special"}
		}
		digest := sha256.Sum256(data)
		return fingerprint{kind: "file", sha256: hex.EncodeToString(digest[:])}
	default:
		return fingerprint{kind: "special"}
	}
}

func validateDestinations(entries []DeployEntry, home string) error {
	safe, err := paths.NewSafeHome(home, false)
	if err != nil {
		return err
	}
	defer safe.Close()
	for _, entry := range entries {
		if _, err := safe.Lstat(entry.Path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
	}
	return nil
}

func writeResolved(entries []resolvedFile, home, snapshotPath string, progressCallback progress.Callback) error {
	applied := make([]string, 0, len(entries))
	total := len(entries)
	safe, err := paths.NewSafeHome(home, false)
	if err != nil {
		return err
	}
	defer safe.Close()
	for index, entry := range entries {
		if entry.link != "" {
			err = safe.ReplaceWithSymlink(entry.path, entry.link)
		} else {
			if entry.content == nil {
				err = fmt.Errorf("missing content for %s", entry.path)
			} else {
				mode, parseErr := strconv.ParseUint(entry.mode, 8, 32)
				if parseErr != nil {
					mode = 0o644
				}
				err = safe.WriteFile(entry.path, entry.content, os.FileMode(mode))
			}
		}
		if err != nil {
			return &kerrors.PartialApplyError{
				KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf("despliegue incompleto en %s: %v", entry.path, err)},
				Applied:     applied,
				Failed:      entry.path,
				Snapshot:    snapshotPath,
			}
		}
		applied = append(applied, entry.path)
		progress.Emit(progressCallback, progress.Writing, index+1, total, entry.path)
	}
	return nil
}

func validateMerges(entries []resolvedFile) error {
	for _, entry := range entries {
		if !entry.write || entry.mergedText == nil || entry.localText == nil || entry.cloneText == nil {
			continue
		}
		validation := conflicts.ValidateMerge(entry.path, *entry.localText, *entry.cloneText, *entry.mergedText)
		if validation.Status == "invalid" {
			return &kerrors.InvalidMergeError{KaminoError: kerrors.KaminoError{
				Msg: fmt.Sprintf("el híbrido de %s no es válido: %s", entry.path, validation.Message),
			}}
		}
	}
	return nil
}

func classification(plan *DeployPlan, resolved []resolvedFile) DeployClassification {
	deployed := make([]string, 0)
	merged := make([]string, 0)
	omitted := append([]string{}, plan.Omitted...)
	unchanged := append([]string{}, plan.Unchanged...)
	for _, entry := range resolved {
		switch entry.category {
		case CategoryDeployed:
			deployed = append(deployed, entry.path)
		case CategoryMerged:
			merged = append(merged, entry.path)
		case CategoryOmitted:
			omitted = append(omitted, entry.path)
		case CategoryUnchanged:
			unchanged = append(unchanged, entry.path)
		}
	}
	return DeployClassification{
		Deployed:         deployed,
		Merged:           merged,
		Omitted:          dedupe(omitted),
		Unchanged:        dedupe(unchanged),
		NotInstalled:     plan.NotInstalled,
		MissingRequested: plan.MissingRequested,
	}
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func isAppInstalled(app archive.AppEntry, definitions catalog.Catalog, applicationDirs []string, pathValue, home string) bool {
	if app.ID == snapshot.SnapshotAppID {
		return true
	}
	definition := definitions[app.ID]
	if definition == nil {
		return false
	}
	return detect.IsInstalled(definition, detect.Options{Home: home, ApplicationDirs: applicationDirs, Path: pathValue})
}

func platformWarnings(manifest *archive.Manifest) []string {
	warnings := make([]string, 0)
	currentArch := sysinfo.Arch()
	if manifest.Arch != "" && currentArch != "" && manifest.Arch != currentArch {
		warnings = append(warnings, fmt.Sprintf(
			"el clon se creó en %s y este Mac es %s; el despliegue puede requerir ajustes", manifest.Arch, currentArch))
	}
	currentMacOS := sysinfo.MacOSVersion()
	if manifest.MacOSVersion != "" && currentMacOS != "" && manifest.MacOSVersion != currentMacOS {
		warnings = append(warnings, fmt.Sprintf(
			"el clon se creó con macOS %s y este Mac usa %s; el despliegue puede requerir ajustes", manifest.MacOSVersion, currentMacOS))
	}
	return warnings
}

func ghosttyPortabilityNotices(manifest *archive.Manifest, cloneFile, home string) []string {
	notices := make([]string, 0)
	for _, app := range manifest.Apps {
		if app.ID != ghostty.GhosttyAppID {
			continue
		}
		for _, file := range app.Files {
			if file.Link != nil {
				continue
			}
			content := archive.ReadFileFromArchive(cloneFile, app.ID, file.Path, ghostty.MaxConfigSize)
			if content == nil {
				continue
			}
			for _, reference := range ghostty.AbsoluteReferences(string(content)) {
				if _, err := paths.ToRelative(reference, home); err != nil {
					notices = append(notices, "aviso: Ghostty conserva una ruta absoluta que puede requerir ajuste manual tras el despliegue")
					break
				}
			}
		}
	}
	return dedupe(notices)
}

// RunDeploy is the non-interactive deploy facade.
func RunDeploy(cloneFile string, options Options, selected *[]string, dryRun bool) (DeployResult, error) {
	session, err := PrepareDeploy(cloneFile, options)
	if err != nil {
		return DeployResult{}, err
	}
	defer session.Close()
	plan, err := session.Plan(selected, dryRun, nil, nil, nil)
	if err != nil {
		return DeployResult{}, err
	}
	if dryRun {
		return session.DryRunResult(plan), nil
	}
	return session.Apply(conflicts.ApplyAll(plan.Conflicts), nil, nil)
}
