package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vectorfy-co/valbridge/adapter"
	"github.com/vectorfy-co/valbridge/config"
	"github.com/vectorfy-co/valbridge/extractor"
	"github.com/vectorfy-co/valbridge/parser"
	"github.com/vectorfy-co/valbridge/retriever"
)

var (
	extractProjectDir string
	extractLangFilter string
	extractSchemaKey  string
	extractEnvFile    string
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract or fetch a single schema as JSON",
	RunE:  runExtract,
}

type extractOutput struct {
	Schema      json.RawMessage      `json:"schema"`
	Diagnostics []adapter.Diagnostic `json:"diagnostics,omitempty"`
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().StringVarP(&extractProjectDir, "project", "p", "", "project root directory (default: current directory)")
	extractCmd.Flags().StringVar(&extractLangFilter, "lang", "", "filter to specific language if multiple detected")
	extractCmd.Flags().StringVar(&extractSchemaKey, "schema", "", "schema key to extract (namespace:id or unique id)")
	extractCmd.Flags().StringVar(&extractEnvFile, "env-file", "", "path to env file for header variable substitution")
	_ = extractCmd.MarkFlagRequired("schema")
}

func runExtract(cmd *cobra.Command, args []string) error {
	root := extractProjectDir
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	if err := config.LoadEnvFile(extractEnvFile, root); err != nil {
		return err
	}

	result, err := parser.Parse(cmd.Context(), root, extractLangFilter)
	if err != nil {
		return err
	}

	decl, err := findDeclaration(result.Declarations, extractSchemaKey)
	if err != nil {
		return err
	}

	output, err := extractDeclaration(cmd, root, decl)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal extract output: %w", err)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
	return err
}

func findDeclaration(decls []parser.Declaration, key string) (parser.Declaration, error) {
	var matches []parser.Declaration
	for _, decl := range decls {
		if decl.Key() == key || decl.ID == key {
			matches = append(matches, decl)
		}
	}

	switch len(matches) {
	case 0:
		return parser.Declaration{}, fmt.Errorf("schema %q not found", key)
	case 1:
		return matches[0], nil
	default:
		return parser.Declaration{}, fmt.Errorf("schema %q is ambiguous; use namespace:id", key)
	}
}

func extractDeclaration(cmd *cobra.Command, root string, decl parser.Declaration) (extractOutput, error) {
	if extractor.IsNativeSourceType(decl.SourceType) {
		results, err := extractor.Extract(cmd.Context(), []parser.Declaration{decl}, extractor.Options{
			ProjectRoot: root,
			Concurrency: 1,
		})
		if err != nil {
			return extractOutput{}, err
		}
		return extractOutput{
			Schema:      results[0].Schema.Schema,
			Diagnostics: results[0].Diagnostics,
		}, nil
	}

	schemas, err := retriever.Retrieve(cmd.Context(), []parser.Declaration{decl}, retriever.DefaultOptions())
	if err != nil {
		return extractOutput{}, err
	}

	return extractOutput{
		Schema: schemas[0].Schema,
	}, nil
}
