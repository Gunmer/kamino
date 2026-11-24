package tui

import (
	"runtime"
	"strings"

	"github.com/Gunmer/kamino/internal/implant"
	"github.com/Gunmer/kamino/internal/install"
	"github.com/Gunmer/kamino/internal/texts"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// installChoice is one missing app offered for installation, with every way to
// install it that can run on this Mac. Nothing is installed unless the user
// confirms it.
type installChoice struct {
	AppID   string
	Name    string
	Recipes []install.Recipe
	// Option is the index in Recipes of the way chosen to install the app.
	Option   int
	Selected bool
}

// installRow addresses one line of the confirmation: an app (option -1) or one
// of its ways to install.
type installRow struct{ choice, option int }

func (s *implantAppsScreen) installRows() []installRow {
	rows := make([]installRow, 0, len(s.installChoices)*2)
	for index, choice := range s.installChoices {
		rows = append(rows, installRow{index, -1})
		for option := range choice.Recipes {
			rows = append(rows, installRow{index, option})
		}
	}
	return rows
}

// beginInstallQuestion switches the apps screen into the install confirmation
// for the missing apps the user marked. It returns false when none of them can
// be installed, so the implant goes straight to the plan.
func (s *implantAppsScreen) beginInstallQuestion(selected, missing []string) bool {
	marked := map[string]bool{}
	for _, id := range missing {
		marked[id] = true
	}
	s.installChoices = make([]installChoice, 0, len(missing))
	for _, candidate := range s.app.implantSession.InstallCandidates(runtime.GOOS, runtime.GOARCH, s.app.installRunner) {
		if marked[candidate.AppID] {
			s.installChoices = append(s.installChoices, installChoice{
				AppID: candidate.AppID, Name: candidate.Name, Recipes: candidate.Recipes, Selected: true,
			})
		}
	}
	if len(s.installChoices) == 0 {
		return false
	}
	s.installing = true
	s.installIndex = 0
	s.pendingSelected = selected
	s.messages = []Message{{Kind: MessageWarning, Text: texts.D1InstallQ}}
	return true
}

// updateInstall handles keys in the install confirmation. Escape returns to the
// app choice; declining an app never cancels the implant.
func (s *implantAppsScreen) updateInstall(keyMessage tea.KeyMsg) (Screen, tea.Cmd) {
	rows := s.installRows()
	count := len(rows)
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
			row := rows[s.installIndex]
			choice := &s.installChoices[row.choice]
			if row.option < 0 {
				choice.Selected = !choice.Selected
			} else {
				choice.Option, choice.Selected = row.option, true
			}
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
func (s *implantAppsScreen) confirmInstalls() (Screen, tea.Cmd) {
	chosen := make([]string, 0)
	recipes := map[string]install.Recipe{}
	for _, choice := range s.installChoices {
		if choice.Selected {
			chosen = append(chosen, choice.AppID)
			recipes[choice.AppID] = choice.Recipes[choice.Option]
		}
	}
	s.installing = false
	s.app.implantSession.SetInstall(implant.InstallSettings{
		AppIDs:    chosen,
		Chosen:    recipes,
		Installer: &install.Installer{Runner: s.app.installRunner, Home: s.app.implantSession.Home},
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	})
	selected := append(append([]string{}, s.pendingSelected...), chosen...)
	if len(selected) == 0 {
		s.messages = []Message{{Kind: MessageWarning, Text: texts.D1NoneSelected}}
		return s, nil
	}
	s.planning = true
	s.current = 0
	s.total = 0
	s.messages = []Message{{Kind: MessageWarning, Text: texts.D1Verifying}}
	return s, startImplantPlan(s.app, s.app.implantSession, selected)
}

func (s *implantAppsScreen) viewInstall(width, height int) string {
	lines := []string{"  " + s.app.theme.Styles.Primary.Render(texts.D1Title), "", "  " + texts.D1InstallQ, "  " + texts.D1InstallHint, ""}
	lineWidth := width - 8
	if lineWidth < 20 {
		lineWidth = 20
	}
	for index, row := range s.installRows() {
		choice := s.installChoices[row.choice]
		cursor := "  "
		style := s.app.theme.Styles.Secondary
		if index == s.installIndex {
			cursor = s.app.theme.Symbols.Cursor + " "
			style = s.app.theme.Styles.Primary
		}
		var line string
		if row.option < 0 {
			checkbox := "[ ]"
			if choice.Selected {
				checkbox = "[x]"
			}
			line = cursor + checkbox + "  " + choice.Name
		} else {
			radio := "( )"
			if row.option == choice.Option {
				radio = "(•)"
			}
			line = cursor + "     " + radio + " " + installWayLabel(choice.Recipes[row.option])
		}
		lines = append(lines, style.Render(Truncate(line, lineWidth, s.app.theme.Symbols.Ellipsis)))
	}
	return Fit(lines, width, height, s.app.theme.Symbols.Ellipsis)
}

// installWayLabel describes one way to install an app: its method and, when it
// downloads something, where from and whether the download can be verified.
func installWayLabel(recipe install.Recipe) string {
	label := strings.NewReplacer("{method}", recipe.Method).Replace(texts.D1InstallRow)
	if method := recipe.Method; method == install.MethodInstaller || method == install.MethodGithubRelease {
		label += " · " + strings.NewReplacer("{url}", recipe.Source()).Replace(texts.D1InstallFrom)
	}
	if recipe.Unverified() {
		label += " · " + texts.D1InstallNoSum
	}
	return label
}

func implantInstallKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{" "}, "Espacio", texts.KeyMark),
		bind([]string{"a", "n"}, "a/n", texts.KeyAllNone),
		bind([]string{"enter"}, "Enter", texts.KeyContinue),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}
