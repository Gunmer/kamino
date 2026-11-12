// Package detect checks whether catalog applications are installed.
package detect

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/plist"
)

type Options struct {
	Home            string
	ApplicationDirs []string
	Path            string
}

// IsInstalled reports whether the app is installed by bundle id or command.
func IsInstalled(app *catalog.AppDefinition, options Options) bool {
	home := options.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if app.InstallMarker != nil {
		if !markerExists(*app.InstallMarker, home) {
			return false
		}
	}
	hasBundle := false
	if app.BundleID != nil {
		hasBundle = findAppBundle(*app.BundleID, applicationDirs(options, home)) != ""
	}
	commands := make([]string, 0, 1+len(app.CommandAliases))
	if app.Command != nil {
		commands = append(commands, *app.Command)
	}
	commands = append(commands, app.CommandAliases...)
	hasCommand := false
	for _, command := range commands {
		if command == "" {
			continue
		}
		if findCommand(command, app.SearchPaths, options.Path, home) != "" {
			hasCommand = true
			break
		}
	}
	return hasBundle || hasCommand
}

// DetectVersion returns the version of a .app bundle, if applicable.
func DetectVersion(app *catalog.AppDefinition, options Options) string {
	if app.BundleID == nil {
		return ""
	}
	home := options.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	bundle := findAppBundle(*app.BundleID, applicationDirs(options, home))
	if bundle == "" {
		return ""
	}
	data := plistOf(bundle)
	if data == nil {
		return ""
	}
	if value, ok := data["CFBundleShortVersionString"].(string); ok && value != "" {
		return value
	}
	if value, ok := data["CFBundleVersion"].(string); ok {
		return value
	}
	return ""
}

func applicationDirs(options Options, home string) []string {
	if options.ApplicationDirs != nil {
		return options.ApplicationDirs
	}
	return []string{"/Applications", filepath.Join(home, "Applications"), "/System/Applications"}
}

func markerExists(marker, home string) bool {
	safe, err := paths.NewSafeHome(home, false)
	if err != nil {
		return false
	}
	defer safe.Close()
	info, err := safe.Lstat(marker)
	if err != nil {
		return false
	}
	return info.Kind == paths.KindFile
}

func appBundles(directory string) []string {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	candidates := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app") {
			continue
		}
		candidates = append(candidates, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(candidates)
	return candidates
}

func findAppBundle(bundleID string, directories []string) string {
	for _, directory := range directories {
		for _, candidate := range appBundles(directory) {
			data := plistOf(candidate)
			if data == nil {
				continue
			}
			if value, ok := data["CFBundleIdentifier"].(string); ok && value == bundleID {
				return candidate
			}
		}
	}
	return ""
}

func plistOf(appBundle string) map[string]any {
	infoPath := filepath.Join(appBundle, "Contents", "Info.plist")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return nil
	}
	value, err := plist.Parse(data)
	if err != nil {
		return nil
	}
	result, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return result
}

func findCommand(command string, searchPaths []string, pathValue, home string) string {
	if strings.Contains(command, "/") {
		if isExecutableFile(command) {
			return command
		}
		return ""
	}
	pathValue = effectivePath(pathValue, home)
	for _, directory := range filepath.SplitList(pathValue) {
		if directory == "" {
			directory = "."
		}
		candidate := filepath.Join(directory, command)
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	for _, directory := range searchPaths {
		candidate := filepath.Join(directory, command)
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	return ""
}

func effectivePath(pathValue, home string) string {
	if pathValue != "" {
		return pathValue
	}
	if value := os.Getenv("PATH"); value != "" {
		return value
	}
	return strings.Join([]string{
		filepath.Join(home, ".local", "bin"),
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	}, string(os.PathListSeparator))
}

func isExecutableFile(candidate string) bool {
	info, err := os.Stat(candidate)
	if err != nil {
		return false
	}
	if !info.Mode().IsRegular() {
		return false
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}
