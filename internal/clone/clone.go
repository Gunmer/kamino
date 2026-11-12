// Package clone plans and writes a clone of the installed applications.
package clone

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	Home            string
	Selected        *[]string
	Output          string
	OutputDir       string
	Catalog         catalog.Catalog
	CatalogDir      string
	ApplicationDirs []string
	Path            string
	Metadata        map[string]string
	Now             time.Time
	Progress        progress.Callback
	Cancelled       progress.Cancel
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

// PlanClone builds the plan, manifest and destination without writing.
func PlanClone(options PlanOptions) (*ClonePlan, error) {
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
	apps := selectInstalled(definitions, options.Selected, options.ApplicationDirs, options.Path, root)
	built, err := plan.BuildPlan(apps, root, options.Progress, options.Cancelled)
	if err != nil {
		return nil, err
	}
	manifest := buildManifest(apps, built, options.ApplicationDirs, options.Metadata, root)
	target, err := destination(options.Output, options.OutputDir, root, options.Now)
	if err != nil {
		return nil, err
	}
	return &ClonePlan{Plan: built, Manifest: manifest, Output: target, Apps: apps}, nil
}

// WriteClone writes the planned archive atomically.
func WriteClone(planned *ClonePlan, home string, force bool, expectedState *archive.DestState, progressCallback progress.Callback, cancelled progress.Cancel) (CloneResult, error) {
	root := home
	if root == "" {
		root, _ = os.UserHomeDir()
	}
	if err := archive.WriteArchive(planned.Output, planned.Manifest, archive.WriteOptions{
		SourceRoot:    root,
		Overwrite:     force,
		ExpectedState: expectedState,
		Progress:      progressCallback,
		Cancelled:     cancelled,
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
func RunClone(options PlanOptions, dryRun, force bool) (CloneResult, error) {
	planned, err := PlanClone(options)
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
	return WriteClone(planned, options.Home, force, nil, options.Progress, options.Cancelled)
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
		directory = defaults.VaultDir(root)
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

// RelativeSources returns the plan operations ordered by app and path.
func RelativeSources(operations []plan.Operation) []string {
	result := make([]string, 0, len(operations))
	for _, operation := range operations {
		result = append(result, operation.Path)
	}
	sort.Strings(result)
	return result
}

var _ = paths.ToAbsolute
