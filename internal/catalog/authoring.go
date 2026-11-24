package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// DefaultExcludes is the standard list used by generated definitions. It keeps
// credentials, caches, histories and repository metadata out of a clone.
var DefaultExcludes = []string{
	"**/.env", "**/.env.*", "**/auth.json", "**/.credentials.json",
	"**/secrets.yaml", "**/secrets.yml", "**/hosts.yml", "**/*.log",
	"**/logs", "**/logs/**", "**/sessions", "**/sessions/**",
	"**/history", "**/history/**", "**/*history*", "**/.cache",
	"**/.cache/**", "**/.git", "**/.git/**",
}

// TemplateOptions carries the values rendered by RenderTemplate.
type TemplateOptions struct {
	ID            string
	Name          string
	Command       string
	BundleID      string
	SearchPaths   []string
	Paths         []string
	OptionalPaths []string
	// Skeleton renders empty id, name, detection and paths so the TUI can open
	// an intentionally invalid draft that the user must complete.
	Skeleton bool
}

// DefinitionResult is the per-file result of inspecting the user catalog.
type DefinitionResult struct {
	File       string
	ID         string
	Name       string
	Definition *AppDefinition
	Err        error
	Warnings   []string
}

// CreateResult is the result of writing one definition.
type CreateResult struct {
	Path       string
	Definition *AppDefinition
	Warnings   []string
}

// Parse parses and validates one definition. It is the same contract used by
// LoadCatalog, exposed for the authoring commands.
func Parse(name string, data []byte) (*AppDefinition, error) {
	return parseDefinition(name, string(data))
}

// Embedded returns the built-in catalog.
func Embedded() (Catalog, error) {
	return loadEmbedded()
}

// EmbeddedSource returns the raw TOML of the built-in definition with the given
// id, comments and install recipes included. The lookup uses the declared id,
// not the file name.
func EmbeddedSource(id string) ([]byte, error) {
	entries, err := fs.ReadDir(embeddedFS, "apps")
	if err != nil {
		return nil, catalogError("no se pudo leer el catálogo embebido: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		data, err := embeddedFS.ReadFile(path.Join("apps", entry.Name()))
		if err != nil {
			return nil, catalogError("no se pudo leer %q: %v", entry.Name(), err)
		}
		definition, err := parseDefinition(entry.Name(), string(data))
		if err != nil {
			return nil, err
		}
		if definition.ID == id {
			return data, nil
		}
	}
	return nil, &NotBuiltinError{ID: id}
}

// NotBuiltinError means no built-in definition has the requested id.
type NotBuiltinError struct{ ID string }

func (e *NotBuiltinError) Error() string {
	return fmt.Sprintf("no hay una app integrada con el id %q", e.ID)
}

// InspectFile parses one definition and reports its warnings without aborting.
func InspectFile(file string) DefinitionResult {
	name := filepath.Base(file)
	result := DefinitionResult{File: name}
	data, err := os.ReadFile(file)
	if err != nil {
		result.Err = catalogError("no se pudo leer %q: %v", name, err)
		return result
	}
	definition, err := Parse(name, data)
	if err != nil {
		result.Err = err
		return result
	}
	result.ID = definition.ID
	result.Name = definition.Name
	result.Definition = definition
	result.Warnings = definitionWarnings(definition, name)
	return result
}

// InspectDir parses every *.toml in a user catalog without failing on the first
// bad file. It detects duplicate ids across files. A missing directory is an
// empty catalog.
func InspectDir(directory string) ([]DefinitionResult, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, catalogError("no se pudo leer el catálogo del usuario %q: %v", directory, err)
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	results := make([]DefinitionResult, 0, len(names))
	for _, name := range names {
		results = append(results, InspectFile(filepath.Join(directory, name)))
	}
	firstByID := map[string]int{}
	for index := range results {
		result := &results[index]
		if result.Err != nil || result.Definition == nil {
			continue
		}
		if first, exists := firstByID[result.ID]; exists {
			result.Err = catalogError(
				"id duplicado %q en el catálogo del usuario: %q y %q",
				result.ID, results[first].File, result.File,
			)
			continue
		}
		firstByID[result.ID] = index
	}
	return results, nil
}

// RenderTemplate writes an annotated TOML definition. The CLI fills the
// required fields; the TUI asks for a skeleton and lets the editor complete it.
func RenderTemplate(options TemplateOptions) (string, error) {
	id := options.ID
	name := options.Name
	command := options.Command
	bundle := options.BundleID
	if options.Skeleton {
		id, name, command, bundle = "", "", "", ""
	}
	if name == "" {
		name = id
	}
	var builder strings.Builder
	builder.WriteString("# Definición de app para Kamino.\n")
	builder.WriteString("# Edita los valores y guarda; Kamino valida la definición antes de usarla.\n")
	builder.WriteString("# Las rutas son relativas al HOME, sin \"..\" ni rutas absolutas.\n\n")
	builder.WriteString("id = " + tomlQuote(id) + "\n")
	builder.WriteString("name = " + tomlQuote(name) + "\n\n")
	builder.WriteString("# Detección: comando en PATH o bundle de macOS. Al menos uno.\n")
	if command != "" || bundle == "" {
		builder.WriteString("command = " + tomlQuote(command) + "\n")
	} else {
		builder.WriteString("# command = \"miapp\"\n")
	}
	if bundle != "" {
		builder.WriteString("bundle_id = " + tomlQuote(bundle) + "\n")
	} else {
		builder.WriteString("# bundle_id = \"com.ejemplo.miapp\"\n")
	}
	if len(options.SearchPaths) > 0 {
		builder.WriteString("search_paths = " + tomlArray(options.SearchPaths) + "\n")
	} else {
		builder.WriteString("# search_paths = [\"/opt/homebrew/bin\", \"/usr/local/bin\"]\n")
	}
	builder.WriteString("# command_aliases = [\"miapp-cli\"]\n")
	builder.WriteString("# install_marker = \".miapp/installed\"\n\n")
	builder.WriteString("# Archivos globales que se clonan. Al menos una ruta.\n")
	builder.WriteString("paths = " + tomlArray(options.Paths) + "\n")
	if len(options.OptionalPaths) > 0 {
		builder.WriteString("optional_paths = " + tomlArray(options.OptionalPaths) + "\n")
	} else {
		builder.WriteString("# optional_paths = [\".config/miapp/themes\"]\n")
	}
	builder.WriteString("# optional_patterns = [\".config/miapp/perfiles/*.toml\"]\n")
	builder.WriteString("# env_paths = [\"MIAPP_CONFIG\"]\n")
	builder.WriteString("# xdg_paths = [\"miapp/config.toml\"]\n\n")
	builder.WriteString("# Exclusiones de credenciales, cachés e historiales.\n")
	builder.WriteString("exclude = [\n")
	for _, pattern := range DefaultExcludes {
		builder.WriteString("  " + tomlQuote(pattern) + ",\n")
	}
	builder.WriteString("]\n")
	return builder.String(), nil
}

// CreateDefinition validates data and writes it atomically as <id>.toml. It
// refuses to overwrite an existing file unless force is set, and refuses an id
// already declared by another file in the directory.
func CreateDefinition(directory string, data []byte, force bool) (CreateResult, error) {
	definition, err := Parse("nueva definición", data)
	if err != nil {
		return CreateResult{}, err
	}
	results, err := InspectDir(directory)
	if err != nil {
		return CreateResult{}, err
	}
	targetName := definition.ID + ".toml"
	target := filepath.Join(directory, targetName)
	warnings := make([]string, 0)
	for _, result := range results {
		if result.Definition == nil {
			if result.Err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: %v", result.File, result.Err))
			}
			continue
		}
		if result.ID == definition.ID && result.File != targetName {
			return CreateResult{}, catalogError("id duplicado %q: ya está en %q", definition.ID, result.File)
		}
		if result.Err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", result.File, result.Err))
		}
	}
	if !force {
		if _, err := os.Lstat(target); err == nil {
			return CreateResult{}, catalogError("%q ya existe; usa --force para reemplazarlo", target)
		}
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return CreateResult{}, catalogError("no se pudo crear el catálogo %q: %v", directory, err)
	}
	file, err := os.CreateTemp(directory, ".kamino-*.tmp")
	if err != nil {
		return CreateResult{}, catalogError("no se pudo escribir %q: %v", target, err)
	}
	temp := file.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(temp)
		}
	}()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return CreateResult{}, catalogError("no se pudo escribir %q: %v", target, err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return CreateResult{}, catalogError("no se pudo escribir %q: %v", target, err)
	}
	if err := file.Close(); err != nil {
		return CreateResult{}, catalogError("no se pudo escribir %q: %v", target, err)
	}
	if err := os.Chmod(temp, 0o644); err != nil {
		return CreateResult{}, catalogError("no se pudo escribir %q: %v", target, err)
	}
	if err := os.Rename(temp, target); err != nil {
		return CreateResult{}, catalogError("no se pudo escribir %q: %v", target, err)
	}
	cleanup = false
	embedded, err := loadEmbedded()
	if err == nil {
		if _, exists := embedded[definition.ID]; exists {
			warnings = append(warnings, fmt.Sprintf("'%s' reemplaza la definición integrada", definition.ID))
		}
	}
	return CreateResult{Path: target, Definition: definition, Warnings: warnings}, nil
}

func definitionWarnings(definition *AppDefinition, file string) []string {
	warnings := make([]string, 0)
	if len(definition.Exclude) == 0 {
		warnings = append(warnings, "no declara exclusiones; puede copiar credenciales, cachés o historiales")
	}
	if base := strings.TrimSuffix(file, ".toml"); base != definition.ID {
		warnings = append(warnings, fmt.Sprintf("el id %q no coincide con el nombre del archivo %q", definition.ID, file))
	}
	return warnings
}

func tomlArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, tomlQuote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func tomlQuote(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, char := range value {
		switch char {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\b':
			builder.WriteString(`\b`)
		case '\t':
			builder.WriteString(`\t`)
		case '\n':
			builder.WriteString(`\n`)
		case '\f':
			builder.WriteString(`\f`)
		case '\r':
			builder.WriteString(`\r`)
		default:
			if char < 0x20 || char == 0x7f || !utf8.ValidRune(char) {
				fmt.Fprintf(&builder, `\u%04X`, char)
				continue
			}
			builder.WriteRune(char)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}
