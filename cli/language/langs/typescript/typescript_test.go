package typescript

import (
	"strings"
	"testing"

	"github.com/vectorfy-co/valbridge/config"
	"github.com/vectorfy-co/valbridge/language"
)

func TestAdapterInvokerPrefersPublishedPackagesByDefault(t *testing.T) {
	t.Setenv(config.EnvWorkspaceRoot, "")
	t.Setenv(config.EnvPreferWorkspace, "")

	spec, err := adapterInvoker{}.BuildAdapterCommand(t.Context(), language.AdapterCommandInput{
		ProjectRoot: t.TempDir(),
		AdapterRef:  "@vectorfyco/valbridge-zod",
	})
	if err != nil {
		t.Fatalf("BuildAdapterCommand: %v", err)
	}

	if spec.Cmd != "pnpm" && spec.Cmd != "npx" {
		t.Fatalf("expected published package runner, got %q", spec.Cmd)
	}
	if len(spec.Args) == 0 || !strings.Contains(strings.Join(spec.Args, " "), "@vectorfyco/valbridge-zod") {
		t.Fatalf("expected published package ref in args, got %#v", spec.Args)
	}
}
