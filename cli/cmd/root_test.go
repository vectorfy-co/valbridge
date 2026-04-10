package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpBranding(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	origOut := rootCmd.OutOrStdout()
	origErr := rootCmd.ErrOrStderr()

	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	t.Cleanup(func() {
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
	})

	if err := rootCmd.Help(); err != nil {
		t.Fatalf("root help failed: %v", err)
	}

	helpText := stdout.String() + stderr.String()

	if !strings.Contains(helpText, "Convert between Zod and Pydantic validators") {
		t.Fatalf("expected updated branding in help output, got:\n%s", helpText)
	}

	if strings.Contains(helpText, "JSON Schema to native validators") {
		t.Fatalf("unexpected stale branding in help output, got:\n%s", helpText)
	}
}

func TestGenerateCommandBranding(t *testing.T) {
	if got, want := generateCmd.Short, "Generate Zod or Pydantic code from valbridge configs"; got != want {
		t.Fatalf("generate short description mismatch: got %q want %q", got, want)
	}
}
