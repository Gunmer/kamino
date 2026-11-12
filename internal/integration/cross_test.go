package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo resolver la raíz")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func pythonPath(t *testing.T) string {
	t.Helper()
	if value := os.Getenv("KAMINO_PYTHON"); value != "" {
		return value
	}
	candidates := []string{
		filepath.Join(repoRoot(t), "legacy", "python", ".venv", "bin", "python"),
		filepath.Join(repoRoot(t), ".venv", "bin", "python"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Skip("Python no disponible; define KAMINO_PYTHON")
	return ""
}

func runCommand(t *testing.T, directory, home string, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "HOME="+home, "KAMINO_PLAIN=1", "PYTHONPATH="+filepath.Join(repoRoot(t), "legacy", "python", "src"))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}

func buildGo(t *testing.T) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "kamino-go")
	command := exec.Command("go", "build", "-trimpath", "-o", output, "./cmd/kamino")
	command.Dir = repoRoot(t)
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, data)
	}
	return output
}

func seedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]\n\tname = Cross\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "git", "ignore"), []byte("ignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestCrossPythonCloneGoDeploy(t *testing.T) {
	python := pythonPath(t)
	binary := buildGo(t)
	source := seedHome(t)
	cloneFile := filepath.Join(t.TempDir(), "python-clone.tar.gz")
	runCommand(t, repoRoot(t), source, python, "-m", "kamino", "clone", "-a", "git", "-o", cloneFile)
	destination := t.TempDir()
	runCommand(t, repoRoot(t), destination, binary, "deploy", cloneFile, "--yes")
	if data, err := os.ReadFile(filepath.Join(destination, ".gitconfig")); err != nil || !strings.Contains(string(data), "Cross") {
		t.Fatalf("archivo no desplegado: %v", err)
	}
}

func TestCrossGoClonePythonDeploy(t *testing.T) {
	python := pythonPath(t)
	binary := buildGo(t)
	source := seedHome(t)
	cloneFile := filepath.Join(t.TempDir(), "go-clone.tar.gz")
	runCommand(t, repoRoot(t), source, binary, "clone", "-a", "git", "-o", cloneFile)
	destination := t.TempDir()
	runCommand(t, repoRoot(t), destination, python, "-m", "kamino", "deploy", cloneFile, "--yes")
	if data, err := os.ReadFile(filepath.Join(destination, ".gitconfig")); err != nil || !strings.Contains(string(data), "Cross") {
		t.Fatalf("archivo no desplegado por Python: %v", err)
	}
}

func TestCrossSnapshotDeploy(t *testing.T) {
	python := pythonPath(t)
	binary := buildGo(t)
	source := seedHome(t)
	cloneFile := filepath.Join(t.TempDir(), "go-clone.tar.gz")
	runCommand(t, repoRoot(t), source, binary, "clone", "-a", "git", "-o", cloneFile)
	destination := t.TempDir()
	configPath := filepath.Join(destination, ".gitconfig")
	if err := os.WriteFile(configPath, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCommand(t, repoRoot(t), destination, binary, "deploy", cloneFile, "--yes")
	snapshots, err := filepath.Glob(filepath.Join(destination, ".config", "kamino", "vault", "snapshot", "*.tar.gz"))
	if err != nil || len(snapshots) == 0 {
		t.Fatalf("no se creó el holocrón: %v", err)
	}
	runCommand(t, repoRoot(t), destination, python, "-m", "kamino", "deploy", snapshots[0], "--yes")
	if data, _ := os.ReadFile(configPath); string(data) != "original\n" {
		t.Fatalf("el holocrón no restauró el original: %q", data)
	}
}
