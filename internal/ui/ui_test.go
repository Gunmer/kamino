package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/implant"
	"github.com/Gunmer/kamino/internal/plan"
)

func TestCloneSummaryShowsRejectionsWithoutVerbose(t *testing.T) {
	result := clone.CloneResult{Manifest: &archive.Manifest{}, Plan: plan.Plan{Skipped: []plan.SkippedPath{
		{Path: ".local/bin/tool", Reason: "ejecutable no clonado"},
	}}}
	var out bytes.Buffer
	PrintCloneSummary(&out, result, false)
	if !strings.Contains(out.String(), ".local/bin/tool (ejecutable no clonado)") {
		t.Fatalf("el rechazo debe verse sin --verbose:\n%s", out.String())
	}
}

func TestCloneSummaryWithoutRejectionsPrintsNoSkippedBlock(t *testing.T) {
	var out bytes.Buffer
	PrintCloneSummary(&out, clone.CloneResult{Manifest: &archive.Manifest{}}, false)
	if strings.Contains(out.String(), "~") {
		t.Fatalf("sin rechazos no debe haber bloque de omitidos:\n%s", out.String())
	}
}

func TestImplantSummaryShowsInstallPlanInDryRun(t *testing.T) {
	result := implant.ImplantResult{
		DryRun: true,
		Plan:   &implant.ImplantPlan{},
		Installs: []implant.InstallOutcome{
			{AppID: "alpha", Method: "installer", URL: "https://example.com/i.sh", Status: implant.InstallPlanned},
		},
	}
	var out bytes.Buffer
	PrintImplantSummary(&out, result, false)
	want := "Se instalaría: alpha con installer desde https://example.com/i.sh"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("la simulación debe mostrar la instalación:\n%s", out.String())
	}
}

func TestImplantSummaryWarnsWhenDownloadHasNoChecksum(t *testing.T) {
	result := implant.ImplantResult{
		DryRun: true,
		Plan:   &implant.ImplantPlan{},
		Installs: []implant.InstallOutcome{
			{AppID: "alpha", Method: "github-release", URL: "https://github.com/acme/alpha/releases", Unverified: true, Status: implant.InstallPlanned},
			{AppID: "beta", Method: "github-release", URL: "https://github.com/acme/beta/releases", Status: implant.InstallPlanned},
		},
	}
	var out bytes.Buffer
	PrintImplantSummary(&out, result, false)
	if !strings.Contains(out.String(), "alpha con github-release desde https://github.com/acme/alpha/releases (sin verificación de suma") {
		t.Fatalf("falta el aviso de verificación:\n%s", out.String())
	}
	if strings.Count(out.String(), "sin verificación de suma") != 1 {
		t.Fatalf("solo la receta sin suma debe llevar el aviso:\n%s", out.String())
	}
}
