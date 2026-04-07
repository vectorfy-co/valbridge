package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vectorfy-co/valbridge/adapter"
	"github.com/vectorfy-co/valbridge/parser"
	"github.com/vectorfy-co/valbridge/retriever"
)

type Options struct {
	ProjectRoot string
	Concurrency int
}

type ExtractedSchema struct {
	Schema      retriever.RetrievedSchema
	Diagnostics []adapter.Diagnostic
}

type extractorPayload struct {
	Schema      json.RawMessage      `json:"schema"`
	Diagnostics []adapter.Diagnostic `json:"diagnostics,omitempty"`
}

func IsNativeSourceType(sourceType parser.SourceType) bool {
	return sourceType == parser.SourcePydantic || sourceType == parser.SourceZod
}

func Extract(ctx context.Context, decls []parser.Declaration, opts Options) ([]ExtractedSchema, error) {
	if len(decls) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(opts.ProjectRoot) == "" {
		return nil, fmt.Errorf("project root is required for native extraction")
	}

	results := make([]ExtractedSchema, 0, len(decls))
	for _, decl := range decls {
		var extracted ExtractedSchema
		var err error

		switch decl.SourceType {
		case parser.SourcePydantic:
			extracted, err = extractPydantic(ctx, decl, opts.ProjectRoot)
		case parser.SourceZod:
			extracted, err = extractZod(ctx, decl, opts.ProjectRoot)
		default:
			err = fmt.Errorf("unsupported native source type %q", decl.SourceType)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to extract schema %s: %w", decl.Key(), err)
		}

		results = append(results, extracted)
	}

	return results, nil
}

func extractPydantic(ctx context.Context, decl parser.Declaration, projectRoot string) (ExtractedSchema, error) {
	target, err := parseSourceString(decl)
	if err != nil {
		return ExtractedSchema{}, err
	}

	workspaceRoot, err := findWorkspaceRoot(
		filepath.Join("python", "packages", "pydantic-extractor"),
		workspaceSearchRoots(projectRoot)...,
	)
	if err != nil {
		return ExtractedSchema{}, err
	}

	workdir := filepath.Join(workspaceRoot, "python")
	args := []string{"run", "valbridge-pydantic-extractor", target}
	if decl.ModuleRoot != "" {
		args = append(args, "--module-root", resolveRelativeToConfig(decl.ConfigPath, decl.ModuleRoot))
	}
	for _, entry := range decl.PythonPath {
		args = append(args, "--python-path", resolveRelativeToConfig(decl.ConfigPath, entry))
	}
	for _, requirement := range decl.Requirements {
		args = append(args, "--requirement", requirement)
	}
	for _, key := range sortedKeys(decl.Env) {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, decl.Env[key]))
	}
	for _, moduleName := range decl.StubModules {
		args = append(args, "--stub-module", moduleName)
	}

	payload, err := runExtractorCommand(ctx, workdir, "uv", args)
	if err != nil {
		return ExtractedSchema{}, err
	}

	sourceURI := fmt.Sprintf("pydantic://%s", target)
	return buildExtractedSchema(decl, sourceURI, payload)
}

func extractZod(ctx context.Context, decl parser.Declaration, projectRoot string) (ExtractedSchema, error) {
	modulePath, err := parseSourceString(decl)
	if err != nil {
		return ExtractedSchema{}, err
	}

	absModulePath := resolveRelativeToConfig(decl.ConfigPath, modulePath)
	workspaceRoot, err := findWorkspaceRoot(
		filepath.Join("typescript", "packages", "zod-extractor"),
		workspaceSearchRoots(projectRoot)...,
	)
	if err != nil {
		return ExtractedSchema{}, err
	}

	workdir := filepath.Join(workspaceRoot, "typescript")

	cmdName, args, err := buildZodCommand(workspaceRoot, absModulePath, decl.Export, decl.Runner)
	if err != nil {
		return ExtractedSchema{}, err
	}

	payload, err := runExtractorCommand(ctx, workdir, cmdName, args)
	if err != nil {
		return ExtractedSchema{}, err
	}

	return buildExtractedSchema(decl, absModulePath, payload)
}

func buildExtractedSchema(
	decl parser.Declaration,
	sourceURI string,
	payload extractorPayload,
) (ExtractedSchema, error) {
	if len(payload.Schema) == 0 || string(payload.Schema) == "null" {
		if len(payload.Diagnostics) > 0 {
			return ExtractedSchema{}, fmt.Errorf("%s", payload.Diagnostics[0].Message)
		}
		return ExtractedSchema{}, fmt.Errorf("extractor did not return a schema")
	}

	return ExtractedSchema{
		Schema: retriever.RetrievedSchema{
			Namespace: decl.Namespace,
			ID:        decl.ID,
			Schema:    payload.Schema,
			Adapter:   decl.Adapter,
			SourceURI: sourceURI,
		},
		Diagnostics: payload.Diagnostics,
	}, nil
}

func runExtractorCommand(
	ctx context.Context,
	dir string,
	cmdName string,
	args []string,
) (extractorPayload, error) {
	var payload extractorPayload

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		if runErr != nil {
			return extractorPayload{}, fmt.Errorf("%w: %s", runErr, strings.TrimSpace(stderr.String()))
		}
		return extractorPayload{}, fmt.Errorf("invalid extractor output: %w", err)
	}

	if runErr != nil {
		if len(payload.Diagnostics) > 0 {
			return payload, nil
		}
		return extractorPayload{}, fmt.Errorf("%w: %s", runErr, strings.TrimSpace(stderr.String()))
	}

	return payload, nil
}

func buildZodCommand(
	projectRoot string,
	modulePath string,
	exportName string,
	runner string,
) (string, []string, error) {
	if strings.TrimSpace(exportName) == "" {
		return "", nil, fmt.Errorf("zod source requires export")
	}

	sourceScript := filepath.Join(projectRoot, "typescript", "packages", "zod-extractor", "src", "index.ts")
	distScript := filepath.Join(projectRoot, "typescript", "packages", "zod-extractor", "dist", "index.js")

	switch strings.TrimSpace(runner) {
	case "", "pnpm":
		return "pnpm", []string{"exec", "tsx", sourceScript, "--module-path", modulePath, "--export-name", exportName}, nil
	case "tsx":
		return "npx", []string{"tsx", sourceScript, "--module-path", modulePath, "--export-name", exportName}, nil
	case "node":
		return "node", []string{distScript, "--module-path", modulePath, "--export-name", exportName}, nil
	default:
		return "", nil, fmt.Errorf("unsupported zod runner %q", runner)
	}
}

func parseSourceString(decl parser.Declaration) (string, error) {
	var value string
	if err := json.Unmarshal(decl.Source, &value); err != nil {
		return "", fmt.Errorf("invalid string source for %s", decl.Key())
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("source for %s must not be empty", decl.Key())
	}
	return value, nil
}

func resolveRelativeToConfig(configPath string, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(filepath.Dir(configPath), value)
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func workspaceSearchRoots(projectRoot string) []string {
	roots := []string{projectRoot}
	if executablePath, err := os.Executable(); err == nil {
		if resolvedPath, resolveErr := filepath.EvalSymlinks(executablePath); resolveErr == nil {
			executablePath = resolvedPath
		}
		roots = append(roots, filepath.Dir(executablePath))
	}

	deduped := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		cleanRoot := filepath.Clean(root)
		if _, ok := seen[cleanRoot]; ok {
			continue
		}
		seen[cleanRoot] = struct{}{}
		deduped = append(deduped, cleanRoot)
	}
	return deduped
}

func findWorkspaceRoot(relativeProbe string, searchRoots ...string) (string, error) {
	tried := make([]string, 0, len(searchRoots))
	for _, root := range searchRoots {
		if root == "" {
			continue
		}

		current := filepath.Clean(root)
		tried = append(tried, current)
		for {
			if _, err := os.Stat(filepath.Join(current, relativeProbe)); err == nil {
				return current, nil
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}

	return "", fmt.Errorf(
		"failed to locate valbridge workspace root using probe %q from %s",
		relativeProbe,
		strings.Join(tried, ", "),
	)
}
