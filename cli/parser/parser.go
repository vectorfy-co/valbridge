package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
	"github.com/vectorfy-co/valbridge/language"
	"github.com/vectorfy-co/valbridge/sourceprofile"
	"github.com/vectorfy-co/valbridge/ui"
)

func validateStringSource(source json.RawMessage, sourceType SourceType, namespace string, id string) error {
	var value string
	if err := json.Unmarshal(source, &value); err != nil {
		return fmt.Errorf("schema %q in namespace %q: sourceType %q requires string source", id, namespace, sourceType)
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("schema %q in namespace %q: sourceType %q requires non-empty string source", id, namespace, sourceType)
	}
	return nil
}

func validateSchemaEntry(namespace string, schema SchemaEntryRaw) error {
	if schema.SourceProfile != "" {
		if _, err := sourceprofile.Parse(schema.SourceProfile); err != nil {
			return fmt.Errorf("schema %q in namespace %q: %w", schema.ID, namespace, err)
		}
	}

	switch schema.SourceType {
	case SourceURL, SourceFile, SourcePydantic, SourceZod:
		if err := validateStringSource(schema.Source, schema.SourceType, namespace, schema.ID); err != nil {
			return err
		}
	case SourceJSON:
		if len(schema.Source) == 0 || schema.Source[0] != '{' {
			return fmt.Errorf("schema %q in namespace %q: sourceType %q requires inline JSON object source", schema.ID, namespace, schema.SourceType)
		}
	default:
		return fmt.Errorf("schema %q in namespace %q: unsupported sourceType %q", schema.ID, namespace, schema.SourceType)
	}

	if len(schema.Headers) > 0 && schema.SourceType != SourceURL {
		return fmt.Errorf("schema %q in namespace %q: headers are only allowed for sourceType \"url\"",
			schema.ID, namespace)
	}

	switch schema.SourceType {
	case SourcePydantic:
		if schema.Export != "" || schema.Runner != "" {
			return fmt.Errorf("schema %q in namespace %q: export and runner are only allowed for sourceType \"zod\"", schema.ID, namespace)
		}
	case SourceZod:
		if strings.TrimSpace(schema.Export) == "" {
			return fmt.Errorf("schema %q in namespace %q: sourceType \"zod\" requires export", schema.ID, namespace)
		}
		if schema.ModuleRoot != "" || len(schema.PythonPath) > 0 || len(schema.Requirements) > 0 || len(schema.Env) > 0 || len(schema.StubModules) > 0 {
			return fmt.Errorf("schema %q in namespace %q: moduleRoot, pythonPath, requirements, env, and stubModules are only allowed for sourceType \"pydantic\"", schema.ID, namespace)
		}
	default:
		if schema.ModuleRoot != "" || len(schema.PythonPath) > 0 || len(schema.Requirements) > 0 || len(schema.Env) > 0 || len(schema.StubModules) > 0 {
			return fmt.Errorf("schema %q in namespace %q: moduleRoot, pythonPath, requirements, env, and stubModules are only allowed for sourceType \"pydantic\"", schema.ID, namespace)
		}
		if schema.Export != "" || schema.Runner != "" {
			return fmt.Errorf("schema %q in namespace %q: export and runner are only allowed for sourceType \"zod\"", schema.ID, namespace)
		}
	}

	return nil
}

// Parse finds all valbridge config files in the project and returns merged declarations
// langFilter can be empty (auto-detect) or a language name to filter by
func Parse(ctx context.Context, projectRoot string, langFilter string) (*ParseResult, error) {
	ui.Verbosef("parsing project: root=%s, langFilter=%s", projectRoot, langFilter)

	// Find all JSON/JSONC files
	files, err := getConfigFiles(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}

	ui.Verbosef("found potential config files: count=%d", len(files))

	// Parse each file, filter by github.com/vectorfy-co/valbridge $schema
	var configs []ConfigFile
	var detectedLang *language.Language
	languageConflict := false

	for _, path := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		config, err := parseConfigFile(path)
		if err != nil {
			ui.Verbosef("skipping file (parse error): path=%s, error=%v", path, err)
			continue
		}
		if config == nil {
			// Not an valbridge config file
			continue
		}

		ui.Verbosef("found valbridge config: path=%s, namespace=%s, language=%s, schemas=%d",
			path, config.Namespace, config.Language.Name, len(config.Schemas))

		// Check language consistency
		if detectedLang == nil {
			detectedLang = config.Language
		} else if detectedLang.Name != config.Language.Name {
			languageConflict = true
		}

		configs = append(configs, *config)
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no valbridge config files found in %s", projectRoot)
	}

	// Handle language filter/conflict
	if languageConflict {
		if langFilter == "" {
			// List detected languages
			langs := make(map[string]bool)
			for _, c := range configs {
				langs[c.Language.Name] = true
			}
			var langList []string
			for l := range langs {
				langList = append(langList, l)
			}
			return nil, fmt.Errorf("multiple languages detected (%s). Use --lang to specify which one to use",
				strings.Join(langList, ", "))
		}
		// Filter configs by language
		var filtered []ConfigFile
		for _, c := range configs {
			if c.Language.Name == langFilter {
				filtered = append(filtered, c)
			}
		}
		configs = filtered
		detectedLang = language.ByName(langFilter)
		if detectedLang == nil {
			return nil, fmt.Errorf("unknown language: %s", langFilter)
		}
	}

	// Merge declarations, checking for conflicts
	declarations, err := mergeDeclarations(configs)
	if err != nil {
		return nil, err
	}

	ui.Verbosef("parsed %d configs, %d declarations", len(configs), len(declarations))

	return &ParseResult{
		Language:     detectedLang,
		Configs:      configs,
		Declarations: declarations,
	}, nil
}

// getConfigFiles returns all JSON/JSONC files in the project
func getConfigFiles(ctx context.Context, projectRoot string) ([]string, error) {
	// Try git ls-files first
	ui.Verbosef("getting config files using git in %s", projectRoot)
	args := []string{"ls-files", "--cached", "--others", "--exclude-standard", "*.json", "*.jsonc"}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		ui.Verbose("git not available, using directory walk")
		return walkDirForConfigs(ctx, projectRoot)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		ui.Verbosef("no files found via git in %s", projectRoot)
		return nil, nil
	}

	files := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			files = append(files, filepath.Join(projectRoot, line))
		}
	}
	ui.Verbosef("found files via git: count=%d", len(files))
	return files, nil
}

// walkDirForConfigs walks directory manually when git is not available
func walkDirForConfigs(ctx context.Context, projectRoot string) ([]string, error) {
	ui.Verbosef("walking directory for configs: %s", projectRoot)

	// Get all language-specific ignore dirs
	ignoreDirs := language.AllIgnoreDirs()
	// Add common dirs that should always be skipped
	ignoreDirs[".git"] = true
	ignoreDirs["vendor"] = true // Go vendor

	var files []string
	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if ignoreDirs[name] {
				ui.Verbosef("skipping directory: %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".json" || ext == ".jsonc" {
			files = append(files, path)
		}
		return nil
	})

	ui.Verbosef("directory walk complete: files=%d", len(files))
	return files, err
}

// parseConfigFile parses a single config file.
// Returns nil if the file is not a valbridge config (no matching $schema).
func parseConfigFile(path string) (*ConfigFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Standardize JSONC to JSON using hujson
	standardized, err := hujson.Standardize(content)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON/JSONC: %w", err)
	}

	// Parse JSON
	var raw ConfigFileRaw
	if err := json.Unmarshal(standardized, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check if this is a valbridge config file.
	if !language.IsValbridgeURL(raw.Schema) {
		return nil, nil
	}

	// Detect language from $schema URL
	lang := language.BySchemaURL(raw.Schema)
	if lang == nil {
		return nil, fmt.Errorf("unknown valbridge language in $schema: %s", raw.Schema)
	}

	// Derive namespace from filename or use explicit override
	namespace := raw.Namespace
	if namespace == "" {
		base := filepath.Base(path)

		legacyJSONSuffix := "." + "xschema" + ".json"
		legacyJSONCSuffix := "." + "xschema" + ".jsonc"
		if strings.HasSuffix(base, legacyJSONSuffix) || strings.HasSuffix(base, legacyJSONCSuffix) {
			return nil, fmt.Errorf("legacy config filename patterns are no longer supported: rename %q to use a .valbridge.json(c) or plain .json(c) filename", base)
		}

		// Support configs named like "user.valbridge.json". In those cases the
		// namespace should be "user", not "user.valbridge".
		switch {
		case strings.HasSuffix(base, ".valbridge.json"):
			namespace = strings.TrimSuffix(base, ".valbridge.json")
		case strings.HasSuffix(base, ".valbridge.jsonc"):
			namespace = strings.TrimSuffix(base, ".valbridge.jsonc")
		default:
			// Use filename without extension
			ext := filepath.Ext(base)
			namespace = strings.TrimSuffix(base, ext)
		}
	}

	return &ConfigFile{
		Path:      path,
		Namespace: namespace,
		Language:  lang,
		Schemas:   raw.Schemas,
	}, nil
}

// mergeDeclarations merges all config files into a flat list of declarations
// Same namespace from different files is merged; duplicate IDs within namespace are an error
func mergeDeclarations(configs []ConfigFile) ([]Declaration, error) {
	// Track seen IDs per namespace for duplicate detection
	seenIDs := make(map[string]map[string]string) // namespace -> id -> config path

	var declarations []Declaration

	for _, config := range configs {
		if seenIDs[config.Namespace] == nil {
			seenIDs[config.Namespace] = make(map[string]string)
		}

		for _, schema := range config.Schemas {
			// Check for duplicate ID in this namespace
			if existingPath, exists := seenIDs[config.Namespace][schema.ID]; exists {
				return nil, fmt.Errorf("duplicate schema ID %q in namespace %q: defined in both %s and %s",
					schema.ID, config.Namespace, existingPath, config.Path)
			}
			seenIDs[config.Namespace][schema.ID] = config.Path

			if err := validateSchemaEntry(config.Namespace, schema); err != nil {
				return nil, err
			}

			declarations = append(declarations, Declaration{
				Namespace:    config.Namespace,
				ID:           schema.ID,
				SourceType:   schema.SourceType,
				SourceProfile: inferSourceProfile(schema),
				Source:       schema.Source,
				Adapter:      schema.Adapter,
				ConfigPath:   config.Path,
				Headers:      schema.Headers,
				ModuleRoot:   schema.ModuleRoot,
				PythonPath:   schema.PythonPath,
				Requirements: schema.Requirements,
				Env:          schema.Env,
				StubModules:  schema.StubModules,
				Export:       schema.Export,
				Runner:       schema.Runner,
			})
		}
	}

	return declarations, nil
}

func inferSourceProfile(schema SchemaEntryRaw) sourceprofile.Profile {
	if schema.SourceProfile != "" {
		profile, err := sourceprofile.Parse(schema.SourceProfile)
		if err == nil {
			return profile
		}
	}

	return sourceprofile.InferFromSourceType(string(schema.SourceType))
}
