// Package update implements self-update from GitHub releases.
package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Gunmer/kamino/internal/buildinfo"
	kerrors "github.com/Gunmer/kamino/internal/errors"
	"github.com/Gunmer/kamino/internal/progress"
	"github.com/Gunmer/kamino/internal/version"
)

const (
	Repository = "Gunmer/kamino"
	AssetName  = "kamino-arm64"
	Timeout    = 30 * time.Second
	ChunkSize  = 64 * 1024
)

var (
	APIURL     = "https://api.github.com/repos/" + Repository + "/releases/latest"
	HTTPClient = &http.Client{Timeout: Timeout}
	UserAgent  = "kamino/" + version.Version
)

var (
	versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
	digestPattern  = regexp.MustCompile(`^sha256:([0-9a-fA-F]{64})$`)
)

// ReleaseInfo is the latest published asset.
type ReleaseInfo struct {
	Version     string
	AssetURL    string
	AssetSize   int64
	AssetDigest string
}

// Outcome is the stable update result category.
type Outcome string

const (
	OutcomeUpdated  Outcome = "updated"
	OutcomeUpToDate Outcome = "up_to_date"
	OutcomeNewer    Outcome = "newer"
)

// Result is the result of an update check or update.
type Result struct {
	CurrentVersion string
	LatestVersion  string
	Target         string
	Updated        bool
	Outcome        Outcome
}

// Check is a resolved update query.
type Check struct {
	CurrentVersion string
	Release        ReleaseInfo
	Target         string
	ShouldUpdate   bool
	Outcome        Outcome
}

// ParseVersion converts vX.Y.Z or X.Y.Z into comparable numbers.
func ParseVersion(value string) ([3]int, error) {
	var result [3]int
	match := versionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return result, updateError(kerrors.UpdateOther, "versión no válida: %q", value)
	}
	for index := 0; index < 3; index++ {
		var number int
		if _, err := fmt.Sscanf(match[index+1], "%d", &number); err != nil {
			return result, updateError(kerrors.UpdateOther, "versión no válida: %q", value)
		}
		result[index] = number
	}
	return result, nil
}

// NormalizeVersion returns the version without a leading v or spaces.
func NormalizeVersion(value string) (string, error) {
	parsed, err := ParseVersion(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d.%d", parsed[0], parsed[1], parsed[2]), nil
}

// InstalledBinaryPath returns the running binary or fails if not self-contained.
func InstalledBinaryPath() (string, error) {
	if !buildinfo.IsSelfContained() {
		return "", updateError(kerrors.UpdateNotSelfContained,
			"kamino update solo está disponible en el binario autocontenido. "+
				"En desarrollo usa `go build`; para una instalación con uv, "+
				"usa `uv tool upgrade kamino-cli`; con pip, actualiza el paquete por la vía habitual.")
	}
	executable, err := os.Executable()
	if err != nil {
		return "", updateError(kerrors.UpdateNotSelfContained, "no se encontró el binario instalado: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", updateError(kerrors.UpdateNotSelfContained, "no se pudo resolver el binario instalado: %v", err)
	}
	return resolved, nil
}

func ensureSupportedPlatform() error {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return updateError(kerrors.UpdatePlatform, "kamino update solo está disponible para macOS en Apple Silicon (arm64).")
	}
	return nil
}

type releasePayload struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               *int64 `json:"size"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

// LatestRelease queries the latest published release and validates its asset.
func LatestRelease() (ReleaseInfo, error) {
	request, err := http.NewRequest(http.MethodGet, APIURL, nil)
	if err != nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "no se pudo consultar la última versión: %v", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", UserAgent)
	response, err := HTTPClient.Do(request)
	if err != nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "no se pudo consultar la última versión: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork,
			"GitHub ha limitado las peticiones de la API; inténtalo más tarde "+
				"o descarga la release desde https://github.com/Gunmer/kamino/releases.")
	}
	if response.StatusCode != http.StatusOK {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork,
			"no se pudo consultar la última versión: HTTP %d", response.StatusCode)
	}
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "no se pudo consultar la última versión: %v", err)
	}
	var payload releasePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "la respuesta de GitHub no es JSON válido")
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "la release no incluye un tag de versión válido")
	}
	normalized, err := NormalizeVersion(payload.TagName)
	if err != nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "la release no tiene una versión válida: %q", payload.TagName)
	}
	if payload.Assets == nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "la release no incluye una lista de assets válida")
	}
	var asset *struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               *int64 `json:"size"`
		Digest             string `json:"digest"`
	}
	for index := range payload.Assets {
		if payload.Assets[index].Name == AssetName {
			asset = &payload.Assets[index]
			break
		}
	}
	if asset == nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "la release %s no incluye el asset %s", payload.TagName, AssetName)
	}
	if !strings.HasPrefix(asset.BrowserDownloadURL, "https://") {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "el asset %s no tiene una URL de descarga válida", AssetName)
	}
	if asset.Size == nil || *asset.Size <= 0 {
		return ReleaseInfo{}, updateError(kerrors.UpdateNetwork, "el asset %s no tiene un tamaño válido", AssetName)
	}
	if digestPattern.FindStringSubmatch(asset.Digest) == nil {
		return ReleaseInfo{}, updateError(kerrors.UpdateIntegrity, "el asset %s no publica un SHA-256 válido", AssetName)
	}
	return ReleaseInfo{
		Version:     normalized,
		AssetURL:    asset.BrowserDownloadURL,
		AssetSize:   *asset.Size,
		AssetDigest: strings.ToLower(asset.Digest),
	}, nil
}

func digestHex(digest string) (string, error) {
	match := digestPattern.FindStringSubmatch(digest)
	if match == nil {
		return "", updateError(kerrors.UpdateIntegrity, "SHA-256 no válido: %q", digest)
	}
	return strings.ToLower(match[1]), nil
}

// QueryUpdate checks platform, binary and latest release without downloading.
func QueryUpdate(currentVersion string, target string, release *ReleaseInfo) (Check, error) {
	targetPath := target
	if targetPath == "" {
		var err error
		targetPath, err = InstalledBinaryPath()
		if err != nil {
			return Check{}, err
		}
	} else {
		resolved, err := filepath.Abs(targetPath)
		if err != nil {
			return Check{}, updateError(kerrors.UpdateNotSelfContained, "no se encontró el binario instalado en %s", targetPath)
		}
		targetPath = resolved
	}
	if err := ensureSupportedPlatform(); err != nil {
		return Check{}, err
	}
	current, err := NormalizeVersion(currentVersion)
	if err != nil {
		return Check{}, err
	}
	if info, err := os.Stat(targetPath); err != nil || !info.Mode().IsRegular() {
		return Check{}, updateError(kerrors.UpdateNotSelfContained, "no se encontró el binario instalado en %s", targetPath)
	}
	found := ReleaseInfo{}
	if release != nil {
		found = *release
	} else {
		found, err = LatestRelease()
		if err != nil {
			return Check{}, err
		}
	}
	currentParts, _ := ParseVersion(current)
	foundParts, _ := ParseVersion(found.Version)
	var outcome Outcome
	shouldUpdate := false
	switch {
	case compareParts(foundParts, currentParts) > 0:
		outcome = OutcomeUpdated
		shouldUpdate = true
	case compareParts(foundParts, currentParts) == 0:
		outcome = OutcomeUpToDate
	default:
		outcome = OutcomeNewer
	}
	return Check{
		CurrentVersion: current,
		Release:        found,
		Target:         targetPath,
		ShouldUpdate:   shouldUpdate,
		Outcome:        outcome,
	}, nil
}

func compareParts(left, right [3]int) int {
	for index := 0; index < 3; index++ {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

// DownloadUpdate downloads and verifies the asset in a temporary file.
func DownloadUpdate(release ReleaseInfo, targetPath string, progressCallback progress.Callback, cancelled progress.Cancel) (string, error) {
	directory := filepath.Dir(targetPath)
	temp, err := os.CreateTemp(directory, ".kamino-update-")
	if err != nil {
		return "", updateError(kerrors.UpdateReplace, "no se pudo crear un archivo temporal en %s: %v", directory, err)
	}
	tempPath := temp.Name()
	_ = temp.Close()
	if err := downloadAsset(release, tempPath, progressCallback, cancelled); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return tempPath, nil
}

func downloadAsset(release ReleaseInfo, destination string, progressCallback progress.Callback, cancelled progress.Cancel) error {
	request, err := http.NewRequest(http.MethodGet, release.AssetURL, nil)
	if err != nil {
		return updateError(kerrors.UpdateNetwork, "no se pudo descargar %s: %v", AssetName, err)
	}
	request.Header.Set("User-Agent", UserAgent)
	response, err := HTTPClient.Do(request)
	if err != nil {
		return updateError(kerrors.UpdateNetwork, "no se pudo descargar %s: %v", AssetName, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return updateError(kerrors.UpdateNetwork, "no se pudo descargar %s: HTTP %d", AssetName, response.StatusCode)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return updateError(kerrors.UpdateReplace, "no se pudo escribir el temporal: %v", err)
	}
	defer file.Close()
	digest := sha256.New()
	var size int64
	buffer := make([]byte, ChunkSize)
	for {
		if err := progress.EnsureNotCancelled(cancelled); err != nil {
			return err
		}
		read, readErr := response.Body.Read(buffer)
		if read > 0 {
			if _, err := file.Write(buffer[:read]); err != nil {
				return updateError(kerrors.UpdateNetwork, "no se pudo descargar %s: %v", AssetName, err)
			}
			_, _ = digest.Write(buffer[:read])
			size += int64(read)
			progress.Emit(progressCallback, progress.Downloading, int(size), int(release.AssetSize), "")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return updateError(kerrors.UpdateNetwork, "no se pudo descargar %s: %v", AssetName, readErr)
		}
	}
	if size != release.AssetSize {
		return updateError(kerrors.UpdateIntegrity,
			"la descarga de %s está incompleta: se esperaban %d bytes y se recibieron %d", AssetName, release.AssetSize, size)
	}
	expected, err := digestHex(release.AssetDigest)
	if err != nil {
		return err
	}
	if hex.EncodeToString(digest.Sum(nil)) != expected {
		return updateError(kerrors.UpdateIntegrity, "el SHA-256 de %s no coincide con el publicado", AssetName)
	}
	if err := file.Sync(); err != nil {
		return updateError(kerrors.UpdateReplace, "no se pudo sincronizar el temporal: %v", err)
	}
	return nil
}

// ReplaceBinary chmods and atomically replaces the target.
func ReplaceBinary(temp, target string) error {
	mode := os.FileMode(0o755)
	if info, err := os.Stat(target); err == nil {
		if info.Mode().Perm()&0o111 != 0 {
			mode = info.Mode().Perm()
		}
	}
	if err := os.Chmod(temp, mode); err != nil {
		return updateError(kerrors.UpdateReplace, "no se pudo reemplazar %s: %v", target, err)
	}
	if err := os.Rename(temp, target); err != nil {
		return updateError(kerrors.UpdateReplace, "no se pudo reemplazar %s: %v", target, err)
	}
	return nil
}

// RunUpdate queries, downloads, verifies and replaces the binary.
func RunUpdate(currentVersion string, target string, release *ReleaseInfo, progressCallback progress.Callback, cancelled progress.Cancel) (Result, error) {
	check, err := QueryUpdate(currentVersion, target, release)
	if err != nil {
		return Result{}, err
	}
	if !check.ShouldUpdate {
		return Result{
			CurrentVersion: check.CurrentVersion,
			LatestVersion:  check.Release.Version,
			Target:         check.Target,
			Updated:        false,
			Outcome:        check.Outcome,
		}, nil
	}
	temp, err := DownloadUpdate(check.Release, check.Target, progressCallback, cancelled)
	if err != nil {
		return Result{}, err
	}
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(temp)
		}
	}()
	if err := progress.EnsureNotCancelled(cancelled); err != nil {
		return Result{}, err
	}
	progress.Emit(progressCallback, progress.Replacing, 0, 1, check.Target)
	if err := ReplaceBinary(temp, check.Target); err != nil {
		return Result{}, err
	}
	keepTemp = false
	return Result{
		CurrentVersion: check.CurrentVersion,
		LatestVersion:  check.Release.Version,
		Target:         check.Target,
		Updated:        true,
		Outcome:        OutcomeUpdated,
	}, nil
}

func updateError(kind kerrors.UpdateErrorKind, format string, args ...any) error {
	return &kerrors.UpdateError{
		KaminoError: kerrors.KaminoError{Msg: fmt.Sprintf(format, args...)},
		Kind:        kind,
	}
}
