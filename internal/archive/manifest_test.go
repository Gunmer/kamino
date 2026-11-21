package archive

import "testing"

func TestManifestValidateRejectsUnsafePaths(t *testing.T) {
	hash := sha([]byte("x"))
	link := "../../x"
	cases := []struct {
		name string
		file FileEntry
	}{
		{"empty", FileEntry{Path: "", SHA256: &hash}},
		{"absolute", FileEntry{Path: "/absolute", SHA256: &hash}},
		{"parent", FileEntry{Path: "../escape", SHA256: &hash}},
		{"home", FileEntry{Path: "~/config", SHA256: &hash}},
		{"link", FileEntry{Path: "a", Link: &link}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manifest := &Manifest{
				FormatVersion: FormatVersion,
				Apps:          []AppEntry{{ID: "git", Name: "Git", Files: []FileEntry{testCase.file}}},
			}
			if err := manifest.Validate(); err == nil {
				t.Fatalf("Validate accepted unsafe file %#v", testCase.file)
			}
		})
	}
}

func TestManifestValidateAcceptsRelativePath(t *testing.T) {
	hash := sha([]byte("x"))
	manifest := &Manifest{
		FormatVersion: FormatVersion,
		Apps:          []AppEntry{{ID: "git", Name: "Git", Files: []FileEntry{{Path: "a/b", SHA256: &hash}}}},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate rejected a valid path: %v", err)
	}
}

func TestManifestValidateRejectsInvalidMode(t *testing.T) {
	hash := sha([]byte("x"))
	for _, mode := range []string{"rw-r--r--", "0999", "1777", "-1", "0644 "} {
		manifest := &Manifest{
			FormatVersion: FormatVersion,
			Apps:          []AppEntry{{ID: "git", Name: "Git", Files: []FileEntry{{Path: "a", SHA256: &hash, Mode: mode}}}},
		}
		if err := manifest.Validate(); err == nil {
			t.Fatalf("Validate accepted mode %q", mode)
		}
	}
	for _, mode := range []string{"", "0644", "644", "0600", "0755"} {
		manifest := &Manifest{
			FormatVersion: FormatVersion,
			Apps:          []AppEntry{{ID: "git", Name: "Git", Files: []FileEntry{{Path: "a", SHA256: &hash, Mode: mode}}}},
		}
		if err := manifest.Validate(); err != nil {
			t.Fatalf("Validate rejected mode %q: %v", mode, err)
		}
	}
}
