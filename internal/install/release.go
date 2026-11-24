package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Gunmer/kamino/internal/paths"
)

const githubBase = "https://github.com/"

// tagPattern accepts the tag segment of a release URL.
var tagPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// githubRelease downloads the asset of the latest release, verifies it against
// the release checksums when the recipe declares them, and only then writes the
// executables under the home. Any failure before that leaves the home intact.
func (i *Installer) githubRelease(ctx context.Context, recipe Recipe) error {
	tag, err := i.latestTag(ctx, recipe.Repo)
	if err != nil {
		return err
	}
	version := strings.TrimPrefix(tag, "v")
	asset := strings.ReplaceAll(recipe.Asset, "{version}", version)
	if !assetPattern.MatchString(asset) || strings.ContainsAny(asset, "{}") {
		return fmt.Errorf("nombre de asset no válido tras resolver la versión: %q", asset)
	}
	base := githubBase + recipe.Repo + "/releases/download/" + tag + "/"
	dir, err := os.MkdirTemp("", "kamino-install-")
	if err != nil {
		return fmt.Errorf("no se pudo crear el temporal: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	download := filepath.Join(dir, "asset")
	if err := i.download(ctx, base+asset, download); err != nil {
		return err
	}
	if recipe.Checksums != "" {
		sums := filepath.Join(dir, "checksums")
		name := strings.ReplaceAll(recipe.Checksums, "{version}", version)
		if err := i.download(ctx, base+name, sums); err != nil {
			return err
		}
		if err := verifyListedSHA256(download, asset, sums); err != nil {
			return err
		}
	}
	safe, err := paths.NewSafeHome(i.Home, false)
	if err != nil {
		return err
	}
	defer func() { _ = safe.Close() }()
	switch recipe.Format {
	case FormatRaw:
		file, err := os.Open(download)
		if err != nil {
			return fmt.Errorf("no se pudo leer la descarga: %w", err)
		}
		defer func() { _ = file.Close() }()
		return safe.CopyFile(recipe.Dest, file, 0o755)
	case FormatZip:
		return extractZip(download, expandExtract(recipe.Extract, version), safe)
	default:
		return extractTarGz(download, expandExtract(recipe.Extract, version), safe)
	}
}

// latestTag resolves the latest release by reading the redirect of
// /releases/latest, which needs no API token and has no rate limit.
func (i *Installer) latestTag(ctx context.Context, repo string) (string, error) {
	args := []string{
		"--fail", "--silent", "--show-error",
		"--proto", "=https", "--tlsv1.2",
		"--output", os.DevNull, "--write-out", "%{redirect_url}",
	}
	args = append(args, i.curlArgs...)
	latest := githubBase + repo + "/releases/latest"
	args = append(args, latest)
	output, err := i.Runner.Run(ctx, "curl", args...)
	if err != nil {
		return "", commandError("la consulta del último release de "+repo, output, err)
	}
	prefix := githubBase + repo + "/releases/tag/"
	redirect := strings.TrimSpace(string(output))
	tag := strings.TrimPrefix(redirect, prefix)
	if tag == redirect || !tagPattern.MatchString(tag) {
		return "", fmt.Errorf("no se pudo resolver el último release de %s: respuesta inesperada", repo)
	}
	return tag, nil
}

func expandExtract(files []ExtractFile, version string) []ExtractFile {
	expanded := make([]ExtractFile, len(files))
	for index, file := range files {
		expanded[index] = ExtractFile{File: strings.ReplaceAll(file.File, "{version}", version), Dest: file.Dest}
	}
	return expanded
}

// verifyListedSHA256 checks the asset against the checksum file of the release.
// The file either lists "<hash>  <name>" lines or holds a single bare hash for
// this asset. A missing entry is an error: nothing is installed.
func verifyListedSHA256(assetPath, asset, sumsPath string) error {
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return fmt.Errorf("no se pudo leer el fichero de sumas: %w", err)
	}
	lines := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	expected := ""
	for _, line := range lines {
		fields := strings.Fields(line)
		switch {
		case len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset:
			expected = fields[0]
		case len(lines) == 1 && len(fields) == 1:
			expected = fields[0]
		}
	}
	if !sha256Pattern.MatchString(expected) {
		return fmt.Errorf("el fichero de sumas no lista %q: no se instaló nada", asset)
	}
	return verifySHA256(assetPath, expected)
}

func memberName(name string) string { return strings.TrimPrefix(name, "./") }

func extractZip(path string, files []ExtractFile, safe *paths.SafeHome) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el zip: %w", err)
	}
	defer func() { _ = reader.Close() }()
	members := map[string]*zip.File{}
	for _, member := range reader.File {
		if _, seen := members[memberName(member.Name)]; !seen {
			members[memberName(member.Name)] = member
		}
	}
	for _, file := range files {
		member, found := members[file.File]
		if !found {
			return fmt.Errorf("el archivo no contiene %q: no se instaló nada", file.File)
		}
		if !member.Mode().IsRegular() {
			return fmt.Errorf("%q no es un fichero regular: no se instaló nada", file.File)
		}
	}
	for _, file := range files {
		if err := copyZipMember(members[file.File], file.Dest, safe); err != nil {
			return err
		}
	}
	return nil
}

func copyZipMember(member *zip.File, dest string, safe *paths.SafeHome) error {
	content, err := member.Open()
	if err != nil {
		return fmt.Errorf("no se pudo leer %q del zip: %w", member.Name, err)
	}
	defer func() { _ = content.Close() }()
	return safe.CopyFile(dest, content, 0o755)
}

func extractTarGz(path string, files []ExtractFile, safe *paths.SafeHome) error {
	found, err := scanTar(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		kind, ok := found[file.File]
		if !ok {
			return fmt.Errorf("el archivo no contiene %q: no se instaló nada", file.File)
		}
		if kind != tar.TypeReg {
			return fmt.Errorf("%q no es un fichero regular: no se instaló nada", file.File)
		}
	}
	dests := map[string]string{}
	for _, file := range files {
		dests[file.File] = file.Dest
	}
	return walkTar(path, func(header *tar.Header, content io.Reader) error {
		dest, wanted := dests[memberName(header.Name)]
		if !wanted || header.Typeflag != tar.TypeReg {
			return nil
		}
		delete(dests, memberName(header.Name))
		return safe.CopyFile(dest, content, 0o755)
	})
}

// scanTar maps each member name to the type of its first occurrence.
func scanTar(path string) (map[string]byte, error) {
	found := map[string]byte{}
	err := walkTar(path, func(header *tar.Header, _ io.Reader) error {
		name := memberName(header.Name)
		if _, seen := found[name]; !seen {
			found[name] = header.Typeflag
		}
		return nil
	})
	return found, err
}

func walkTar(path string, visit func(*tar.Header, io.Reader) error) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("no se pudo leer la descarga: %w", err)
	}
	defer func() { _ = file.Close() }()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el tar.gz: %w", err)
	}
	defer func() { _ = gz.Close() }()
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("el tar.gz está dañado: %w", err)
		}
		if err := visit(header, reader); err != nil {
			return err
		}
	}
}
