// Package snapshot creates the previous-state archive used before deploying.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/defaults"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/sysinfo"
	"github.com/Gunmer/kamino/internal/version"
)

const (
	SnapshotAppID = "snapshot"
	maxSuffix     = 1000
)

// snapshotName matches the files this package writes and may prune.
var snapshotName = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)(?:-(\d+))?\.tar\.gz$`)

// snapshotFile is one prunable snapshot ordered by timestamp and suffix.
type snapshotFile struct {
	name   string
	stamp  string
	suffix int
}

// CreateSnapshot stores the current state of paths in a snapshot archive and
// prunes the oldest snapshots so that at most limit remain. A limit of zero
// keeps every snapshot.
func CreateSnapshot(targets []string, home, snapshotsDir, timestamp string, limit int) (string, error) {
	if len(targets) == 0 {
		return "", nil
	}
	root := home
	if root == "" {
		root, _ = os.UserHomeDir()
	}
	when := timestamp
	if when == "" {
		when = sysinfo.NowISO()
	}
	if err := checkTimestamp(when); err != nil {
		return "", err
	}
	definition, err := snapshotDefinition(targets, root)
	if err != nil {
		return "", err
	}
	built, err := plan.BuildPlan([]*catalog.AppDefinition{definition}, root, nil, nil)
	if err != nil {
		return "", err
	}
	if len(built.Operations) == 0 {
		return "", nil
	}
	files := make([]archive.FileEntry, 0, len(built.Operations))
	for _, operation := range built.Operations {
		files = append(files, archive.FileEntry{
			Path:   operation.Path,
			SHA256: operation.SHA256,
			Size:   operation.Size,
			Mode:   operation.Mode,
			Link:   operation.Link,
		})
	}
	manifest := &archive.Manifest{
		FormatVersion: archive.FormatVersion,
		ToolVersion:   version.Version,
		CreatedAt:     when,
		MacOSVersion:  sysinfo.MacOSVersion(),
		Arch:          sysinfo.Arch(),
		Hostname:      sysinfo.Hostname(),
		Apps: []archive.AppEntry{{
			ID:    SnapshotAppID,
			Name:  SnapshotAppID,
			Files: files,
		}},
	}
	directory := snapshotsDir
	if directory == "" {
		directory = defaults.SnapshotsDir(root)
	}
	output, err := outputPath(directory, when)
	if err != nil {
		return "", err
	}
	if err := archive.WriteArchive(output, manifest, archive.WriteOptions{SourceRoot: root}); err != nil {
		return "", err
	}
	if _, err := PruneSnapshots(directory, limit); err != nil {
		return "", err
	}
	return output, nil
}

// PruneSnapshots deletes the oldest snapshots of directory until at most limit
// remain and returns the deleted paths. A limit of zero keeps every snapshot.
// Only regular files named like a snapshot are eligible.
func PruneSnapshots(directory string, limit int) ([]string, error) {
	if limit <= 0 || directory == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("no se pudo leer el directorio de holocrones %s: %v", directory, err),
		}}
	}
	files := make([]snapshotFile, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		matches := snapshotName.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		suffix := 0
		if matches[2] != "" {
			suffix, err = strconv.Atoi(matches[2])
			if err != nil {
				continue
			}
		}
		files = append(files, snapshotFile{name: entry.Name(), stamp: matches[1], suffix: suffix})
	}
	if len(files) <= limit {
		return nil, nil
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].stamp != files[j].stamp {
			return files[i].stamp < files[j].stamp
		}
		return files[i].suffix < files[j].suffix
	})
	removed := make([]string, 0, len(files)-limit)
	for _, file := range files[:len(files)-limit] {
		path := filepath.Join(directory, file.name)
		if err := os.Remove(path); err != nil {
			return removed, &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
				Msg: fmt.Sprintf("no se pudo borrar el holocrón antiguo %s: %v", path, err),
			}}
		}
		removed = append(removed, path)
	}
	return removed, nil
}

// SnapshotFiles counts the files a snapshot would protect without writing.
func SnapshotFiles(targets []string, home string) int {
	if len(targets) == 0 {
		return 0
	}
	root := home
	if root == "" {
		root, _ = os.UserHomeDir()
	}
	definition, err := snapshotDefinition(targets, root)
	if err != nil {
		return 0
	}
	built, err := plan.BuildPlan([]*catalog.AppDefinition{definition}, root, nil, nil)
	if err != nil {
		return 0
	}
	return len(built.Operations)
}

func snapshotDefinition(targets []string, root string) (*catalog.AppDefinition, error) {
	relative, err := relativeTargets(targets, root)
	if err != nil {
		return nil, err
	}
	command := SnapshotAppID
	return &catalog.AppDefinition{ID: SnapshotAppID, Name: SnapshotAppID, Command: &command, Paths: relative}, nil
}

func checkTimestamp(timestamp string) error {
	if timestamp == "" || filepath.Base(timestamp) != timestamp || strings.ContainsAny(timestamp, `/\`) {
		return &kerrors.PathError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("timestamp no admitido como nombre de archivo: %q", timestamp),
		}}
	}
	return nil
}

func outputPath(directory, timestamp string) (string, error) {
	for index := 0; index <= maxSuffix; index++ {
		suffix := ""
		if index > 0 {
			suffix = fmt.Sprintf("-%d", index)
		}
		candidate := filepath.Join(directory, timestamp+suffix+".tar.gz")
		if _, err := os.Lstat(candidate); err != nil {
			if os.IsNotExist(err) {
				return candidate, nil
			}
		}
	}
	return "", &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
		Msg: fmt.Sprintf("no se pudo elegir un nombre libre para %s.tar.gz", timestamp),
	}}
}

func relativeTargets(targets []string, root string) ([]string, error) {
	seen := map[string]bool{}
	relative := make([]string, 0, len(targets))
	for _, target := range targets {
		rel, err := paths.ToRelative(target, root)
		if err != nil {
			return nil, err
		}
		if !seen[rel] {
			seen[rel] = true
			relative = append(relative, rel)
		}
	}
	result := make([]string, 0, len(relative))
	for _, rel := range relative {
		covered := false
		for _, other := range relative {
			if isAncestor(other, rel) {
				covered = true
				break
			}
		}
		if !covered {
			result = append(result, rel)
		}
	}
	return result, nil
}

func isAncestor(other, rel string) bool {
	otherParts := strings.Split(other, "/")
	parts := strings.Split(rel, "/")
	if len(otherParts) >= len(parts) {
		return false
	}
	for index := range otherParts {
		if otherParts[index] != parts[index] {
			return false
		}
	}
	return true
}
