package app

import (
	"fmt"
	"os"
	"sort"

	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/detect"
)

// AppListEntry is one row of the merged catalog listing.
type AppListEntry struct {
	ID       string
	Name     string
	File     string
	Source   string
	Err      error
	Warnings []string
}

// AppsListOutcome is the result of listing the catalog.
type AppsListOutcome struct {
	Directory   string
	ConfigError error
	Entries     []AppListEntry
}

// AppValidateEntry is one validated file.
type AppValidateEntry struct {
	File     string
	ID       string
	Name     string
	Err      error
	Warnings []string
}

// AppsValidateOutcome is the result of validating a file, a directory or the
// effective user catalog.
type AppsValidateOutcome struct {
	Directory   string
	ConfigError error
	Entries     []AppValidateEntry
	Valid       bool
}

// AppsNewOptions carries the flags of `kamino apps new`.
type AppsNewOptions struct {
	ID            string
	Name          string
	Command       string
	BundleID      string
	Paths         []string
	OptionalPaths []string
	SearchPaths   []string
	Force         bool
	Print         bool
}

// AppsNewOutcome is the result of creating one definition.
type AppsNewOutcome struct {
	Content     string
	Path        string
	Directory   string
	ConfigError error
	Warnings    []string
	Detected    bool
	Definition  *catalog.AppDefinition
}

// ExecuteAppsList returns the merged catalog without failing on invalid files.
func ExecuteAppsList(home string) (AppsListOutcome, error) {
	directory, loaded, err := catalogResolution(home)
	if err != nil {
		return AppsListOutcome{}, err
	}
	embedded, err := catalog.Embedded()
	if err != nil {
		return AppsListOutcome{}, err
	}
	user, err := catalog.InspectDir(directory)
	if err != nil {
		return AppsListOutcome{}, err
	}
	valid := map[string]AppListEntry{}
	for id, definition := range embedded {
		valid[id] = AppListEntry{ID: id, Name: definition.Name, Source: "integrado"}
	}
	invalid := make([]AppListEntry, 0)
	for _, result := range user {
		entry := AppListEntry{
			ID: result.ID, Name: result.Name, File: result.File,
			Source: "usuario", Err: result.Err, Warnings: result.Warnings,
		}
		if result.Definition != nil {
			if _, exists := embedded[result.ID]; exists {
				entry.Warnings = append(entry.Warnings, "reemplaza la definición integrada")
			}
		}
		if result.Err != nil || result.Definition == nil {
			invalid = append(invalid, entry)
			continue
		}
		valid[result.ID] = entry
	}
	entries := make([]AppListEntry, 0, len(valid)+len(invalid))
	for _, entry := range valid {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	sort.Slice(invalid, func(i, j int) bool { return invalid[i].File < invalid[j].File })
	entries = append(entries, invalid...)
	return AppsListOutcome{Directory: directory, ConfigError: loaded.Error, Entries: entries}, nil
}

// ExecuteAppsValidate validates a file, a directory or the effective user
// catalog. A configuration error does not make the catalog invalid.
func ExecuteAppsValidate(home, path string) (AppsValidateOutcome, error) {
	outcome := AppsValidateOutcome{Valid: true}
	var results []catalog.DefinitionResult
	if path == "" {
		directory, loaded, err := catalogResolution(home)
		if err != nil {
			return AppsValidateOutcome{}, err
		}
		outcome.Directory = directory
		outcome.ConfigError = loaded.Error
		results, err = catalog.InspectDir(directory)
		if err != nil {
			return AppsValidateOutcome{}, err
		}
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return AppsValidateOutcome{}, err
		}
		if info.IsDir() {
			outcome.Directory = path
			results, err = catalog.InspectDir(path)
			if err != nil {
				return AppsValidateOutcome{}, err
			}
		} else {
			results = []catalog.DefinitionResult{catalog.InspectFile(path)}
		}
	}
	embedded, err := catalog.Embedded()
	if err != nil {
		return AppsValidateOutcome{}, err
	}
	for _, result := range results {
		entry := AppValidateEntry{
			File: result.File, ID: result.ID, Name: result.Name,
			Err: result.Err, Warnings: result.Warnings,
		}
		if entry.Err != nil {
			outcome.Valid = false
		}
		if result.Definition != nil {
			if _, exists := embedded[result.ID]; exists {
				entry.Warnings = append(entry.Warnings, "reemplaza la definición integrada")
			}
		}
		outcome.Entries = append(outcome.Entries, entry)
	}
	return outcome, nil
}

// ExecuteAppsNew validates one definition and writes it atomically, or returns
// the rendered TOML when Print is set.
func ExecuteAppsNew(home string, options AppsNewOptions, applicationDirs []string, pathValue string) (AppsNewOutcome, error) {
	content, err := catalog.RenderTemplate(catalog.TemplateOptions{
		ID:            options.ID,
		Name:          options.Name,
		Command:       options.Command,
		BundleID:      options.BundleID,
		SearchPaths:   options.SearchPaths,
		Paths:         options.Paths,
		OptionalPaths: options.OptionalPaths,
	})
	if err != nil {
		return AppsNewOutcome{}, err
	}
	definition, err := catalog.Parse(options.ID+".toml", []byte(content))
	if err != nil {
		return AppsNewOutcome{}, err
	}
	outcome := AppsNewOutcome{Content: content, Definition: definition}
	if options.Print {
		return outcome, nil
	}
	directory, loaded, err := catalogResolution(home)
	if err != nil {
		return AppsNewOutcome{}, err
	}
	outcome.Directory = directory
	outcome.ConfigError = loaded.Error
	result, err := catalog.CreateDefinition(directory, []byte(content), options.Force)
	if err != nil {
		return AppsNewOutcome{}, err
	}
	outcome.Path = result.Path
	outcome.Definition = result.Definition
	outcome.Warnings = result.Warnings
	outcome.Detected = detect.IsInstalled(result.Definition, detect.Options{
		Home: home, ApplicationDirs: applicationDirs, Path: pathValue,
	})
	if !outcome.Detected {
		outcome.Warnings = append(outcome.Warnings, fmt.Sprintf(
			"'%s' no está instalada; aparecerá en Clonar cuando lo esté", result.Definition.ID,
		))
	}
	return outcome, nil
}

// AppsCustomizeOptions carries the flags of `kamino apps customize`.
type AppsCustomizeOptions struct {
	ID    string
	Force bool
	Print bool
}

// ExecuteAppsCustomize copies the built-in definition of one app into the user
// catalog, or returns its TOML when Print is set.
func ExecuteAppsCustomize(home string, options AppsCustomizeOptions) (AppsNewOutcome, error) {
	data, err := catalog.EmbeddedSource(options.ID)
	if err != nil {
		return AppsNewOutcome{}, err
	}
	definition, err := catalog.Parse(options.ID+".toml", data)
	if err != nil {
		return AppsNewOutcome{}, err
	}
	outcome := AppsNewOutcome{Content: string(data), Definition: definition}
	if options.Print {
		return outcome, nil
	}
	directory, loaded, err := catalogResolution(home)
	if err != nil {
		return AppsNewOutcome{}, err
	}
	outcome.Directory = directory
	outcome.ConfigError = loaded.Error
	result, err := catalog.CreateDefinition(directory, data, options.Force)
	if err != nil {
		return AppsNewOutcome{}, err
	}
	outcome.Path = result.Path
	outcome.Definition = result.Definition
	outcome.Warnings = result.Warnings
	return outcome, nil
}
