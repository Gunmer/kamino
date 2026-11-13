package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/catalog"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/conflicts"
	"github.com/Gunmer/kamino/internal/deploy"
	"github.com/Gunmer/kamino/internal/plan"
	"github.com/Gunmer/kamino/internal/update"
)

func TestMain(m *testing.M) {
	_ = os.Unsetenv("NO_COLOR")
	_ = os.Unsetenv("KAMINO_NO_LOGO")
	_ = os.Unsetenv("TEXTUAL_THEME")
	os.Exit(m.Run())
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 13, 10, 30, 0, 0, time.UTC)
}

func fixtureCatalog() catalog.Catalog {
	names := []string{"herdr", "ghostty", "git", "zsh", "vscode", "atuin", "starship", "iterm2", "yazi"}
	result := catalog.Catalog{}
	for _, name := range names {
		result[name] = &catalog.AppDefinition{ID: name, Name: name}
	}
	return result
}

func fixtureApps() []appChoice {
	return []appChoice{
		{ID: "herdr", Name: "herdr", Detected: true, Selected: true, Count: 12},
		{ID: "ghostty", Name: "ghostty", Detected: true, Selected: true, Count: 8},
		{ID: "git", Name: "git", Detected: true, Count: 4},
		{ID: "zsh", Name: "zsh", Detected: true, Count: 3},
		{ID: "vscode", Name: "vscode", Detected: true, Count: 0},
		{ID: "atuin", Name: "atuin", Detected: true, Count: 6},
		{ID: "starship", Name: "starship", Detected: true, Count: 2},
		{ID: "iterm2", Name: "iterm2"},
		{ID: "yazi", Name: "yazi"},
	}
}

func fixtureClonePlan() *clone.ClonePlan {
	operations := make([]plan.Operation, 0, 20)
	for index := 0; index < 20; index++ {
		operations = append(operations, plan.Operation{
			AppID: "herdr", Path: fmt.Sprintf(".config/herdr/file-%02d.conf", index), Size: 62914,
		})
	}
	skipped := []plan.SkippedPath{
		{Path: ".config/herdr/cache", Reason: "excluido"},
		{Path: ".config/herdr/socket", Reason: "no es un archivo regular"},
		{Path: ".config/herdr/lock", Reason: "excluido"},
	}
	return &clone.ClonePlan{
		Plan: plan.Plan{Operations: operations, Skipped: skipped},
		Manifest: &archive.Manifest{Apps: []archive.AppEntry{
			{ID: "herdr", Name: "herdr", Files: make([]archive.FileEntry, 12)},
			{ID: "ghostty", Name: "ghostty", Files: make([]archive.FileEntry, 8)},
		}},
		Output: "/Users/test/.config/kamino/vault/clone-20261113.tar.gz",
		Apps: []*catalog.AppDefinition{
			{ID: "herdr", Name: "herdr"}, {ID: "ghostty", Name: "ghostty"},
		},
	}
}

func fixtureCloneResult() *clone.CloneResult {
	planned := fixtureClonePlan()
	return &clone.CloneResult{
		Plan: planned.Plan, Manifest: planned.Manifest, Output: planned.Output,
		PlanOutput: planned.Output,
	}
}

func makeConflict(appID, path, local, cloneText string) conflicts.Conflict {
	entry := archive.FileEntry{Path: path, Size: int64(len(cloneText)), Mode: "0644"}
	localFile := conflicts.LocalFile{Kind: "file", Data: []byte(local), Mode: "0644"}
	destination := "/Users/test/" + path
	value := conflicts.BuildConflict(appID, path, destination, entry, localFile, []byte(cloneText))
	if value == nil {
		panic("fixture conflict did not differ: " + path)
	}
	return *value
}

func fixtureConflicts() []conflicts.Conflict {
	localOne := strings.Join([]string{
		"# herdr", "alpha = 1", "beta = 2", "gamma = 3", "delta = 4",
		"one", "two", "three", "four", "five", "six", "gap1", "gap2",
		"epsilon = 5", "zeta = 6", "eta = 7", "theta = 8",
		"seven", "eight", "nine", "ten", "eleven", "twelve",
	}, "\n") + "\n"
	cloneOne := strings.Join([]string{
		"# herdr", "alpha = 10", "beta = 20", "gamma = 30", "delta = 40",
		"one", "two", "three", "four", "five", "six", "gap1", "gap2",
		"epsilon = 50", "zeta = 60", "eta = 70", "theta = 80",
		"seven", "eight", "nine", "ten", "eleven", "twelve",
	}, "\n") + "\n"
	localTwo := "key = 1\nvalue = 2\n"
	cloneTwo := "key = 9\nvalue = 2\n"
	return []conflicts.Conflict{
		makeConflict("herdr", ".config/herdr/config.toml", localOne, cloneOne),
		makeConflict("ghostty", ".config/ghostty/config", localTwo, cloneTwo),
	}
}

func fixtureConflictStates() []*conflictState {
	all := fixtureConflicts()
	states := make([]*conflictState, 0, len(all))
	for _, conflict := range all {
		states = append(states, &conflictState{
			Conflict: conflict, Action: conflicts.ActionReview,
			Rows: conflicts.ReviewAll(conflict, conflicts.RowClone).Rows,
		})
	}
	return states
}

func fixtureDeployPlan(states []*conflictState) *deploy.DeployPlan {
	entries := make([]deploy.DeployEntry, 0)
	for _, state := range states {
		entries = append(entries, deploy.DeployEntry{
			AppID: state.Conflict.AppID, Path: state.Conflict.Path,
			Destination: state.Conflict.Destination, Status: deploy.StatusDifferent,
		})
	}
	entries = append(entries,
		deploy.DeployEntry{AppID: "git", Path: ".config/git/config", Destination: "/Users/test/.config/git/config", Status: deploy.StatusNew},
		deploy.DeployEntry{AppID: "zsh", Path: ".config/zsh/.zshrc", Destination: "/Users/test/.config/zsh/.zshrc", Status: deploy.StatusIdentical},
		deploy.DeployEntry{AppID: "vscode", Path: ".config/vscode/settings.json", Destination: "/Users/test/.config/vscode/settings.json", Status: deploy.StatusUnsupported},
	)
	return &deploy.DeployPlan{
		Entries: entries, Conflicts: fixtureConflictList(states),
		Unchanged: []string{".config/zsh/.zshrc"}, Omitted: []string{".config/vscode/settings.json"},
		Notices: []string{"Arquitectura distinta: arm64 → x86_64"},
	}
}

func fixtureConflictList(states []*conflictState) []conflicts.Conflict {
	result := make([]conflicts.Conflict, 0, len(states))
	for _, state := range states {
		result = append(result, state.Conflict)
	}
	return result
}

func fixtureDeployResult() *deploy.DeployResult {
	plan := fixtureDeployPlan(fixtureConflictStates())
	return &deploy.DeployResult{
		Plan: plan, Output: "/Users/test/.config/kamino/vault/clone-20261113.tar.gz",
		Snapshot: "/Users/test/.config/kamino/vault/snapshot/20261113T103000Z.tar.gz",
		Deployed: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
		Merged:   []string{"j", "k"}, Omitted: []string{"l", "m"},
		Unchanged: []string{"n", "o"}, NotInstalled: []string{"p", "q"},
		SnapshotFiles: 8,
	}
}

func fixtureDeployCandidates() []deployCandidate {
	return []deployCandidate{
		{
			Path: "/Users/test/.config/kamino/vault/clone-20261113.tar.gz",
			Name: "clone-20261113.tar.gz", Origin: "por defecto",
			Date: time.Date(2026, 11, 13, 10, 30, 0, 0, time.UTC), Size: 1258291,
		},
		{
			Path: "/Users/test/Projects/kamino/clone-20261112.tar.gz",
			Name: "clone-20261112.tar.gz", Origin: "este directorio",
			Date: time.Date(2026, 11, 12, 18, 5, 0, 0, time.UTC), Size: 1003520,
		},
		{
			Path: "/Users/test/.config/kamino/vault/clone-20261001.tar.gz",
			Name: "clone-20261001.tar.gz", Origin: "por defecto",
			Date: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), Size: 1153433,
		},
	}
}

func fixtureUpdateCheck() update.Check {
	return update.Check{
		CurrentVersion: "0.9.0",
		Release: update.ReleaseInfo{
			Version: "0.9.1", AssetURL: "https://example.invalid/kamino-arm64",
			AssetSize: 12 * 1024 * 1024, AssetDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		Target: "/usr/local/bin/kamino", ShouldUpdate: true, Outcome: update.OutcomeUpdated,
	}
}

func fixtureUpdateResult() *update.Result {
	return &update.Result{
		CurrentVersion: "0.9.0", LatestVersion: "0.9.1",
		Target: "/usr/local/bin/kamino", Updated: true, Outcome: update.OutcomeUpdated,
	}
}

func testApp(noColor, unicode bool) *App {
	value := unicode
	app := NewApp(AppOptions{
		Home: "/Users/test", Cwd: "/Users/test/Projects/kamino",
		Now: fixedNow, Phrase: "La Fuerza estará contigo.",
		Unicode: &value, NoColor: noColor, ThemeName: "ansi-dark",
		Catalog: fixtureCatalog(), PathValue: "/opt/homebrew/bin",
	})
	app.width, app.height = 100, 26
	app.snapshotTimestamp = "2026-11-13T10:30:00Z"
	app.cloneOutput = "/Users/test/.config/kamino/vault/clone-20261113.tar.gz"
	return app
}
