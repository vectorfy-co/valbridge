package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

// Version information set via ldflags at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "valbridge",
	Short: "JSON Schema to native validators",
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("valbridge {{.Version}} (" + commit + ", " + date + ")\n")
}
