package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type releaseAsset struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	Size  int64  `json:"size"`
}

type releaseInfo struct {
	Draft  bool           `json:"draft"`
	Assets []releaseAsset `json:"assets"`
}

// Publish creates or completes the GitHub release for a tag.
func Publish(root, repository, tag, binary string) error {
	if _, err := ValidateTag(tag); err != nil {
		return err
	}
	changelog, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		return err
	}
	notes, err := ReleaseNotes(string(changelog), tag)
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "kamino-release-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(directory) }()
	notesFile := filepath.Join(directory, "notes.md")
	if err := os.WriteFile(notesFile, []byte(notes), 0o644); err != nil {
		return err
	}

	release, err := githubRelease(repository, tag)
	if err != nil {
		return err
	}
	if release == nil {
		if err := runGH("release", "create", tag, "--verify-tag", "--draft",
			"--title", tag, "--notes-file", notesFile, "--repo", repository); err != nil {
			return err
		}
	} else if release.Draft {
		if err := runGH("release", "edit", tag, "--notes-file", notesFile, "--repo", repository); err != nil {
			return err
		}
	}

	uploaded := false
	if release != nil {
		for _, asset := range release.Assets {
			if asset.Name != AssetName {
				continue
			}
			if asset.State == "uploaded" {
				uploaded = true
				continue
			}
			if asset.State != "starter" || asset.Size != 0 {
				return fmt.Errorf("el asset existente está incompleto y requiere revisión manual")
			}
			if err := runGH("api", "--method", "DELETE",
				fmt.Sprintf("repos/%s/releases/assets/%d", repository, asset.ID)); err != nil {
				return err
			}
		}
	}
	if !uploaded {
		if err := runGH("release", "upload", tag, binary, "--repo", repository); err != nil {
			return err
		}
	}
	if release == nil || release.Draft {
		if err := runGH("release", "edit", tag, "--draft=false", "--repo", repository); err != nil {
			return err
		}
	}
	fmt.Printf("Release %s publicada con %s.\n", tag, filepath.Base(binary))
	return nil
}

func githubRelease(repository, tag string) (*releaseInfo, error) {
	command := exec.Command("gh", "api", fmt.Sprintf("repos/%s/releases/tags/%s", repository, tag))
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if strings.Contains(stderr.String(), "HTTP 404") {
			return nil, nil
		}
		return nil, fmt.Errorf("no se pudo consultar la release: %s", stderr.String())
	}
	var release releaseInfo
	if err := json.Unmarshal(stdout.Bytes(), &release); err != nil {
		return nil, err
	}
	return &release, nil
}

func runGH(arguments ...string) error {
	command := exec.Command("gh", arguments...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh %s: %w\n%s", strings.Join(arguments, " "), err, output)
	}
	return nil
}
