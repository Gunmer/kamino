package tui

import (
	"fmt"
	"strings"

	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
)

const (
	colorReset  = "\x1b[0m"
	colorBold   = "\x1b[1m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorRed    = "\x1b[31m"
	colorGray   = "\x1b[90m"
	colorCyan   = "\x1b[36m"
)

func (m *Model) paint(code, value string) string {
	if m.NoColor {
		return value
	}
	return code + value + colorReset
}

// View renders the current state.
func (m *Model) View() string {
	if m.Width < 80 || m.Height < 24 {
		return m.frame([]string{
			m.paint(colorRed+colorBold, texts.ErrorSmallTitle),
			"",
			texts.ErrorSmallBody,
			"",
			m.paint(colorGray, "?"+texts.HelpTitle),
		})
	}
	if m.Error != nil {
		return m.viewError()
	}
	if m.Screen == ScreenHelp {
		return m.viewHelp()
	}
	switch m.Screen {
	case ScreenMenu:
		return m.viewMenu()
	case ScreenC1:
		return m.viewCloneApps()
	case ScreenC2:
		return m.viewCloneDestination()
	case ScreenC3:
		return m.viewCloneReview()
	case ScreenC4:
		return m.viewCloneSummary()
	case ScreenD0:
		return m.viewDeployFile()
	case ScreenD1:
		return m.viewDeployApps()
	case ScreenD2:
		return m.viewDeployConflicts()
	case ScreenD3:
		return m.viewDeployDetail()
	case ScreenD4:
		return m.viewDeployConfirm()
	case ScreenD5:
		return m.viewDeploySummary()
	case ScreenU1:
		return m.viewUpdate()
	case ScreenSettings:
		return m.viewSettings()
	}
	return m.frame([]string{string(m.Screen)})
}

func (m *Model) frame(lines []string) string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if len([]rune(stripANSI(line))) > m.Width {
			runes := []rune(stripANSI(line))
			line = string(runes[:m.Width])
		}
		result = append(result, line)
	}
	for len(result) < m.Height-1 {
		result = append(result, "")
	}
	if len(result) > m.Height-1 {
		result = result[:m.Height-1]
	}
	return strings.Join(result, "\n") + "\n"
}

func stripANSI(value string) string {
	var builder strings.Builder
	inEscape := false
	for _, character := range value {
		if inEscape {
			if character == 'm' {
				inEscape = false
			}
			continue
		}
		if character == 0x1b {
			inEscape = true
			continue
		}
		builder.WriteRune(character)
	}
	return builder.String()
}

func (m *Model) viewMenu() string {
	lines := make([]string, 0)
	for _, line := range texts.Wordmark {
		lines = append(lines, m.paint(colorCyan+colorBold, line))
	}
	lines = append(lines, "", m.paint(colorGray, texts.Subtitle), "")
	options := []string{texts.MenuClone, texts.MenuDeploy, texts.MenuUpdate, texts.MenuConfig, texts.MenuExit}
	helps := []string{texts.MenuCloneHelp, texts.MenuDeployHelp, texts.MenuUpdateHelp, texts.MenuConfigHelp, texts.MenuExitHelp}
	for index, option := range options {
		marker := texts.MarkerNotChosen
		if index == m.MenuIndex {
			marker = texts.MarkerChosen
		}
		line := fmt.Sprintf("%s %-14s %s", marker, option, m.paint(colorGray, helps[index]))
		if index == m.MenuIndex {
			line = m.paint(colorGreen, line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsMenu))
	lines = append(lines, m.paint(colorGray, texts.PrivacyWarning))
	return m.frame(lines)
}

func (m *Model) viewCloneApps() string {
	lines := []string{
		m.paint(colorBold, "Paso 1 de 3 · apps"),
		m.paint(colorBold, texts.CloneAppsTitle),
	}
	if m.CloneOutputError != "" {
		lines = append(lines, m.paint(colorYellow, m.CloneOutputError))
	}
	if m.CloneFiltering {
		lines = append(lines, texts.CloneFilter+": "+m.CloneFilter+"_")
	} else if m.CloneFilter != "" {
		lines = append(lines, texts.CloneFilter+": "+m.CloneFilter)
	}
	visible := m.visibleCloneApps()
	if len(visible) == 0 {
		lines = append(lines, m.paint(colorGray, "sin apps"))
	}
	start := 0
	maxRows := m.Height - 10
	if maxRows < 3 {
		maxRows = 3
	}
	if m.CloneIndex >= maxRows {
		start = m.CloneIndex - maxRows + 1
	}
	for index := start; index < len(visible) && index < start+maxRows; index++ {
		choice := visible[index]
		marker := texts.MarkerNotChosen
		if choice.Selected {
			marker = texts.MarkerChosen
		}
		line := fmt.Sprintf("%s %-22s %s", marker, choice.Name, m.paint(colorGray, choice.ID))
		if !choice.Detected {
			line = m.paint(colorGray, line+" · sin detectar")
		}
		if index == m.CloneIndex {
			line = m.paint(colorGreen, "> "+line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.paint(colorGray, "Espacio marca · a/n todas/ninguna · / filtra · Enter continuar · Esc volver"))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewCloneDestination() string {
	lines := []string{
		m.paint(colorBold, "Paso 2 de 3 · destino"),
		m.paint(colorBold, texts.CloneDestinationTitle),
		"",
		"> " + m.CloneOutput + "_",
		"",
		m.paint(colorGray, texts.LabelDefaultName),
		m.paint(colorGray, texts.CloneSecrets),
		"",
		m.paint(colorGray, "Enter continuar · F2 directorio actual · F3 Descargas · F4 Escritorio · Esc volver"),
	}
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewCloneReview() string {
	lines := []string{
		m.paint(colorBold, "Paso 3 de 3 · revisión"),
		m.paint(colorBold, texts.CloneReviewTitle),
	}
	if m.ClonePlan == nil {
		lines = append(lines, texts.ClonePlanning)
		return m.frame(lines)
	}
	lines = append(lines,
		fmt.Sprintf("%s: %d", texts.CloneAppsLabel, len(m.ClonePlan.Apps)),
		fmt.Sprintf("%s: %d", texts.CloneFileCount, m.ClonePlan.FileCount()),
		fmt.Sprintf("%s: %s", texts.CloneSizeLabel, texts.SizeLabel(m.ClonePlan.Size())),
		fmt.Sprintf("%s: %s", texts.ClonePathLabel, m.ClonePlan.Output),
	)
	if m.CloneDryRun {
		lines = append(lines, m.paint(colorYellow, texts.CloneDryRunNote))
	}
	if m.CloneShowSkipped {
		lines = append(lines, "", m.paint(colorBold, texts.CloneSkippedTitle))
		for _, skipped := range m.ClonePlan.Plan.Skipped {
			lines = append(lines, fmt.Sprintf("  ~ %s (%s)", skipped.Path, skipped.Reason))
		}
	}
	if m.CloneShowFiles {
		lines = append(lines, "", m.paint(colorBold, texts.CloneIncludedTitle))
		for _, operation := range m.ClonePlan.Plan.Operations {
			lines = append(lines, "  + "+operation.Path)
		}
	}
	for _, notice := range m.ClonePlan.Plan.Notices {
		lines = append(lines, m.paint(colorYellow, "  ! "+notice))
	}
	action := "Enter clonar"
	if m.CloneDryRun {
		action = "Enter terminar simulación"
	}
	lines = append(lines, "", m.paint(colorGray, "d omitidos · f archivos · "+action+" · Esc volver"))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewCloneSummary() string {
	lines := []string{m.paint(colorGreen+colorBold, texts.CloneSummaryTitle)}
	if m.CloneResult == nil {
		return m.frame(lines)
	}
	if m.CloneResult.DryRun {
		lines[0] = m.paint(colorYellow+colorBold, texts.CloneDryRunTitle)
	}
	lines = append(lines,
		fmt.Sprintf("%s: %d", texts.CloneAppsLabel, len(m.CloneResult.Manifest.Apps)),
		fmt.Sprintf("%s: %d", texts.CloneFileCount, len(m.CloneResult.Plan.Operations)),
	)
	if m.CloneResult.Output != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", texts.ClonePathLabel, m.CloneResult.Output))
	}
	if len(m.CloneResult.Plan.Skipped) > 0 {
		lines = append(lines, "", m.paint(colorYellow, texts.CloneSkippedTitle))
		for _, skipped := range m.CloneResult.Plan.Skipped {
			lines = append(lines, fmt.Sprintf("  ~ %s (%s)", skipped.Path, skipped.Reason))
		}
		lines = append(lines, m.paint(colorYellow, texts.CloneExitLine))
	}
	lines = append(lines, "", m.paint(colorGray, "c copiar ruta · Enter/Esc continuar"))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewDeployFile() string {
	lines := []string{
		m.paint(colorBold, texts.DeployTitle),
		m.paint(colorBold, texts.DeployFileTitle),
	}
	if m.DeployError != "" {
		lines = append(lines, m.paint(colorRed, m.DeployError))
	}
	if m.DeployManual {
		lines = append(lines, "", texts.LabelCloneFilePlaceholder+": "+m.DeployFile+"_")
		lines = append(lines, m.paint(colorGray, "Enter abrir · Esc volver"))
		return m.frame(lines)
	}
	if m.DeployFiltering {
		lines = append(lines, texts.CloneFilter+": "+m.DeployFilter+"_")
	}
	files := m.visibleDeployFiles()
	if len(files) == 0 {
		lines = append(lines, m.paint(colorGray, texts.DeployNoFile))
	}
	start := 0
	maxRows := m.Height - 8
	if maxRows < 3 {
		maxRows = 3
	}
	if m.DeployIndex >= maxRows {
		start = m.DeployIndex - maxRows + 1
	}
	for index := start; index < len(files) && index < start+maxRows; index++ {
		line := files[index]
		if index == m.DeployIndex {
			line = m.paint(colorGreen, "> "+line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsD0))
	for _, notice := range m.deployNotices() {
		lines = append(lines, m.paint(colorYellow, "! "+notice))
	}
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) deployNotices() []string {
	if m.DeploySession == nil {
		return nil
	}
	return append(append([]string{}, m.DeploySession.Warnings...), m.DeploySession.Notices...)
}

func (m *Model) viewDeployApps() string {
	lines := []string{
		m.paint(colorBold, "Paso 2 de 4 · apps"),
		m.paint(colorBold, texts.DeployAppsTitle),
	}
	if m.DeploySession != nil {
		lines = append(lines, m.paint(colorGray, texts.DeployOpenApps))
		for _, warning := range m.DeploySession.Warnings {
			lines = append(lines, m.paint(colorYellow, warning))
		}
	}
	maxRows := m.Height - 9
	if maxRows < 3 {
		maxRows = 3
	}
	start := 0
	if m.DeployIndex >= maxRows {
		start = m.DeployIndex - maxRows + 1
	}
	for index := start; index < len(m.DeployApps) && index < start+maxRows; index++ {
		choice := m.DeployApps[index]
		marker := texts.MarkerNotChosen
		if choice.Selected {
			marker = texts.MarkerChosen
		}
		line := fmt.Sprintf("%s %s", marker, choice.Name)
		if index == m.DeployIndex {
			line = m.paint(colorGreen, "> "+line)
		}
		lines = append(lines, line)
	}
	if m.DeploySession != nil && len(m.DeploySession.NotInstalledApps) > 0 {
		lines = append(lines, "", m.paint(colorGray, texts.DeployNotInstalledTitle))
		for _, item := range m.DeploySession.NotInstalledApps {
			lines = append(lines, m.paint(colorGray, "  "+item.Name))
		}
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsD1))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewDeployConflicts() string {
	lines := []string{
		m.paint(colorBold, "Paso 3 de 4 · conflictos"),
		m.paint(colorBold, texts.DeployConflictsTitle),
	}
	if m.DeployPlan != nil {
		lines = append(lines, fmt.Sprintf("%s · %s · %s",
			replaceCount(texts.DeployNew, len(m.DeployPlan.Entries)),
			replaceCount(texts.DeployPerfect, len(m.DeployPlan.Unchanged)),
			replaceCount(texts.DeployChanged, len(m.Conflicts))))
	}
	if len(m.Conflicts) == 0 {
		lines = append(lines, m.paint(colorGreen, "Sin conflictos: todo se puede aplicar."))
	}
	maxRows := m.Height - 10
	if maxRows < 3 {
		maxRows = 3
	}
	start := 0
	if m.ConflictIndex >= maxRows {
		start = m.ConflictIndex - maxRows + 1
	}
	for index := start; index < len(m.Conflicts) && index < start+maxRows; index++ {
		state := m.Conflicts[index]
		action := string(state.Action)
		switch state.Action {
		case conflicts.ActionApply:
			action = texts.DeployApply
		case conflicts.ActionOmit:
			action = texts.DeployOmit
		case conflicts.ActionReview:
			action = texts.DeployReview
		}
		line := fmt.Sprintf("%-40s %s", state.Conflict.Path, m.paint(colorGray, action))
		if index == m.ConflictIndex {
			line = m.paint(colorGreen, "> "+line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.paint(colorGray, texts.DeployActions))
	lines = append(lines, m.paint(colorGray, texts.ShortcutsD2))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewDeployDetail() string {
	if len(m.Conflicts) == 0 {
		return m.frame([]string{texts.DeployConflictsTitle})
	}
	state := m.Conflicts[m.ConflictIndex%len(m.Conflicts)]
	lines := []string{
		m.paint(colorBold, texts.DeployDetailTitle),
		m.paint(colorBold, state.Conflict.Path),
	}
	if len(state.Conflict.Hunks) == 0 {
		lines = append(lines, texts.StatusConflictWholeFile)
		lines = append(lines, m.paint(colorGray, "Esc volver"))
		return m.frame(lines)
	}
	hunk := state.Conflict.Hunks[state.HunkIndex%len(state.Conflict.Hunks)]
	lines = append(lines, fmt.Sprintf("%s · %s",
		replacePair(texts.DeployBlock, "{index}", fmt.Sprintf("%d", hunk.Index), "{total}", fmt.Sprintf("%d", len(state.Conflict.Hunks))),
		replacePair(texts.DeployBlockCounts, "{clone}", fmt.Sprintf("%d", hunk.CloneEnd-hunk.CloneStart), "{original}", fmt.Sprintf("%d", hunk.LocalEnd-hunk.LocalStart))))
	decidable := decidableRows(&state, state.HunkIndex)
	decisionPosition := 0
	for _, row := range hunk.Rows {
		prefix := "  "
		marker := " "
		text := row.Clone
		if row.Kind == conflicts.RowLocalOnly {
			text = row.Local
		}
		if row.Kind == conflicts.RowReplace {
			text = row.Clone
		}
		if row.Kind != conflicts.RowEqual {
			decision := conflicts.RowClone
			if decisionPosition < len(decidable) && decidable[decisionPosition] < len(state.Rows) {
				decision = state.Rows[decidable[decisionPosition]]
			}
			if decision == conflicts.RowLocal {
				marker = texts.MarkerNotChosen
				if row.Kind == conflicts.RowReplace || row.Kind == conflicts.RowLocalOnly {
					text = row.Local
				}
			} else {
				marker = texts.MarkerChosen
			}
			if decisionPosition < len(decidable) && state.RowIndex%max(1, len(decidable)) == decisionPosition {
				prefix = "> "
			}
			decisionPosition++
		}
		lines = append(lines, prefix+marker+" "+strings.TrimRight(text, "\r\n"))
	}
	lines = append(lines, "", m.paint(colorGray, texts.DeployDecisionHint), m.paint(colorGray, texts.ShortcutsD3))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewDeployConfirm() string {
	lines := []string{
		m.paint(colorBold, "Paso 4 de 4 · confirmación"),
		m.paint(colorBold, texts.DeployConfirmTitle),
	}
	if m.DeployPlan != nil {
		lines = append(lines,
			fmt.Sprintf("%s: %d", texts.CloneFilesLabel, len(m.DeployPlan.Entries)),
			fmt.Sprintf("%s: %d", texts.DeployChanged, len(m.Conflicts)),
			fmt.Sprintf("%s: %d", texts.DeployOmitted, len(m.DeployPlan.Omitted)),
		)
	}
	lines = append(lines, "", m.paint(colorYellow, texts.DeployAfterSnapshot))
	options := []string{texts.DeployBack, texts.DeployAbort, texts.DeployApplyButton}
	for index, option := range options {
		line := "  " + option
		if index == m.ConfirmIndex {
			line = m.paint(colorGreen, "> "+option)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsD4))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewDeploySummary() string {
	lines := []string{m.paint(colorGreen+colorBold, texts.DeploySummaryTitle)}
	if m.DeployResult == nil {
		return m.frame(lines)
	}
	lines = append(lines,
		replaceCount(texts.DeployDeployed, len(m.DeployResult.Deployed)),
		replaceCount(texts.DeployHybrids, len(m.DeployResult.Merged)),
		replaceCount(texts.DeployOmitted, len(m.DeployResult.Omitted)),
		replaceCount(texts.DeployUnchanged, len(m.DeployResult.Unchanged)),
		fmt.Sprintf("%s: %d", texts.NotInstalled, len(m.DeployResult.NotInstalled)),
	)
	if m.DeployResult.Snapshot != "" {
		lines = append(lines,
			replacePair(texts.DeploySnapshotPath, "{path}", m.DeployResult.Snapshot, "", ""),
			replacePair(texts.DeployUndo, "{path}", m.DeployResult.Snapshot, "", ""))
	} else {
		lines = append(lines, texts.DeployNoSnapshot)
	}
	for _, notice := range m.DeployResult.Plan.Notices {
		lines = append(lines, m.paint(colorYellow, "! "+notice))
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsD5))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewUpdate() string {
	lines := []string{m.paint(colorBold, texts.UpdateTitle)}
	if m.UpdateError != "" {
		lines = append(lines, m.paint(colorRed, m.UpdateError))
		lines = append(lines, "", m.paint(colorGray, texts.ShortcutsU1))
		return m.frame(lines)
	}
	if m.UpdateResult == nil {
		lines = append(lines, texts.UpdateChecking)
		return m.frame(lines)
	}
	switch m.UpdateResult.Outcome {
	case update.OutcomeUpdated:
		lines = append(lines, replaceTriple(texts.UpdateUpdated, m.UpdateResult.CurrentVersion, m.UpdateResult.LatestVersion, m.UpdateResult.Target))
		lines = append(lines, texts.UpdateRestart)
	case update.OutcomeNewer:
		lines = append(lines, replacePair(texts.UpdateNewer, "{current}", m.UpdateResult.CurrentVersion, "{latest}", m.UpdateResult.LatestVersion))
	default:
		lines = append(lines, replacePair(texts.UpdateUpToDate, "{current}", m.UpdateResult.CurrentVersion, "{latest}", m.UpdateResult.LatestVersion))
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsU1))
	return m.frame(lines)
}

func (m *Model) viewSettings() string {
	lines := []string{
		m.paint(colorBold, texts.SettingsTitle),
		m.paint(colorGray, texts.SettingsSubtitle),
		"",
		m.paint(colorGray, texts.SettingsCLINote),
	}
	labels := []string{texts.SettingsCatalogLabel, texts.SettingsCloneLabel, texts.SettingsSnapshotsLabel}
	hints := []string{texts.SettingsCatalogHint, texts.SettingsCloneHint, texts.SettingsSnapshotsHint}
	for index, label := range labels {
		marker := "  "
		if index == m.SettingsIndex {
			marker = "> "
		}
		lines = append(lines, marker+m.paint(colorBold, label))
		lines = append(lines, "    "+m.SettingsFields[index]+"_")
		lines = append(lines, "    "+m.paint(colorGray, hints[index]))
	}
	if m.SettingsError != "" {
		lines = append(lines, "", m.paint(colorRed, m.SettingsError))
	}
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsSettings))
	lines = append(lines, m.messageLine())
	return m.frame(lines)
}

func (m *Model) viewHelp() string {
	screen := m.HelpReturn
	shortcuts := helpForScreen(screen)
	steps, current := []string(nil), ""
	switch screen {
	case ScreenC1, ScreenC2, ScreenC3, ScreenC4:
		steps, current = texts.CloneSteps, cloneStep(m.HelpReturn)
	case ScreenD0, ScreenD1, ScreenD2, ScreenD3, ScreenD4, ScreenD5:
		steps, current = texts.DeploySteps, deployStep(m.HelpReturn)
	}
	body := texts.HelpTexts(helpScreenName(screen), shortcuts, steps, current, screen == ScreenC3 || screen == ScreenD2, true)
	lines := []string{m.paint(colorBold, texts.HelpTitle), ""}
	lines = append(lines, strings.Split(body, "\n")...)
	lines = append(lines, "", m.paint(colorGray, texts.ShortcutsHelp))
	return m.frame(lines)
}

func (m *Model) viewError() string {
	lines := []string{m.paint(colorRed+colorBold, m.Error.What)}
	if m.Error.Safety != "" {
		lines = append(lines, "", m.paint(colorYellow, texts.ErrorSafety+": "+m.Error.Safety))
	}
	if m.Error.Action != "" {
		lines = append(lines, texts.ErrorAction+": "+m.Error.Action)
	}
	if m.ErrorDetail && m.Error.Detail != "" {
		lines = append(lines, "", texts.ErrorDetail+":", m.Error.Detail)
	}
	lines = append(lines, "", m.paint(colorGray, texts.ErrorShowDetail), m.paint(colorGray, texts.ErrorBackToMenu+": Esc"))
	return m.frame(lines)
}

func (m *Model) messageLine() string {
	if m.Message == "" {
		return ""
	}
	return m.paint(colorYellow, m.Message)
}

func helpForScreen(screen Screen) [][2]string {
	switch screen {
	case ScreenMenu:
		return texts.HelpMenu
	case ScreenC1:
		return texts.HelpC1
	case ScreenC2:
		return texts.HelpC2
	case ScreenC3:
		return texts.HelpC3
	case ScreenC4:
		return texts.HelpC4
	case ScreenD0:
		return texts.HelpD0
	case ScreenD1:
		return texts.HelpD1
	case ScreenD2:
		return texts.HelpD2
	case ScreenD3:
		return texts.HelpD3
	case ScreenD4:
		return texts.HelpD4
	case ScreenD5:
		return texts.HelpD5
	case ScreenU1:
		return texts.HelpU1
	case ScreenSettings:
		return texts.HelpSettings
	}
	return nil
}

func helpScreenName(screen Screen) string {
	switch screen {
	case ScreenMenu:
		return texts.ScreenMenu
	case ScreenC1:
		return texts.ScreenC1
	case ScreenC2:
		return texts.ScreenC2
	case ScreenC3:
		return texts.ScreenC3
	case ScreenC4:
		return texts.ScreenC4
	case ScreenD0:
		return texts.ScreenD0
	case ScreenD1:
		return texts.ScreenD1
	case ScreenD2:
		return texts.ScreenD2
	case ScreenD3:
		return texts.ScreenD3
	case ScreenD4:
		return texts.ScreenD4
	case ScreenD5:
		return texts.ScreenD5
	case ScreenU1:
		return texts.ScreenU1
	case ScreenSettings:
		return texts.ScreenSettings
	}
	return string(screen)
}

func cloneStep(screen Screen) string {
	switch screen {
	case ScreenC1:
		return "apps"
	case ScreenC2:
		return "destino"
	case ScreenC3:
		return "revisión"
	}
	return ""
}

func deployStep(screen Screen) string {
	switch screen {
	case ScreenD0:
		return "clon"
	case ScreenD1:
		return "apps"
	case ScreenD2, ScreenD3:
		return "conflictos"
	case ScreenD4:
		return "confirmación"
	}
	return ""
}

func replaceCount(template string, count int) string {
	return strings.ReplaceAll(template, "{count}", fmt.Sprintf("%d", count))
}

func replacePair(template, key, value, secondKey, secondValue string) string {
	template = strings.ReplaceAll(template, key, value)
	if secondKey != "" {
		template = strings.ReplaceAll(template, secondKey, secondValue)
	}
	return template
}

func replaceTriple(template, current, latest, target string) string {
	replacer := strings.NewReplacer("{current}", current, "{latest}", latest, "{path}", target)
	return replacer.Replace(template)
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
