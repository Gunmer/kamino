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
	MethodBrew      = "brew"
	MethodInstaller = "installer"
	MethodBinary    = "binary"
)

var (
	namePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9@+._/-]*$`)
	sha256Pattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	knownOS       = map[string]bool{"darwin": true}
	knownArch     = map[string]bool{"arm64": true, "amd64": true}
)

// ErrNoRecipe reports that no recipe applies to the current platform or
// environment, so the app cannot be offered for installation.
var ErrNoRecipe = errors.New("ninguna receta de instalación aplicable")

// Recipe is one installation method declared in a `[[install]]` block.
type Recipe struct {
	Method  string `toml:"method"`
	Formula string `toml:"formula"`
	Cask    string `toml:"cask"`
	URL     string `toml:"url"`
	SHA256  string `toml:"sha256"`
	Dest    string `toml:"dest"`
	OS      string `toml:"os"`
	Arch    string `toml:"arch"`
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
		if r.URL != "" || r.SHA256 != "" || r.Dest != "" {
			return errors.New("'brew' no admite 'url', 'sha256' ni 'dest'")
		}
	case MethodInstaller:
		if err := requireHTTPS(r.URL); err != nil {
			return err
		}
		if r.Formula != "" || r.Cask != "" || r.SHA256 != "" || r.Dest != "" {
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
		if r.Formula != "" || r.Cask != "" {
			return errors.New("'binary' no admite 'formula' ni 'cask'")
		}
	default:
		return fmt.Errorf("método de instalación %q no admitido: usa 'brew', 'installer' o 'binary'", r.Method)
	}
	if r.OS != "" && !knownOS[r.OS] {
		return fmt.Errorf("'os' no reconocido %q: usa 'darwin'", r.OS)
	}
	if r.Arch != "" && !knownArch[r.Arch] {
		return fmt.Errorf("'arch' no reconocida %q: usa 'arm64' o 'amd64'", r.Arch)
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
