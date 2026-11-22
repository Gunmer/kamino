package deploy

import (
	"context"
	"fmt"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/conflicts"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/snapshot"
)

// Apply applies the decisions with a snapshot and atomic per-file writes.
func (s *DeploySession) Apply(ctx context.Context, decisions conflicts.ConflictDecisions, progressCallback progress.Callback) (DeployResult, error) {
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
		if err := progress.EnsureNotCancelled(ctx); err != nil {
			_ = safe.Close()
			return DeployResult{}, err
		}
		if err := s.verifyUnchanged(safe, item.path); err != nil {
			_ = safe.Close()
			return DeployResult{}, err
		}
	}
	_ = safe.Close()

	destinations := make([]string, 0, len(writes))
	for _, item := range writes {
		destinations = append(destinations, item.destination)
	}
	// Hold the snapshots directory for the whole snapshot-and-apply window so a
	// concurrent deploy cannot prune the recovery snapshot of this one.
	directory, err := s.SnapshotDirectory()
	if err != nil {
		return DeployResult{}, err
	}
	release, err := snapshot.Lock(ctx, directory)
	if err != nil {
		return DeployResult{}, err
	}
	defer func() { _ = release() }()
	// Waiting for the lock can take long: check again before the snapshot.
	if err := s.verifyAll(writes); err != nil {
		return DeployResult{}, err
	}
	protected, err := snapshot.SnapshotFiles(ctx, destinations, s.Home)
	if err != nil {
		return DeployResult{}, err
	}
	if err := progress.EnsureNotCancelled(ctx); err != nil {
		return DeployResult{}, err
	}
	snapshotPath, err := snapshot.CreateSnapshot(ctx, destinations, s.Home, directory, "", s.SnapshotsLimit)
	if err != nil {
		return DeployResult{}, err
	}
	if err := s.writeResolved(writes, snapshotPath, progressCallback); err != nil {
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

// verifyUnchanged fails when the destination no longer matches the state the
// plan was built from.
func (s *DeploySession) verifyUnchanged(safe *paths.SafeHome, rel string) error {
	current, err := fingerprintOf(safe, rel)
	if err != nil {
		return err
	}
	expected, ok := s.fingerprints[rel]
	if !ok || current != expected {
		return &kerrors.ConcurrentChangeError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("el destino cambió durante la revisión: %s", rel),
		}}
	}
	return nil
}

func (s *DeploySession) verifyAll(entries []resolvedFile) error {
	safe, err := paths.NewSafeHome(s.Home, false)
	if err != nil {
		return err
	}
	defer func() { _ = safe.Close() }()
	for _, entry := range entries {
		if err := s.verifyUnchanged(safe, entry.path); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeploySession) writeResolved(entries []resolvedFile, snapshotPath string, progressCallback progress.Callback) error {
	applied := make([]string, 0, len(entries))
	total := len(entries)
	safe, err := paths.NewSafeHome(s.Home, false)
	if err != nil {
		return err
	}
	defer func() { _ = safe.Close() }()
	for index, entry := range entries {
		if err := s.verifyUnchanged(safe, entry.path); err != nil {
			if len(applied) == 0 {
				return err
			}
			return &kerrors.PartialApplyError{
				KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf("implantación incompleta en %s: %v", entry.path, err)},
				Applied:     applied,
				Failed:      entry.path,
				Snapshot:    snapshotPath,
			}
		}
		if entry.link != "" {
			err = safe.ReplaceWithSymlink(entry.path, entry.link)
		} else {
			if entry.content == nil {
				err = fmt.Errorf("missing content for %s", entry.path)
			} else {
				mode, parseErr := archive.ParseMode(entry.mode)
				if parseErr != nil {
					err = parseErr
				} else {
					err = safe.WriteFile(entry.path, entry.content, mode)
				}
			}
		}
		if err != nil {
			return &kerrors.PartialApplyError{
				KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf("implantación incompleta en %s: %v", entry.path, err)},
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
