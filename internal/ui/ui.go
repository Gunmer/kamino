// Package ui prints non-interactive warnings, summaries and diffs.
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
)

// PrintPrivacyWarning shows the sensitive-content warning before cloning.
func PrintPrivacyWarning(w io.Writer) {
	fmt.Fprintln(w, texts.PrivacyWarning)
}

// PrintUpdateStart shows that the version check is starting.
func PrintUpdateStart(w io.Writer) {
	fmt.Fprintln(w, texts.UpdateChecking)
}

// PrintUpdateResult prints an update result.
func PrintUpdateResult(w io.Writer, result update.Result) {
	switch result.Outcome {
	case update.OutcomeUpdated:
		fmt.Fprintln(w, strings.NewReplacer(
			"{current}", result.CurrentVersion,
			"{latest}", result.LatestVersion,
			"{path}", result.Target,
		).Replace(texts.UpdateUpdated))
		fmt.Fprintln(w, texts.UpdateRestart)
	case update.OutcomeNewer:
		fmt.Fprintln(w, strings.NewReplacer(
			"{current}", result.CurrentVersion,
			"{latest}", result.LatestVersion,
		).Replace(texts.UpdateNewer))
	default:
		fmt.Fprintln(w, strings.ReplaceAll(texts.UpdateUpToDate, "{current}", result.CurrentVersion))
	}
}

// PrintCloneSummary prints the clone summary or the dry-run plan.
func PrintCloneSummary(w io.Writer, result clone.CloneResult, verbose bool) {
	if result.DryRun {
		fmt.Fprintln(w, texts.CloneDryRunTitle)
		for _, operation := range result.Plan.Operations {
			fmt.Fprintf(w, "  + %s\n", operation.Path)
		}
		for _, skipped := range result.Plan.Skipped {
			fmt.Fprintf(w, "  ~ %s (%s)\n", skipped.Path, skipped.Reason)
		}
		for _, notice := range result.Plan.Notices {
			fmt.Fprintf(w, "  ! %s\n", notice)
		}
		if result.PlanOutput != "" {
			fmt.Fprintln(w, strings.ReplaceAll(texts.CloneDestination, "{path}", result.PlanOutput))
		}
		return
	}
	fmt.Fprintln(w, texts.CloneSummaryTitle)
	fmt.Fprintf(w, "%s\t%s\n", texts.CloneAppsLabel, texts.CloneFilesLabel)
	for _, app := range result.Manifest.Apps {
		fmt.Fprintf(w, "%s\t%d\n", app.Name, len(app.Files))
	}
	fmt.Fprintln(w, strings.ReplaceAll(texts.CloneDestination, "{path}", result.Output))
	if verbose {
		fmt.Fprintln(w, texts.CloneIncludedTitle)
		for _, operation := range result.Plan.Operations {
			fmt.Fprintf(w, "  + %s\n", operation.Path)
		}
		for _, skipped := range result.Plan.Skipped {
			fmt.Fprintf(w, "  ~ %s (%s)\n", skipped.Path, skipped.Reason)
		}
		for _, notice := range result.Plan.Notices {
			fmt.Fprintf(w, "  ! %s\n", notice)
		}
	}
}

// PrintDeploySummary prints the deploy summary or the dry-run plan and diffs.
func PrintDeploySummary(w io.Writer, result deploy.DeployResult, verbose bool) {
	if result.DryRun {
		fmt.Fprintln(w, texts.TitleDeployDryRun)
		for _, entry := range result.Plan.Entries {
			fmt.Fprintf(w, "  [%s] %s\n", entry.Status, entry.Path)
		}
		for _, conflict := range result.Plan.Conflicts {
			printConflictDiff(w, conflict)
		}
		fmt.Fprintln(w, strings.ReplaceAll(texts.DeployOmitted, "{count}", itoa(len(result.Omitted))))
		fmt.Fprintln(w, strings.ReplaceAll(texts.DeployUnchanged, "{count}", itoa(len(result.Unchanged))))
		fmt.Fprintf(w, "%s: %d\n", texts.NotInstalled, len(result.NotInstalled))
		for _, notice := range result.Plan.Notices {
			fmt.Fprintf(w, "  ! %s\n", notice)
		}
		return
	}
	fmt.Fprintln(w, strings.ReplaceAll(texts.DeployDeployed, "{count}", itoa(len(result.Deployed))))
	fmt.Fprintln(w, strings.ReplaceAll(texts.DeployHybrids, "{count}", itoa(len(result.Merged))))
	fmt.Fprintln(w, strings.ReplaceAll(texts.DeployOmitted, "{count}", itoa(len(result.Omitted))))
	fmt.Fprintln(w, strings.ReplaceAll(texts.DeployUnchanged, "{count}", itoa(len(result.Unchanged))))
	fmt.Fprintf(w, "%s: %d\n", texts.NotInstalled, len(result.NotInstalled))
	if result.Snapshot != "" {
		fmt.Fprintln(w, strings.ReplaceAll(texts.DeploySnapshotPath, "{path}", result.Snapshot))
		fmt.Fprintln(w, strings.ReplaceAll(texts.DeployUndo, "{path}", result.Snapshot))
	} else {
		fmt.Fprintln(w, texts.DeployNoSnapshot)
	}
	for _, notice := range result.Plan.Notices {
		fmt.Fprintf(w, "! %s\n", notice)
	}
	if verbose {
		fmt.Fprintln(w, texts.TitleDetail)
		for _, path := range result.Deployed {
			fmt.Fprintf(w, "  + %s\n", path)
		}
		for _, path := range result.Merged {
			fmt.Fprintf(w, "  * %s\n", path)
		}
		for _, path := range result.Omitted {
			fmt.Fprintf(w, "  ~ %s\n", path)
		}
		for _, path := range result.Unchanged {
			fmt.Fprintf(w, "  = %s\n", path)
		}
	}
}

func printConflictDiff(w io.Writer, conflict conflicts.Conflict) {
	switch conflict.Kind {
	case conflicts.KindText:
		if conflict.LocalText == nil || conflict.CloneText == nil {
			return
		}
		diff := conflicts.UnifiedDiff(
			conflicts.SplitLines(*conflict.LocalText),
			conflicts.SplitLines(*conflict.CloneText),
			texts.Original+"/"+conflict.Path,
			"clon/"+conflict.Path,
		)
		fmt.Fprintln(w, strings.Join(diff, "\n"))
	case conflicts.KindLink:
		fmt.Fprintln(w, "  ! "+strings.NewReplacer(
			"{path}", conflict.Path,
			"{original}", texts.Original,
			"{local}", conflict.LocalLink,
			"{clone}", conflict.CloneLink,
		).Replace(texts.ConflictSymlink))
	default:
		reason := texts.ConflictBinary
		if conflict.Kind == conflicts.KindLarge {
			reason = texts.ConflictLarge
		}
		fmt.Fprintln(w, "  ! "+strings.NewReplacer("{path}", conflict.Path, "{reason}", reason).Replace(texts.ConflictWholeFile))
	}
}

func itoa(value int) string { return fmt.Sprintf("%d", value) }
