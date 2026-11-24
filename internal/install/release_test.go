package install

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	releasePage = "https://github.com/acme/tool/releases/"
	releaseTag  = releasePage + "tag/v2.105.0"
	downloadURL = releasePage + "download/v2.105.0/"
)

func zipOf(t *testing.T, members map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range members {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(content))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func tarGzOf(t *testing.T, members map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for name, content := range members {
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(content))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func releaseRunner(bodies map[string][]byte) *fakeRunner {
	return &fakeRunner{redirect: releaseTag, bodies: bodies}
}

func installRelease(t *testing.T, runner Runner, home string, recipe Recipe) error {
	t.Helper()
	installer := &Installer{Runner: runner, Home: home}
	return installer.Install(context.Background(), []Recipe{recipe}, "darwin", "arm64")
}

func TestGithubReleaseValidateRejectsInvalidRecipes(t *testing.T) {
	valid := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool.zip", Format: FormatZip,
		Extract: []ExtractFile{{File: "tool", Dest: ".local/bin/tool"}}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("receta válida rechazada: %v", err)
	}
	mutate := map[string]func(*Recipe){
		"repo con ..":           func(r *Recipe) { r.Repo = "../otro/sitio" },
		"repo sin propietario":  func(r *Recipe) { r.Repo = "tool" },
		"repo vacío":            func(r *Recipe) { r.Repo = "" },
		"asset con barra":       func(r *Recipe) { r.Asset = "../tool.zip" },
		"asset vacío":           func(r *Recipe) { r.Asset = "" },
		"formato desconocido":   func(r *Recipe) { r.Format = "rar" },
		"formato vacío":         func(r *Recipe) { r.Format = "" },
		"zip sin extract":       func(r *Recipe) { r.Extract = nil },
		"zip con dest":          func(r *Recipe) { r.Dest = ".local/bin/tool" },
		"extract con dest ..":   func(r *Recipe) { r.Extract = []ExtractFile{{File: "tool", Dest: "../tool"}} },
		"extract con file ..":   func(r *Recipe) { r.Extract = []ExtractFile{{File: "../tool", Dest: ".local/bin/tool"}} },
		"extract con file abs":  func(r *Recipe) { r.Extract = []ExtractFile{{File: "/tool", Dest: ".local/bin/tool"}} },
		"extract incompleto":    func(r *Recipe) { r.Extract = []ExtractFile{{File: "tool"}} },
		"raw sin dest":          func(r *Recipe) { r.Format, r.Extract = FormatRaw, nil },
		"raw con extract":       func(r *Recipe) { r.Format, r.Dest = FormatRaw, ".local/bin/tool" },
		"raw con dest absoluto": func(r *Recipe) { r.Format, r.Extract, r.Dest = FormatRaw, nil, "/usr/bin/tool" },
		"con url":               func(r *Recipe) { r.URL = "https://example.com/x" },
		"con sha256":            func(r *Recipe) { r.SHA256 = validSHA },
		"con package":           func(r *Recipe) { r.Package = "tool" },
		"con formula":           func(r *Recipe) { r.Formula = "tool" },
		"checksums con barra":   func(r *Recipe) { r.Checksums = "../sums" },
	}
	for name, change := range mutate {
		t.Run(name, func(t *testing.T) {
			recipe := valid
			change(&recipe)
			if err := recipe.Validate(); err == nil {
				t.Fatalf("se esperaba un error para %+v", recipe)
			}
		})
	}
	raw := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool", Format: FormatRaw, Dest: ".local/bin/tool", Checksums: "tool.sha"}
	if err := raw.Validate(); err != nil {
		t.Fatalf("receta raw válida rechazada: %v", err)
	}
}

func TestGithubReleaseFieldsOutsideMethodAreRejected(t *testing.T) {
	for _, method := range []Recipe{
		{Method: MethodBinary, URL: "https://example.com/b", SHA256: validSHA, Dest: ".local/bin/b", Repo: "a/b"},
		{Method: MethodBrew, Formula: "a", Checksums: "sums"},
		{Method: MethodNpm, Package: "a", Format: FormatRaw},
	} {
		if err := method.Validate(); err == nil {
			t.Fatalf("los campos de github-release no deben admitirse en %+v", method)
		}
	}
}

func TestGithubReleaseSourceAndUnverified(t *testing.T) {
	recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool"}
	if recipe.Source() != "https://github.com/acme/tool/releases" || !recipe.Unverified() {
		t.Fatalf("origen o aviso inesperados: %q, %v", recipe.Source(), recipe.Unverified())
	}
	recipe.Checksums = "sums"
	if recipe.Unverified() {
		t.Fatal("con checksums la receta está verificada")
	}
	if (Recipe{Method: MethodBrew, Formula: "a"}).Unverified() {
		t.Fatal("brew no lleva aviso de verificación")
	}
}

func TestGithubReleaseInstallsAllExtractedFilesWithMode0755(t *testing.T) {
	archive := zipOf(t, map[string]string{"dist/yazi": "yazi-bin", "dist/ya": "ya-bin", "dist/readme": "no"})
	runner := releaseRunner(map[string][]byte{downloadURL + "tool_2.105.0.zip": archive})
	home := t.TempDir()
	recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool_{version}.zip", Format: FormatZip,
		Extract: []ExtractFile{{File: "dist/yazi", Dest: ".local/bin/yazi"}, {File: "dist/ya", Dest: ".local/bin/ya"}}}
	if err := installRelease(t, runner, home, recipe); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"yazi": "yazi-bin", "ya": "ya-bin"} {
		path := filepath.Join(home, ".local/bin", name)
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o755 {
			t.Fatalf("%s: %v, %v", name, info, err)
		}
		if data, _ := os.ReadFile(path); string(data) != want {
			t.Fatalf("%s: contenido %q", name, data)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".local/bin/readme")); !os.IsNotExist(err) {
		t.Fatal("solo se extraen los ficheros nombrados")
	}
}

func TestGithubReleaseResolvesLatestTagFromRedirect(t *testing.T) {
	runner := releaseRunner(map[string][]byte{downloadURL + "tool": []byte("bin")})
	recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool", Format: FormatRaw, Dest: ".local/bin/tool"}
	if err := installRelease(t, runner, t.TempDir(), recipe); err != nil {
		t.Fatal(err)
	}
	first := strings.Join(runner.calls[0], " ")
	if !strings.Contains(first, "--write-out %{redirect_url}") || !strings.HasSuffix(first, releasePage+"latest") ||
		!strings.Contains(first, "--proto =https") || strings.Contains(first, "--location") {
		t.Fatalf("consulta del último release inesperada: %s", first)
	}
}

func TestGithubReleaseWithoutRedirectFails(t *testing.T) {
	for name, redirect := range map[string]string{
		"vacía":          "",
		"otro repo":      "https://github.com/evil/tool/releases/tag/v1",
		"otro host":      "https://example.com/acme/tool/releases/tag/v1",
		"tag con barras": releasePage + "tag/v1/../../x",
	} {
		t.Run(name, func(t *testing.T) {
			runner := &fakeRunner{redirect: redirect}
			home := t.TempDir()
			recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool", Format: FormatRaw, Dest: ".local/bin/tool"}
			if err := installRelease(t, runner, home, recipe); err == nil {
				t.Fatal("se esperaba un error sin redirección válida")
			}
			if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
				t.Fatal("no se debe escribir nada")
			}
		})
	}
}

func TestGithubReleaseTarGzInstallsNamedMember(t *testing.T) {
	archive := tarGzOf(t, map[string]string{"./pkg/bin/tool": "tool-bin", "pkg/other": "x"})
	runner := releaseRunner(map[string][]byte{downloadURL + "tool.tar.gz": archive})
	home := t.TempDir()
	recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool.tar.gz", Format: FormatTarGz,
		Extract: []ExtractFile{{File: "pkg/bin/tool", Dest: ".local/bin/tool"}}}
	if err := installRelease(t, runner, home, recipe); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(home, ".local/bin/tool")); err != nil || string(data) != "tool-bin" {
		t.Fatalf("contenido inesperado: %q, %v", data, err)
	}
}

func TestGithubReleaseMissingMemberWritesNothing(t *testing.T) {
	archives := map[string][]byte{
		"zip":    zipOf(t, map[string]string{"dist/yazi": "yazi-bin"}),
		"tar.gz": tarGzOf(t, map[string]string{"dist/yazi": "yazi-bin"}),
	}
	for format, archive := range archives {
		t.Run(format, func(t *testing.T) {
			runner := releaseRunner(map[string][]byte{downloadURL + "tool": archive})
			home := t.TempDir()
			recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool", Format: format,
				Extract: []ExtractFile{{File: "dist/yazi", Dest: ".local/bin/yazi"}, {File: "dist/ya", Dest: ".local/bin/ya"}}}
			if err := installRelease(t, runner, home, recipe); err == nil {
				t.Fatal("se esperaba un error por el miembro ausente")
			}
			if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
				t.Fatal("un miembro ausente no debe dejar instalación a medias")
			}
		})
	}
}

func TestGithubReleaseDoesNotMatchEscapingMembers(t *testing.T) {
	archive := zipOf(t, map[string]string{"../dist/tool": "evil"})
	runner := releaseRunner(map[string][]byte{downloadURL + "tool.zip": archive})
	home := t.TempDir()
	recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool.zip", Format: FormatZip,
		Extract: []ExtractFile{{File: "dist/tool", Dest: ".local/bin/tool"}}}
	if err := installRelease(t, runner, home, recipe); err == nil {
		t.Fatal("un miembro con .. no debe instalarse")
	}
	if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
		t.Fatal("no se debe escribir nada")
	}
}

func TestGithubReleaseChecksums(t *testing.T) {
	payload := []byte("binary")
	good := checksum(payload)
	cases := map[string]struct {
		sums    map[string][]byte
		wantErr string
	}{
		"suma correcta":      {map[string][]byte{downloadURL + "sums.txt": []byte(good + "  tool\n")}, ""},
		"suma con asterisco": {map[string][]byte{downloadURL + "sums.txt": []byte("deadbeef  other\n" + good + " *tool\n")}, ""},
		"hash suelto":        {map[string][]byte{downloadURL + "sums.txt": []byte(good + "\n")}, ""},
		"suma incorrecta":    {map[string][]byte{downloadURL + "sums.txt": []byte(validSHA + "  tool\n")}, "no coincide"},
		"asset sin línea":    {map[string][]byte{downloadURL + "sums.txt": []byte(good + "  otro\n" + good + "  mas\n")}, "no lista"},
		"fichero ausente":    {nil, "falló"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			bodies := map[string][]byte{downloadURL + "tool": payload}
			for url, body := range test.sums {
				bodies[url] = body
			}
			home := t.TempDir()
			recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool", Format: FormatRaw,
				Dest: ".local/bin/tool", Checksums: "sums.txt"}
			err := installRelease(t, releaseRunner(bodies), home, recipe)
			_, statErr := os.Stat(filepath.Join(home, ".local/bin/tool"))
			if test.wantErr == "" {
				if err != nil || statErr != nil {
					t.Fatalf("se esperaba instalar: %v, %v", err, statErr)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("se esperaba un error con %q, got %v", test.wantErr, err)
			}
			if !os.IsNotExist(statErr) {
				t.Fatal("un fallo de verificación no debe escribir nada")
			}
		})
	}
}

func TestGithubReleaseMissingAssetFails(t *testing.T) {
	runner := releaseRunner(map[string][]byte{})
	home := t.TempDir()
	recipe := Recipe{Method: MethodGithubRelease, Repo: "acme/tool", Asset: "tool", Format: FormatRaw, Dest: ".local/bin/tool"}
	err := installRelease(t, runner, home, recipe)
	if err == nil || !strings.Contains(err.Error(), downloadURL+"tool") {
		t.Fatalf("el error debe nombrar el asset no encontrado: %v", err)
	}
}
