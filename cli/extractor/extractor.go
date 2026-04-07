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
	"github.com/vectorfy-co/valbridge/config"
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

type commandCandidate struct {
	Label string
	Dir   string
	Cmd   string
	Args  []string
}

const (
	publishedZodExtractorPackage      = "@vectorfyco/valbridge-zod-extractor"
	publishedPydanticExtractorPackage = "valbridge-pydantic-extractor"
)

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

	args := []string{target}
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

	candidates, err := buildPydanticExtractorCandidates(projectRoot, args)
	if err != nil {
		return ExtractedSchema{}, err
	}
	payload, err := runExtractorCandidates(ctx, candidates)
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
	candidates, err := buildZodExtractorCandidates(projectRoot, absModulePath, decl.Export, decl.Runner)
	if err != nil {
		return ExtractedSchema{}, err
	}
	payload, err := runExtractorCandidates(ctx, candidates)
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

func runExtractorCandidates(ctx context.Context, candidates []commandCandidate) (extractorPayload, error) {
	if len(candidates) == 0 {
		return extractorPayload{}, fmt.Errorf("no extractor command candidates available")
	}

	var errors []string
	for _, candidate := range candidates {
		payload, err := runExtractorCommand(ctx, candidate.Dir, candidate.Cmd, candidate.Args)
		if err == nil {
			return payload, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", candidate.Label, err))
	}

	return extractorPayload{}, fmt.Errorf("%s", strings.Join(errors, "\n"))
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

func buildPydanticExtractorCandidates(projectRoot string, args []string) ([]commandCandidate, error) {
	var candidates []commandCandidate
	var resolutionErrors []string
	addWorkspaceFirst := config.WorkspaceRoot() != "" || config.PreferWorkspace()

	if addWorkspaceFirst {
		candidate, err := buildWorkspacePydanticExtractorCandidate(projectRoot, args)
		if err != nil {
			resolutionErrors = append(resolutionErrors, err.Error())
		} else {
			candidates = append(candidates, candidate)
		}
	}

	if candidate, err := buildPublishedPydanticExtractorCandidate(args); err == nil {
		candidates = append(candidates, candidate)
	} else {
		resolutionErrors = append(resolutionErrors, err.Error())
	}

	if !addWorkspaceFirst {
		candidate, err := buildWorkspacePydanticExtractorCandidate(projectRoot, args)
		if err == nil {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("%s", strings.Join(resolutionErrors, "\n"))
	}

	return candidates, nil
}

func buildWorkspacePydanticExtractorCandidate(projectRoot string, args []string) (commandCandidate, error) {
	workspaceRoot, err := resolveWorkspaceRoot(
		projectRoot,
		filepath.Join("python", "packages", "pydantic-extractor"),
	)
	if err != nil {
		return commandCandidate{}, err
	}

	return commandCandidate{
		Label: "workspace-python-extractor",
		Dir:   filepath.Join(workspaceRoot, "python"),
		Cmd:   "uv",
		Args:  append([]string{"run", publishedPydanticExtractorPackage}, args...),
	}, nil
}

func buildPublishedPydanticExtractorCandidate(args []string) (commandCandidate, error) {
	packageRef := config.PublishedPackageRef(
		config.EnvPydanticExtractorPackage,
		publishedPydanticExtractorPackage,
	)

	switch {
	case commandExists("uvx"):
		return commandCandidate{
			Label: "published-python-extractor-uvx",
			Cmd:   "uvx",
			Args:  append([]string{packageRef}, args...),
		}, nil
	case commandExists("uv"):
		return commandCandidate{
			Label: "published-python-extractor-uv-tool-run",
			Cmd:   "uv",
			Args:  append([]string{"tool", "run", packageRef}, args...),
		}, nil
	case commandExists("pipx"):
		return commandCandidate{
			Label: "published-python-extractor-pipx-run",
			Cmd:   "pipx",
			Args:  append([]string{"run", packageRef}, args...),
		}, nil
	default:
		return commandCandidate{}, fmt.Errorf(
			"no published python extractor runner available for %q; install uv/uvx or pipx",
			packageRef,
		)
	}
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

func buildZodExtractorCandidates(
	projectRoot string,
	modulePath string,
	exportName string,
	runner string,
) ([]commandCandidate, error) {
	var candidates []commandCandidate
	var resolutionErrors []string
	addWorkspaceFirst := config.WorkspaceRoot() != "" || config.PreferWorkspace()

	if addWorkspaceFirst {
		candidate, err := buildWorkspaceZodExtractorCandidate(projectRoot, modulePath, exportName, runner)
		if err != nil {
			resolutionErrors = append(resolutionErrors, err.Error())
		} else {
			candidates = append(candidates, candidate)
		}
	}

	if candidate, err := buildPublishedZodExtractorCandidate(modulePath, exportName); err == nil {
		candidates = append(candidates, candidate)
	} else {
		resolutionErrors = append(resolutionErrors, err.Error())
	}

	if !addWorkspaceFirst {
		candidate, err := buildWorkspaceZodExtractorCandidate(projectRoot, modulePath, exportName, runner)
		if err == nil {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("%s", strings.Join(resolutionErrors, "\n"))
	}

	return candidates, nil
}

func buildWorkspaceZodExtractorCandidate(
	projectRoot string,
	modulePath string,
	exportName string,
	runner string,
) (commandCandidate, error) {
	workspaceRoot, err := resolveWorkspaceRoot(
		projectRoot,
		filepath.Join("typescript", "packages", "zod-extractor"),
	)
	if err != nil {
		return commandCandidate{}, err
	}

	cmdName, args, err := buildZodCommand(workspaceRoot, modulePath, exportName, runner)
	if err != nil {
		return commandCandidate{}, err
	}

	return commandCandidate{
		Label: "workspace-zod-extractor",
		Dir:   filepath.Join(workspaceRoot, "typescript"),
		Cmd:   cmdName,
		Args:  args,
	}, nil
}

func buildPublishedZodExtractorCandidate(
	modulePath string,
	exportName string,
) (commandCandidate, error) {
	packageRef := config.PublishedPackageRef(
		config.EnvZodExtractorPackage,
		publishedZodExtractorPackage,
	)
	args := []string{"--module-path", modulePath, "--export-name", exportName}

	switch {
	case commandExists("pnpm"):
		return commandCandidate{
			Label: "published-zod-extractor-pnpm-dlx",
			Cmd:   "pnpm",
			Args:  append([]string{"dlx", packageRef}, args...),
		}, nil
	case commandExists("npx"):
		return commandCandidate{
			Label: "published-zod-extractor-npx",
			Cmd:   "npx",
			Args:  append([]string{"-y", packageRef}, args...),
		}, nil
	default:
		return commandCandidate{}, fmt.Errorf(
			"no published typescript extractor runner available for %q; install pnpm or npm/npx",
			packageRef,
		)
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

func resolveWorkspaceRoot(projectRoot string, relativeProbe string) (string, error) {
	if explicitRoot := config.WorkspaceRoot(); explicitRoot != "" {
		if _, err := os.Stat(filepath.Join(explicitRoot, relativeProbe)); err == nil {
			return explicitRoot, nil
		}
		return "", fmt.Errorf(
			"explicit workspace root %q does not contain %q",
			explicitRoot,
			relativeProbe,
		)
	}

	return findWorkspaceRoot(relativeProbe, workspaceSearchRoots(projectRoot)...)
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

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
