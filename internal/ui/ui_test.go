package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/archive"
	"github.com/Gunmer/kamino/internal/clone"
	"github.com/Gunmer/kamino/internal/deploy"
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

func TestDeploySummaryShowsInstallPlanInDryRun(t *testing.T) {
	result := deploy.DeployResult{
		DryRun: true,
		Plan:   &deploy.DeployPlan{},
		Installs: []deploy.InstallOutcome{
			{AppID: "alpha", Method: "installer", URL: "https://example.com/i.sh", Status: deploy.InstallPlanned},
		},
	}
	var out bytes.Buffer
	PrintDeploySummary(&out, result, false)
	want := "Se instalaría: alpha con installer desde https://example.com/i.sh"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("la simulación debe mostrar la instalación:\n%s", out.String())
	}
}
