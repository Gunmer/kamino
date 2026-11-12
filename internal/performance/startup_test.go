package performance

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"time"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "kamino")
	command := exec.Command("go", "build", "-trimpath", "-o", output, "../../cmd/kamino")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		command.Env = append(command.Environ(), "CGO_ENABLED=0")
	}
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, data)
	}
	return output
}

func median(values []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

func TestStartupBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary startup test in short mode")
	}
	binary := buildBinary(t)
	budget := 30 * time.Millisecond
	if value := getenvInt("KAMINO_PERF_BUDGET_MS"); value > 0 {
		budget = time.Duration(value) * time.Millisecond
	}
	for _, argument := range []string{"--version", "--help"} {
		times := make([]time.Duration, 0, 7)
		for run := 0; run < 7; run++ {
			start := time.Now()
			command := exec.Command(binary, argument)
			if err := command.Run(); err != nil {
				t.Fatalf("%s: %v", argument, err)
			}
			elapsed := time.Since(start)
			if run > 0 {
				times = append(times, elapsed)
			}
		}
		value := median(times)
		t.Logf("%s median: %s", argument, value)
		if value > budget {
			t.Fatalf("%s median %s exceeds budget %s", argument, value, budget)
		}
	}
}

func getenvInt(name string) int {
	value, _ := strconv.Atoi(os.Getenv(name))
	return value
}
