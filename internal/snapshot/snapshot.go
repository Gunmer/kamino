// Package snapshot creates the previous-state archive used before deploying.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
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

// CreateSnapshot stores the current state of paths in a snapshot archive.
func CreateSnapshot(targets []string, home, snapshotsDir, timestamp string) (string, error) {
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
	return output, nil
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
