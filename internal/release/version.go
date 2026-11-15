package release

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Bump is the semantic version increment requested by a set of commits.
type Bump int

const (
	BumpNone Bump = iota
	BumpPatch
	BumpMinor
	BumpMajor
)

var (
	tagPattern     = regexp.MustCompile(`^v(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$`)
	versionPattern = regexp.MustCompile(`^var Version = "(.*)"$`)
	sectionPattern = regexp.MustCompile(`(?m)^## (v\d+\.\d+\.\d+) \([^\n]+\)$`)
)

// ParseVersion parses a strict X.Y.Z version.
func ParseVersion(value string) ([3]int, error) {
	var parsed [3]int
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return parsed, fmt.Errorf("versión inválida: %q", value)
	}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return parsed, fmt.Errorf("versión inválida: %q", value)
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return parsed, fmt.Errorf("versión inválida: %q", value)
		}
		parsed[index] = number
	}
	return parsed, nil
}

// ValidateTag checks a vX.Y.Z tag and returns its version.
func ValidateTag(tag string) (string, error) {
	if !tagPattern.MatchString(tag) {
		return "", fmt.Errorf("el tag debe tener el formato vX.Y.Z: %q", tag)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

// NextVersion computes the next version from the current one and the pending commits.
func NextVersion(current string, commits []Commit) (string, Bump, error) {
	parsed, err := ParseVersion(current)
	if err != nil {
		return "", BumpNone, err
	}

	bump := BumpNone
	for _, commit := range commits {
		if commit.Breaking {
			if parsed[0] == 0 {
				bump = maxBump(bump, BumpMinor)
			} else {
				bump = maxBump(bump, BumpMajor)
			}
			continue
		}
		if metadata, ok := changelogTypes[commit.Type]; ok {
			bump = maxBump(bump, metadata.Bump)
		}
	}

	switch bump {
	case BumpMajor:
		parsed[0]++
		parsed[1] = 0
		parsed[2] = 0
	case BumpMinor:
		parsed[1]++
		parsed[2] = 0
	case BumpPatch:
		parsed[2]++
	}
	return fmt.Sprintf("%d.%d.%d", parsed[0], parsed[1], parsed[2]), bump, nil
}

func maxBump(left, right Bump) Bump {
	if right > left {
		return right
	}
	return left
}

// ReleaseNotes extracts exactly one version section from a changelog.
func ReleaseNotes(changelog, tag string) (string, error) {
	if _, err := ValidateTag(tag); err != nil {
		return "", err
	}
	sections := sectionPattern.FindAllStringSubmatchIndex(changelog, -1)
	var matches []int
	for _, section := range sections {
		if changelog[section[2]:section[3]] == tag {
			matches = append(matches, section[0])
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("el changelog debe contener exactamente una entrada para %s", tag)
	}
	start := matches[0]
	end := len(changelog)
	for _, section := range sections {
		if section[0] > start {
			end = section[0]
			break
		}
	}
	return strings.TrimSpace(changelog[start:end]) + "\n", nil
}

// WriteVersionFile updates the version variable in a Go source file.
func WriteVersionFile(path, version string) error {
	if _, err := ParseVersion(version); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`(?m)^var Version = ".*"$`)
	if len(pattern.FindAll(data, -1)) != 1 {
		return fmt.Errorf("no se encontró una única declaración var Version en %s", path)
	}
	updated := pattern.ReplaceAll(data, []byte(`var Version = "`+version+`"`))
	return os.WriteFile(path, updated, 0o644)
}

// ReadVersionFile reads the version variable from a Go source file.
func ReadVersionFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	line := ""
	for _, candidate := range strings.Split(string(data), "\n") {
		if versionPattern.MatchString(candidate) {
			if line != "" {
				return "", fmt.Errorf("no se encontró una única declaración var Version en %s", path)
			}
			line = candidate
		}
	}
	if line == "" {
		return "", fmt.Errorf("no se encontró var Version en %s", path)
	}
	version := versionPattern.FindStringSubmatch(line)[1]
	if _, err := ParseVersion(version); err != nil {
		return "", err
	}
	return version, nil
}
