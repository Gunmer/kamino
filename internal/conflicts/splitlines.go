package conflicts

import "strings"

func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', '\x1c', '\x1d', '\x1e', '\x85', '\u2028', '\u2029':
		return true
	}
	return false
}

// splitLinesKeepEnds mirrors Python str.splitlines(keepends=True).
func splitLinesKeepEnds(text string) []string {
	if text == "" {
		return nil
	}
	runes := []rune(text)
	lines := make([]string, 0)
	start := 0
	for index := 0; index < len(runes); index++ {
		if !isLineBreak(runes[index]) {
			continue
		}
		end := index + 1
		if runes[index] == '\r' && index+1 < len(runes) && runes[index+1] == '\n' {
			end = index + 2
			index++
		}
		lines = append(lines, string(runes[start:end]))
		start = end
	}
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}
	return lines
}

// splitLines mirrors Python str.splitlines() without endings.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	runes := []rune(text)
	lines := make([]string, 0)
	start := 0
	for index := 0; index < len(runes); index++ {
		if !isLineBreak(runes[index]) {
			continue
		}
		lines = append(lines, string(runes[start:index]))
		if runes[index] == '\r' && index+1 < len(runes) && runes[index+1] == '\n' {
			index++
		}
		start = index + 1
	}
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}
	return lines
}

func normaliseNewlines(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// UnifiedDiff mirrors difflib.unified_diff with lineterm="".
func UnifiedDiff(a, b []string, fromFile, toFile string) []string {
	matcher := newSequenceMatcher(a, b, true)
	output := make([]string, 0)
	first := true
	for _, group := range matcher.GetGroupedOpcodes(3) {
		if !first {
			output = append(output, "")
		}
		first = false
		firstOpcode := group[0]
		lastOpcode := group[len(group)-1]
		output = append(output, "--- "+fromFile)
		output = append(output, "+++ "+toFile)
		output = append(output, "@@ -"+formatRangeUnified(firstOpcode.I1, lastOpcode.I2)+" +"+formatRangeUnified(firstOpcode.J1, lastOpcode.J2)+" @@")
		for _, opcode := range group {
			switch opcode.Tag {
			case "equal":
				for _, line := range a[opcode.I1:opcode.I2] {
					output = append(output, " "+line)
				}
			case "replace":
				for _, line := range a[opcode.I1:opcode.I2] {
					output = append(output, "-"+line)
				}
				for _, line := range b[opcode.J1:opcode.J2] {
					output = append(output, "+"+line)
				}
			case "delete":
				for _, line := range a[opcode.I1:opcode.I2] {
					output = append(output, "-"+line)
				}
			case "insert":
				for _, line := range b[opcode.J1:opcode.J2] {
					output = append(output, "+"+line)
				}
			}
		}
	}
	return output
}

func formatRangeUnified(start, stop int) string {
	beginning := start + 1
	length := stop - start
	if length == 1 {
		return itoa(beginning)
	}
	if length == 0 {
		beginning--
	}
	return itoa(beginning) + "," + itoa(length)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	buf := make([]byte, 0, 12)
	for value > 0 {
		buf = append([]byte{byte('0' + value%10)}, buf...)
		value /= 10
	}
	if negative {
		return "-" + string(buf)
	}
	return string(buf)
}

// SplitLines exposes Python-compatible splitlines for presentation.
func SplitLines(text string) []string { return splitLines(text) }
