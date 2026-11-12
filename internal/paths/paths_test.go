package paths

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRelativeRejectsEscapes(t *testing.T) {
	bad := []string{"", "~", "~/x", "/abs", "../x", "a/../b"}
	for _, value := range bad {
		if err := EnsureRelative(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
	for _, value := range []string{"a", "a/b", "./a", "a//b"} {
		if err := EnsureRelative(value); err != nil {
			t.Fatalf("unexpected error for %q: %v", value, err)
		}
	}
}

func TestToAbsoluteAndRelative(t *testing.T) {
	home := t.TempDir()
	absolute, err := ToAbsolute("a/b", home)
	if err != nil {
		t.Fatal(err)
	}
	if absolute != filepath.Join(home, "a", "b") {
		t.Fatalf("unexpected absolute path: %s", absolute)
	}
	relative, err := ToRelative(absolute, home)
	if err != nil || relative != "a/b" {
		t.Fatalf("unexpected relative path: %q %v", relative, err)
	}
	if _, err := ToRelative(filepath.Join(home, "..", "elsewhere"), home); err == nil {
		t.Fatal("expected outside-home error")
	}
}

func TestSafeHomeRejectsIntermediateSymlink(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "sub", "file"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "file"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, "linkdir")); err != nil {
		t.Fatal(err)
	}
	safe, err := NewSafeHome(home, false)
	if err != nil {
		t.Fatal(err)
	}
	defer safe.Close()
	if _, err := safe.ReadBytes("sub/file"); err != nil {
		t.Fatalf("expected safe read: %v", err)
	}
	if _, err := safe.ReadBytes("linkdir/file"); err == nil {
		t.Fatal("expected intermediate symlink rejection")
	}
}

func TestSafeHomeFinalSymlink(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "target"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(home, "link")); err != nil {
		t.Fatal(err)
	}
	safe, _ := NewSafeHome(home, false)
	defer safe.Close()
	info, err := safe.Lstat("link")
	if err != nil || info.Kind != KindLink {
		t.Fatalf("expected final symlink: %v %v", info.Kind, err)
	}
	target, err := safe.Readlink("link")
	if err != nil || target != "target" {
		t.Fatalf("unexpected link target: %q %v", target, err)
	}
	if _, err := safe.ReadBytes("link"); err == nil {
		t.Fatal("expected final symlink read rejection")
	}
}

func TestSafeHomeWritesAndWalk(t *testing.T) {
	home := t.TempDir()
	safe, _ := NewSafeHome(home, false)
	defer safe.Close()
	if err := safe.WriteFile("a/b/file", []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := safe.ReadBytes("a/b/file")
	if err != nil || string(data) != "hello" {
		t.Fatalf("unexpected read: %q %v", data, err)
	}
	if err := safe.ReplaceWithSymlink("a/link", "b/file"); err != nil {
		t.Fatal(err)
	}
	if target, err := safe.Readlink("a/link"); err != nil || target != "b/file" {
		t.Fatalf("unexpected link: %q %v", target, err)
	}
	var items []WalkItem
	if err := safe.Walk("a", nil, func(item WalkItem) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two items, got %d: %#v", len(items), items)
	}
	if items[0].Path != "a/link" || items[1].Path != "a/b/file" {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func TestSafeHomeWriteRejectsSymlinkParent(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, "linkdir")); err != nil {
		t.Fatal(err)
	}
	safe, _ := NewSafeHome(home, false)
	defer safe.Close()
	err := safe.WriteFile("linkdir/file", []byte("x"), 0o644)
	if err == nil {
		t.Fatal("expected write rejection through symlink")
	}
}

var _ = errors.Is
