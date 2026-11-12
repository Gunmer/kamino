// Package paths normalizes home-relative paths and provides confined access.
package paths

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"

	kerrors "github.com/Gunmer/kamino/internal/errors"
)

const tempPrefix = ".kamino-"

// Kind classifies a filesystem entry discovered by Walk.
type Kind string

const (
	KindFile    Kind = "file"
	KindLink    Kind = "link"
	KindSpecial Kind = "special"
	KindSkipped Kind = "skipped"
)

// Info is a portable, minimal lstat result.
type Info struct {
	Kind  Kind
	Perm  os.FileMode
	Size  int64
	UID   uint32
	GID   uint32
	Mtime int64
	Ino   uint64
}

func (i Info) IsDir() bool     { return i.Kind == "dir" }
func (i Info) IsRegular() bool { return i.Kind == KindFile }
func (i Info) IsSymlink() bool { return i.Kind == KindLink }

// ModeOctal renders the permission bits as four octal digits.
func (i Info) ModeOctal() string { return fmt.Sprintf("%04o", i.Perm.Perm()) }

// WalkItem is one entry discovered by SafeHome.Walk.
type WalkItem struct {
	Path string
	Kind Kind
	Info *Info
}

// EnsureRelative rejects empty, absolute, home-prefixed or escaping paths.
func EnsureRelative(raw string) error {
	if raw == "" {
		return pathError("ruta vacía no admitida")
	}
	if strings.HasPrefix(raw, "~") {
		return pathError(fmt.Sprintf("ruta con '~' no admitida, usa una ruta relativa al home: %q", raw))
	}
	if strings.HasPrefix(raw, "/") {
		return pathError(fmt.Sprintf("ruta absoluta no admitida: %q", raw))
	}
	for _, part := range strings.Split(raw, "/") {
		if part == ".." {
			return pathError(fmt.Sprintf("ruta fuera del home no admitida: %q", raw))
		}
	}
	return nil
}

func pathError(msg string) error {
	return &kerrors.PathError{KaminoError: kerrors.KaminoError{Msg: msg}}
}

// Parts validates a home-relative path and returns its normalized components.
func Parts(rel string) ([]string, error) {
	if err := EnsureRelative(rel); err != nil {
		return nil, err
	}
	cleaned := path.Clean(rel)
	if cleaned == "." {
		return nil, nil
	}
	raw := strings.Split(cleaned, "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return parts, nil
}

// ToAbsolute resolves a home-relative path.
func ToAbsolute(rel string, home string) (string, error) {
	if err := EnsureRelative(rel); err != nil {
		return "", err
	}
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", pathError(fmt.Sprintf("no se pudo resolver el home: %v", err))
		}
	}
	return filepath.Join(home, filepath.FromSlash(rel)), nil
}

// ToRelative converts an absolute path inside home to its POSIX relative form.
func ToRelative(candidate string, home string) (string, error) {
	c := filepath.Clean(candidate)
	if !filepath.IsAbs(c) {
		return "", pathError(fmt.Sprintf("se requiere una ruta absoluta: %q", candidate))
	}
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", pathError(fmt.Sprintf("no se pudo resolver el home: %v", err))
		}
	}
	root := filepath.Clean(home)
	rel, err := filepath.Rel(root, c)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", pathError(fmt.Sprintf("ruta fuera del home %q: %q", root, candidate))
	}
	return filepath.ToSlash(rel), nil
}

// SafeHome confines every access to a home without following symlinks.
type SafeHome struct {
	Root string
	fd   int
}

// NewSafeHome opens the home descriptor.
func NewSafeHome(home string, create bool) (*SafeHome, error) {
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, pathError(fmt.Sprintf("no se pudo resolver el home: %v", err))
		}
	}
	if create {
		if err := os.MkdirAll(home, 0o755); err != nil {
			return nil, pathError(fmt.Sprintf("no se pudo crear el home %q: %v", home, err))
		}
	}
	fd, err := unix.Open(home, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, pathError(fmt.Sprintf("no se pudo abrir el home %q: %v", home, err))
	}
	return &SafeHome{Root: home, fd: fd}, nil
}

// Close releases the root descriptor.
func (s *SafeHome) Close() error {
	if s.fd >= 0 {
		err := unix.Close(s.fd)
		s.fd = -1
		return err
	}
	return nil
}

func (s *SafeHome) stepDir(parent int, part, rel string) (int, error) {
	child, err := unix.Openat(parent, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return -1, fs.ErrNotExist
		}
		return -1, pathError(fmt.Sprintf("no se pudo acceder a %q sin seguir enlaces: %v", rel, err))
	}
	_ = unix.Close(parent)
	return child, nil
}

func (s *SafeHome) parent(rel string) (int, string, error) {
	parts, err := Parts(rel)
	if err != nil {
		return -1, "", err
	}
	if len(parts) == 0 {
		return -1, "", pathError("se requiere una ruta dentro del home, no el home")
	}
	fd, err := unix.Dup(s.fd)
	if err != nil {
		return -1, "", pathError(fmt.Sprintf("no se pudo duplicar el descriptor del home: %v", err))
	}
	for _, part := range parts[:len(parts)-1] {
		child, err := s.stepDir(fd, part, rel)
		if err != nil {
			_ = unix.Close(fd)
			return -1, "", err
		}
		fd = child
	}
	return fd, parts[len(parts)-1], nil
}

func (s *SafeHome) openDir(rel string) (int, error) {
	parts, err := Parts(rel)
	if err != nil {
		return -1, err
	}
	fd, err := unix.Dup(s.fd)
	if err != nil {
		return -1, pathError(fmt.Sprintf("no se pudo duplicar el descriptor del home: %v", err))
	}
	for _, part := range parts {
		child, err := s.stepDir(fd, part, rel)
		if err != nil {
			_ = unix.Close(fd)
			return -1, err
		}
		fd = child
	}
	return fd, nil
}

// Lstat returns the final component without following symlinks.
func (s *SafeHome) Lstat(rel string) (Info, error) {
	parent, name, err := s.parent(rel)
	if err != nil {
		return Info{}, err
	}
	defer unix.Close(parent)
	var st unix.Stat_t
	if err := unix.Fstatat(parent, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Info{}, fs.ErrNotExist
		}
		return Info{}, pathError(fmt.Sprintf("no se pudo inspeccionar %q: %v", rel, err))
	}
	return infoFromStat(&st), nil
}

// ReadBytes reads a regular file without following intermediate or final links.
func (s *SafeHome) ReadBytes(rel string) ([]byte, error) {
	parent, name, err := s.parent(rel)
	if err != nil {
		return nil, err
	}
	defer unix.Close(parent)
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, pathError(fmt.Sprintf("no se pudo leer %q sin seguir enlaces: %v", rel, err))
	}
	file := os.NewFile(uintptr(fd), rel)
	defer file.Close()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return nil, pathError(fmt.Sprintf("no se pudo inspeccionar %q: %v", rel, err))
	}
	if infoFromStat(&st).Kind != KindFile {
		return nil, pathError(fmt.Sprintf("no es un archivo regular: %q", rel))
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, pathError(fmt.Sprintf("no se pudo leer %q: %v", rel, err))
	}
	return data, nil
}

// ReadText reads a file as UTF-8 with replacement, optionally limited by size.
func (s *SafeHome) ReadText(rel string, limit int64) (string, error) {
	if limit > 0 {
		info, err := s.Lstat(rel)
		if err != nil {
			return "", err
		}
		if info.Size > limit {
			return "", pathError(fmt.Sprintf("archivo demasiado grande para analizar: %q", rel))
		}
	}
	data, err := s.ReadBytes(rel)
	if err != nil {
		return "", err
	}
	return strings.ToValidUTF8(string(data), "\uFFFD"), nil
}

// Readlink returns the final symlink target without following it.
func (s *SafeHome) Readlink(rel string) (string, error) {
	parent, name, err := s.parent(rel)
	if err != nil {
		return "", err
	}
	defer unix.Close(parent)
	buf := make([]byte, 256)
	for {
		n, err := unix.Readlinkat(parent, name, buf)
		if err != nil {
			return "", pathError(fmt.Sprintf("no es un enlace simbólico: %q: %v", rel, err))
		}
		if n < len(buf) {
			return string(buf[:n]), nil
		}
		buf = make([]byte, len(buf)*2)
	}
}

// Listdir returns the sorted names of a home directory.
func (s *SafeHome) Listdir(rel string) ([]string, error) {
	fd, err := s.openDir(rel)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), rel)
	defer file.Close()
	entries, err := file.ReadDir(-1)
	if err != nil {
		return nil, pathError(fmt.Sprintf("no se pudo listar %q: %v", rel, err))
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// Walk traverses a directory without following symlinks and in deterministic order.
func (s *SafeHome) Walk(rel string, prune func(string) bool, visit func(WalkItem) error) error {
	if prune == nil {
		prune = func(string) bool { return false }
	}
	return s.walkDir(rel, prune, visit)
}

func (s *SafeHome) walkDir(rel string, prune func(string) bool, visit func(WalkItem) error) error {
	fd, err := s.openDir(rel)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return visit(WalkItem{Path: rel, Kind: KindSkipped})
		}
		if _, ok := err.(*kerrors.PathError); ok {
			return visit(WalkItem{Path: rel, Kind: KindSkipped})
		}
		return err
	}
	file := os.NewFile(uintptr(fd), rel)
	defer file.Close()
	entries, readErr := file.ReadDir(-1)
	if readErr != nil {
		return visit(WalkItem{Path: rel, Kind: KindSkipped})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	directories := make([]string, 0)
	for _, entry := range entries {
		childRel := path.Join(rel, entry.Name())
		if prune(childRel) {
			continue
		}
		var st unix.Stat_t
		if err := unix.Fstatat(fd, entry.Name(), &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if err := visit(WalkItem{Path: childRel, Kind: KindSkipped}); err != nil {
				return err
			}
			continue
		}
		info := infoFromStat(&st)
		switch info.Kind {
		case KindLink:
			if err := visit(WalkItem{Path: childRel, Kind: KindLink, Info: &info}); err != nil {
				return err
			}
		case KindFile:
			if err := visit(WalkItem{Path: childRel, Kind: KindFile, Info: &info}); err != nil {
				return err
			}
		case "dir":
			directories = append(directories, entry.Name())
		default:
			if err := visit(WalkItem{Path: childRel, Kind: KindSpecial, Info: &info}); err != nil {
				return err
			}
		}
	}
	for _, name := range directories {
		if err := s.walkDir(path.Join(rel, name), prune, visit); err != nil {
			return err
		}
	}
	return nil
}

// ensureParent creates missing parent directories and returns (fd, name).
func (s *SafeHome) ensureParent(rel string) (int, string, error) {
	parts, err := Parts(rel)
	if err != nil {
		return -1, "", err
	}
	if len(parts) == 0 {
		return -1, "", pathError("se requiere una ruta dentro del home, no el home")
	}
	fd, err := unix.Dup(s.fd)
	if err != nil {
		return -1, "", pathError(fmt.Sprintf("no se pudo duplicar el descriptor del home: %v", err))
	}
	for _, part := range parts[:len(parts)-1] {
		child, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
		if errors.Is(openErr, fs.ErrNotExist) {
			if mkErr := unix.Mkdirat(fd, part, 0o755); mkErr != nil && !errors.Is(mkErr, fs.ErrExist) {
				_ = unix.Close(fd)
				return -1, "", pathError(fmt.Sprintf("no se pudo crear el directorio de %q: %v", rel, mkErr))
			}
			child, openErr = unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
		}
		if openErr != nil {
			_ = unix.Close(fd)
			return -1, "", pathError(fmt.Sprintf("no se pudo acceder a %q sin seguir enlaces: %v", rel, openErr))
		}
		_ = unix.Close(fd)
		fd = child
	}
	return fd, parts[len(parts)-1], nil
}

func randomName() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return tempPrefix + hex.EncodeToString(buf) + ".tmp"
}

// WriteFile writes a regular file with atomic replacement.
func (s *SafeHome) WriteFile(rel string, data []byte, mode os.FileMode) error {
	parent, name, err := s.ensureParent(rel)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	temp := randomName()
	fd, err := unix.Openat(parent, temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return pathError(fmt.Sprintf("no se pudo crear el temporal para %q: %v", rel, err))
	}
	cleanup := func() {
		_ = unix.Unlinkat(parent, temp, 0)
	}
	file := os.NewFile(uintptr(fd), temp)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		cleanup()
		return pathError(fmt.Sprintf("no se pudo escribir %q: %v", rel, err))
	}
	if err := unix.Fchmod(fd, uint32(mode.Perm())); err != nil {
		_ = file.Close()
		cleanup()
		return pathError(fmt.Sprintf("no se pudo ajustar el modo de %q: %v", rel, err))
	}
	if err := unix.Fsync(fd); err != nil {
		_ = file.Close()
		cleanup()
		return pathError(fmt.Sprintf("no se pudo sincronizar %q: %v", rel, err))
	}
	if err := file.Close(); err != nil {
		cleanup()
		return pathError(fmt.Sprintf("no se pudo cerrar %q: %v", rel, err))
	}
	if err := unix.Renameat(parent, temp, parent, name); err != nil {
		cleanup()
		return pathError(fmt.Sprintf("no se pudo reemplazar %q: %v", rel, err))
	}
	return nil
}

// ReplaceWithSymlink atomically creates a symlink at rel.
func (s *SafeHome) ReplaceWithSymlink(rel, target string) error {
	parent, name, err := s.ensureParent(rel)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	temp := randomName()
	if err := unix.Symlinkat(target, parent, temp); err != nil {
		return pathError(fmt.Sprintf("no se pudo crear el enlace %q: %v", rel, err))
	}
	if err := unix.Renameat(parent, temp, parent, name); err != nil {
		_ = unix.Unlinkat(parent, temp, 0)
		return pathError(fmt.Sprintf("no se pudo reemplazar %q: %v", rel, err))
	}
	return nil
}

// EnsureParent exposes the safe parent creation used by callers that need an fd.
func (s *SafeHome) EnsureParent(rel string) (int, string, error) { return s.ensureParent(rel) }

// JoinPosix joins POSIX-style relative components.
func JoinPosix(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return path.Join(filtered...)
}

// DirnamePosix returns the POSIX parent of a relative path.
func DirnamePosix(rel string) string { return path.Dir(rel) }
