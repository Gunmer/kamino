package release

import (
	"fmt"
	"strings"
	"time"
)

// ChangelogMarker is the insertion point used by the generated changelog.
const ChangelogMarker = "<!-- version list -->"

// UnreleasedHeading marks a manual changelog section that is replaced by the
// next generated release entry.
const UnreleasedHeading = "## Sin publicar"

type changelogType struct {
	Heading string
	Bump    Bump
}

var changelogTypes = map[string]changelogType{
	"feat":     {"Funcionalidades", BumpMinor},
	"fix":      {"Correcciones", BumpPatch},
	"perf":     {"Rendimiento", BumpNone},
	"docs":     {"Documentación", BumpNone},
	"ci":       {"Integración continua", BumpNone},
	"build":    {"Construcción", BumpNone},
	"chore":    {"Mantenimiento", BumpNone},
	"refactor": {"Refactorización", BumpNone},
	"style":    {"Estilo de código", BumpNone},
	"test":     {"Pruebas", BumpNone},
}

// changelogOrder follows the alphabetical order of the English type keys used
// by the previous release tooling.
var changelogOrder = []string{
	"fix", "build", "chore", "style", "ci", "docs", "feat", "perf", "refactor", "test",
}

// RenderChangelogEntry renders one version section.
func RenderChangelogEntry(version string, date time.Time, commits []Commit, commitURL func(string) string) (string, error) {
	if _, err := ParseVersion(version); err != nil {
		return "", err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "## v%s (%s)\n", version, date.Format("2006-01-02"))
	for _, commitType := range changelogOrder {
		var entries []Commit
		for _, commit := range commits {
			if commit.Type == commitType {
				entries = append(entries, commit)
			}
		}
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(&builder, "\n### %s\n\n", changelogTypes[commitType].Heading)
		for _, commit := range entries {
			short := commit.SHA
			if len(short) > 7 {
				short = short[:7]
			}
			fmt.Fprintf(&builder, "- %s ([`%s`](%s))\n", commit.Description, short, commitURL(commit.SHA))
			if commit.Breaking {
				note := strings.Join(commit.BreakingNotes, " ")
				if note == "" {
					note = commit.Description
				}
				fmt.Fprintf(&builder, "\n  **Cambio incompatible.** %s\n", note)
			}
		}
	}
	return strings.TrimRight(builder.String(), "\n"), nil
}

// InsertChangelogEntry adds a version section after the marker and replaces a
// manual unreleased section if one exists.
func InsertChangelogEntry(previous, entry string) (string, error) {
	previous = stripUnreleased(previous)
	index := strings.Index(previous, ChangelogMarker)
	if index < 0 {
		return "", fmt.Errorf("el changelog no contiene el marcador %q", ChangelogMarker)
	}
	before := previous[:index+len(ChangelogMarker)]
	after := strings.TrimLeft(previous[index+len(ChangelogMarker):], "\n")
	result := before + "\n\n" + entry
	if after != "" {
		result += "\n\n" + after
	}
	return result + "\n", nil
}

func stripUnreleased(changelog string) string {
	index := strings.Index(changelog, UnreleasedHeading)
	if index < 0 {
		return changelog
	}
	rest := changelog[index+len(UnreleasedHeading):]
	next := strings.Index(rest, "\n## ")
	if next < 0 {
		return strings.TrimRight(changelog[:index], "\n") + "\n"
	}
	return changelog[:index] + rest[next+1:]
}
