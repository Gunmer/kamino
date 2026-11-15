// Package release automates versioning, changelog updates, builds and publication.
package release

import (
	"regexp"
	"strings"
)

// Commit is one parsed Conventional Commit.
type Commit struct {
	SHA           string
	Type          string
	Scope         string
	Description   string
	Breaking      bool
	BreakingNotes []string
}

var (
	headerPattern   = regexp.MustCompile(`^([A-Za-z]+)(?:\(([^)]*)\))?(!)?:\s+(.+)$`)
	breakingPattern = regexp.MustCompile(`^BREAKING[ -]CHANGE:\s*(.*)$`)
)

// ParseCommit converts one raw commit message into one or more Conventional
// Commits. Squash commits can contain several headers in the body.
func ParseCommit(sha, message string) []Commit {
	message = strings.ReplaceAll(message, "\r\n", "\n")
	lines := strings.Split(message, "\n")
	if len(lines) == 0 || strings.HasPrefix(strings.TrimSpace(lines[0]), "Merge ") {
		return nil
	}

	var commits []Commit
	var current *Commit
	var body []string

	flush := func() {
		if current == nil {
			return
		}
		if notes := breakingNotes(body); len(notes) > 0 {
			current.Breaking = true
			current.BreakingNotes = notes
		}
		commits = append(commits, *current)
		current = nil
		body = nil
	}

	for _, line := range lines {
		if match := headerPattern.FindStringSubmatch(line); match != nil {
			flush()
			current = &Commit{
				SHA:         sha,
				Type:        strings.ToLower(match[1]),
				Scope:       match[2],
				Description: strings.TrimSpace(match[4]),
				Breaking:    match[3] == "!",
			}
			continue
		}
		if current != nil {
			body = append(body, line)
		}
	}
	flush()
	return commits
}

func breakingNotes(lines []string) []string {
	var notes []string
	for index := 0; index < len(lines); index++ {
		match := breakingPattern.FindStringSubmatch(lines[index])
		if match == nil {
			continue
		}
		note := strings.TrimSpace(match[1])
		for next := index + 1; next < len(lines); next++ {
			line := lines[next]
			if strings.TrimSpace(line) == "" {
				break
			}
			if headerPattern.MatchString(line) || breakingPattern.MatchString(line) {
				break
			}
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				break
			}
			note = strings.TrimSpace(note + " " + strings.TrimSpace(line))
			index = next
		}
		if note != "" {
			notes = append(notes, note)
		}
	}
	return notes
}
