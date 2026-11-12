package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func sha(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func manifestFor(files map[string][]byte) *Manifest {
	entries := make([]FileEntry, 0)
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sortStrings(paths)
	for _, path := range paths {
		value := sha(files[path])
		entries = append(entries, FileEntry{Path: path, SHA256: &value, Size: int64(len(files[path])), Mode: "0644"})
	}
	return &Manifest{
		FormatVersion: FormatVersion, ToolVersion: "0.8.0", CreatedAt: "2026-01-01T00:00:00Z",
		MacOSVersion: "15.0", Arch: "arm64", Hostname: "host",
		Apps: []AppEntry{{ID: "git", Name: "Git", Files: entries}},
	}
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func writeSource(t *testing.T, home string, files map[string][]byte) {
	t.Helper()
	for path, data := range files {
		target := filepath.Join(home, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	home := t.TempDir()
	files := map[string][]byte{"a.txt": []byte("alpha"), "dir/b.txt": []byte("beta")}
	writeSource(t, home, files)
	manifest := manifestFor(files)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := WriteArchive(output, manifest, WriteOptions{SourceRoot: home}); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadManifest(output)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.FormatVersion != 1 || len(loaded.Apps) != 1 || len(loaded.Apps[0].Files) != 2 {
		t.Fatalf("unexpected manifest: %#v", loaded)
	}
	if err := VerifyArchive(output, loaded, nil, nil); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if err := ExtractArchive(output, destination, nil, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "apps", "git", "dir", "b.txt"))
	if err != nil || string(data) != "beta" {
		t.Fatalf("unexpected extraction: %q %v", data, err)
	}
}

func TestWriteRefusesOverwrite(t *testing.T) {
	home := t.TempDir()
	files := map[string][]byte{"a": []byte("a")}
	writeSource(t, home, files)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := WriteArchive(output, manifestFor(files), WriteOptions{SourceRoot: home}); err != nil {
		t.Fatal(err)
	}
	if err := WriteArchive(output, manifestFor(files), WriteOptions{SourceRoot: home}); err == nil {
		t.Fatal("expected overwrite refusal")
	}
}

func TestOverwriteRevalidatesDestination(t *testing.T) {
	home := t.TempDir()
	files := map[string][]byte{"a": []byte("a")}
	writeSource(t, home, files)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := WriteArchive(output, manifestFor(files), WriteOptions{SourceRoot: home}); err != nil {
		t.Fatal(err)
	}
	state := DestinationState(output)
	if state == nil {
		t.Fatal("expected state")
	}
	state.Ino++
	if err := WriteArchive(output, manifestFor(files), WriteOptions{SourceRoot: home, Overwrite: true, ExpectedState: state}); err == nil {
		t.Fatal("expected revalidation failure")
	}
}

func TestSymlinkRoundTrip(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "target"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(home, "link")); err != nil {
		t.Fatal(err)
	}
	link := "target"
	manifest := &Manifest{
		FormatVersion: 1, ToolVersion: "0.8.0", CreatedAt: "x", MacOSVersion: "", Arch: "arm64", Hostname: "h",
		Apps: []AppEntry{{ID: "git", Name: "Git", Files: []FileEntry{{Path: "link", Link: &link, Mode: "0777"}}}},
	}
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := WriteArchive(output, manifest, WriteOptions{SourceRoot: home}); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadManifest(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyArchive(output, loaded, nil, nil); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if err := ExtractArchive(output, destination, nil, nil); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(destination, "apps", "git", "link"))
	if err != nil || target != "target" {
		t.Fatalf("unexpected link: %q %v", target, err)
	}
}

func TestVerifyDetectsChecksumMismatch(t *testing.T) {
	home := t.TempDir()
	files := map[string][]byte{"a": []byte("a")}
	writeSource(t, home, files)
	manifest := manifestFor(files)
	output := filepath.Join(t.TempDir(), "clone.tar.gz")
	if err := WriteArchive(output, manifest, WriteOptions{SourceRoot: home}); err != nil {
		t.Fatal(err)
	}
	loaded, _ := ReadManifest(output)
	bad := "0" + (*loaded.Apps[0].Files[0].SHA256)[1:]
	loaded.Apps[0].Files[0].SHA256 = &bad
	if err := VerifyArchive(output, loaded, nil, nil); err == nil {
		t.Fatal("expected checksum failure")
	}
}

func rawArchive(t *testing.T, members []struct {
	name string
	data []byte
	link string
},
) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "raw.tar.gz")
	file, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	for _, member := range members {
		header := &tar.Header{Name: member.name, Mode: 0o644, Size: int64(len(member.data)), Typeflag: tar.TypeReg}
		if member.link != "" {
			header.Typeflag = tar.TypeSymlink
			header.Linkname = member.link
			header.Size = 0
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeReg {
			if _, err := tw.Write(member.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return output
}

func TestReadManifestRejectsDuplicateAndOversized(t *testing.T) {
	manifest := manifestFor(map[string][]byte{"a": []byte("a")})
	raw, _ := manifest.MarshalJSONBytes()
	duplicate := rawArchive(t, []struct {
		name string
		data []byte
		link string
	}{{ManifestName, raw, ""}, {ManifestName, raw, ""}})
	if _, err := ReadManifest(duplicate); err == nil {
		t.Fatal("expected duplicate manifest failure")
	}
	big := make([]byte, MaxManifestBytes+1)
	oversized := rawArchive(t, []struct {
		name string
		data []byte
		link string
	}{{ManifestName, big, ""}})
	if _, err := ReadManifest(oversized); err == nil {
		t.Fatal("expected oversized manifest failure")
	}
}

func TestExtractRejectsUnsafeMembers(t *testing.T) {
	cases := []struct {
		name string
		link string
	}{
		{"../evil", ""},
		{"/absolute", ""},
		{"link", "../../outside"},
	}
	for _, testCase := range cases {
		archivePath := rawArchive(t, []struct {
			name string
			data []byte
			link string
		}{{testCase.name, []byte("x"), testCase.link}})
		if err := ExtractArchive(archivePath, t.TempDir(), nil, nil); err == nil {
			t.Fatalf("expected rejection for %q", testCase.name)
		}
	}
}

func TestManifestJSONMatchesPythonShape(t *testing.T) {
	value := "a" + string(bytes.Repeat([]byte("0"), 63))
	manifest := &Manifest{
		FormatVersion: 1, ToolVersion: "0.8.0", CreatedAt: "2026-01-01T00:00:00Z",
		MacOSVersion: "", Arch: "arm64", Hostname: "h",
		Apps: []AppEntry{{ID: "git", Name: "Git", Files: []FileEntry{{Path: "a", SHA256: &value, Size: 1, Mode: "0644"}}}},
	}
	raw, err := manifest.MarshalJSONBytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`"format_version": 1`, `"tool_version": "0.8.0"`, `"bundle_id": null`,
		`"version": null`, `"sha256": "` + value + `"`, `"link": null`,
	} {
		if !bytes.Contains(raw, []byte(expected)) {
			t.Fatalf("missing %q in %s", expected, raw)
		}
	}
}
