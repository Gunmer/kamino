// Package catalog loads embedded and user application definitions.
package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	kerrors "github.com/Gunmer/kamino/internal/errors"
)

//go:embed apps/*.toml
var embeddedFS embed.FS

var (
	idPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	envPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

// AppDefinition is one validated catalog definition.
type AppDefinition struct {
	ID               string   `toml:"id"`
	Name             string   `toml:"name"`
	BundleID         *string  `toml:"bundle_id"`
	Command          *string  `toml:"command"`
	CommandAliases   []string `toml:"command_aliases"`
	SearchPaths      []string `toml:"search_paths"`
	InstallMarker    *string  `toml:"install_marker"`
	Paths            []string `toml:"paths"`
	OptionalPaths    []string `toml:"optional_paths"`
	OptionalPatterns []string `toml:"optional_patterns"`
	EnvPaths         []string `toml:"env_paths"`
	XDGPaths         []string `toml:"xdg_paths"`
	Exclude          []string `toml:"exclude"`
}

// Validate checks the definition against the catalog contract.
func (a *AppDefinition) Validate() error {
	if !idPattern.MatchString(a.ID) {
		return fmt.Errorf("'id' debe usar minúsculas y guiones (ej. 'herdr')")
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("'name' no puede estar vacío")
	}
	for _, value := range append(append([]string{}, a.Paths...), append(a.OptionalPaths, a.XDGPaths...)...) {
		if err := ensureRelative(value); err != nil {
			return err
		}
	}
	for _, pattern := range a.OptionalPatterns {
		if err := ensurePattern(pattern); err != nil {
			return err
		}
	}
	if a.InstallMarker != nil {
		if err := ensureRelative(*a.InstallMarker); err != nil {
			return err
		}
	}
	for _, name := range a.EnvPaths {
		if !envPattern.MatchString(name) {
			return fmt.Errorf("nombre de variable de entorno no válido: %q", name)
		}
	}
	for _, alias := range a.CommandAliases {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("'command_aliases' no admite entradas vacías")
		}
	}
	if a.BundleID != nil && strings.TrimSpace(*a.BundleID) == "" {
		return fmt.Errorf("'bundle_id' no puede estar vacío")
	}
	if a.Command != nil && strings.TrimSpace(*a.Command) == "" {
		return fmt.Errorf("'command' no puede estar vacío")
	}
	if a.BundleID == nil && a.Command == nil {
		return fmt.Errorf("se requiere 'bundle_id' o 'command' para detectar la app")
	}
	if len(a.Paths) == 0 && len(a.OptionalPaths) == 0 && len(a.OptionalPatterns) == 0 {
		return fmt.Errorf("se requiere al menos una entrada en 'paths', 'optional_paths' o 'optional_patterns'")
	}
	return nil
}

func ensureRelative(raw string) error {
	if raw == "" {
		return fmt.Errorf("no se admiten rutas vacías")
	}
	if strings.HasPrefix(raw, "~") {
		return fmt.Errorf("ruta con '~' no admitida, usa una ruta relativa al home: %q", raw)
	}
	if strings.HasPrefix(raw, "/") {
		return fmt.Errorf("ruta absoluta no admitida: %q", raw)
	}
	for _, part := range strings.Split(raw, "/") {
		if part == ".." {
			return fmt.Errorf("ruta fuera del home no admitida: %q", raw)
		}
	}
	return nil
}

func ensurePattern(pattern string) error {
	if err := ensureRelative(pattern); err != nil {
		return err
	}
	if strings.Contains(pattern, "**") {
		return fmt.Errorf("patrón recursivo no admitido: %q", pattern)
	}
	parts := strings.Split(path.Clean(pattern), "/")
	if len(parts) < 2 {
		return fmt.Errorf("el patrón debe incluir un directorio fijo: %q", pattern)
	}
	for _, part := range parts[:len(parts)-1] {
		if strings.ContainsAny(part, "*?[") {
			return fmt.Errorf("solo el último componente puede contener comodines: %q", pattern)
		}
	}
	if !strings.ContainsAny(parts[len(parts)-1], "*?[") {
		return fmt.Errorf("el último componente debe contener un comodín: %q", pattern)
	}
	return nil
}

// Catalog is a map of id to definition.
type Catalog map[string]*AppDefinition

// LoadCatalog returns the embedded catalog overridden by the user directory.
func LoadCatalog(userDir string) (Catalog, error) {
	catalog, err := loadEmbedded()
	if err != nil {
		return nil, err
	}
	user, err := loadUser(userDir)
	if err != nil {
		return nil, err
	}
	for id, definition := range user {
		catalog[id] = definition
	}
	return catalog, nil
}

func loadEmbedded() (Catalog, error) {
	entries, err := fs.ReadDir(embeddedFS, "apps")
	if err != nil {
		return nil, catalogError("no se pudo leer el catálogo embebido: %v", err)
	}
	catalog := Catalog{}
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
		if _, exists := catalog[definition.ID]; exists {
			return nil, catalogError("id duplicado %q en el catálogo embebido", definition.ID)
		}
		catalog[definition.ID] = definition
	}
	return catalog, nil
}

func loadUser(directory string) (Catalog, error) {
	catalog := Catalog{}
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return catalog, nil
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
	for _, name := range names {
		file := filepath.Join(directory, name)
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, catalogError("no se pudo leer %q: %v", name, err)
		}
		definition, err := parseDefinition(name, string(data))
		if err != nil {
			return nil, err
		}
		if _, exists := catalog[definition.ID]; exists {
			return nil, catalogError("id duplicado %q en el catálogo del usuario", definition.ID)
		}
		catalog[definition.ID] = definition
	}
	return catalog, nil
}

func parseDefinition(name, data string) (*AppDefinition, error) {
	var definition AppDefinition
	metadata, err := toml.Decode(data, &definition)
	if err != nil {
		return nil, catalogError("TOML inválido en %q: %v", name, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		return nil, catalogError("definición inválida en %q: campo desconocido: %s", name, strings.Join(keys, ", "))
	}
	if err := definition.Validate(); err != nil {
		return nil, catalogError("definición inválida en %q: %v", name, err)
	}
	return &definition, nil
}

func catalogError(format string, args ...any) error {
	return &kerrors.CatalogError{KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf(format, args...)}}
}

// SortedIDs returns the catalog ids in display order.
func (c Catalog) SortedIDs() []string {
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
