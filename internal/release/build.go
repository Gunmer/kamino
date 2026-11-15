package release

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// AssetName is the release asset produced by Build.
const AssetName = "kamino-arm64"

// Build compiles the standalone macOS ARM64 executable with the release version.
func Build(root, version, output string) error {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("el binario de release debe construirse en macOS ARM64")
	}
	if _, err := ParseVersion(version); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	ldflags := "-s -w " +
		"-X github.com/Gunmer/kamino/internal/version.Version=" + version + " " +
		"-X github.com/Gunmer/kamino/internal/buildinfo.SelfContained=true"
	command := exec.Command(
		"go", "build", "-trimpath", "-ldflags", ldflags, "-o", output, "./cmd/kamino",
	)
	command.Dir = root
	command.Env = withEnv(os.Environ(), map[string]string{"CGO_ENABLED": "0"})
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("go build: %v\n%s", err, stderr.String())
	}
	return nil
}

// Smoke checks the version, help output, embedded catalog and HOME isolation.
func Smoke(binary, version string) error {
	directory, err := os.MkdirTemp("", "kamino-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)

	environment := []string{"HOME=" + directory, "KAMINO_PLAIN=1"}
	if _, err := runBinary(binary, environment, "--help"); err != nil {
		return err
	}
	output, err := runBinary(binary, environment, "--version")
	if err != nil {
		return err
	}
	if strings.TrimSpace(output) != "kamino "+version {
		return fmt.Errorf("la versión del binario no coincide con la release: %q", output)
	}

	apps := filepath.Join(directory, ".config", "kamino", "apps")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		return err
	}
	invalid := filepath.Join(apps, "smoke.toml")
	if err := os.WriteFile(invalid, []byte("invalid = ["), 0o644); err != nil {
		return err
	}
	before, err := snapshotHome(directory)
	if err != nil {
		return err
	}
	command := exec.Command(binary, "clone", "--dry-run")
	command.Dir = directory
	command.Env = withEnv(os.Environ(), map[string]string{"HOME": directory, "KAMINO_PLAIN": "1"})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	exitError, ok := err.(*exec.ExitError)
	if !ok || exitError.ExitCode() != 1 || !strings.Contains(stderr.String(), "TOML inválido") {
		return fmt.Errorf("falló la comprobación del catálogo embebido: %v\n%s", err, stderr.String())
	}
	after, err := snapshotHome(directory)
	if err != nil {
		return err
	}
	if !sameSnapshot(before, after) {
		return fmt.Errorf("la comprobación del binario escribió en el HOME temporal")
	}
	content, err := os.ReadFile(invalid)
	if err != nil || string(content) != "invalid = [" {
		return fmt.Errorf("la comprobación del binario modificó el catálogo temporal")
	}
	return nil
}

func runBinary(binary string, environment []string, arguments ...string) (string, error) {
	command := exec.Command(binary, arguments...)
	command.Env = withEnv(os.Environ(), envMap(environment))
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("%s %v: %v\n%s", binary, arguments, err, stderr.String())
	}
	return string(output), nil
}

func envMap(environment []string) map[string]string {
	result := make(map[string]string, len(environment))
	for _, item := range environment {
		key, value, found := strings.Cut(item, "=")
		if found {
			result[key] = value
		}
	}
	return result
}

func withEnv(environment []string, overrides map[string]string) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, item := range environment {
		key, _, found := strings.Cut(item, "=")
		if found {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		result = append(result, item)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func snapshotHome(home string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(home, path)
		if err != nil || relative == "." {
			return err
		}
		if entry.IsDir() {
			result[relative] = "<dir>"
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[relative] = string(content)
		return nil
	})
	return result, err
}

func sameSnapshot(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, content := range left {
		if right[path] != content {
			return false
		}
	}
	return true
}
