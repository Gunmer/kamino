package plan

import "strings"

// fnmatch implements the subset of Python fnmatch used by the catalog.
// Unlike path.Match, '*' also matches '/'.
func fnmatch(name, pattern string) bool {
	if pattern == "" {
		return name == ""
	}
	// Iterative glob with backtracking for '*'.
	var nameIndex, patternIndex int
	starPattern := -1
	starName := -1
	for nameIndex < len(name) {
		if patternIndex < len(pattern) {
			switch pattern[patternIndex] {
			case '*':
				starPattern = patternIndex
				starName = nameIndex
				patternIndex++
				continue
			case '?':
				patternIndex++
				nameIndex++
				continue
			case '[':
				matched, next, ok := matchClass(name[nameIndex], pattern[patternIndex:])
				if ok {
					if matched {
						patternIndex += next
						nameIndex++
						continue
					}
				} else if name[nameIndex] == '[' {
					patternIndex++
					nameIndex++
					continue
				}
			default:
				if pattern[patternIndex] == name[nameIndex] {
					patternIndex++
					nameIndex++
					continue
				}
			}
		}
		if starPattern >= 0 {
			starName++
			nameIndex = starName
			patternIndex = starPattern + 1
			continue
		}
		return false
	}
	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}

// matchClass matches one character against a [...] class. It returns whether
// the character matched, the index after the class and whether the class was
// well formed.
func matchClass(character byte, pattern string) (bool, int, bool) {
	index := 1
	negated := false
	if index < len(pattern) && (pattern[index] == '!' || pattern[index] == '^') {
		negated = true
		index++
	}
	matched := false
	first := true
	for index < len(pattern) {
		if pattern[index] == ']' && !first {
			if negated {
				matched = !matched
			}
			return matched, index + 1, true
		}
		first = false
		if index+2 < len(pattern) && pattern[index+1] == '-' && pattern[index+2] != ']' {
			start, end := pattern[index], pattern[index+2]
			if start <= character && character <= end {
				matched = true
			}
			index += 3
			continue
		}
		if pattern[index] == character {
			matched = true
		}
		index++
	}
	return false, 0, false
}

func matchesPath(value, pattern string) bool {
	if fnmatch(value, pattern) {
		return true
	}
	return strings.HasPrefix(pattern, "**/") && fnmatch(value, pattern[3:])
}
