package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/config"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/defaults"
	"github.com/Gunmer/kamino/internal/deploy"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/Gunmer/kamino/internal/update"
	"github.com/Gunmer/kamino/internal/version"
)

func (m *Model) selectedCloneIDs() []string {
	selected := make([]string, 0)
	for _, choice := range m.CloneApps {
		if choice.Selected && choice.Detected {
			selected = append(selected, choice.ID)
		}
	}
	return selected
}

func (m *Model) updateCloneApps(key Key) {
	visible := m.visibleCloneApps()
	if m.CloneFiltering {
		switch key.Kind {
		case KeyEsc:
			m.CloneFiltering = false
			m.CloneFilter = ""
		case KeyEnter:
			m.CloneFiltering = false
		case KeyBackspace:
			if len(m.CloneFilter) > 0 {
				runes := []rune(m.CloneFilter)
				m.CloneFilter = string(runes[:len(runes)-1])
			}
		case KeyRune:
			m.CloneFilter += string(key.Rune)
		}
		return
	}
	switch key.Kind {
	case KeyUp:
		if len(visible) > 0 {
			m.CloneIndex = (m.CloneIndex + len(visible) - 1) % len(visible)
		}
	case KeyDown:
		if len(visible) > 0 {
			m.CloneIndex = (m.CloneIndex + 1) % len(visible)
		}
	case KeyRune:
		switch key.Rune {
		case '/':
			m.CloneFiltering = true
		case ' ':
			m.toggleVisible(visible, m.CloneIndex)
		case 'a':
			for index := range m.CloneApps {
				if m.CloneApps[index].Detected && m.matchesCloneFilter(m.CloneApps[index]) {
					m.CloneApps[index].Selected = true
				}
			}
		case 'n':
			for index := range m.CloneApps {
				if m.matchesCloneFilter(m.CloneApps[index]) {
					m.CloneApps[index].Selected = false
				}
			}
		}
	case KeyEnter:
		if len(m.selectedCloneIDs()) == 0 {
			m.Message = texts.CloneNoSelection
			return
		}
		if m.CloneOutput == "" {
			m.Screen = ScreenC2
			if m.CloneOutput == "" {
				m.CloneOutput = filepath.Join(defaults.VaultDir(m.Home), clone.DefaultName(m.Now()))
			}
			return
		}
		m.planClone()
	case KeyEsc:
		m.Screen = ScreenMenu
		m.Stack = nil
	}
}

func (m *Model) visibleCloneApps() []appChoice {
	result := make([]appChoice, 0)
	for _, choice := range m.CloneApps {
		if m.matchesCloneFilter(choice) {
			result = append(result, choice)
		}
	}
	return result
}

func (m *Model) matchesCloneFilter(choice appChoice) bool {
	if m.CloneFilter == "" {
		return true
	}
	query := strings.ToLower(m.CloneFilter)
	return strings.Contains(strings.ToLower(choice.ID), query) || strings.Contains(strings.ToLower(choice.Name), query)
}

func (m *Model) toggleVisible(visible []appChoice, index int) {
	if len(visible) == 0 {
		return
	}
	target := visible[index%len(visible)]
	for position := range m.CloneApps {
		if m.CloneApps[position].ID == target.ID {
			m.CloneApps[position].Selected = !m.CloneApps[position].Selected
			return
		}
	}
}

func (m *Model) updateCloneDestination(key Key) {
	switch key.Kind {
	case KeyEsc:
		m.Screen = ScreenC1
	case KeyEnter:
		if strings.TrimSpace(m.CloneOutput) == "" {
			m.Message = "escribe una ruta para el clon"
			return
		}
		message := clone.ValidateDestination(m.CloneOutput, true)
		if message != "" {
			m.Message = message
			return
		}
		m.planClone()
	case KeyF2:
		if current, err := os.Getwd(); err == nil {
			m.CloneOutput = filepath.Join(current, clone.DefaultName(m.Now()))
		}
	case KeyF3:
		m.CloneOutput = filepath.Join(m.Home, "Downloads", clone.DefaultName(m.Now()))
	case KeyF4:
		m.CloneOutput = filepath.Join(m.Home, "Desktop", clone.DefaultName(m.Now()))
	case KeyBackspace:
		if len(m.CloneOutput) > 0 {
			runes := []rune(m.CloneOutput)
			m.CloneOutput = string(runes[:len(runes)-1])
		}
	case KeyRune:
		m.CloneOutput += string(key.Rune)
	}
}

func (m *Model) planClone() {
	selected := m.selectedCloneIDs()
	planned, err := clone.PlanClone(clone.PlanOptions{
		Home: m.Home, Selected: &selected, Output: m.CloneOutput, Catalog: m.Catalog,
		ApplicationDirs: m.ApplicationDirs, Path: m.PathValue, Now: m.Now(),
	})
	if err != nil {
		m.showError("no se pudo planificar el clon", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error())
		return
	}
	m.ClonePlan = planned
	m.CloneError = ""
	m.Screen = ScreenC3
}

func (m *Model) updateCloneReview(key Key) {
	switch key.Kind {
	case KeyEsc:
		m.Screen = ScreenC2
	case KeyRune:
		switch key.Rune {
		case 'd':
			m.CloneShowSkipped = !m.CloneShowSkipped
		case 'f':
			m.CloneShowFiles = !m.CloneShowFiles
		}
	case KeyCtrlC:
		if m.CloneWriting {
			m.Message = texts.StatusRenameStarted
			return
		}
		m.Message = texts.CloneCancelled
		m.ClonePlan = nil
		m.Screen = ScreenC1
	case KeyEnter:
		if m.ClonePlan == nil {
			return
		}
		if m.CloneDryRun {
			result := clone.CloneResult{Plan: m.ClonePlan.Plan, Manifest: m.ClonePlan.Manifest, DryRun: true, PlanOutput: m.ClonePlan.Output}
			m.CloneResult = &result
			m.Screen = ScreenC4
			return
		}
		m.CloneWriting = true
		result, err := clone.WriteClone(m.ClonePlan, m.Home, m.CloneForce, nil, nil, nil)
		m.CloneWriting = false
		if err != nil {
			m.showError("no se pudo escribir el clon", texts.ErrorCloneWritten, texts.StatusReviewDefault, err.Error())
			return
		}
		m.CloneResult = &result
		m.Screen = ScreenC4
	}
}

func (m *Model) updateCloneSummary(key Key) {
	switch key.Kind {
	case KeyEnter, KeyEsc:
		m.finishClone()
	case KeyRune:
		if key.Rune == 'c' {
			path := ""
			if m.CloneResult != nil {
				path = m.CloneResult.Output
			}
			m.copy(path)
		}
	}
}

func (m *Model) finishClone() {
	m.ExitCode = 0
	if m.CloneResult != nil && len(m.CloneResult.Plan.Skipped) > 0 {
		m.ExitCode = 3
	}
	m.Quit = true
}

func (m *Model) updateDeployFile(key Key) {
	if m.DeployManual || m.DeployFiltering {
		field := &m.DeployFile
		if m.DeployFiltering {
			field = &m.DeployFilter
		}
		switch key.Kind {
		case KeyEsc:
			m.DeployManual = false
			m.DeployFiltering = false
			m.DeployFilter = ""
		case KeyEnter:
			if m.DeployFiltering {
				m.DeployFiltering = false
				return
			}
			m.DeployManual = false
			m.openDeployFile(*field)
		case KeyBackspace:
			if len(*field) > 0 {
				runes := []rune(*field)
				*field = string(runes[:len(runes)-1])
			}
		case KeyRune:
			*field += string(key.Rune)
		}
		return
	}
	files := m.visibleDeployFiles()
	switch key.Kind {
	case KeyUp:
		if len(files) > 0 {
			m.DeployIndex = (m.DeployIndex + len(files) - 1) % len(files)
		}
	case KeyDown:
		if len(files) > 0 {
			m.DeployIndex = (m.DeployIndex + 1) % len(files)
		}
	case KeyRune:
		switch key.Rune {
		case '/':
			m.DeployFiltering = true
		case 'o':
			m.DeployManual = true
			m.DeployFile = ""
		}
	case KeyEnter:
		if len(files) > 0 {
			m.openDeployFile(files[m.DeployIndex%len(files)])
		} else {
			m.Message = texts.DeployNoFile
		}
	case KeyEsc:
		m.Screen = ScreenMenu
		m.Stack = nil
	}
}

func (m *Model) visibleDeployFiles() []string {
	if m.DeployFilter == "" {
		return m.DeployFiles
	}
	result := make([]string, 0)
	for _, path := range m.DeployFiles {
		if strings.Contains(strings.ToLower(path), strings.ToLower(m.DeployFilter)) {
			result = append(result, path)
		}
	}
	return result
}

func (m *Model) updateDeployApps(key Key) {
	switch key.Kind {
	case KeyUp:
		if len(m.DeployApps) > 0 {
			m.DeployIndex = (m.DeployIndex + len(m.DeployApps) - 1) % len(m.DeployApps)
		}
	case KeyDown:
		if len(m.DeployApps) > 0 {
			m.DeployIndex = (m.DeployIndex + 1) % len(m.DeployApps)
		}
	case KeyRune:
		switch key.Rune {
		case ' ':
			if len(m.DeployApps) > 0 {
				m.DeployApps[m.DeployIndex%len(m.DeployApps)].Selected = !m.DeployApps[m.DeployIndex%len(m.DeployApps)].Selected
			}
		case 'a':
			for index := range m.DeployApps {
				m.DeployApps[index].Selected = true
			}
		case 'n':
			for index := range m.DeployApps {
				m.DeployApps[index].Selected = false
			}
		}
	case KeyEnter:
		selected := make([]string, 0)
		for _, choice := range m.DeployApps {
			if choice.Selected {
				selected = append(selected, choice.ID)
			}
		}
		if len(selected) == 0 {
			m.Message = texts.DeployNoneSelected
			return
		}
		if m.DeploySession == nil {
			return
		}
		plan, err := m.DeploySession.Plan(&selected, false, selected, nil, nil)
		if err != nil {
			m.showError("no se pudo preparar el despliegue", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error())
			return
		}
		m.DeployPlan = plan
		m.Conflicts = make([]conflictState, 0, len(plan.Conflicts))
		for _, conflict := range plan.Conflicts {
			rows := make([]conflicts.RowDecision, conflicts.RowCount(conflict))
			for index := range rows {
				rows[index] = conflicts.RowClone
			}
			m.Conflicts = append(m.Conflicts, conflictState{Conflict: conflict, Action: conflicts.ActionReview, Rows: rows})
		}
		m.ConflictIndex = 0
		m.Screen = ScreenD2
	case KeyEsc:
		m.Screen = ScreenD0
	}
}

func (m *Model) updateDeployConflicts(key Key) {
	switch key.Kind {
	case KeyUp:
		if len(m.Conflicts) > 0 {
			m.ConflictIndex = (m.ConflictIndex + len(m.Conflicts) - 1) % len(m.Conflicts)
		}
	case KeyDown:
		if len(m.Conflicts) > 0 {
			m.ConflictIndex = (m.ConflictIndex + 1) % len(m.Conflicts)
		}
	case KeyRune:
		switch key.Rune {
		case 'c':
			m.setConflictAction(m.ConflictIndex, conflicts.ActionApply)
		case 'o':
			m.setConflictAction(m.ConflictIndex, conflicts.ActionOmit)
		case 'C':
			for index := range m.Conflicts {
				m.setConflictAction(index, conflicts.ActionApply)
			}
		case 'O':
			for index := range m.Conflicts {
				m.setConflictAction(index, conflicts.ActionOmit)
			}
		case 'r':
			if len(m.Conflicts) > 0 {
				m.Conflicts[m.ConflictIndex].Action = conflicts.ActionReview
				m.Screen = ScreenD3
			}
		}
	case KeyEnter:
		if len(m.Conflicts) > 0 {
			m.Screen = ScreenD4
			m.ConfirmIndex = 0
		} else if m.DeployPlan != nil {
			m.Screen = ScreenD4
			m.ConfirmIndex = 0
		}
	case KeyEsc:
		m.Screen = ScreenD1
	}
}

func (m *Model) setConflictAction(index int, action conflicts.FileAction) {
	if index < 0 || index >= len(m.Conflicts) {
		return
	}
	m.Conflicts[index].Action = action
	if action == conflicts.ActionReview && len(m.Conflicts[index].Rows) == 0 {
		for row := 0; row < conflicts.RowCount(m.Conflicts[index].Conflict); row++ {
			m.Conflicts[index].Rows = append(m.Conflicts[index].Rows, conflicts.RowClone)
		}
	}
}

func (m *Model) updateDeployDetail(key Key) {
	if len(m.Conflicts) == 0 {
		m.Screen = ScreenD2
		return
	}
	state := &m.Conflicts[m.ConflictIndex%len(m.Conflicts)]
	if len(state.Conflict.Hunks) == 0 {
		m.Screen = ScreenD2
		return
	}
	decidable := decidableRows(state, state.HunkIndex)
	switch key.Kind {
	case KeyUp:
		if len(decidable) > 0 {
			state.RowIndex = (state.RowIndex + len(decidable) - 1) % len(decidable)
		}
	case KeyDown:
		if len(decidable) > 0 {
			state.RowIndex = (state.RowIndex + 1) % len(decidable)
		}
	case KeyTab:
		state.HunkIndex = (state.HunkIndex + 1) % len(state.Conflict.Hunks)
		state.RowIndex = 0
	case KeyShiftTab:
		state.HunkIndex = (state.HunkIndex + len(state.Conflict.Hunks) - 1) % len(state.Conflict.Hunks)
		state.RowIndex = 0
	case KeyRune:
		switch key.Rune {
		case ' ':
			m.toggleDetailRow(state)
		case 'c':
			m.setHunkSide(state, state.Conflict.Hunks[state.HunkIndex%len(state.Conflict.Hunks)], conflicts.RowClone)
		case 'o':
			m.setHunkSide(state, state.Conflict.Hunks[state.HunkIndex%len(state.Conflict.Hunks)], conflicts.RowLocal)
		case 'C':
			for index := range state.Rows {
				state.Rows[index] = conflicts.RowClone
			}
		case 'O':
			for index := range state.Rows {
				state.Rows[index] = conflicts.RowLocal
			}
		}
	case KeyEnter:
		m.toggleDetailRow(state)
	case KeyEsc:
		m.Screen = ScreenD2
	}
}

func (m *Model) toggleDetailRow(state *conflictState) {
	decidable := decidableRows(state, state.HunkIndex)
	if len(decidable) == 0 {
		return
	}
	rowIndex := decidable[state.RowIndex%len(decidable)]
	if rowIndex < 0 || rowIndex >= len(state.Rows) {
		return
	}
	if state.Rows[rowIndex] == conflicts.RowClone {
		state.Rows[rowIndex] = conflicts.RowLocal
	} else {
		state.Rows[rowIndex] = conflicts.RowClone
	}
}

func (m *Model) setHunkSide(state *conflictState, hunk conflicts.Hunk, side conflicts.RowDecision) {
	decidable := decidableRows(state, state.HunkIndex)
	for _, rowIndex := range decidable {
		if rowIndex >= 0 && rowIndex < len(state.Rows) {
			state.Rows[rowIndex] = side
		}
	}
}

func decidableRows(state *conflictState, hunkIndex int) []int {
	index := 0
	offset := 0
	result := make([]int, 0)
	for position, hunk := range state.Conflict.Hunks {
		for _, row := range hunk.Rows {
			if row.Kind == conflicts.RowEqual {
				continue
			}
			if position == hunkIndex%len(state.Conflict.Hunks) {
				result = append(result, offset+index)
			}
			index++
		}
	}
	return result
}

func (m *Model) updateDeployConfirm(key Key) {
	switch key.Kind {
	case KeyUp:
		m.ConfirmIndex = (m.ConfirmIndex + 2) % 3
	case KeyDown:
		m.ConfirmIndex = (m.ConfirmIndex + 1) % 3
	case KeyEsc:
		m.Screen = ScreenD2
	case KeyEnter:
		switch m.ConfirmIndex {
		case 0:
			m.Screen = ScreenD2
		case 1:
			m.Screen = ScreenMenu
			m.Stack = nil
		case 2:
			m.applyDeploy()
		}
	}
}

func (m *Model) applyDeploy() {
	if m.DeploySession == nil || m.DeployPlan == nil {
		return
	}
	decisions := conflicts.ConflictDecisions{Decisions: map[string]conflicts.FileDecision{}}
	for _, state := range m.Conflicts {
		rows := make([]conflicts.RowDecision, len(state.Rows))
		copy(rows, state.Rows)
		decisions.Set(state.Conflict.Path, conflicts.FileDecision{Action: state.Action, Rows: rows})
	}
	m.DeployCannotCancel = true
	result, err := m.DeploySession.Apply(decisions, nil, nil)
	m.DeployCannotCancel = false
	if err != nil {
		var partial *kerrors.PartialApplyError
		if errors.As(err, &partial) {
			safety := texts.ErrorSnapshotWritten
			if partial.Snapshot == "" {
				safety = texts.ErrorNothingWritten
			}
			m.showError("despliegue incompleto", safety, texts.StatusReviewDefault, err.Error())
			return
		}
		m.showError("no se pudo aplicar el despliegue", texts.ErrorNothingWritten, texts.StatusReviewDefault, err.Error())
		return
	}
	m.DeployResult = &result
	m.Screen = ScreenD5
}

func (m *Model) updateDeploySummary(key Key) {
	switch key.Kind {
	case KeyEnter, KeyEsc:
		m.ExitCode = 0
		if m.DeployResult != nil {
			m.ExitCode = deploy.ResultCode(*m.DeployResult)
		}
		m.Quit = true
	case KeyRune:
		if key.Rune == 'c' && m.DeployResult != nil {
			m.copy(m.DeployResult.Snapshot)
		}
	}
}

func (m *Model) updateUpdate(key Key) {
	switch key.Kind {
	case KeyEsc:
		m.Screen = ScreenMenu
	case KeyEnter:
		if m.UpdateResult == nil {
			m.Screen = ScreenMenu
			return
		}
		if m.UpdateCheck.ShouldUpdate && m.UpdateRelease != nil {
			result, err := update.RunUpdate(version.Version, m.UpdateCheck.Target, m.UpdateRelease, nil, nil)
			if err != nil {
				m.showError("no se pudo actualizar", texts.ErrorCloneWritten, texts.StatusReviewDefault, err.Error())
				return
			}
			m.UpdateResult = &result
			return
		}
		m.Screen = ScreenMenu
	}
}

func (m *Model) updateSettings(key Key) {
	field := &m.SettingsFields[m.SettingsIndex]
	switch key.Kind {
	case KeyUp, KeyShiftTab:
		m.SettingsIndex = (m.SettingsIndex + 2) % 3
	case KeyDown, KeyTab, KeyEnter:
		m.SettingsIndex = (m.SettingsIndex + 1) % 3
	case KeyEsc:
		m.Screen = ScreenMenu
	case KeyCtrlS:
		m.saveSettings()
	case KeyBackspace:
		if len(*field) > 0 {
			runes := []rune(*field)
			*field = string(runes[:len(runes)-1])
		}
	case KeyRune:
		*field += string(key.Rune)
	}
}

func (m *Model) saveSettings() {
	values := make([]string, 3)
	for index, field := range m.SettingsFields {
		resolved, err := config.ResolvePath(field, m.Home, []string{config.CatalogKey, config.CloneKey, config.SnapshotsKey}[index])
		if err != nil {
			m.SettingsError = err.Error()
			return
		}
		values[index] = resolved
	}
	value := config.AppConfig{CatalogDir: values[0], CloneOutputDir: values[1], SnapshotsDir: values[2]}
	if _, err := config.SaveConfig(value, m.Home, ""); err != nil {
		m.SettingsError = err.Error()
		return
	}
	m.Settings = value
	m.SettingsError = ""
	m.Message = texts.SettingsSaved
}

func (m *Model) copy(value string) {
	if value == "" {
		return
	}
	m.Clipboard = value
	m.Message = texts.CloneCopyPath
}

func (m *Model) startRealUpdate() {
	check, err := update.QueryUpdate(version.Version, "", nil)
	if err != nil {
		m.UpdateError = err.Error()
		return
	}
	m.UpdateCheck = check
	m.UpdateRelease = &check.Release
	m.UpdateResult = &update.Result{
		CurrentVersion: check.CurrentVersion,
		LatestVersion:  check.Release.Version,
		Target:         check.Target,
		Updated:        false,
		Outcome:        check.Outcome,
	}
}
