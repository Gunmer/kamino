package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validSHA = "0000000000000000000000000000000000000000000000000000000000000000"

func TestRecipeValidateRejectsInvalidRecipes(t *testing.T) {
	cases := map[string]Recipe{
		"método no permitido":        {Method: "script", URL: "https://example.com/x"},
		"brew sin fórmula ni cask":   {Method: MethodBrew},
		"brew con fórmula y cask":    {Method: MethodBrew, Formula: "a", Cask: "b"},
		"brew con nombre con guion":  {Method: MethodBrew, Formula: "-evil"},
		"brew con url":               {Method: MethodBrew, Formula: "a", URL: "https://example.com"},
		"installer con http":         {Method: MethodInstaller, URL: "http://example.com/i.sh"},
		"installer sin url":          {Method: MethodInstaller},
		"installer con sha256":       {Method: MethodInstaller, URL: "https://example.com/i.sh", SHA256: validSHA},
		"binary sin sha256":          {Method: MethodBinary, URL: "https://example.com/b", Dest: ".local/bin/b"},
		"binary con sha256 corto":    {Method: MethodBinary, URL: "https://example.com/b", SHA256: "abc", Dest: ".local/bin/b"},
		"binary sin dest":            {Method: MethodBinary, URL: "https://example.com/b", SHA256: validSHA},
		"binary con dest absoluto":   {Method: MethodBinary, URL: "https://example.com/b", SHA256: validSHA, Dest: "/usr/local/bin/b"},
		"binary con dest que escapa": {Method: MethodBinary, URL: "https://example.com/b", SHA256: validSHA, Dest: "../b"},
		"binary con http":            {Method: MethodBinary, URL: "http://example.com/b", SHA256: validSHA, Dest: ".local/bin/b"},
		"npm sin package":            {Method: MethodNpm},
		"npm con metacaracteres":     {Method: MethodNpm, Package: "foo; rm -rf ~"},
		"npm con guion inicial":      {Method: MethodNpm, Package: "-g"},
		"npm con formula":            {Method: MethodNpm, Package: "foo", Formula: "foo"},
		"npm con url":                {Method: MethodNpm, Package: "foo", URL: "https://example.com/x"},
		"brew con package":           {Method: MethodBrew, Formula: "a", Package: "foo"},
		"installer con repo":         {Method: MethodInstaller, URL: "https://example.com/i.sh", Repo: "a/b"},
		"os no reconocido":           {Method: MethodBrew, Formula: "a", OS: "windows"},
		"arch no reconocida":         {Method: MethodBrew, Formula: "a", Arch: "386"},
	}
	for name, recipe := range cases {
		t.Run(name, func(t *testing.T) {
			if err := recipe.Validate(); err == nil {
				t.Fatalf("se esperaba un error para %+v", recipe)
			}
		})
	}
}

func TestRecipeValidateAcceptsValidRecipes(t *testing.T) {
	valid := []Recipe{
		{Method: MethodBrew, Formula: "git"},
		{Method: MethodBrew, Cask: "ghostty", OS: "darwin", Arch: "arm64"},
		{Method: MethodInstaller, URL: "https://example.com/install.sh"},
		{Method: MethodBinary, URL: "https://example.com/tool", SHA256: validSHA, Dest: ".local/bin/tool"},
		{Method: MethodNpm, Package: "@openai/codex"},
		{Method: MethodNpm, Package: "opencode-ai"},
	}
	for _, recipe := range valid {
		if err := recipe.Validate(); err != nil {
			t.Fatalf("receta válida rechazada %+v: %v", recipe, err)
		}
	}
}

func TestSelectFiltersByPlatform(t *testing.T) {
	recipes := []Recipe{
		{Method: MethodBinary, URL: "https://example.com/b", SHA256: validSHA, Dest: ".local/bin/b", Arch: "amd64"},
		{Method: MethodInstaller, URL: "https://example.com/i.sh"},
	}
	recipe, err := Select(recipes, "darwin", "arm64", &fakeRunner{})
	if err != nil || recipe.Method != MethodInstaller {
		t.Fatalf("se esperaba la receta installer en arm64, got %+v, %v", recipe, err)
	}
	if _, err := Select(recipes[:1], "darwin", "arm64", &fakeRunner{}); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("sin receta aplicable se esperaba ErrNoRecipe, got %v", err)
	}
}

type fakeRunner struct {
	tools map[string]bool
	calls [][]string
	// onCurl writes the body for a download; nil means curl fails.
	body    []byte
	curlErr error
	shErr   error
	npmErr  error
	// redirect answers the latest-release lookup; bodies maps a download URL to
	// its content and, when set, makes any other URL a 404.
	redirect string
	bodies   map[string][]byte
}

func (f *fakeRunner) Lookup(name string) bool { return f.tools[name] }

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	switch name {
	case "curl":
		if f.curlErr != nil {
			return []byte("curl: (22) HTTP 404"), f.curlErr
		}
		for _, arg := range args {
			if arg == "--write-out" {
				return []byte(f.redirect), nil
			}
		}
		body := f.body
		if f.bodies != nil {
			found := false
			if body, found = f.bodies[args[len(args)-1]]; !found {
				return []byte("curl: (22) HTTP 404"), errors.New("exit status 22")
			}
		}
		for index, arg := range args {
			if arg == "--output" {
				return nil, os.WriteFile(args[index+1], body, 0o600)
			}
		}
	case "npm":
		return nil, f.npmErr
	case "sh":
		return nil, f.shErr
	}
	return nil, nil
}

func TestSelectSkipsBrewWithoutAlternative(t *testing.T) {
	brewOnly := []Recipe{{Method: MethodBrew, Formula: "git"}}
	if _, err := Select(brewOnly, "darwin", "arm64", &fakeRunner{}); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("brew ausente sin alternativa debe dar ErrNoRecipe, got %v", err)
	}
	withAlternative := []Recipe{
		{Method: MethodBrew, Formula: "git"},
		{Method: MethodInstaller, URL: "https://example.com/i.sh"},
	}
	recipe, err := Select(withAlternative, "darwin", "arm64", &fakeRunner{})
	if err != nil || recipe.Method != MethodInstaller {
		t.Fatalf("debe elegir la alternativa cuando brew falta, got %+v, %v", recipe, err)
	}
}

func TestBrewInstallRunsExpectedCommand(t *testing.T) {
	runner := &fakeRunner{tools: map[string]bool{"brew": true}}
	installer := &Installer{Runner: runner, Home: t.TempDir()}
	recipes := []Recipe{{Method: MethodBrew, Cask: "ghostty"}}
	if err := installer.Install(context.Background(), recipes, "darwin", "arm64"); err != nil {
		t.Fatal(err)
	}
	want := []string{"brew", "install", "--cask", "ghostty"}
	if len(runner.calls) != 1 || strings.Join(runner.calls[0], " ") != strings.Join(want, " ") {
		t.Fatalf("comando inesperado: %v", runner.calls)
	}
}

func TestBrewMissingExecutesNothing(t *testing.T) {
	runner := &fakeRunner{}
	installer := &Installer{Runner: runner, Home: t.TempDir()}
	recipes := []Recipe{{Method: MethodBrew, Formula: "git"}}
	err := installer.Install(context.Background(), recipes, "darwin", "arm64")
	if !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("se esperaba ErrNoRecipe, got %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("sin brew no debe ejecutarse nada: %v", runner.calls)
	}
}

func TestInstallerRunsScriptFromFile(t *testing.T) {
	runner := &fakeRunner{body: []byte("echo installed\n")}
	installer := &Installer{Runner: runner, Home: t.TempDir()}
	recipes := []Recipe{{Method: MethodInstaller, URL: "https://example.com/i.sh"}}
	if err := installer.Install(context.Background(), recipes, "darwin", "arm64"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 2 || runner.calls[0][0] != "curl" || runner.calls[1][0] != "sh" {
		t.Fatalf("se esperaba curl y luego sh: %v", runner.calls)
	}
	script := runner.calls[1][1]
	if len(runner.calls[1]) != 2 {
		t.Fatalf("sh debe recibir solo la ruta del script, got %v", runner.calls[1])
	}
	if !strings.HasSuffix(script, "installer.sh") {
		t.Fatalf("ruta de script inesperada: %s", script)
	}
}

func TestInstallerDownloadFailureExecutesNothing(t *testing.T) {
	runner := &fakeRunner{curlErr: errors.New("exit status 22")}
	installer := &Installer{Runner: runner, Home: t.TempDir()}
	recipes := []Recipe{{Method: MethodInstaller, URL: "https://example.com/i.sh"}}
	err := installer.Install(context.Background(), recipes, "darwin", "arm64")
	if err == nil {
		t.Fatal("se esperaba error de descarga")
	}
	for _, call := range runner.calls {
		if call[0] == "sh" {
			t.Fatalf("una descarga fallida no debe ejecutar nada: %v", runner.calls)
		}
	}
}

func TestInstallerRejectsHTTPSLessRecipe(t *testing.T) {
	recipe := Recipe{Method: MethodInstaller, URL: "http://example.com/i.sh"}
	if err := recipe.Validate(); err == nil {
		t.Fatal("una URL http debe rechazarse")
	}
}

// newTLSServer serves body at /tool and returns the server plus a curl flag
// that trusts its certificate.
func newTLSServer(t *testing.T, handler http.Handler) (*httptest.Server, []string) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	certPath := filepath.Join(t.TempDir(), "cert.pem")
	block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(certPath, block, 0o600); err != nil {
		t.Fatal(err)
	}
	return server, []string{"--cacert", certPath}
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestBinaryInstallsVerifiedFileWithMode0755(t *testing.T) {
	payload := []byte("#!/bin/sh\necho tool\n")
	server, trust := newTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	home := t.TempDir()
	installer := &Installer{Runner: ExecRunner{}, Home: home, curlArgs: trust}
	recipes := []Recipe{{Method: MethodBinary, URL: server.URL + "/tool", SHA256: checksum(payload), Dest: ".local/bin/tool"}}
	if err := installer.Install(context.Background(), recipes, "darwin", "arm64"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(home, ".local/bin/tool"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("permisos %o, se esperaban 0755", info.Mode().Perm())
	}
}

func TestBinaryChecksumMismatchKeepsDestination(t *testing.T) {
	server, trust := newTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("otro contenido"))
	}))
	home := t.TempDir()
	destination := filepath.Join(home, ".local/bin/tool")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("previo"), 0o755); err != nil {
		t.Fatal(err)
	}
	installer := &Installer{Runner: ExecRunner{}, Home: home, curlArgs: trust}
	recipes := []Recipe{{Method: MethodBinary, URL: server.URL + "/tool", SHA256: validSHA, Dest: ".local/bin/tool"}}
	if err := installer.Install(context.Background(), recipes, "darwin", "arm64"); err == nil {
		t.Fatal("se esperaba error por SHA-256 incorrecto")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "previo" {
		t.Fatalf("el destino debe quedar intacto, got %q, %v", data, err)
	}
}

func TestBinaryDownloadFailureWritesNothing(t *testing.T) {
	server, trust := newTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	home := t.TempDir()
	installer := &Installer{Runner: ExecRunner{}, Home: home, curlArgs: trust}
	recipes := []Recipe{{Method: MethodBinary, URL: server.URL + "/tool", SHA256: validSHA, Dest: ".local/bin/tool"}}
	if err := installer.Install(context.Background(), recipes, "darwin", "arm64"); err == nil {
		t.Fatal("se esperaba error de descarga")
	}
	if _, err := os.Stat(filepath.Join(home, ".local/bin/tool")); !os.IsNotExist(err) {
		t.Fatalf("una descarga fallida no debe crear el destino: %v", err)
	}
}

func TestMethodErrorNamesAllFiveMethods(t *testing.T) {
	err := Recipe{Method: "script"}.Validate()
	if err == nil {
		t.Fatal("un método desconocido debe rechazarse")
	}
	for _, method := range []string{"brew", "npm", "installer", "binary", "github-release"} {
		if !strings.Contains(err.Error(), "'"+method+"'") {
			t.Fatalf("el error debe nombrar %q: %v", method, err)
		}
	}
}

func TestSelectPrefersCatalogOrderAmongAvailableTools(t *testing.T) {
	recipes := []Recipe{
		{Method: MethodBrew, Formula: "demo"},
		{Method: MethodNpm, Package: "demo"},
		{Method: MethodInstaller, URL: "https://example.com/i.sh"},
	}
	cases := map[string]struct {
		tools map[string]bool
		want  string
	}{
		"brew y npm": {map[string]bool{"brew": true, "npm": true}, MethodBrew},
		"solo npm":   {map[string]bool{"npm": true}, MethodNpm},
		"ninguna":    {nil, MethodInstaller},
		"solo brew":  {map[string]bool{"brew": true}, MethodBrew},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			recipe, err := Select(recipes, "darwin", "arm64", &fakeRunner{tools: test.tools})
			if err != nil || recipe.Method != test.want {
				t.Fatalf("se esperaba %s, got %+v, %v", test.want, recipe, err)
			}
		})
	}
	npmOnly := recipes[:2]
	if _, err := Select(npmOnly, "darwin", "arm64", &fakeRunner{}); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("sin brew ni npm debe dar ErrNoRecipe, got %v", err)
	}
}

func TestNpmInstallRunsGlobalInstall(t *testing.T) {
	runner := &fakeRunner{tools: map[string]bool{"npm": true}}
	installer := &Installer{Runner: runner, Home: t.TempDir()}
	recipes := []Recipe{{Method: MethodBrew, Formula: "codex"}, {Method: MethodNpm, Package: "@openai/codex"}}
	if err := installer.Install(context.Background(), recipes, "darwin", "arm64"); err != nil {
		t.Fatal(err)
	}
	want := "npm install --global @openai/codex"
	if len(runner.calls) != 1 || strings.Join(runner.calls[0], " ") != want {
		t.Fatalf("comando inesperado: %v", runner.calls)
	}
}

func TestNpmFailureIsReported(t *testing.T) {
	runner := &fakeRunner{tools: map[string]bool{"npm": true}, npmErr: errors.New("exit status 243")}
	installer := &Installer{Runner: runner, Home: t.TempDir()}
	err := installer.Install(context.Background(), []Recipe{{Method: MethodNpm, Package: "demo"}}, "darwin", "arm64")
	if err == nil || !strings.Contains(err.Error(), "npm install") {
		t.Fatalf("se esperaba un error de npm, got %v", err)
	}
}
