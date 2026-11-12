// Package ghostty interprets the bounded Ghostty configuration format.
package ghostty

import (
	"path"
	"strings"

	"github.com/Gunmer/kamino/internal/paths"
)

const (
	GhosttyAppID  = "ghostty"
	MaxConfigSize = 1024 * 1024
)

var ConfigFilenames = map[string]bool{"config": true, "config.ghostty": true}

// IterDirectives yields key/value pairs ignoring comments and blank lines.
func IterDirectives(text string) [][2]string {
	result := make([][2]string, 0)
	for _, raw := range splitLines(text) {
		line := strings.TrimSpace(StripComment(raw))
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		result = append(result, [2]string{strings.TrimSpace(key), strings.TrimSpace(value)})
	}
	return result
}

// ConfigReference returns the config-file path and whether it was optional.
func ConfigReference(value string) (string, bool) {
	raw := Unquote(strings.TrimSpace(value))
	optional := strings.HasPrefix(raw, "?")
	if optional {
		raw = Unquote(strings.TrimSpace(raw[1:]))
	}
	return raw, optional
}

// ThemeReferences returns the themes of a simple or light/dark value.
func ThemeReferences(value string) []string {
	result := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		candidate := strings.TrimSpace(part)
		if prefix, rest, ok := strings.Cut(candidate, ":"); ok {
			if trimmed := strings.TrimSpace(prefix); trimmed == "light" || trimmed == "dark" {
				candidate = strings.TrimSpace(rest)
			}
		}
		candidate = Unquote(candidate)
		if candidate != "" {
			result = append(result, candidate)
		}
	}
	return result
}

// IsPathReference reports whether a theme value looks like a path.
func IsPathReference(value string) bool {
	return strings.HasPrefix(value, "/") || strings.Contains(value, "/")
}

// ResolveReference resolves a reference to a home-relative POSIX path.
func ResolveReference(raw, baseRel, home string) (string, bool) {
	candidate := Unquote(strings.TrimSpace(raw))
	if candidate == "" || strings.HasPrefix(candidate, "~") {
		return "", false
	}
	if strings.HasPrefix(candidate, "/") {
		rel, err := paths.ToRelative(candidate, home)
		if err != nil {
			return "", false
		}
		return rel, true
	}
	base := path.Dir(baseRel)
	resolved := path.Clean(path.Join(base, candidate))
	if resolved == "" || resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", false
	}
	return resolved, true
}

// AbsoluteReferences returns absolute references declared in a config file.
func AbsoluteReferences(text string) []string {
	references := make([]string, 0)
	for _, directive := range IterDirectives(text) {
		key, value := directive[0], directive[1]
		if key == "config-file" {
			raw, _ := ConfigReference(value)
			if strings.HasPrefix(raw, "/") {
				references = append(references, raw)
			}
		} else if key == "theme" {
			for _, raw := range ThemeReferences(value) {
				if strings.HasPrefix(raw, "/") {
					references = append(references, raw)
				}
			}
		}
	}
	return references
}

// StripComment removes an unquoted trailing comment.
func StripComment(line string) string {
	var quote rune
	for index, character := range line {
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
		} else if character == '#' {
			return line[:index]
		}
	}
	return line
}

// Unquote removes one matching pair of single or double quotes.
func Unquote(value string) string {
	runes := []rune(value)
	if len(runes) >= 2 && runes[0] == runes[len(runes)-1] && (runes[0] == '\'' || runes[0] == '"') {
		return string(runes[1 : len(runes)-1])
	}
	return value
}

func splitLines(text string) []string {
	// Ghostty config is line-oriented; Python splitlines is sufficient here.
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}
