// Package clone plans and writes a clone of the installed applications.
package clone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/detect"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/sysinfo"
	"github.com/Gunmer/kamino/internal/version"
)

// PlanOptions controls PlanClone.
type PlanOptions struct {
	Home     string
	Selected *[]string
	// Files maps an app id to the only paths to include for it. An app absent
	// from the map includes every file the plan resolves.
	Files           map[string][]string
	Output          string
	OutputDir       string
	Catalog         catalog.Catalog
	CatalogDir      string
	ApplicationDirs []string
	Path            string
	Metadata        map[string]string
	Now             time.Time
	Progress        progress.Callback
}

// ClonePlan is a complete clone plan that has not written anything yet.
type ClonePlan struct {
	Plan     plan.Plan
	Manifest *archive.Manifest
	Output   string
	Apps     []*catalog.AppDefinition
}

// FileCount returns the planned file count.
func (c ClonePlan) FileCount() int { return len(c.Plan.Operations) }

// Size returns the planned total size.
func (c ClonePlan) Size() int64 {
	total := int64(0)
	for _, operation := range c.Plan.Operations {
		total += operation.Size
	}
	return total
}

// CloneResult is the outcome of a clone.
type CloneResult struct {
	Plan       plan.Plan
	Manifest   *archive.Manifest
	Output     string
	DryRun     bool
	PlanOutput string
}

// Size returns the planned total size.
func (c CloneResult) Size() int64 {
	total := int64(0)
	for _, operation := range c.Plan.Operations {
		total += operation.Size
	}
	return total
}

// PlanClone builds the plan, manifest and destination without writing.
func PlanClone(ctx context.Context, options PlanOptions) (*ClonePlan, error) {
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
	apps := selectInstalled(definitions, options.Selected, options.ApplicationDirs, options.Path, root)
	built, err := plan.BuildPlan(ctx, apps, root, options.Progress)
	if err != nil {
		return nil, err
	}
	built.Operations = filterOperations(built.Operations, options.Files)
	manifest := buildManifest(apps, built, options.ApplicationDirs, options.Metadata, root)
	target, err := destination(options.Output, options.OutputDir, root, options.Now)
	if err != nil {
		return nil, err
	}
	return &ClonePlan{Plan: built, Manifest: manifest, Output: target, Apps: apps}, nil
}

// WriteClone writes the planned archive atomically.
func WriteClone(ctx context.Context, planned *ClonePlan, home string, force bool, expectedState *archive.DestState, progressCallback progress.Callback) (CloneResult, error) {
	root, err := paths.ResolveHome(home)
	if err != nil {
		return CloneResult{}, err
	}
	if err := archive.WriteArchive(ctx, planned.Output, planned.Manifest, archive.WriteOptions{
		SourceRoot:    root,
		Overwrite:     force,
		ExpectedState: expectedState,
		Progress:      progressCallback,
	}); err != nil {
		return CloneResult{}, err
	}
	return CloneResult{
		Plan:       planned.Plan,
		Manifest:   planned.Manifest,
		Output:     planned.Output,
		DryRun:     false,
		PlanOutput: planned.Output,
	}, nil
}

// RunClone is the CLI facade: plan and, unless dry-run, write.
func RunClone(ctx context.Context, options PlanOptions, dryRun, force bool) (CloneResult, error) {
	planned, err := PlanClone(ctx, options)
	if err != nil {
		return CloneResult{}, err
	}
	if dryRun {
		return CloneResult{
			Plan:       planned.Plan,
			Manifest:   planned.Manifest,
			Output:     "",
			DryRun:     true,
			PlanOutput: planned.Output,
		}, nil
	}
	return WriteClone(ctx, planned, options.Home, force, nil, options.Progress)
}

func selectInstalled(definitions catalog.Catalog, selected *[]string, applicationDirs []string, pathValue, home string) []*catalog.AppDefinition {
	wanted := map[string]bool(nil)
	if selected != nil {
		wanted = map[string]bool{}
		for _, id := range *selected {
			wanted[id] = true
		}
	}
	ids := definitions.SortedIDs()
	apps := make([]*catalog.AppDefinition, 0)
	for _, id := range ids {
		app := definitions[id]
		if wanted != nil && !wanted[app.ID] {
			continue
		}
		if detect.IsInstalled(app, detect.Options{Home: home, ApplicationDirs: applicationDirs, Path: pathValue}) {
			apps = append(apps, app)
		}
	}
	return apps
}

// filterOperations keeps, for each app listed in files, only the listed paths.
func filterOperations(operations []plan.Operation, files map[string][]string) []plan.Operation {
	if len(files) == 0 {
		return operations
	}
	allowed := make(map[string]map[string]bool, len(files))
	for id, list := range files {
		set := make(map[string]bool, len(list))
		for _, path := range list {
			set[path] = true
		}
		allowed[id] = set
	}
	kept := make([]plan.Operation, 0, len(operations))
	for _, operation := range operations {
		if set, ok := allowed[operation.AppID]; ok && !set[operation.Path] {
			continue
		}
		kept = append(kept, operation)
	}
	return kept
}

func buildManifest(apps []*catalog.AppDefinition, built plan.Plan, applicationDirs []string, metadata map[string]string, home string) *archive.Manifest {
	filesByApp := map[string][]archive.FileEntry{}
	for _, operation := range built.Operations {
		filesByApp[operation.AppID] = append(filesByApp[operation.AppID], archive.FileEntry{
			Path:   operation.Path,
			SHA256: operation.SHA256,
			Size:   operation.Size,
			Mode:   operation.Mode,
			Link:   operation.Link,
		})
	}
	entries := make([]archive.AppEntry, 0, len(apps))
	for _, app := range apps {
		entry := archive.AppEntry{
			ID:       app.ID,
			Name:     app.Name,
			BundleID: app.BundleID,
			Files:    filesByApp[app.ID],
		}
		if entry.Files == nil {
			entry.Files = []archive.FileEntry{}
		}
		if detected := detect.DetectVersion(app, detect.Options{Home: home, ApplicationDirs: applicationDirs}); detected != "" {
			entry.Version = &detected
		}
		entries = append(entries, entry)
	}
	return &archive.Manifest{
		FormatVersion: archive.FormatVersion,
		ToolVersion:   version.Version,
		CreatedAt:     metadataValue(metadata, "created_at", sysinfo.NowISO()),
		MacOSVersion:  metadataValue(metadata, "macos_version", sysinfo.MacOSVersion()),
		Arch:          metadataValue(metadata, "arch", sysinfo.Arch()),
		Hostname:      metadataValue(metadata, "hostname", sysinfo.Hostname()),
		Apps:          entries,
	}
}

func metadataValue(metadata map[string]string, key, fallback string) string {
	if value, ok := metadata[key]; ok {
		return value
	}
	return fallback
}

// DefaultName returns clone-yyyyMMdd.tar.gz using local time.
func DefaultName(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	return "clone-" + now.Format("20060102") + ".tar.gz"
}

func destination(output, outputDir, root string, now time.Time) (string, error) {
	if output != "" {
		return output, nil
	}
	directory := outputDir
	if directory == "" {
		resolved, err := defaults.VaultDir(root)
		if err != nil {
			return "", err
		}
		directory = resolved
	}
	if info, err := os.Stat(directory); err == nil && !info.IsDir() {
		return "", &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("la ruta del directorio no es válida: %s", directory),
		}}
	}
	return filepath.Join(directory, DefaultName(now)), nil
}

// FreeSuffix returns the first free numbered name for the same base.
func FreeSuffix(output string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(output), ".tar.gz")
	for index := 1; index <= 1000; index++ {
		candidate := filepath.Join(filepath.Dir(output), fmt.Sprintf("%s-%d.tar.gz", base, index))
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
		Msg: fmt.Sprintf("no se pudo elegir un nombre libre para %s", filepath.Base(output)),
	}}
}

// ValidateDestination validates the destination without creating anything.
func ValidateDestination(output string, allowMissingParent bool) string {
	name := filepath.Base(output)
	if name == "" || name == "." || name == ".." || strings.HasSuffix(output, "/") {
		return "el nombre del clon no es válido"
	}
	parent := filepath.Dir(output)
	if parent == "" {
		parent = "."
	}
	info, err := os.Stat(parent)
	if err != nil {
		if os.IsNotExist(err) {
			if allowMissingParent {
				return ""
			}
			return fmt.Sprintf("el directorio no existe: %s", parent)
		}
		return fmt.Sprintf("la ruta del directorio no es válida: %s", parent)
	}
	if !info.IsDir() {
		return fmt.Sprintf("la ruta del directorio no es válida: %s", parent)
	}
	if info, err := os.Lstat(output); err == nil && info.IsDir() {
		return fmt.Sprintf("el destino es un directorio: %s", output)
	}
	return ""
}
