// Package config reads and writes the TUI configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/Gunmer/kamino/internal/defaults"
	kerrors "github.com/Gunmer/kamino/internal/errors"
)

const (
	CatalogKey   = "catalog.dir"
	CloneKey     = "clone.output_dir"
	SnapshotsKey = "deploy.snapshots_dir"
)

var sectionKeys = map[string]map[string]string{
	"catalog": {"dir": CatalogKey},
	"clone":   {"output_dir": CloneKey},
	"deploy":  {"snapshots_dir": SnapshotsKey},
}

// AppConfig holds the three effective TUI paths.
type AppConfig struct {
	CatalogDir     string
	CloneOutputDir string
	SnapshotsDir   string
}

// LoadResult is the result of loading the configuration.
type LoadResult struct {
	Config AppConfig
	Error  error
	Exists bool
	Raw    map[string]string
}

// ConfigPath returns the fixed TUI configuration path.
func ConfigPath(home string) string {
	return filepath.Join(homeOrUser(home), ".config", "kamino", "config.toml")
}

// DefaultConfig returns the built-in paths.
func DefaultConfig(home string) AppConfig {
	root := homeOrUser(home)
	return AppConfig{
		CatalogDir:     defaults.CatalogDir(root),
		CloneOutputDir: defaults.VaultDir(root),
		SnapshotsDir:   defaults.SnapshotsDir(root),
	}
}

// LoadConfig loads the TUI configuration, keeping readable values on failure.
func LoadConfig(home, path string) LoadResult {
	root := homeOrUser(home)
	target := path
	if target == "" {
		target = ConfigPath(root)
	}
	defaultsValue := DefaultConfig(root)
	if _, err := os.Lstat(target); err != nil {
		return LoadResult{Config: defaultsValue, Raw: map[string]string{}}
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return LoadResult{Config: defaultsValue, Exists: true, Raw: map[string]string{}, Error: invalid(target, "no se pudo leer el archivo: "+err.Error())}
	}
	var parsed map[string]any
	if _, err := toml.Decode(string(data), &parsed); err != nil {
		return LoadResult{Config: defaultsValue, Exists: true, Raw: map[string]string{}, Error: invalid(target, "TOML inválido: "+err.Error())}
	}

	values := map[string]string{}
	raw := map[string]string{}
	errorsList := make([]string, 0)
	for section, value := range parsed {
		allowed, ok := sectionKeys[section]
		if !ok {
			errorsList = append(errorsList, fmt.Sprintf("sección desconocida: [%s]", section))
			continue
		}
		table, ok := value.(map[string]any)
		if !ok {
			errorsList = append(errorsList, fmt.Sprintf("[%s] debe ser una tabla", section))
			continue
		}
		for key, rawValue := range table {
			fullKey := section + "." + key
			if _, ok := allowed[key]; !ok {
				errorsList = append(errorsList, "clave desconocida: "+fullKey)
				continue
			}
			text, ok := rawValue.(string)
			if !ok {
				errorsList = append(errorsList, fullKey+" debe ser una cadena")
				continue
			}
			raw[fullKey] = text
			resolved, err := ResolvePath(text, root, fullKey)
			if err != nil {
				errorsList = append(errorsList, err.Error())
				continue
			}
			values[fullKey] = resolved
		}
	}

	config := AppConfig{
		CatalogDir:     fallback(values[CatalogKey], defaultsValue.CatalogDir),
		CloneOutputDir: fallback(values[CloneKey], defaultsValue.CloneOutputDir),
		SnapshotsDir:   fallback(values[SnapshotsKey], defaultsValue.SnapshotsDir),
	}
	result := LoadResult{Config: config, Exists: true, Raw: raw}
	if len(errorsList) > 0 {
		result.Error = invalid(target, strings.Join(errorsList, "; "))
	}
	return result
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}

// ResolvePath validates and expands a configured path against the home.
func ResolvePath(value, home, key string) (string, error) {
	root := homeOrUser(home)
	text := strings.TrimSpace(value)
	if text == "" {
		return "", configError(key + " no puede estar vacía")
	}
	var resolved string
	switch {
	case text == "~":
		resolved = root
	case strings.HasPrefix(text, "~/"):
		rest := text[2:]
		if strings.HasPrefix(rest, "/") {
			return "", configError(key + ": solo se admite ~ o ~/ al inicio")
		}
		resolved = filepath.Join(root, filepath.FromSlash(rest))
	case strings.HasPrefix(text, "~"):
		return "", configError(key + ": solo se admite ~ o ~/ al inicio")
	default:
		if !filepath.IsAbs(text) {
			return "", configError(key + ": usa una ruta absoluta o que empiece por ~")
		}
		resolved = filepath.Clean(text)
	}
	if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
		return "", configError(key + ": la ruta existe y no es un directorio")
	}
	return resolved, nil
}

// SaveConfig writes the three paths atomically and returns the file path.
func SaveConfig(config AppConfig, home, path string) (string, error) {
	root := homeOrUser(home)
	target := path
	if target == "" {
		target = ConfigPath(root)
	}
	if err := validateConfigPaths(config); err != nil {
		return "", err
	}
	content := serialize(config)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", configError("no se pudo guardar " + target + ": " + err.Error())
	}
	temp := filepath.Join(filepath.Dir(target), ".config-"+randomSuffix()+".tmp")
	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", configError("no se pudo guardar " + target + ": " + err.Error())
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(temp)
		}
	}()
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return "", configError("no se pudo guardar " + target + ": " + err.Error())
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", configError("no se pudo guardar " + target + ": " + err.Error())
	}
	if err := file.Close(); err != nil {
		return "", configError("no se pudo guardar " + target + ": " + err.Error())
	}
	if err := os.Rename(temp, target); err != nil {
		return "", configError("no se pudo guardar " + target + ": " + err.Error())
	}
	cleanup = false
	return target, nil
}

func validateConfigPaths(config AppConfig) error {
	values := [][2]string{
		{CatalogKey, config.CatalogDir},
		{CloneKey, config.CloneOutputDir},
		{SnapshotsKey, config.SnapshotsDir},
	}
	for _, item := range values {
		if !filepath.IsAbs(item[1]) {
			return configError(item[0] + ": usa una ruta absoluta o que empiece por ~")
		}
		if info, err := os.Stat(item[1]); err == nil && !info.IsDir() {
			return configError(item[0] + ": la ruta existe y no es un directorio")
		}
	}
	return nil
}

func serialize(config AppConfig) string {
	return "[catalog]\n" +
		"dir = " + tomlString(config.CatalogDir) + "\n\n" +
		"[clone]\n" +
		"output_dir = " + tomlString(config.CloneOutputDir) + "\n\n" +
		"[deploy]\n" +
		"snapshots_dir = " + tomlString(config.SnapshotsDir) + "\n"
}

func tomlString(value string) string {
	return strconv.Quote(value)
}

func invalid(target, detail string) error {
	return configError("configuración inválida en " + target + ": " + detail)
}

func configError(message string) error {
	return &kerrors.ConfigError{KaminoError: kerrors.KaminoError{Msg: message}}
}

func homeOrUser(home string) string {
	if home != "" {
		return home
	}
	value, _ := os.UserHomeDir()
	return value
}

func randomSuffix() string {
	return strconv.FormatInt(int64(os.Getpid()), 36) + strconv.FormatInt(hashNow(), 36)
}

func hashNow() int64 {
	info, err := os.Stat(".")
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano()
}
