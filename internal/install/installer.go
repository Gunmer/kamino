package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gunmer/kamino/internal/paths"
)

// Installer runs the selected recipe of an app. It never installs Homebrew and
// never executes anything that was not downloaded from the recipe's URL.
type Installer struct {
	Runner Runner
	// Home is the absolute home where binary recipes write their destination.
	Home string
	// curlArgs are extra curl flags. Only tests set them, to trust a local
	// HTTPS server; production always uses the fixed flags below.
	curlArgs []string
}

// Applicable returns the first recipe that matches the platform, without
// checking tools on the PATH. The TUI and the plan use it to decide whether an
// app can be offered for installation.
func Applicable(recipes []Recipe, goos, goarch string) (Recipe, error) {
	for _, recipe := range recipes {
		if recipe.Applies(goos, goarch) {
			return recipe, nil
		}
	}
	return Recipe{}, ErrNoRecipe
}

// Select returns the first recipe that matches the platform and whose tool is
// available. A brew recipe without brew on the PATH is skipped in favour of the
// next recipe, so the app is omitted only when none can run.
func Select(recipes []Recipe, goos, goarch string, runner Runner) (Recipe, error) {
	for _, recipe := range recipes {
		if !recipe.Applies(goos, goarch) {
			continue
		}
		if recipe.Method == MethodBrew && !runner.Lookup("brew") {
			continue
		}
		return recipe, nil
	}
	return Recipe{}, ErrNoRecipe
}

// Install runs the first viable recipe for the platform. It returns ErrNoRecipe
// when none applies; any other error means the attempt failed and nothing was
// written to the destination.
func (i *Installer) Install(ctx context.Context, recipes []Recipe, goos, goarch string) error {
	recipe, err := Select(recipes, goos, goarch, i.Runner)
	if err != nil {
		return err
	}
	switch recipe.Method {
	case MethodBrew:
		return i.brew(ctx, recipe)
	case MethodInstaller:
		return i.installer(ctx, recipe)
	default:
		return i.binary(ctx, recipe)
	}
}

func (i *Installer) brew(ctx context.Context, recipe Recipe) error {
	args := []string{"install"}
	if recipe.Cask != "" {
		args = append(args, "--cask", recipe.Cask)
	} else {
		args = append(args, recipe.Formula)
	}
	output, err := i.Runner.Run(ctx, "brew", args...)
	if err != nil {
		return commandError("brew install", output, err)
	}
	return nil
}

// installer downloads the script to a private temporary file and runs it from
// there. The script is never piped into an interpreter, so it can be kept for
// inspection and an interrupted download never executes.
func (i *Installer) installer(ctx context.Context, recipe Recipe) error {
	dir, err := os.MkdirTemp("", "kamino-install-")
	if err != nil {
		return fmt.Errorf("no se pudo crear el temporal: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	script := filepath.Join(dir, "installer.sh")
	if err := i.download(ctx, recipe.URL, script); err != nil {
		return err
	}
	output, err := i.Runner.Run(ctx, "sh", script)
	if err != nil {
		return commandError("el instalador", output, err)
	}
	return nil
}

// binary downloads the file, checks its SHA-256 while streaming, and only then
// moves it into the home with mode 0755. A mismatch leaves the destination
// untouched.
func (i *Installer) binary(ctx context.Context, recipe Recipe) error {
	dir, err := os.MkdirTemp("", "kamino-install-")
	if err != nil {
		return fmt.Errorf("no se pudo crear el temporal: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	download := filepath.Join(dir, "download")
	if err := i.download(ctx, recipe.URL, download); err != nil {
		return err
	}
	if err := verifySHA256(download, recipe.SHA256); err != nil {
		return err
	}
	safe, err := paths.NewSafeHome(i.Home, false)
	if err != nil {
		return err
	}
	defer func() { _ = safe.Close() }()
	file, err := os.Open(download)
	if err != nil {
		return fmt.Errorf("no se pudo leer la descarga: %w", err)
	}
	defer func() { _ = file.Close() }()
	return safe.CopyFile(recipe.Dest, file, 0o755)
}

func (i *Installer) download(ctx context.Context, rawURL, destination string) error {
	args := []string{
		"--fail", "--silent", "--show-error", "--location",
		"--proto", "=https", "--proto-redir", "=https", "--tlsv1.2",
	}
	args = append(args, i.curlArgs...)
	args = append(args, "--output", destination, rawURL)
	output, err := i.Runner.Run(ctx, "curl", args...)
	if err != nil {
		return commandError("la descarga de "+rawURL, output, err)
	}
	return nil
}

func verifySHA256(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("no se pudo leer la descarga: %w", err)
	}
	defer func() { _ = file.Close() }()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return fmt.Errorf("no se pudo calcular el SHA-256: %w", err)
	}
	if !strings.EqualFold(hex.EncodeToString(digest.Sum(nil)), expected) {
		return fmt.Errorf("el SHA-256 de la descarga no coincide con el declarado: no se instaló nada")
	}
	return nil
}

// commandError keeps the last line of the tool output so the user sees why it
// failed without flooding the terminal.
func commandError(what string, output []byte, err error) error {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return fmt.Errorf("%s falló: %w", what, err)
	}
	return fmt.Errorf("%s falló: %w: %s", what, err, last)
}
