// Package conflicts detects, diffs, composes and validates deploy conflicts.
package conflicts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/plist"
)

const (
	MaxTextBytes = 5 * 1024 * 1024
	MaxDiffLines = 20_000
	ContextLines = 3
)

// ConflictKind explains why a file needs a decision.
type ConflictKind string

const (
	KindText   ConflictKind = "text"
	KindBinary ConflictKind = "binary"
	KindLarge  ConflictKind = "large"
	KindLink   ConflictKind = "symlink"
)

// RowKind describes a hunk row.
type RowKind string

const (
	RowEqual     RowKind = "equal"
	RowReplace   RowKind = "replace"
	RowLocalOnly RowKind = "local_only"
	RowCloneOnly RowKind = "clone_only"
)

// RowDecision selects the side written for a row.
type RowDecision string

const (
	RowClone RowDecision = "clone"
	RowLocal RowDecision = "local"
)

// FileAction selects how a conflicted file is resolved.
type FileAction string

const (
	ActionApply  FileAction = "apply"
	ActionOmit   FileAction = "omit"
	ActionReview FileAction = "review"
)

// LocalFile is the destination state read during planning.
type LocalFile struct {
	Kind string
	Data []byte
	Link string
	Mode string
}

// HunkRow is one line of a hunk.
type HunkRow struct {
	Kind  RowKind
	Local string
	Clone string
}

// Hunk is a contiguous change block with context.
type Hunk struct {
	Index      int
	LocalStart int
	LocalEnd   int
	CloneStart int
	CloneEnd   int
	Rows       []HunkRow
}

// FileDecision is the decision for one conflicted file.
type FileDecision struct {
	Action FileAction
	Rows   []RowDecision
}

// ConflictDecisions maps a relative path to its decision.
type ConflictDecisions struct {
	Decisions map[string]FileDecision
}

// Set stores a decision.
func (d *ConflictDecisions) Set(path string, decision FileDecision) {
	if d.Decisions == nil {
		d.Decisions = map[string]FileDecision{}
	}
	d.Decisions[path] = decision
}

// Get returns a decision, if any.
func (d *ConflictDecisions) Get(path string) (FileDecision, bool) {
	decision, ok := d.Decisions[path]
	return decision, ok
}

// Missing returns the conflicts without a decision.
func (d *ConflictDecisions) Missing(conflicts []Conflict) []Conflict {
	missing := make([]Conflict, 0)
	for _, conflict := range conflicts {
		if _, ok := d.Decisions[conflict.Path]; !ok {
			missing = append(missing, conflict)
		}
	}
	return missing
}

// MergeValidation is the result of validating a composed file.
type MergeValidation struct {
	Status  string
	Message string
}

// Conflict is a destination and clone file that differ.
type Conflict struct {
	AppID            string
	Path             string
	Destination      string
	Kind             ConflictKind
	LocalSize        int64
	CloneSize        int64
	LocalMode        string
	CloneMode        string
	LocalSHA256      *string
	CloneSHA256      *string
	LocalLink        string
	CloneLink        string
	LocalText        *string
	CloneText        *string
	Hunks            []Hunk
	ValidationFormat string
}

// BuildConflict builds a conflict, or nil when the file needs no decision.
func BuildConflict(appID, rel, destination string, entry archive.FileEntry, local LocalFile, cloneContent []byte) *Conflict {
	var localSHA *string
	if local.Kind == "file" {
		digest := sha256.Sum256(local.Data)
		value := hex.EncodeToString(digest[:])
		localSHA = &value
	}
	cloneSHA := entry.SHA256

	if entry.Link != nil {
		if local.Kind != "symlink" || local.Link == "" {
			return nil
		}
		if local.Link == *entry.Link {
			return nil
		}
		return &Conflict{
			AppID: appID, Path: rel, Destination: destination, Kind: KindLink,
			LocalMode: local.Mode, CloneMode: entry.Mode,
			LocalLink: local.Link, CloneLink: *entry.Link,
		}
	}
	if local.Kind != "file" {
		return nil
	}
	if cloneSHA != nil && localSHA != nil && *localSHA == *cloneSHA {
		return nil
	}

	localText, localIsText := decodeText(local.Data)
	if cloneContent == nil {
		if entry.Size > MaxTextBytes || int64(len(local.Data)) > MaxTextBytes {
			return wholeFileConflict(appID, rel, destination, entry, local, localSHA, KindLarge)
		}
		return wholeFileConflict(appID, rel, destination, entry, local, localSHA, KindBinary)
	}
	cloneText, cloneIsText := decodeText(cloneContent)
	if !localIsText || !cloneIsText {
		kind := KindBinary
		if entry.Size > MaxTextBytes || int64(len(local.Data)) > MaxTextBytes {
			kind = KindLarge
		}
		return wholeFileConflict(appID, rel, destination, entry, local, localSHA, kind)
	}
	if normaliseNewlines(localText) == normaliseNewlines(cloneText) {
		return nil
	}
	if entry.Size > MaxTextBytes || int64(len(local.Data)) > MaxTextBytes ||
		lineCount(cloneText) > MaxDiffLines || lineCount(localText) > MaxDiffLines {
		return wholeFileConflict(appID, rel, destination, entry, local, localSHA, KindLarge)
	}
	hunks := buildHunks(localText, cloneText)
	return &Conflict{
		AppID: appID, Path: rel, Destination: destination, Kind: KindText,
		LocalSize: int64(len(local.Data)), CloneSize: entry.Size,
		LocalMode: local.Mode, CloneMode: entry.Mode,
		LocalSHA256: localSHA, CloneSHA256: cloneSHA,
		LocalText: &localText, CloneText: &cloneText,
		Hunks: hunks, ValidationFormat: ValidationFormat(rel),
	}
}

func wholeFileConflict(appID, rel, destination string, entry archive.FileEntry, local LocalFile, localSHA *string, kind ConflictKind) *Conflict {
	return &Conflict{
		AppID: appID, Path: rel, Destination: destination, Kind: kind,
		LocalSize: int64(len(local.Data)), CloneSize: entry.Size,
		LocalMode: local.Mode, CloneMode: entry.Mode,
		LocalSHA256: localSHA, CloneSHA256: entry.SHA256,
		ValidationFormat: ValidationFormat(rel),
	}
}

func buildHunks(localText, cloneText string) []Hunk {
	localLines := splitLinesKeepEnds(localText)
	cloneLines := splitLinesKeepEnds(cloneText)
	localNorm := splitLinesKeepEnds(normaliseNewlines(localText))
	cloneNorm := splitLinesKeepEnds(normaliseNewlines(cloneText))
	matcher := newSequenceMatcher(localNorm, cloneNorm, false)

	hunks := make([]Hunk, 0)
	for _, group := range matcher.GetGroupedOpcodes(ContextLines) {
		changed := false
		for _, opcode := range group {
			if opcode.Tag != "equal" {
				changed = true
				break
			}
		}
		if !changed {
			continue
		}
		rows := make([]HunkRow, 0)
		for _, opcode := range group {
			rows = append(rows, rowsForOpcode(opcode, localLines, cloneLines)...)
		}
		first := group[0]
		last := group[len(group)-1]
		hunks = append(hunks, Hunk{
			Index:      len(hunks) + 1,
			LocalStart: first.I1 + 1,
			LocalEnd:   last.I2 + 1,
			CloneStart: first.J1 + 1,
			CloneEnd:   last.J2 + 1,
			Rows:       rows,
		})
	}
	return hunks
}

func rowsForOpcode(opcode Opcode, localLines, cloneLines []string) []HunkRow {
	localSlice := localLines[opcode.I1:opcode.I2]
	cloneSlice := cloneLines[opcode.J1:opcode.J2]
	switch opcode.Tag {
	case "equal":
		rows := make([]HunkRow, 0, len(localSlice))
		for index := range localSlice {
			rows = append(rows, HunkRow{Kind: RowEqual, Local: localSlice[index], Clone: cloneSlice[index]})
		}
		return rows
	case "replace":
		paired := len(localSlice)
		if len(cloneSlice) < paired {
			paired = len(cloneSlice)
		}
		rows := make([]HunkRow, 0, len(localSlice)+len(cloneSlice))
		for index := 0; index < paired; index++ {
			rows = append(rows, HunkRow{Kind: RowReplace, Local: localSlice[index], Clone: cloneSlice[index]})
		}
		for _, line := range localSlice[paired:] {
			rows = append(rows, HunkRow{Kind: RowLocalOnly, Local: line})
		}
		for _, line := range cloneSlice[paired:] {
			rows = append(rows, HunkRow{Kind: RowCloneOnly, Clone: line})
		}
		return rows
	case "delete":
		rows := make([]HunkRow, 0, len(localSlice))
		for _, line := range localSlice {
			rows = append(rows, HunkRow{Kind: RowLocalOnly, Local: line})
		}
		return rows
	case "insert":
		rows := make([]HunkRow, 0, len(cloneSlice))
		for _, line := range cloneSlice {
			rows = append(rows, HunkRow{Kind: RowCloneOnly, Clone: line})
		}
		return rows
	}
	return nil
}

// Compose builds the merged text from the selected side of every row.
func Compose(localText, cloneText string, decisions []RowDecision) (string, error) {
	localLines := splitLinesKeepEnds(localText)
	cloneLines := splitLinesKeepEnds(cloneText)
	localNorm := splitLinesKeepEnds(normaliseNewlines(localText))
	cloneNorm := splitLinesKeepEnds(normaliseNewlines(cloneText))
	matcher := newSequenceMatcher(localNorm, cloneNorm, false)

	expected := 0
	for _, opcode := range matcher.GetOpcodes() {
		if opcode.Tag == "equal" {
			continue
		}
		expected += len(rowsForOpcode(opcode, localLines, cloneLines))
	}
	if len(decisions) != expected {
		return "", fmt.Errorf("the decision list does not match the conflict rows")
	}

	eol := dominantEOL(localText)
	output := make([]string, 0)
	decisionIndex := 0
	for _, opcode := range matcher.GetOpcodes() {
		if opcode.Tag == "equal" {
			output = append(output, localLines[opcode.I1:opcode.I2]...)
			continue
		}
		rows := rowsForOpcode(opcode, localLines, cloneLines)
		for _, row := range rows {
			decision := decisions[decisionIndex]
			decisionIndex++
			switch row.Kind {
			case RowReplace:
				if decision == RowLocal {
					output = append(output, row.Local)
				} else {
					output = append(output, adaptLine(row.Clone, eol))
				}
			case RowLocalOnly:
				if decision == RowLocal {
					output = append(output, row.Local)
				}
			case RowCloneOnly:
				if decision == RowClone {
					output = append(output, adaptLine(row.Clone, eol))
				}
			}
		}
	}
	return ensureFinalNewline(output, eol), nil
}

func ensureFinalNewline(lines []string, eol string) string {
	result := make([]string, len(lines))
	for index, line := range lines {
		if index == len(lines)-1 || strings.HasSuffix(line, "\n") {
			result[index] = line
		} else {
			result[index] = line + eol
		}
	}
	return strings.Join(result, "")
}

func dominantEOL(text string) string {
	crlf := strings.Count(text, "\r\n")
	lf := strings.Count(text, "\n") - crlf
	if crlf > lf {
		return "\r\n"
	}
	return "\n"
}

func adaptLine(line, eol string) string {
	if strings.HasSuffix(line, "\r\n") {
		return strings.TrimSuffix(line, "\r\n") + eol
	}
	if strings.HasSuffix(line, "\n") {
		return strings.TrimSuffix(line, "\n") + eol
	}
	return line
}

// ValidateMerge validates a composed file with the standard library.
func ValidateMerge(path, localText, cloneText, mergedText string) MergeValidation {
	parser, ok := parserFor(path)
	if !ok {
		return MergeValidation{Status: "unvalidated", Message: "este formato no se valida automáticamente"}
	}
	if parser(localText) != nil || parser(cloneText) != nil {
		return MergeValidation{Status: "warning", Message: "uno de los originales ya era inválido; se permite aplicar el híbrido"}
	}
	if err := parser(mergedText); err != nil {
		return MergeValidation{Status: "invalid", Message: err.Error()}
	}
	return MergeValidation{Status: "valid"}
}

// ValidationFormat returns the display name of the validator for path.
func ValidationFormat(rel string) string {
	switch strings.ToLower(path.Ext(rel)) {
	case ".toml":
		return "TOML"
	case ".json":
		return "JSON"
	case ".ini", ".cfg", ".gitconfig":
		return "INI"
	case ".plist":
		return "plist"
	}
	return ""
}

// ApplyAll decides every conflict by applying the clone side.
func ApplyAll(conflicts []Conflict) ConflictDecisions {
	decisions := ConflictDecisions{Decisions: map[string]FileDecision{}}
	for _, conflict := range conflicts {
		decisions.Set(conflict.Path, FileDecision{Action: ActionApply})
	}
	return decisions
}

// OmitAll decides every conflict by keeping the local side.
func OmitAll(conflicts []Conflict) ConflictDecisions {
	decisions := ConflictDecisions{Decisions: map[string]FileDecision{}}
	for _, conflict := range conflicts {
		decisions.Set(conflict.Path, FileDecision{Action: ActionOmit})
	}
	return decisions
}

// RowCount returns how many decidable rows the conflict has.
func RowCount(conflict Conflict) int {
	total := 0
	for _, hunk := range conflict.Hunks {
		for _, row := range hunk.Rows {
			if row.Kind != RowEqual {
				total++
			}
		}
	}
	return total
}

// ReviewAll builds a REVIEW decision with every row on the same side.
func ReviewAll(conflict Conflict, decision RowDecision) FileDecision {
	rows := make([]RowDecision, RowCount(conflict))
	for index := range rows {
		rows[index] = decision
	}
	return FileDecision{Action: ActionReview, Rows: rows}
}

func decodeText(data []byte) (string, bool) {
	for _, b := range data {
		if b == 0 {
			return "", false
		}
	}
	if !utf8.Valid(data) {
		return "", false
	}
	return string(data), true
}

func lineCount(text string) int { return len(splitLines(normaliseNewlines(text))) }

func parserFor(rel string) (func(string) error, bool) {
	switch strings.ToLower(path.Ext(rel)) {
	case ".toml":
		return parseTOML, true
	case ".json":
		return parseJSON, true
	case ".ini", ".cfg", ".gitconfig":
		return parseINI, true
	case ".plist":
		return parsePlistText, true
	}
	return nil, false
}

func parseTOML(text string) error {
	var value map[string]any
	if _, err := toml.Decode(text, &value); err != nil {
		return fmt.Errorf("TOML inválido: %v", err)
	}
	return nil
}

func parseJSON(text string) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := parseJSONValue(decoder); err != nil {
		return fmt.Errorf("JSON inválido: %v", err)
	}
	if _, err := decoder.Token(); err == nil {
		return fmt.Errorf("JSON inválido: datos extra")
	}
	return nil
}

func parseJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("clave no textual")
			}
			if seen[key] {
				return fmt.Errorf("clave duplicada: %q", key)
			}
			seen[key] = true
			if err := parseJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := parseJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	}
	return nil
}

func parseINI(text string) error {
	type section struct {
		name string
		keys map[string]bool
	}
	var current *section
	sections := map[string]bool{}
	seenKey := false
	for lineNumber, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			if !strings.HasSuffix(trimmed, "]") {
				return fmt.Errorf("INI inválido: línea %d: sección sin cerrar", lineNumber+1)
			}
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if name == "" {
				return fmt.Errorf("INI inválido: línea %d: sección vacía", lineNumber+1)
			}
			if sections[name] {
				return fmt.Errorf("INI inválido: línea %d: sección duplicada %q", lineNumber+1, name)
			}
			sections[name] = true
			current = &section{name: name, keys: map[string]bool{}}
			seenKey = false
			continue
		}
		if line != "" && (line[0] == ' ' || line[0] == '\t') && current != nil && seenKey {
			continue // continuation line
		}
		separator := strings.IndexAny(line, "=:")
		if separator < 0 {
			return fmt.Errorf("INI inválido: línea %d: falta el separador", lineNumber+1)
		}
		if current == nil {
			return fmt.Errorf("INI inválido: línea %d: clave sin sección", lineNumber+1)
		}
		key := strings.ToLower(strings.TrimSpace(line[:separator]))
		if key == "" {
			return fmt.Errorf("INI inválido: línea %d: clave vacía", lineNumber+1)
		}
		if current.keys[key] {
			return fmt.Errorf("INI inválido: línea %d: clave duplicada %q", lineNumber+1, key)
		}
		current.keys[key] = true
		seenKey = true
	}
	return nil
}

func parsePlistText(text string) error {
	data := []byte(text)
	if _, err := plist.Parse(data); err != nil {
		return fmt.Errorf("plist inválido: %v", err)
	}
	if duplicate := plist.DuplicateKey(data); duplicate != "" {
		return fmt.Errorf("plist inválido: clave duplicada: %q", duplicate)
	}
	return nil
}

// BuildHunks exposes the hunk builder for integration tests.
func BuildHunks(localText, cloneText string) []Hunk { return buildHunks(localText, cloneText) }
