package tui

import (
	"runtime"
	"strings"

	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/install"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// installChoice is one missing app offered for installation. Nothing is
// installed unless the user marks it.
type installChoice struct {
	AppID    string
	Name     string
	Recipe   install.Recipe
	Selected bool
}

// beginInstallQuestion switches the apps screen into the install question when
// some missing app has a recipe for this Mac. It returns false when there is
// nothing to ask, so the deploy goes straight to the plan.
func (s *deployAppsScreen) beginInstallQuestion(selected []string) bool {
	candidates := s.app.deploySession.InstallCandidates(runtime.GOOS, runtime.GOARCH)
	if len(candidates) == 0 {
		return false
	}
	s.installChoices = make([]installChoice, 0, len(candidates))
	for _, candidate := range candidates {
		s.installChoices = append(s.installChoices, installChoice{
			AppID: candidate.AppID, Name: candidate.Name, Recipe: candidate.Recipe,
		})
	}
	s.installing = true
	s.installIndex = 0
	s.pendingSelected = selected
	s.messages = []Message{{Kind: MessageWarning, Text: texts.D1InstallQ}}
	return true
}

// updateInstall handles keys in the install question. Escape returns to the app
// choice; declining an app never cancels the deploy.
func (s *deployAppsScreen) updateInstall(keyMessage tea.KeyMsg) (Screen, tea.Cmd) {
	count := len(s.installChoices)
	switch keyMessage.String() {
	case "up", "k":
		if count > 0 {
			s.installIndex = (s.installIndex + count - 1) % count
		}
	case "down", "j":
		if count > 0 {
			s.installIndex = (s.installIndex + 1) % count
		}
	case " ":
		if count > 0 {
			choice := &s.installChoices[s.installIndex]
			choice.Selected = !choice.Selected
		}
	case "a":
		for index := range s.installChoices {
			s.installChoices[index].Selected = true
		}
	case "n":
		for index := range s.installChoices {
			s.installChoices[index].Selected = false
		}
	case "esc":
		s.installing = false
		s.messages = []Message{{Kind: MessageWarning, Text: texts.D1CloseApps}}
	case "enter":
		return s.confirmInstalls()
	}
	return s, nil
}

// confirmInstalls hands the chosen installs to the session and starts the plan,
// which runs them after the archive is verified and before classification.
func (s *deployAppsScreen) confirmInstalls() (Screen, tea.Cmd) {
	chosen := make([]string, 0)
	for _, choice := range s.installChoices {
		if choice.Selected {
			chosen = append(chosen, choice.AppID)
		}
	}
	s.installing = false
	s.app.deploySession.SetInstall(deploy.InstallSettings{
		AppIDs:    chosen,
		Installer: &install.Installer{Runner: s.app.installRunner, Home: s.app.deploySession.Home},
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	})
	selected := append(append([]string{}, s.pendingSelected...), chosen...)
	s.planning = true
	s.current = 0
	s.total = 0
	s.messages = []Message{{Kind: MessageWarning, Text: texts.D1Verifying}}
	return s, startDeployPlan(s.app, s.app.deploySession, selected)
}

func (s *deployAppsScreen) viewInstall(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.D1Title), "", "  " + texts.D1InstallQ, "  " + texts.D1InstallHint, ""}
	lineWidth := width - 8
	if lineWidth < 20 {
		lineWidth = 20
	}
	for index, choice := range s.installChoices {
		cursor := "  "
		style := s.app.theme.Styles.Secondary
		if index == s.installIndex {
			cursor = s.app.theme.Symbols.Cursor + " "
			style = s.app.theme.Styles.Primary
		}
		checkbox := "[ ]"
		if choice.Selected {
			checkbox = "[x]"
		}
		detail := strings.NewReplacer("{method}", choice.Recipe.Method).Replace(texts.D1InstallRow)
		if choice.Recipe.Method == install.MethodInstaller {
			detail += " · " + strings.NewReplacer("{url}", choice.Recipe.URL).Replace(texts.D1InstallFrom)
		}
		line := cursor + checkbox + "  " + choice.Name + "  " + detail
		lines = append(lines, style.Render(Truncate(line, lineWidth, s.app.theme.Symbols.Ellipsis)))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

func deployInstallKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{" "}, "Espacio", texts.KeyMark),
		bind([]string{"a", "n"}, "a/n", texts.KeyAllNone),
		bind([]string{"enter"}, "Enter", texts.KeyContinue),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}
