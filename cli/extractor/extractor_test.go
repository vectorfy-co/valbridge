package extractor

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vectorfy-co/valbridge/config"
	"github.com/vectorfy-co/valbridge/parser"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func requireCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
}

func TestExtractPydanticSource(t *testing.T) {
	requireCommand(t, "uv")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "app.valbridge.jsonc")
	modelPath := filepath.Join(tmpDir, "models.py")
	if err := os.WriteFile(modelPath, []byte("from pydantic import BaseModel\n\nclass UserModel(BaseModel):\n    count: int = 3\n"), 0o644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}

	results, err := Extract(context.Background(), []parser.Declaration{{
		Namespace:  "test",
		ID:         "UserModel",
		SourceType: parser.SourcePydantic,
		Source:     json.RawMessage(`"models:UserModel"`),
		Adapter:    "@vectorfyco/valbridge-zod",
		ConfigPath: configPath,
		ModuleRoot: ".",
	}}, Options{ProjectRoot: repoRoot(t), Concurrency: 1})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 extracted schema, got %d", len(results))
	}

	var payload map[string]any
	if err := json.Unmarshal(results[0].Schema.Schema, &payload); err != nil {
		t.Fatalf("invalid extracted schema: %v", err)
	}

	properties := payload["properties"].(map[string]any)
	count := properties["count"].(map[string]any)
	valbridge := count["x-valbridge"].(map[string]any)
	defaultBehavior := valbridge["defaultBehavior"].(map[string]any)
	if defaultBehavior["kind"] != "default" || defaultBehavior["value"] != float64(3) {
		t.Fatalf("expected preserved literal default, got %#v", defaultBehavior)
	}
}

func TestExtractZodSource(t *testing.T) {
	requireCommand(t, "pnpm")

	tmpDir, err := os.MkdirTemp(filepath.Join(repoRoot(t), "typescript", "packages", "zod-extractor"), "extractor-test-*")
	if err != nil {
		t.Fatalf("failed to create workspace temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})
	configPath := filepath.Join(tmpDir, "app.valbridge.jsonc")
	modulePath := filepath.Join(tmpDir, "schema.mjs")
	if err := os.WriteFile(modulePath, []byte("import { z } from 'zod';\nexport const userSchema = z.object({ name: z.string().trim() }).strict();\n"), 0o644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}

	results, err := Extract(context.Background(), []parser.Declaration{{
		Namespace:  "test",
		ID:         "UserSchema",
		SourceType: parser.SourceZod,
		Source:     json.RawMessage(`"./schema.mjs"`),
		Adapter:    "vectorfyco/valbridge-pydantic",
		ConfigPath: configPath,
		Export:     "userSchema",
	}}, Options{ProjectRoot: repoRoot(t), Concurrency: 1})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 extracted schema, got %d", len(results))
	}

	var payload map[string]any
	if err := json.Unmarshal(results[0].Schema.Schema, &payload); err != nil {
		t.Fatalf("invalid extracted schema: %v", err)
	}

	properties := payload["properties"].(map[string]any)
	name := properties["name"].(map[string]any)
	valbridge := name["x-valbridge"].(map[string]any)
	transforms := valbridge["transforms"].([]any)
	if len(transforms) != 1 {
		t.Fatalf("expected trim transform, got %#v", transforms)
	}
}

func TestFindWorkspaceRootFallsBackAcrossSearchRoots(t *testing.T) {
	workspaceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspaceRoot, "typescript", "packages", "zod-extractor"), 0o755); err != nil {
		t.Fatalf("failed to create fake workspace probe: %v", err)
	}

	externalProject := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(externalProject, 0o755); err != nil {
		t.Fatalf("failed to create external project: %v", err)
	}

	found, err := findWorkspaceRoot(
		filepath.Join("typescript", "packages", "zod-extractor"),
		externalProject,
		workspaceRoot,
	)
	if err != nil {
		t.Fatalf("findWorkspaceRoot: %v", err)
	}
	if found != workspaceRoot {
		t.Fatalf("expected fallback workspace root %q, got %q", workspaceRoot, found)
	}
}

func TestResolveWorkspaceRootUsesExplicitEnv(t *testing.T) {
	workspaceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspaceRoot, "typescript", "packages", "zod-extractor"), 0o755); err != nil {
		t.Fatalf("failed to create fake workspace probe: %v", err)
	}

	t.Setenv(config.EnvWorkspaceRoot, workspaceRoot)

	found, err := resolveWorkspaceRoot(
		t.TempDir(),
		filepath.Join("typescript", "packages", "zod-extractor"),
	)
	if err != nil {
		t.Fatalf("resolveWorkspaceRoot: %v", err)
	}
	if found != workspaceRoot {
		t.Fatalf("expected explicit workspace root %q, got %q", workspaceRoot, found)
	}
}

func TestBuildZodExtractorCandidatesPrefersPublishedByDefault(t *testing.T) {
	t.Setenv(config.EnvWorkspaceRoot, "")
	t.Setenv(config.EnvPreferWorkspace, "")

	candidates, err := buildZodExtractorCandidates(repoRoot(t), "/tmp/schema.ts", "Schema", "pnpm")
	if err != nil {
		t.Fatalf("buildZodExtractorCandidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}
	if !strings.HasPrefix(candidates[0].Label, "published-zod-extractor-") {
		t.Fatalf("expected published extractor first, got %q", candidates[0].Label)
	}
}

func TestBuildPydanticExtractorCandidatesPrefersPublishedByDefault(t *testing.T) {
	t.Setenv(config.EnvWorkspaceRoot, "")
	t.Setenv(config.EnvPreferWorkspace, "")

	candidates, err := buildPydanticExtractorCandidates(repoRoot(t), []string{"app.models:User"})
	if err != nil {
		t.Fatalf("buildPydanticExtractorCandidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}
	if !strings.HasPrefix(candidates[0].Label, "published-python-extractor-") {
		t.Fatalf("expected published extractor first, got %q", candidates[0].Label)
	}
}
