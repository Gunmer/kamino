// Package ui prints non-interactive warnings, summaries and diffs.
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/implant"
	"github.com/Gunmer/kamino/internal/plan"
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
	} else {
		// Rejections such as binaries or app bundles must stay visible without
		// --verbose: the user needs to know what the clone does not contain.
		printSkipped(w, result.Plan.Skipped)
	}
}

func printSkipped(w io.Writer, skipped []plan.SkippedPath) {
	if len(skipped) == 0 {
		return
	}
	fmt.Fprintln(w, texts.CloneSkippedTitle)
	for _, item := range skipped {
		fmt.Fprintf(w, "  ~ %s (%s)\n", item.Path, item.Reason)
	}
}

// PrintImplantSummary prints the implant summary or the dry-run plan and diffs.
func PrintImplantSummary(w io.Writer, result implant.ImplantResult, verbose bool) {
	if result.DryRun {
		fmt.Fprintln(w, texts.TitleImplantDryRun)
		printInstalls(w, result.Installs)
		for _, entry := range result.Plan.Entries {
			fmt.Fprintf(w, "  [%s] %s\n", entry.Status, entry.Path)
		}
		for _, conflict := range result.Plan.Conflicts {
			printConflictDiff(w, conflict)
		}
		fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantOmitted, "{count}", itoa(len(result.Omitted))))
		fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantUnchanged, "{count}", itoa(len(result.Unchanged))))
		fmt.Fprintf(w, "%s: %d\n", texts.NotInstalled, len(result.NotInstalled))
		for _, notice := range result.Plan.Notices {
			fmt.Fprintf(w, "  ! %s\n", notice)
		}
		return
	}
	printInstalls(w, result.Installs)
	fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantImplanted, "{count}", itoa(len(result.Implanted))))
	fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantHybrids, "{count}", itoa(len(result.Merged))))
	fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantOmitted, "{count}", itoa(len(result.Omitted))))
	fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantUnchanged, "{count}", itoa(len(result.Unchanged))))
	fmt.Fprintf(w, "%s: %d\n", texts.NotInstalled, len(result.NotInstalled))
	if result.Snapshot != "" {
		fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantSnapshotPath, "{path}", result.Snapshot))
		fmt.Fprintln(w, strings.ReplaceAll(texts.ImplantUndo, "{path}", result.Snapshot))
	} else {
		fmt.Fprintln(w, texts.ImplantNoSnapshot)
	}
	for _, notice := range result.Plan.Notices {
		fmt.Fprintf(w, "! %s\n", notice)
	}
	if verbose {
		fmt.Fprintln(w, texts.TitleDetail)
		for _, path := range result.Implanted {
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

// printInstalls lists what --install-missing did or would do, one line per app.
func printInstalls(w io.Writer, outcomes []implant.InstallOutcome) {
	if len(outcomes) == 0 {
		return
	}
	fmt.Fprintln(w, texts.ImplantInstallTitle)
	for _, outcome := range outcomes {
		switch outcome.Status {
		case implant.InstallPlanned:
			url := ""
			if outcome.URL != "" {
				url = " desde " + outcome.URL
			}
			fmt.Fprintln(w, "  "+strings.NewReplacer("{app}", outcome.AppID, "{method}", outcome.Method, "{url}", url).Replace(texts.ImplantInstallPlanned))
		case implant.InstallDone:
			fmt.Fprintln(w, "  "+strings.NewReplacer("{app}", outcome.AppID, "{method}", outcome.Method).Replace(texts.ImplantInstallDone))
		case implant.InstallFailed:
			fmt.Fprintln(w, "  "+strings.NewReplacer("{app}", outcome.AppID, "{reason}", outcome.Reason).Replace(texts.ImplantInstallFailed))
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
