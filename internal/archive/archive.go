package archive

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
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
	"syscall"
	"time"

	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/paths"
	"github.com/Gunmer/kamino/internal/progress"
)

// DestState is a cheap destination fingerprint for forced replacement.
type DestState struct {
	Ino    uint64
	Size   int64
	Mtime  int64
	Exists bool
}

// WriteOptions controls WriteArchive.
type WriteOptions struct {
	SourceRoot    string
	Overwrite     bool
	ExpectedState *DestState
	Progress      progress.Callback
	Cancelled     progress.Cancel
}

// DestinationState returns the current destination fingerprint, if any.
func DestinationState(output string) *DestState {
	info, err := os.Lstat(output)
	if err != nil {
		return nil
	}
	state := &DestState{Size: info.Size(), Mtime: info.ModTime().UnixNano(), Exists: true}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		state.Ino = st.Ino
	}
	return state
}

// WriteArchive writes manifest.json and every manifest file to a gzip tar.
func WriteArchive(output string, manifest *Manifest, options WriteOptions) error {
	if err := manifest.Validate(); err != nil {
		return &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{Msg: err.Error()}}
	}
	root := options.SourceRoot
	if root == "" {
		var err error
		root, err = os.UserHomeDir()
		if err != nil {
			return archiveError("no se pudo resolver el home: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return archiveError("no se pudo crear el directorio del clon: %v", err)
	}
	initial := DestinationState(output)
	if options.ExpectedState != nil && !statesEqual(initial, options.ExpectedState) {
		return archiveError("el archivo del clon cambió antes de escribir: %s", output)
	}

	type entryRef struct {
		appID string
		entry FileEntry
	}
	entries := make([]entryRef, 0)
	for _, app := range manifest.Apps {
		for _, entry := range app.Files {
			entries = append(entries, entryRef{appID: app.ID, entry: entry})
		}
	}

	temp := filepath.Join(filepath.Dir(output), fmt.Sprintf(".%s.%s.tmp", filepath.Base(output), randomSuffix()))
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(temp)
		}
	}()

	safe, err := paths.NewSafeHome(root, false)
	if err != nil {
		return err
	}
	defer safe.Close()

	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return archiveError("no se pudo crear el temporal del clon: %v", err)
	}
	gz := gzip.NewWriter(file)
	gz.Header.ModTime = time.Time{}
	tw := tar.NewWriter(gz)
	writeErr := func() error {
		if err := addManifest(tw, manifest); err != nil {
			return err
		}
		for index, ref := range entries {
			if err := progress.EnsureNotCancelled(options.Cancelled); err != nil {
				return err
			}
			if err := addFile(tw, safe, ref.appID, ref.entry); err != nil {
				return err
			}
			progress.Emit(options.Progress, progress.Writing, index+1, len(entries), ref.entry.Path)
		}
		return nil
	}()
	if writeErr != nil {
		_ = tw.Close()
		_ = gz.Close()
		_ = file.Close()
		return writeErr
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		_ = file.Close()
		return archiveError("no se pudo cerrar el archivo del clon: %v", err)
	}
	if err := gz.Close(); err != nil {
		_ = file.Close()
		return archiveError("no se pudo cerrar el archivo del clon: %v", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return archiveError("no se pudo sincronizar el archivo del clon: %v", err)
	}
	if err := file.Close(); err != nil {
		return archiveError("no se pudo cerrar el archivo del clon: %v", err)
	}
	if err := progress.EnsureNotCancelled(options.Cancelled); err != nil {
		return err
	}
	if err := commit(temp, output, options.Overwrite, initial); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func commit(temp, output string, overwrite bool, initial *DestState) error {
	if overwrite {
		if !statesEqual(DestinationState(output), initial) {
			return archiveError("el archivo del clon cambió antes del renombrado: %s", output)
		}
		if err := os.Rename(temp, output); err != nil {
			return archiveError("no se pudo reemplazar el clon: %v", err)
		}
		return nil
	}
	if err := os.Link(temp, output); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return archiveError("el archivo del clon ya existe: %s", output)
		}
		return archiveError("no se pudo crear el clon: %v", err)
	}
	_ = os.Remove(temp)
	return nil
}

func statesEqual(left, right *DestState) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Exists == right.Exists && left.Ino == right.Ino && left.Size == right.Size && left.Mtime == right.Mtime
}

func addManifest(tw *tar.Writer, manifest *Manifest) error {
	data, err := manifest.MarshalJSONBytes()
	if err != nil {
		return archiveError("no se pudo serializar el manifiesto: %v", err)
	}
	header := &tar.Header{
		Name:     ManifestName,
		Mode:     0o644,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg,
		Format:   tar.FormatPAX,
	}
	if err := tw.WriteHeader(header); err != nil {
		return archiveError("no se pudo escribir el manifiesto: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		return archiveError("no se pudo escribir el manifiesto: %v", err)
	}
	return nil
}

func addFile(tw *tar.Writer, safe *paths.SafeHome, appID string, entry FileEntry) error {
	info, err := safe.Lstat(entry.Path)
	if err != nil {
		return archiveError("no se pudo leer %s: %v", entry.Path, err)
	}
	arcname := path.Join(AppsDir, appID, entry.Path)
	switch info.Kind {
	case paths.KindFile:
		if entry.Link != nil {
			return archiveError("%s es un archivo regular, no un enlace", entry.Path)
		}
		content, err := safe.ReadBytes(entry.Path)
		if err != nil {
			return archiveError("no se pudo leer %s: %v", entry.Path, err)
		}
		digest := sha256.Sum256(content)
		if entry.SHA256 == nil || hex.EncodeToString(digest[:]) != *entry.SHA256 {
			return archiveError("el archivo cambió durante la clonación: %s", entry.Path)
		}
		header := &tar.Header{
			Name:     arcname,
			Mode:     int64(info.Perm.Perm()),
			Uid:      int(info.UID),
			Gid:      int(info.GID),
			ModTime:  time.Unix(info.Mtime, 0),
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatPAX,
		}
		if err := tw.WriteHeader(header); err != nil {
			return archiveError("no se pudo escribir %s: %v", entry.Path, err)
		}
		if _, err := tw.Write(content); err != nil {
			return archiveError("no se pudo escribir %s: %v", entry.Path, err)
		}
	case paths.KindLink:
		if entry.Link == nil {
			return archiveError("%s es un enlace, no un archivo regular", entry.Path)
		}
		target, err := safe.Readlink(entry.Path)
		if err != nil {
			return archiveError("no se pudo leer el enlace %s: %v", entry.Path, err)
		}
		if target != *entry.Link {
			return archiveError("el destino del enlace cambió para %s: %q != %q", entry.Path, target, *entry.Link)
		}
		header := &tar.Header{
			Name:     arcname,
			Mode:     int64(info.Perm.Perm()),
			Uid:      int(info.UID),
			Gid:      int(info.GID),
			ModTime:  time.Unix(info.Mtime, 0),
			Typeflag: tar.TypeSymlink,
			Linkname: target,
			Format:   tar.FormatPAX,
		}
		if err := tw.WriteHeader(header); err != nil {
			return archiveError("no se pudo escribir %s: %v", entry.Path, err)
		}
	default:
		return archiveError("tipo de archivo no soportado para clonar: %s", entry.Path)
	}
	return nil
}

// ReadManifest reads and validates the unique manifest member.
func ReadManifest(file string) (*Manifest, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, archiveError("no se pudo leer el archivo del clon: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, archiveError("no se pudo leer el archivo del clon: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var raw []byte
	found := false
	index := 0
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, archiveError("no se pudo leer el archivo del clon: %v", err)
		}
		index++
		if index > MaxArchiveMembers {
			return nil, archiveError("el archivo del clon supera el tope de %d miembros", MaxArchiveMembers)
		}
		if header.Name != ManifestName {
			continue
		}
		if found {
			return nil, archiveError("hay más de un %s en el archivo del clon", ManifestName)
		}
		found = true
		if header.Typeflag != tar.TypeReg {
			return nil, archiveError("%s no es un archivo regular", ManifestName)
		}
		if header.Size > MaxManifestBytes {
			return nil, archiveError("%s supera el tope de %d bytes", ManifestName, MaxManifestBytes)
		}
		raw = make([]byte, header.Size)
		if _, err := io.ReadFull(tr, raw); err != nil {
			return nil, archiveError("no se pudo leer %s: %v", ManifestName, err)
		}
	}
	if !found {
		return nil, archiveError("falta %s en el archivo del clon", ManifestName)
	}
	if len(raw) > MaxManifestBytes {
		return nil, archiveError("%s supera el tope de %d bytes", ManifestName, MaxManifestBytes)
	}
	return ParseManifest(raw)
}

// ReadFileFromArchive reads a bounded regular member without extracting it.
func ReadFileFromArchive(file, appID, rel string, limit int64) []byte {
	name := path.Join(AppsDir, appID, rel)
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err != nil {
			return nil
		}
		if header.Name != name {
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Size > limit {
			return nil
		}
		raw := make([]byte, header.Size)
		if _, err := io.ReadFull(tr, raw); err != nil {
			return nil
		}
		return raw
	}
}

func expectedMembers(manifest *Manifest) map[string]FileEntry {
	expected := make(map[string]FileEntry)
	for _, app := range manifest.Apps {
		for _, file := range app.Files {
			expected[path.Join(AppsDir, app.ID, file.Path)] = file
		}
	}
	return expected
}

// VerifyArchive checks every manifest member and rejects extras.
func VerifyArchive(file string, manifest *Manifest, callback progress.Callback, cancelled progress.Cancel) error {
	expected := expectedMembers(manifest)
	total := len(expected)
	current := 0
	f, err := os.Open(file)
	if err != nil {
		return archiveError("no se pudo verificar el archivo del clon: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return archiveError("no se pudo verificar el archivo del clon: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	index := 0
	for {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			return err
		}
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return archiveError("no se pudo verificar el archivo del clon: %v", err)
		}
		index++
		if index > MaxArchiveMembers+1 {
			return archiveError("el archivo del clon supera el tope de %d miembros", MaxArchiveMembers)
		}
		if header.Name == ManifestName {
			continue
		}
		entry, ok := expected[header.Name]
		if !ok {
			return integrityError("archivo inesperado en el clon: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeReg:
			if entry.Link != nil {
				return integrityError("esperaba un enlace en %s, no un archivo", header.Name)
			}
			digest, err := hashTarMember(tr)
			if err != nil {
				return archiveError("no se pudo leer %s", header.Name)
			}
			if entry.SHA256 == nil || digest != *entry.SHA256 {
				return integrityError("checksum no coincide para %s", header.Name)
			}
		case tar.TypeSymlink:
			if entry.Link == nil {
				return integrityError("esperaba un archivo en %s, no un enlace", header.Name)
			}
			if header.Linkname != *entry.Link {
				return integrityError("destino de enlace no coincide para %s", header.Name)
			}
		default:
			return integrityError("tipo de miembro no soportado: %s", header.Name)
		}
		delete(expected, header.Name)
		current++
		progress.Emit(callback, progress.Verifying, current, total, header.Name)
	}
	if len(expected) > 0 {
		missing := make([]string, 0, len(expected))
		for name := range expected {
			missing = append(missing, name)
		}
		sort.Strings(missing)
		return integrityError("faltan archivos en el clon: %v", missing)
	}
	return nil
}

func hashTarMember(reader io.Reader) (string, error) {
	digest := sha256.New()
	if _, err := io.Copy(digest, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// ExtractArchive extracts safely into a staging directory.
func ExtractArchive(file, dest string, callback progress.Callback, cancelled progress.Cancel) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return archiveError("extracción insegura o fallida: %v", err)
	}
	members, err := scanMembers(file)
	if err != nil {
		return err
	}
	if err := validateMembers(members); err != nil {
		return err
	}

	// Directories first so later members cannot traverse a symlink.
	for _, member := range members {
		if member.Typeflag != tar.TypeDir {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(member.Name))
		if err := os.MkdirAll(target, os.FileMode(member.Mode).Perm()); err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
	}
	total := len(members)
	// Regular files last, reading content from a fresh archive pass.
	f, err := os.Open(file)
	if err != nil {
		return archiveError("extracción insegura o fallida: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return archiveError("extracción insegura o fallida: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	index := 0
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
		index++
		if header.Typeflag != tar.TypeReg {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(header.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode).Perm())
		if err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return archiveError("extracción insegura o fallida: %v", err)
		}
		if err := out.Close(); err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
		progress.Emit(callback, progress.Extracting, index, total, header.Name)
	}
	for _, member := range members {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			return err
		}
		if member.Typeflag != tar.TypeSymlink {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(member.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
		_ = os.Remove(target)
		if err := os.Symlink(member.Linkname, target); err != nil {
			return archiveError("extracción insegura o fallida: %v", err)
		}
		index++
		progress.Emit(callback, progress.Extracting, index, total, member.Name)
	}
	return nil
}

type memberInfo struct {
	Name     string
	Typeflag byte
	Linkname string
	Mode     int64
	Size     int64
}

func scanMembers(file string) ([]memberInfo, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, archiveError("extracción insegura o fallida: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, archiveError("extracción insegura o fallida: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	members := make([]memberInfo, 0)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, archiveError("extracción insegura o fallida: %v", err)
		}
		if len(members) >= MaxArchiveMembers {
			return nil, archiveError("el archivo del clon supera el tope de %d miembros", MaxArchiveMembers)
		}
		members = append(members, memberInfo{
			Name:     header.Name,
			Typeflag: header.Typeflag,
			Linkname: header.Linkname,
			Mode:     header.Mode,
			Size:     header.Size,
		})
	}
	return members, nil
}

func validateMembers(members []memberInfo) error {
	symlinks := make(map[string]bool)
	for _, member := range members {
		if member.Typeflag == tar.TypeSymlink {
			symlinks[path.Clean(member.Name)] = true
		}
	}
	for _, member := range members {
		parent := path.Dir(path.Clean(member.Name))
		for parent != "." && parent != "/" {
			if symlinks[parent] {
				return archiveError("miembro dentro de un enlace no permitido: %q", member.Name)
			}
			parent = path.Dir(parent)
		}
		name := path.Clean(strings.ReplaceAll(member.Name, "\\", "/"))
		if strings.HasPrefix(member.Name, "/") || path.IsAbs(name) {
			return archiveError("ruta absoluta no permitida en el archivo: %q", member.Name)
		}
		for _, part := range strings.Split(member.Name, "/") {
			if part == ".." {
				return archiveError("ruta con '..' no permitida en el archivo: %q", member.Name)
			}
		}
		switch member.Typeflag {
		case tar.TypeReg, tar.TypeDir:
		case tar.TypeSymlink:
			if path.IsAbs(member.Linkname) {
				return archiveError("enlace absoluto no permitido en el archivo: %q", member.Name)
			}
			resolved := path.Clean(path.Join(path.Dir(member.Name), member.Linkname))
			if resolved == ".." || strings.HasPrefix(resolved, "../") {
				return archiveError("enlace fuera del destino en el archivo: %q", member.Name)
			}
		default:
			return archiveError("tipo de miembro no soportado en el archivo: %q", member.Name)
		}
	}
	return nil
}

func archiveError(format string, args ...any) error {
	return &kerrors.ArchiveError{KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf(format, args...)}}
}

func integrityError(format string, args ...any) error {
	return &kerrors.IntegrityError{KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf(format, args...)}}
}

func randomSuffix() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
