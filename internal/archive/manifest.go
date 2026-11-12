package archive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	kerrors "github.com/Gunmer/kamino/internal/errors"
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

// Manifest is the clone metadata shared with the Python implementation.
type Manifest struct {
	FormatVersion int        `json:"format_version"`
	ToolVersion   string     `json:"tool_version"`
	CreatedAt     string     `json:"created_at"`
	MacOSVersion  string     `json:"macos_version"`
	Arch          string     `json:"arch"`
	Hostname      string     `json:"hostname"`
	Apps          []AppEntry `json:"apps"`
}

// EnsureRelative rejects empty, absolute or escaping manifest paths.
func EnsureRelative(raw string) error {
	if raw == "" {
		return fmt.Errorf("ruta vacía no admitida")
	}
	if strings.HasPrefix(raw, "/") {
		return fmt.Errorf("ruta absoluta no admitida: %q", raw)
	}
	for _, part := range strings.Split(raw, "/") {
		if part == ".." {
			return fmt.Errorf("ruta con '..' no admitida: %q", raw)
		}
	}
	return nil
}

func (f *FileEntry) normalize() {
	if f.Mode == "" {
		f.Mode = defaultFileMode
	}
}

func (f *FileEntry) validate() error {
	f.normalize()
	if err := EnsureRelative(f.Path); err != nil {
		return err
	}
	if f.SHA256 != nil && !sha256Pattern.MatchString(*f.SHA256) {
		return fmt.Errorf("'sha256' debe ser 64 caracteres hexadecimales")
	}
	if f.Link != nil {
		if err := EnsureRelative(*f.Link); err != nil {
			return err
		}
	}
	if f.Size < 0 {
		return fmt.Errorf("'size' no puede ser negativo")
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
	for index := range m.Apps {
		app := &m.Apps[index]
		for fileIndex := range app.Files {
			if err := app.Files[fileIndex].validate(); err != nil {
				return fmt.Errorf("app %q: %w", app.ID, err)
			}
		}
	}
	return nil
}

// MarshalJSONBytes serializes the manifest with the Python field order.
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
