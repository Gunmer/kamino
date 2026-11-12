package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/buildinfo"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func payload(tag, url, digest string, size int64) []byte {
	value := map[string]any{
		"tag_name": tag,
		"assets": []map[string]any{{
			"name": AssetName, "browser_download_url": url, "size": size, "digest": digest,
		}},
	}
	raw, _ := json.Marshal(value)
	return raw
}

func fakeResponse(status int, body []byte) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(string(body))), Header: http.Header{}}
}

func TestVersionParsing(t *testing.T) {
	if version, err := NormalizeVersion("v1.2.3"); err != nil || version != "1.2.3" {
		t.Fatalf("unexpected version: %q %v", version, err)
	}
	if _, err := ParseVersion("main"); err == nil {
		t.Fatal("expected invalid version")
	}
}

func TestLatestRelease(t *testing.T) {
	originalClient := HTTPClient
	defer func() { HTTPClient = originalClient }()
	digest := "sha256:" + strings.Repeat("a", 64)
	HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fakeResponse(200, payload("v0.9.0", "https://example.test/kamino-arm64", digest, 42)), nil
	})}
	release, err := LatestRelease()
	if err != nil {
		t.Fatal(err)
	}
	if release.Version != "0.9.0" || release.AssetSize != 42 || release.AssetDigest != digest {
		t.Fatalf("unexpected release: %#v", release)
	}
}

func TestLatestReleaseErrors(t *testing.T) {
	originalClient := HTTPClient
	defer func() { HTTPClient = originalClient }()
	cases := []struct {
		name   string
		status int
		body   []byte
	}{
		{"rate limit", 403, nil},
		{"invalid json", 200, []byte("nope")},
		{"missing asset", 200, []byte(`{"tag_name":"v1.0.0","assets":[]}`)},
		{"insecure url", 200, payload("v1.0.0", "http://example.test/x", "sha256:"+strings.Repeat("a", 64), 1)},
		{"missing digest", 200, payload("v1.0.0", "https://example.test/x", "", 1)},
		{"invalid tag", 200, payload("main", "https://example.test/x", "sha256:"+strings.Repeat("a", 64), 1)},
	}
	for _, testCase := range cases {
		HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return fakeResponse(testCase.status, testCase.body), nil
		})}
		if _, err := LatestRelease(); err == nil {
			t.Fatalf("%s: expected error", testCase.name)
		}
	}
}

func requireDarwinArm64(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("update is only supported on darwin/arm64")
	}
}

func TestQueryOutcomes(t *testing.T) {
	requireDarwinArm64(t)
	target := filepath.Join(t.TempDir(), "kamino")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	newer := &ReleaseInfo{Version: "0.9.0", AssetSize: 1, AssetDigest: "sha256:" + strings.Repeat("a", 64), AssetURL: "https://example.test/x"}
	check, err := QueryUpdate("0.8.0", target, newer)
	if err != nil || !check.ShouldUpdate || check.Outcome != OutcomeUpdated {
		t.Fatalf("unexpected update: %#v %v", check, err)
	}
	check, _ = QueryUpdate("0.9.0", target, newer)
	if check.ShouldUpdate || check.Outcome != OutcomeUpToDate {
		t.Fatalf("unexpected up-to-date: %#v", check)
	}
	check, _ = QueryUpdate("1.0.0", target, newer)
	if check.ShouldUpdate || check.Outcome != OutcomeNewer {
		t.Fatalf("unexpected newer: %#v", check)
	}
}

func TestDownloadAndRunUpdate(t *testing.T) {
	requireDarwinArm64(t)
	newBinary := []byte("#!/bin/sh\necho new\n")
	digest := sha256.Sum256(newBinary)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, _ = w.Write(newBinary)
	}))
	defer server.Close()
	release := ReleaseInfo{
		Version: "0.9.0", AssetURL: server.URL + "/kamino-arm64",
		AssetSize: int64(len(newBinary)), AssetDigest: "sha256:" + hex.EncodeToString(digest[:]),
	}
	target := filepath.Join(t.TempDir(), "kamino")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := RunUpdate("0.8.0", target, &release, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Updated || result.Outcome != OutcomeUpdated {
		t.Fatalf("unexpected result: %#v", result)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != string(newBinary) {
		t.Fatalf("unexpected target: %q %v", data, err)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatal("target must remain executable")
	}
}

func TestDownloadRejectsBadDigestAndCleansTemp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, _ = w.Write([]byte("new"))
	}))
	defer server.Close()
	directory := t.TempDir()
	target := filepath.Join(directory, "kamino")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := ReleaseInfo{Version: "0.9.0", AssetURL: server.URL, AssetSize: 3, AssetDigest: "sha256:" + strings.Repeat("0", 64)}
	if _, err := DownloadUpdate(release, target, nil, nil); err == nil {
		t.Fatal("expected digest failure")
	}
	entries, _ := os.ReadDir(directory)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".kamino-update-") {
			t.Fatalf("temporary file not removed: %s", entry.Name())
		}
	}
}

func TestDevelopmentBuildCannotUpdate(t *testing.T) {
	original := buildinfo.SelfContained
	buildinfo.SelfContained = "false"
	defer func() { buildinfo.SelfContained = original }()
	if _, err := InstalledBinaryPath(); err == nil {
		t.Fatal("development build must not update")
	}
}
