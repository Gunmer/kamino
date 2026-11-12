package detect

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gunmer/kamino/internal/catalog"
)

const binaryPlistHex = "62706c6973743030d30102030405065f1012434642756e646c654964656e7469666965725f101a434642756e646c6553686f727456657273696f6e537472696e675f100f434642756e646c6556657273696f6e5f1010636f6d2e6578616d706c652e7465737455312e322e33523432080f244153666c000000000000010100000000000000070000000000000000000000000000006f"

func ptr(value string) *string { return &value }

func writeBundle(t *testing.T, directory, name, identifier string, binary bool) {
	t.Helper()
	bundle := filepath.Join(directory, name+".app", "Contents")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if binary {
		data, err := hex.DecodeString(binaryPlistHex)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bundle, "Info.plist"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + identifier + `</string>
<key>CFBundleShortVersionString</key><string>9.8.7</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(bundle, "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBundleDetectionXMLAndBinary(t *testing.T) {
	apps := t.TempDir()
	writeBundle(t, apps, "Example", "com.example.test", false)
	app := &catalog.AppDefinition{ID: "example", Name: "Example", BundleID: ptr("com.example.test")}
	if !IsInstalled(app, Options{Home: t.TempDir(), ApplicationDirs: []string{apps}}) {
		t.Fatal("expected XML bundle detection")
	}
	if version := DetectVersion(app, Options{Home: t.TempDir(), ApplicationDirs: []string{apps}}); version != "9.8.7" {
		t.Fatalf("unexpected version: %q", version)
	}

	binaryApps := t.TempDir()
	writeBundle(t, binaryApps, "Binary", "", true)
	binaryApp := &catalog.AppDefinition{ID: "binary", Name: "Binary", BundleID: ptr("com.example.test")}
	if !IsInstalled(binaryApp, Options{Home: t.TempDir(), ApplicationDirs: []string{binaryApps}}) {
		t.Fatal("expected binary plist detection")
	}
	if version := DetectVersion(binaryApp, Options{Home: t.TempDir(), ApplicationDirs: []string{binaryApps}}); version != "1.2.3" {
		t.Fatalf("unexpected binary version: %q", version)
	}
}

func TestCommandAndMarkerDetection(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := &catalog.AppDefinition{
		ID: "tool", Name: "Tool", Command: ptr("tool"),
		InstallMarker: ptr(".tool/installed"), OptionalPaths: []string{".tool"},
	}
	options := Options{Home: home, Path: bin}
	if IsInstalled(app, options) {
		t.Fatal("marker is missing")
	}
	if err := os.MkdirAll(filepath.Join(home, ".tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".tool", "installed"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsInstalled(app, options) {
		t.Fatal("expected marker and command detection")
	}
	if err := os.Chmod(filepath.Join(bin, "tool"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsInstalled(app, options) {
		t.Fatal("non-executable file must not count as a command")
	}
}

func TestSearchPathsAndAliases(t *testing.T) {
	home := t.TempDir()
	extra := t.TempDir()
	if err := os.WriteFile(filepath.Join(extra, "alias"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := &catalog.AppDefinition{
		ID: "alias", Name: "Alias", Command: ptr("missing"), CommandAliases: []string{"alias"},
		SearchPaths: []string{extra}, OptionalPaths: []string{".alias"},
	}
	if !IsInstalled(app, Options{Home: home, Path: t.TempDir()}) {
		t.Fatal("expected alias in search paths")
	}
}
