package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRunnerInDirUsesUVRunForWorkspaceRoot(t *testing.T) {
	if !commandExists("uv") {
		t.Skip("uv is required for runner detection test")
	}

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "uv.lock"), []byte("lock"), 0o644); err != nil {
		t.Fatalf("failed to write uv.lock: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[tool.uv.workspace]\nmembers = []\n"), 0o644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	cmd, args, err := detectRunnerInDir(tmpDir)
	if err != nil {
		t.Fatalf("detectRunnerInDir: %v", err)
	}

	if cmd != "uv" {
		t.Fatalf("expected uv runner, got %q", cmd)
	}
	if len(args) != 1 || args[0] != "run" {
		t.Fatalf("expected uv run, got %#v", args)
	}
}
