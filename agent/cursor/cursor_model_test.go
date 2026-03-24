package cursor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStripANSIFromCLIOutput(t *testing.T) {
	in := "\x1b[2K\x1b[Gmodel-a - Model A  (current)\n"
	got := stripANSIFromCLIOutput(in)
	want := "model-a - Model A  (current)\n"
	if got != want {
		t.Fatalf("stripANSI = %q, want %q", got, want)
	}
}

func TestParseAgentModelsOutput(t *testing.T) {
	text := "model-b - Model B\nmodel-a - Model A  (current)\n"
	models := parseAgentModelsOutput(text)
	if len(models) != 2 {
		t.Fatalf("len = %d, want 2", len(models))
	}
	if models[0].Name != "model-a" || models[0].Desc != "Model A" {
		t.Errorf("first = %+v, want model-a / Model A", models[0])
	}
	if models[1].Name != "model-b" || models[1].Desc != "Model B" {
		t.Errorf("second = %+v, want model-b / Model B", models[1])
	}
}

func TestParseAgentModelsOutput_SkipsHeaders(t *testing.T) {
	text := "Available models\n\nfoo - Bar  (default)\n"
	models := parseAgentModelsOutput(text)
	if len(models) != 1 || models[0].Name != "foo" || models[0].Desc != "Bar" {
		t.Fatalf("got %+v", models)
	}
}

func TestFetchModelsFromAgentCLI_FailsGracefully(t *testing.T) {
	ctx := context.Background()
	models, ok := fetchModelsFromAgentCLI(ctx, "nonexistent-agent-xyz", "", nil)
	if ok || len(models) != 0 {
		t.Errorf("want ok=false and empty models, got ok=%v len=%d", ok, len(models))
	}
}

func TestAvailableModels_Fallback(t *testing.T) {
	ctx := context.Background()
	a := &Agent{cmd: "nonexistent-cmd-that-will-fail"}
	models := a.AvailableModels(ctx)
	fallback := cursorFallbackModels()
	if len(models) != len(fallback) {
		t.Fatalf("fallback models length = %d, want %d", len(models), len(fallback))
	}
	for i := range models {
		if models[i].Name != fallback[i].Name {
			t.Errorf("models[%d].Name = %q, want %q", i, models[i].Name, fallback[i].Name)
		}
	}
}

func TestAvailableModels_NoFallbackWhenCLIReportsNoModels(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-agent")
	body := "#!/bin/sh\nprintf '%s\\n' 'No models available for this account.'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	a := &Agent{cmd: script, workDir: dir}
	models := a.AvailableModels(ctx)
	if len(models) != 0 {
		t.Fatalf("expected empty list when CLI reports no models, got %d: %+v", len(models), models)
	}
}

func TestAvailableModels_ParsesFakeCLIOutputWithANSI(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-agent")
	body := "#!/bin/sh\nprintf '\\033[2K\\033[Gsonnet-4 - Sonnet 4\\n'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	a := &Agent{cmd: script, workDir: dir}
	models := a.AvailableModels(ctx)
	if len(models) != 1 || models[0].Name != "sonnet-4" || models[0].Desc != "Sonnet 4" {
		t.Fatalf("got %+v", models)
	}
}

func TestAvailableModels_IntegrationAgentCLI(t *testing.T) {
	if _, err := exec.LookPath("agent"); err != nil {
		t.Skip("agent CLI not in PATH")
	}

	ctx := context.Background()
	a := &Agent{cmd: "agent", workDir: "."}
	models := a.AvailableModels(ctx)

	t.Logf("AvailableModels returned %d models", len(models))
	for i, m := range models {
		t.Logf("  %2d. %s - %s", i+1, m.Name, m.Desc)
	}

	// Logged-in accounts get a non-empty list; unauthenticated CI may get none or fallback.
	if len(models) == 0 {
		t.Log("empty model list (no account models and CLI reported none)")
	}
}
