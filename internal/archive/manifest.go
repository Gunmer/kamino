package archive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
)

const (
	FormatVersion      = 1
	ManifestName       = "manifest.json"
	AppsDir            = "apps"
	MaxManifestBytes   = 4 * 1024 * 1024
	MaxArchiveMembers  = 100_000
	defaultFileMode    = "0644"
	paxFormatVersion   = 1
	manifestIndentSize = 2
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// FileEntry is one manifest file: regular (sha256) or symlink (link).
type FileEntry struct {
	Path   string  `json:"path"`
	SHA256 *string `json:"sha256"`
	Size   int64   `json:"size"`
	Mode   string  `json:"mode"`
	Link   *string `json:"link"`
}

// AppEntry groups the cloned files of one application.
type AppEntry struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	BundleID *string     `json:"bundle_id"`
	Version  *string     `json:"version"`
	Files    []FileEntry `json:"files"`
}

// Manifest is the clone metadata stored in every archive.
type Manifest struct {
	FormatVersion int        `json:"format_version"`
	ToolVersion   string     `json:"tool_version"`
	CreatedAt     string     `json:"created_at"`
	MacOSVersion  string     `json:"macos_version"`
	Arch          string     `json:"arch"`
	Hostname      string     `json:"hostname"`
	Apps          []AppEntry `json:"apps"`
}

func (f *FileEntry) normalize() {
	if f.Mode == "" {
		f.Mode = defaultFileMode
	}
}

// ParseMode converts a manifest permission string such as "0644" to a mode.
func ParseMode(value string) (os.FileMode, error) {
	parsed, err := strconv.ParseUint(value, 8, 32)
	if err != nil || parsed > 0o777 {
		return 0, fmt.Errorf("'mode' debe ser un permiso octal entre 0000 y 0777: %q", value)
	}
	return os.FileMode(parsed), nil
}

func (f *FileEntry) validate() error {
	f.normalize()
	if err := paths.EnsureRelative(f.Path); err != nil {
		return err
	}
	if f.SHA256 != nil && !sha256Pattern.MatchString(*f.SHA256) {
		return fmt.Errorf("'sha256' debe ser 64 caracteres hexadecimales")
	}
	if f.Link != nil {
		if err := paths.EnsureRelative(*f.Link); err != nil {
			return err
		}
	}
	if f.Size < 0 {
		return fmt.Errorf("'size' no puede ser negativo")
	}
	if _, err := ParseMode(f.Mode); err != nil {
		return err
	}
	if f.Link == nil && f.SHA256 == nil {
		return fmt.Errorf("se requiere 'sha256' (archivo regular) o 'link' (enlace simbólico)")
	}
	if f.Link != nil && f.SHA256 != nil {
		return fmt.Errorf("un archivo no puede ser regular y enlace a la vez")
	}
	return nil
}

func (m *Manifest) normalize() {
	if m.Apps == nil {
		m.Apps = []AppEntry{}
	}
	for index := range m.Apps {
		if m.Apps[index].Files == nil {
			m.Apps[index].Files = []FileEntry{}
		}
		for fileIndex := range m.Apps[index].Files {
			m.Apps[index].Files[fileIndex].normalize()
		}
	}
}

// Validate checks the manifest and applies defaults in place.
func (m *Manifest) Validate() error {
	m.normalize()
	if m.FormatVersion < 1 {
		return fmt.Errorf("'format_version' debe ser positivo")
	}
	// macOS volumes are case-insensitive by default: two paths that differ only
	// in case would overwrite each other when deployed.
	folded := map[string]string{}
	for index := range m.Apps {
		app := &m.Apps[index]
		for fileIndex := range app.Files {
			file := &app.Files[fileIndex]
			if err := file.validate(); err != nil {
				return fmt.Errorf("app %q: %w", app.ID, err)
			}
			key := strings.ToLower(file.Path)
			if previous, seen := folded[key]; seen && previous != file.Path {
				return fmt.Errorf("app %q: las rutas %q y %q solo difieren en mayúsculas y colisionan en un volumen que no distingue mayúsculas", app.ID, previous, file.Path)
			}
			folded[key] = file.Path
		}
	}
	return nil
}

// MarshalJSONBytes serializes the manifest with the canonical field order.
func (m *Manifest) MarshalJSONBytes() ([]byte, error) {
	m.normalize()
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", strings.Repeat(" ", manifestIndentSize))
	if err := encoder.Encode(m); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// ParseManifest decodes and validates a manifest document.
func ParseManifest(raw []byte) (*Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&manifest); err != nil {
		return nil, &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("manifiesto inválido: %v", err),
		}}
	}
	if err := manifest.Validate(); err != nil {
		return nil, &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("manifiesto inválido: %v", err),
		}}
	}
	if manifest.FormatVersion > FormatVersion {
		return nil, &kerrors.VersionError{KaminoError: kerrors.KaminoError{
			Msg: fmt.Sprintf("format_version %d no soportado (máximo %d)", manifest.FormatVersion, FormatVersion),
		}}
	}
	return &manifest, nil
}
