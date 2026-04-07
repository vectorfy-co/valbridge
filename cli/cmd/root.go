package cmd

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vectorfy-co/valbridge/config"
)

// Version information set via ldflags at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	workspaceRootFlag  string
	preferWorkspaceFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "valbridge",
	Short: "JSON Schema to native validators",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		_ = args

		if strings.TrimSpace(workspaceRootFlag) != "" {
			_ = os.Setenv(config.EnvWorkspaceRoot, workspaceRootFlag)
		}
		if cmd.Flags().Changed("prefer-workspace") {
			_ = os.Setenv(config.EnvPreferWorkspace, strconv.FormatBool(preferWorkspaceFlag))
		}
	},
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("valbridge {{.Version}} (" + commit + ", " + date + ")\n")
	rootCmd.PersistentFlags().StringVar(
		&workspaceRootFlag,
		"workspace-root",
		"",
		"explicit valbridge workspace root for local development adapters/extractors",
	)
	rootCmd.PersistentFlags().BoolVar(
		&preferWorkspaceFlag,
		"prefer-workspace",
		false,
		"prefer workspace-local adapters/extractors over published packages",
	)
}
