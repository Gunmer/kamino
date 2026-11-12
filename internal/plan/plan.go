// Package plan expands catalog definitions into a clone or deploy plan.
package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/ghostty"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/progress"
)

const (
	MaxGhosttyFiles = 500
	MaxGhosttyDepth = 12
)

// SensitivePatterns are never included in automatic Ghostty references.
var SensitivePatterns = []string{
	"**/.env", "**/.env.*", "**/auth.json", "**/.credentials.json",
	"**/credentials.json", "**/secrets.yaml", "**/secrets.yml", "**/hosts.yml",
	"**/.ssh", "**/.ssh/**", "**/.aws/credentials", "**/id_rsa", "**/id_ed25519",
	"**/*.pem", "**/*.key", "**/sessions", "**/sessions/**", "**/history",
	"**/history/**", "**/*history*", "**/.cache", "**/.cache/**", "**/cache",
	"**/cache/**", "**/logs", "**/logs/**", "**/*.log", "**/*.sqlite", "**/*.sqlite3",
}

var ghosttyThemeDirs = []string{
	".config/ghostty/themes",
	"Library/Application Support/com.mitchellh.ghostty/themes",
}

// Operation is one planned regular file or symlink.
type Operation struct {
	AppID  string
	Path   string
	Source string
	Size   int64
	SHA256 *string
	Mode   string
	Link   *string
}

// SkippedPath is a path omitted from the plan with a visible reason.
type SkippedPath struct {
	Path   string
	Reason string
}

// Plan is the result of expanding applications against a home.
type Plan struct {
	Operations []Operation
	Skipped    []SkippedPath
	Notices    []string
}

// CountResult counts files per app without reading their contents.
type CountResult struct {
	Counts  map[string]int
	Skipped []SkippedPath
	Notices []string
}

// IsExcluded reports whether a POSIX relative path matches any pattern.
func IsExcluded(value string, patterns []string) bool {
	parts := strings.Split(value, "/")
	candidates := []string{value}
	for depth := 1; depth < len(parts); depth++ {
		candidates = append(candidates, strings.Join(parts[:depth], "/"))
	}
	for _, candidate := range candidates {
		for _, pattern := range patterns {
			if matchesPath(candidate, pattern) {
				return true
			}
		}
	}
	return false
}

type builder struct {
	safe       *paths.SafeHome
	root       string
	operations []Operation
	skipped    []SkippedPath
	notices    []string
	countOnly  bool
	progress   progress.Callback
	cancelled  progress.Cancel
	opKeys     map[string]bool
	skipKeys   map[string]bool
	noticeKeys map[string]bool
}

func newBuilder(safe *paths.SafeHome, root string, progressCallback progress.Callback, cancelled progress.Cancel) *builder {
	return &builder{
		safe:       safe,
		root:       root,
		progress:   progressCallback,
		cancelled:  cancelled,
		opKeys:     map[string]bool{},
		skipKeys:   map[string]bool{},
		noticeKeys: map[string]bool{},
	}
}

func (b *builder) addOperation(operation Operation) {
	key := operation.AppID + "\x00" + operation.Path
	if b.opKeys[key] {
		return
	}
	b.opKeys[key] = true
	b.operations = append(b.operations, operation)
	progress.Emit(b.progress, progress.Planning, len(b.operations), 0, operation.Path)
}

func (b *builder) skip(path, reason string) {
	if b.skipKeys[path] {
		return
	}
	b.skipKeys[path] = true
	b.skipped = append(b.skipped, SkippedPath{Path: path, Reason: reason})
}

func (b *builder) notice(message string) {
	if b.noticeKeys[message] {
		return
	}
	b.noticeKeys[message] = true
	b.notices = append(b.notices, message)
}

func (b *builder) count(appID string) int {
	total := 0
	for _, operation := range b.operations {
		if operation.AppID == appID {
			total++
		}
	}
	return total
}

func (b *builder) regularFile(appID, rel string) *Operation {
	for index := range b.operations {
		operation := &b.operations[index]
		if operation.AppID == appID && operation.Path == rel && operation.Link == nil {
			return operation
		}
	}
	return nil
}

// BuildPlan expands every application against the home.
func BuildPlan(apps []*catalog.AppDefinition, home string, progressCallback progress.Callback, cancelled progress.Cancel) (Plan, error) {
	root := home
	if root == "" {
		var err error
		root, err = os.UserHomeDir()
		if err != nil {
			return Plan{}, err
		}
	}
	safe, err := paths.NewSafeHome(root, false)
	if err != nil {
		return Plan{}, err
	}
	defer safe.Close()
	b := newBuilder(safe, root, progressCallback, cancelled)
	for _, app := range apps {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			return Plan{}, err
		}
		expandApp(app, b)
	}
	if err := progress.EnsureNotCancelled(cancelled); err != nil {
		return Plan{}, err
	}
	return Plan{Operations: b.operations, Skipped: b.skipped, Notices: b.notices}, nil
}

// CountFiles counts files per app without hashing their contents.
func CountFiles(apps []*catalog.AppDefinition, home string, cancelled progress.Cancel) (CountResult, error) {
	root := home
	if root == "" {
		var err error
		root, err = os.UserHomeDir()
		if err != nil {
			return CountResult{}, err
		}
	}
	safe, err := paths.NewSafeHome(root, false)
	if err != nil {
		return CountResult{}, err
	}
	defer safe.Close()
	b := newBuilder(safe, root, nil, cancelled)
	b.countOnly = true
	for _, app := range apps {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			return CountResult{}, err
		}
		expandApp(app, b)
	}
	counts := make(map[string]int, len(apps))
	for _, app := range apps {
		counts[app.ID] = b.count(app.ID)
	}
	return CountResult{Counts: counts, Skipped: b.skipped, Notices: b.notices}, nil
}

func expandApp(app *catalog.AppDefinition, b *builder) {
	for _, entry := range app.Paths {
		expand(app, entry, b, true, false)
	}
	for _, entry := range app.OptionalPaths {
		expand(app, entry, b, false, false)
	}
	for _, pattern := range app.OptionalPatterns {
		expandPattern(app, pattern, b)
	}
	for _, name := range app.EnvPaths {
		expandEnv(app, name, b)
	}
	for _, entry := range app.XDGPaths {
		expandXDG(app, entry, b)
	}
	if app.ID == ghostty.GhosttyAppID {
		expandGhostty(app, b)
	}
}

func expand(app *catalog.AppDefinition, entry string, b *builder, required, automatic bool) {
	if err := progress.EnsureNotCancelled(b.cancelled); err != nil {
		b.skip(entry, "cancelado")
		return
	}
	rel := path.Clean(entry)
	if _, err := paths.Parts(rel); err != nil {
		b.skip(rel, "ruta insegura o ilegible")
		return
	}
	if IsExcluded(rel, app.Exclude) || (automatic && IsExcluded(rel, SensitivePatterns)) {
		return
	}
	info, err := b.safe.Lstat(rel)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if required {
				b.skip(rel, "no existe")
			}
			return
		}
		b.skip(rel, "ruta insegura o ilegible")
		return
	}
	switch info.Kind {
	case paths.KindLink:
		addLink(app.ID, rel, info, b)
	case paths.KindDir:
		walk(app, rel, b)
	case paths.KindFile:
		addFile(app.ID, rel, info, b)
	default:
		b.skip(rel, "tipo de archivo no soportado")
	}
}

func expandPattern(app *catalog.AppDefinition, pattern string, b *builder) {
	parent := path.Dir(pattern)
	names, err := b.safe.Listdir(parent)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		b.skip(parent, "directorio del patrón inseguro o ilegible")
		return
	}
	base := path.Base(pattern)
	for _, name := range names {
		if fnmatch(name, base) {
			expand(app, path.Join(parent, name), b, false, false)
		}
	}
}

func expandEnv(app *catalog.AppDefinition, name string, b *builder) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	rel, err := paths.ToRelative(value, b.root)
	if err != nil {
		b.skip(name+" fuera del home", "fuera del home")
		return
	}
	expand(app, rel, b, false, false)
}

func expandXDG(app *catalog.AppDefinition, entry string, b *builder) {
	value := os.Getenv("XDG_CONFIG_HOME")
	if value == "" {
		return
	}
	rel, err := paths.ToRelative(value, b.root)
	if err != nil {
		b.skip("XDG_CONFIG_HOME fuera del home", "fuera del home")
		return
	}
	expand(app, path.Join(rel, entry), b, false, false)
}

func walk(app *catalog.AppDefinition, rel string, b *builder) {
	prune := func(child string) bool { return IsExcluded(child, app.Exclude) }
	err := b.safe.Walk(rel, prune, func(item paths.WalkItem) error {
		if err := progress.EnsureNotCancelled(b.cancelled); err != nil {
			return err
		}
		switch item.Kind {
		case paths.KindFile:
			addFile(app.ID, item.Path, *item.Info, b)
		case paths.KindLink:
			addLink(app.ID, item.Path, *item.Info, b)
		default:
			b.skip(item.Path, "tipo de archivo no soportado")
		}
		return nil
	})
	_ = err
}

func addFile(appID, rel string, info paths.Info, b *builder) {
	if err := progress.EnsureNotCancelled(b.cancelled); err != nil {
		return
	}
	source, err := paths.ToAbsolute(rel, b.root)
	if err != nil {
		b.skip(rel, "ruta insegura o ilegible")
		return
	}
	if b.countOnly {
		b.addOperation(Operation{AppID: appID, Path: rel, Source: source, Size: info.Size, Mode: info.ModeOctal()})
		return
	}
	data, err := b.safe.ReadBytes(rel)
	if err != nil {
		b.skip(rel, "no se pudo leer")
		return
	}
	digest := sha256.Sum256(data)
	sha := hex.EncodeToString(digest[:])
	b.addOperation(Operation{AppID: appID, Path: rel, Source: source, Size: int64(len(data)), SHA256: &sha, Mode: info.ModeOctal()})
}

func addLink(appID, rel string, info paths.Info, b *builder) {
	target, err := b.safe.Readlink(rel)
	if err != nil {
		b.skip(rel, "no se pudo leer el enlace")
		return
	}
	if !isSafeLink(target) {
		b.skip(rel, "enlace no seguro")
		return
	}
	source, err := paths.ToAbsolute(rel, b.root)
	if err != nil {
		b.skip(rel, "ruta insegura o ilegible")
		return
	}
	link := target
	b.addOperation(Operation{AppID: appID, Path: rel, Source: source, Size: 0, Mode: info.ModeOctal(), Link: &link})
}

func isSafeLink(target string) bool {
	if target == "" || strings.HasPrefix(target, "/") {
		return false
	}
	for _, part := range strings.Split(target, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func expandGhostty(app *catalog.AppDefinition, b *builder) {
	themeDirs := ghosttyThemeDirsFor(b)
	visited := map[string]bool{}
	stack := make([]string, 0)
	processed := 0

	initial := make([]string, 0)
	for _, operation := range b.operations {
		if operation.AppID == app.ID && operation.Link == nil && ghostty.ConfigFilenames[path.Base(operation.Path)] {
			initial = append(initial, operation.Path)
		}
	}

	var parse func(string, int)
	parse = func(rel string, depth int) {
		if err := progress.EnsureNotCancelled(b.cancelled); err != nil {
			return
		}
		if visited[rel] {
			return
		}
		visited[rel] = true
		if processed >= MaxGhosttyFiles {
			b.notice("se alcanzó el límite de archivos de configuración de Ghostty")
			return
		}
		if depth > MaxGhosttyDepth {
			b.notice("se alcanzó el límite de profundidad en " + rel)
			return
		}
		text, err := b.safe.ReadText(rel, ghostty.MaxConfigSize)
		if err != nil {
			return
		}
		processed++
		stack = append(stack, rel)
		defer func() { stack = stack[:len(stack)-1] }()
		for _, directive := range ghostty.IterDirectives(text) {
			key, value := directive[0], directive[1]
			switch key {
			case "config-file":
				raw, optional := ghostty.ConfigReference(value)
				followConfigFile(app, raw, rel, depth, optional, b, visited, stack, parse)
			case "theme":
				for _, raw := range ghostty.ThemeReferences(value) {
					followTheme(app, raw, rel, b, themeDirs)
				}
			}
		}
	}
	for _, rel := range initial {
		parse(rel, 0)
	}
}

func followConfigFile(app *catalog.AppDefinition, raw, baseRel string, depth int, optional bool, b *builder, visited map[string]bool, stack []string, parse func(string, int)) {
	if raw == "" {
		return
	}
	resolved, ok := ghostty.ResolveReference(raw, baseRel, b.root)
	if !ok {
		b.skip(raw, "referencia de Ghostty fuera del home")
		b.notice("referencia de Ghostty fuera del home: " + raw)
		return
	}
	if blockedReference(app, resolved) {
		b.skip(resolved, "referencia de Ghostty excluida por seguridad")
		b.notice("referencia de Ghostty excluida por seguridad: " + resolved)
		return
	}
	for _, active := range stack {
		if active == resolved {
			b.notice("referencia circular de Ghostty en " + resolved)
			return
		}
	}
	expand(app, resolved, b, !optional, true)
	if b.regularFile(app.ID, resolved) != nil {
		parse(resolved, depth+1)
	} else if !optional {
		b.skip(resolved, "referencia de Ghostty no incluida")
	}
}

func followTheme(app *catalog.AppDefinition, raw, baseRel string, b *builder, themeDirs []string) {
	if raw == "" {
		return
	}
	if ghostty.IsPathReference(raw) {
		resolved, ok := ghostty.ResolveReference(raw, baseRel, b.root)
		if !ok {
			b.skip(raw, "tema de Ghostty fuera del home")
			b.notice("tema de Ghostty fuera del home: " + raw)
			return
		}
		if blockedReference(app, resolved) {
			b.skip(resolved, "tema de Ghostty excluido por seguridad")
			b.notice("tema de Ghostty excluido por seguridad: " + resolved)
			return
		}
		expand(app, resolved, b, true, true)
		return
	}
	for _, directory := range themeDirs {
		candidate := path.Join(directory, raw)
		info, err := b.safe.Lstat(candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			b.skip(candidate, "tema de Ghostty inseguro")
			b.notice("tema de Ghostty inseguro: " + candidate)
			return
		}
		if info.Kind == paths.KindFile {
			expand(app, candidate, b, false, true)
		}
		return
	}
}

func blockedReference(app *catalog.AppDefinition, rel string) bool {
	return IsExcluded(rel, app.Exclude) || IsExcluded(rel, SensitivePatterns)
}

func ghosttyThemeDirsFor(b *builder) []string {
	directories := make([]string, 0, len(ghosttyThemeDirs)+1)
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		if rel, err := paths.ToRelative(xdg, b.root); err == nil && rel != "" {
			directories = append(directories, path.Join(rel, "ghostty", "themes"))
		}
	}
	for _, directory := range ghosttyThemeDirs {
		found := false
		for _, existing := range directories {
			if existing == directory {
				found = true
				break
			}
		}
		if !found {
			directories = append(directories, directory)
		}
	}
	return directories
}
