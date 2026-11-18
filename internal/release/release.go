package release

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var repositoryPattern = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)
var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Run executes the requested release flow.
func Run(root, sha, tag string) error {
	repository, err := repositorySlug(root)
	if err != nil {
		return err
	}
	if err := requireClean(root); err != nil {
		return err
	}
	if err := runGitOK(root, "fetch", "origin", "main", "--tags"); err != nil {
		return err
	}

	if tag != "" {
		if err := recoverRelease(root, tag); err != nil {
			return err
		}
		return Publish(root, repository, tag, filepath.Join(root, "dist", AssetName))
	}

	releaseTag, err := prepareRelease(root, repository, sha)
	if err != nil || releaseTag == "" {
		return err
	}
	return Publish(root, repository, releaseTag, filepath.Join(root, "dist", AssetName))
}

func prepareRelease(root, repository, sha string) (string, error) {
	if !shaPattern.MatchString(sha) {
		return "", fmt.Errorf("el SHA de CI no es válido")
	}
	remote, err := remoteMain(root)
	if err != nil {
		return "", err
	}
	head, err := currentHead(root)
	if err != nil {
		return "", err
	}
	if remote != sha || head != sha {
		fmt.Println("Ejecución obsoleta: main cambió; no se publicará una nueva versión.")
		return "", nil
	}
	branch, err := runGit(root, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	if branch != "main" {
		return "", fmt.Errorf("la publicación automática requiere la rama main")
	}

	previousTag, err := lastTag(root)
	if err != nil {
		return "", err
	}
	var current string
	if previousTag != "" {
		current = strings.TrimPrefix(previousTag, "v")
	} else {
		value, readErr := ReadVersionFile(filepath.Join(root, "internal", "version", "version.go"))
		if readErr != nil {
			return "", readErr
		}
		current = value
	}
	commits, err := commitsSince(root, previousTag, "HEAD")
	if err != nil {
		return "", err
	}
	next, bump, err := NextVersion(current, commits)
	if err != nil {
		return "", err
	}
	if bump == BumpNone {
		fmt.Println("No hay cambios publicables desde el último tag; no se genera una versión.")
		return "", nil
	}
	tag := "v" + next
	exists, err := tagExists(root, tag)
	if err != nil {
		return "", err
	}
	if exists {
		fmt.Println("No hay cambios publicables desde el último tag; no se genera una versión.")
		return "", nil
	}

	entry, err := RenderChangelogEntry(next, time.Now(), commits, func(commit string) string {
		return "https://github.com/" + repository + "/commit/" + commit
	})
	if err != nil {
		return "", err
	}
	if err := writeRelease(root, next, entry); err != nil {
		return "", err
	}

	binary := filepath.Join(root, "dist", AssetName)
	if err := Build(root, next, binary); err != nil {
		return "", err
	}
	if err := Smoke(binary, next); err != nil {
		return "", err
	}
	if err := commitRelease(root, tag, next); err != nil {
		return "", err
	}
	if err := validateRelease(root, tag); err != nil {
		return "", err
	}
	parent, err := runGit(root, "rev-parse", "HEAD^")
	if err != nil {
		return "", err
	}
	if parent != sha {
		return "", fmt.Errorf("main cambió durante la construcción; se cancela el push de la release")
	}
	remote, err = remoteMain(root)
	if err != nil {
		return "", err
	}
	if remote != sha {
		return "", fmt.Errorf("main cambió durante la construcción; se cancela el push de la release")
	}
	if err := runGitOK(root, "push", "--atomic", "origin",
		"HEAD:refs/heads/main", "refs/tags/"+tag+":refs/tags/"+tag); err != nil {
		return "", err
	}
	return tag, nil
}

func writeRelease(root, version, entry string) error {
	changelogPath := filepath.Join(root, "CHANGELOG.md")
	changelog, err := os.ReadFile(changelogPath)
	if err != nil {
		return err
	}
	updated, err := InsertChangelogEntry(string(changelog), entry)
	if err != nil {
		return err
	}
	if err := os.WriteFile(changelogPath, []byte(updated), 0o644); err != nil {
		return err
	}
	return WriteVersionFile(filepath.Join(root, "internal", "version", "version.go"), version)
}

func commitRelease(root, tag, version string) error {
	if err := runGitOK(root, "add", "--all"); err != nil {
		return err
	}
	if err := runGitOK(root, "-c",
		"commit.gpgsign=false", "-c",
		"user.name=github-actions[bot]", "-c",
		"user.email=41898282+github-actions[bot]@users.noreply.github.com", "commit", "--author",
		"github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>",
		"-m", "chore(release): "+version+" [skip ci]"); err != nil {
		return err
	}
	return runGitOK(root, "-c", "tag.gpgSign=false", "tag", tag)
}

func recoverRelease(root, tag string) error {
	if _, err := ValidateTag(tag); err != nil {
		return err
	}
	if err := validateRelease(root, tag); err != nil {
		return err
	}
	if err := runGitOK(root, "merge-base", "--is-ancestor", tag, "origin/main"); err != nil {
		return fmt.Errorf("el tag %s no está integrado en main", tag)
	}
	if err := runGitOK(root, "switch", "--detach", tag); err != nil {
		return err
	}
	binary := filepath.Join(root, "dist", AssetName)
	version, _ := ValidateTag(tag)
	if err := Build(root, version, binary); err != nil {
		return err
	}
	return Smoke(binary, version)
}

func validateRelease(root, tag string) error {
	version, err := ValidateTag(tag)
	if err != nil {
		return err
	}
	subject, err := runGit(root, "log", "-1", "--format=%s", tag)
	if err != nil {
		return err
	}
	if subject != "chore(release): "+version+" [skip ci]" {
		return fmt.Errorf("el tag no identifica un commit de release de Kamino")
	}
	source, err := runGit(root, "show", tag+":internal/version/version.go")
	if err != nil {
		return err
	}
	if !strings.Contains(source, `var Version = "`+version+`"`) {
		return fmt.Errorf("el tag y la versión Go no coinciden")
	}
	changelog, err := runGit(root, "show", tag+":CHANGELOG.md")
	if err != nil {
		return err
	}
	if _, err := ReleaseNotes(changelog, tag); err != nil {
		return err
	}
	return nil
}

func repositorySlug(root string) (string, error) {
	if value := os.Getenv("GH_REPO"); value != "" {
		if !repositoryPattern.MatchString(value) {
			return "", fmt.Errorf("GH_REPO debe identificar el repositorio de GitHub")
		}
		return value, nil
	}
	remote, err := runGit(root, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("GH_REPO no está definido y no se encontró el remoto origin")
	}
	value := strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	for _, prefix := range []string{"git@github.com:", "https://github.com/", "ssh://git@github.com/"} {
		if strings.HasPrefix(value, prefix) {
			slug := strings.TrimPrefix(value, prefix)
			if repositoryPattern.MatchString(slug) {
				return slug, nil
			}
		}
	}
	return "", fmt.Errorf("no se pudo identificar el repositorio de GitHub desde origin")
}

func requireClean(root string) error {
	status, err := runGit(root, "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("la publicación requiere un árbol de trabajo limpio")
	}
	return nil
}

func remoteMain(root string) (string, error) {
	output, err := runGit(root, "ls-remote", "origin", "refs/heads/main")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(output)
	if len(fields) == 0 {
		return "", fmt.Errorf("no se pudo resolver origin/main")
	}
	return fields[0], nil
}

func currentHead(root string) (string, error) {
	return runGit(root, "rev-parse", "HEAD")
}

func lastTag(root string) (string, error) {
	output, err := runGit(root, "describe", "--tags", "--abbrev=0", "--match", "v[0-9]*", "HEAD")
	if err == nil {
		return output, nil
	}
	tags, tagErr := runGit(root, "tag", "--list", "v[0-9]*")
	if tagErr == nil && strings.TrimSpace(tags) == "" {
		return "", nil
	}
	return "", err
}

func tagExists(root, tag string) (bool, error) {
	output, err := runGit(root, "tag", "--list", tag)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func commitsSince(root, from, to string) ([]Commit, error) {
	arguments := []string{"log", "--no-merges", "--format=%H%x1f%B%x1e"}
	if from != "" {
		arguments = append(arguments, from+".."+to)
	} else {
		arguments = append(arguments, to)
	}
	output, err := runGit(root, arguments...)
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, record := range strings.Split(output, "\x1e") {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, "\x1f", 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, ParseCommit(strings.TrimSpace(parts[0]), parts[1])...)
	}
	return commits, nil
}

func runGit(root string, arguments ...string) (string, error) {
	command := exec.Command("git", arguments...)
	command.Dir = root
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(arguments, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func runGitOK(root string, arguments ...string) error {
	_, err := runGit(root, arguments...)
	return err
}
