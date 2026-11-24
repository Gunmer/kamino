// Package install runs the installation recipes declared by catalog
// definitions. Recipes come only from the catalog; a clone never supplies one.
package install

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/Gunmer/kamino/internal/paths"
)

// Methods accepted in a recipe. The set is closed: anything else is rejected
// when the catalog loads.
const (
	MethodBrew          = "brew"
	MethodNpm           = "npm"
	MethodInstaller     = "installer"
	MethodBinary        = "binary"
	MethodGithubRelease = "github-release"
)

// Asset formats accepted by the github-release method.
const (
	FormatRaw   = "raw"
	FormatZip   = "zip"
	FormatTarGz = "tar.gz"
)

var (
	namePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9@+._/-]*$`)
	sha256Pattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	// packagePattern matches an npm package name with an optional @scope/.
	packagePattern = regexp.MustCompile(`^(@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)
	repoPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)
	// assetPattern matches a release asset file name; "{version}" is allowed
	// and replaced by the latest tag without its "v" prefix.
	assetPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+{}-]*$`)
	knownOS      = map[string]bool{"darwin": true}
	knownArch    = map[string]bool{"arm64": true, "amd64": true}
)

// ErrNoRecipe reports that no recipe applies to the current platform or
// environment, so the app cannot be offered for installation.
var ErrNoRecipe = errors.New("ninguna receta de instalación aplicable")

// Recipe is one installation method declared in a `[[install]]` block.
type Recipe struct {
	Method  string `toml:"method"`
	Formula string `toml:"formula"`
	Cask    string `toml:"cask"`
	Package string `toml:"package"`
	URL     string `toml:"url"`
	SHA256  string `toml:"sha256"`
	Dest    string `toml:"dest"`
	OS      string `toml:"os"`
	Arch    string `toml:"arch"`

	// Fields of the github-release method.
	Repo      string        `toml:"repo"`
	Asset     string        `toml:"asset"`
	Format    string        `toml:"format"`
	Extract   []ExtractFile `toml:"extract"`
	Checksums string        `toml:"checksums"`
}

// ExtractFile names one regular file inside a release archive and where it is
// installed under the home.
type ExtractFile struct {
	File string `toml:"file"`
	Dest string `toml:"dest"`
}

// Source returns the address shown to the user before the recipe downloads
// anything, or "" when the method does not download from a fixed place.
func (r Recipe) Source() string {
	switch r.Method {
	case MethodInstaller, MethodBinary:
		return r.URL
	case MethodGithubRelease:
		return "https://github.com/" + r.Repo + "/releases"
	}
	return ""
}

// Unverified reports whether the recipe downloads an executable without any
// checksum: only HTTPS and the GitHub origin vouch for it.
func (r Recipe) Unverified() bool {
	return r.Method == MethodGithubRelease && r.Checksums == ""
}

func (r Recipe) hasReleaseFields() bool {
	return r.Repo != "" || r.Asset != "" || r.Format != "" || len(r.Extract) > 0 || r.Checksums != ""
}

// Validate checks the recipe against the closed catalog contract. Fields that
// do not belong to the chosen method are rejected so a typo never silently
// changes what would run.
func (r Recipe) Validate() error {
	switch r.Method {
	case MethodBrew:
		if (r.Formula == "") == (r.Cask == "") {
			return errors.New("'brew' requiere exactamente una de 'formula' o 'cask'")
		}
		if err := checkName("formula", r.Formula); err != nil {
			return err
		}
		if err := checkName("cask", r.Cask); err != nil {
			return err
		}
		if r.URL != "" || r.SHA256 != "" || r.Dest != "" || r.Package != "" || r.hasReleaseFields() {
			return errors.New("'brew' solo admite 'formula' o 'cask'")
		}
	case MethodNpm:
		if r.Package == "" || !packagePattern.MatchString(r.Package) {
			return fmt.Errorf("'npm' requiere 'package' con un nombre de paquete válido: %q", r.Package)
		}
		if r.Formula != "" || r.Cask != "" || r.URL != "" || r.SHA256 != "" || r.Dest != "" || r.hasReleaseFields() {
			return errors.New("'npm' solo admite 'package'")
		}
	case MethodInstaller:
		if err := requireHTTPS(r.URL); err != nil {
			return err
		}
		if r.Formula != "" || r.Cask != "" || r.Package != "" || r.SHA256 != "" || r.Dest != "" || r.hasReleaseFields() {
			return errors.New("'installer' solo admite 'url'")
		}
	case MethodBinary:
		if err := requireHTTPS(r.URL); err != nil {
			return err
		}
		if !sha256Pattern.MatchString(r.SHA256) {
			return errors.New("'binary' requiere 'sha256' con 64 caracteres hexadecimales")
		}
		if r.Dest == "" {
			return errors.New("'binary' requiere 'dest' con una ruta relativa al home")
		}
		if err := paths.EnsureRelative(r.Dest); err != nil {
			return fmt.Errorf("'dest' no válido: %w", err)
		}
		if r.Formula != "" || r.Cask != "" || r.Package != "" || r.hasReleaseFields() {
			return errors.New("'binary' no admite 'formula', 'cask', 'package' ni campos de 'github-release'")
		}
	case MethodGithubRelease:
		if err := r.validateGithubRelease(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("método de instalación %q no admitido: usa 'brew', 'npm', 'installer', 'binary' o 'github-release'", r.Method)
	}
	if r.OS != "" && !knownOS[r.OS] {
		return fmt.Errorf("'os' no reconocido %q: usa 'darwin'", r.OS)
	}
	if r.Arch != "" && !knownArch[r.Arch] {
		return fmt.Errorf("'arch' no reconocida %q: usa 'arm64' o 'amd64'", r.Arch)
	}
	return nil
}

func (r Recipe) validateGithubRelease() error {
	if r.Formula != "" || r.Cask != "" || r.Package != "" || r.URL != "" || r.SHA256 != "" {
		return errors.New("'github-release' no admite 'formula', 'cask', 'package', 'url' ni 'sha256'")
	}
	if !repoPattern.MatchString(r.Repo) {
		return fmt.Errorf("'github-release' requiere 'repo' con la forma propietario/nombre: %q", r.Repo)
	}
	if !assetPattern.MatchString(r.Asset) {
		return fmt.Errorf("'github-release' requiere 'asset' con un nombre de fichero válido: %q", r.Asset)
	}
	if r.Checksums != "" && !assetPattern.MatchString(r.Checksums) {
		return fmt.Errorf("'checksums' no es un nombre de fichero válido: %q", r.Checksums)
	}
	switch r.Format {
	case FormatRaw:
		if len(r.Extract) > 0 {
			return errors.New("'format = \"raw\"' no admite 'extract': usa 'dest'")
		}
		if r.Dest == "" {
			return errors.New("'github-release' con formato 'raw' requiere 'dest' con una ruta relativa al home")
		}
		if err := paths.EnsureRelative(r.Dest); err != nil {
			return fmt.Errorf("'dest' no válido: %w", err)
		}
	case FormatZip, FormatTarGz:
		if r.Dest != "" {
			return fmt.Errorf("'format = %q' no admite 'dest': usa 'extract'", r.Format)
		}
		if len(r.Extract) == 0 {
			return fmt.Errorf("'github-release' con formato %q requiere al menos un 'extract'", r.Format)
		}
		for _, file := range r.Extract {
			if file.File == "" || file.Dest == "" {
				return errors.New("cada 'extract' requiere 'file' y 'dest'")
			}
			if err := paths.EnsureRelative(file.File); err != nil {
				return fmt.Errorf("'extract.file' no válido: %w", err)
			}
			if err := paths.EnsureRelative(file.Dest); err != nil {
				return fmt.Errorf("'extract.dest' no válido: %w", err)
			}
		}
	default:
		return fmt.Errorf("'format' %q no admitido: usa 'raw', 'zip' o 'tar.gz'", r.Format)
	}
	return nil
}

// Applies reports whether the recipe matches the given platform. Empty os or
// arch fields match any value.
func (r Recipe) Applies(goos, goarch string) bool {
	return (r.OS == "" || r.OS == goos) && (r.Arch == "" || r.Arch == goarch)
}

func checkName(field, value string) error {
	if value == "" || namePattern.MatchString(value) {
		return nil
	}
	return fmt.Errorf("'%s' no es un nombre válido: %q", field, value)
}

func requireHTTPS(raw string) error {
	if raw == "" {
		return errors.New("se requiere 'url' con https")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("'url' debe usar https: %q", raw)
	}
	return nil
}
