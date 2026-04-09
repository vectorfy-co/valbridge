package normalizer

import (
	"encoding/json"
	"fmt"

	"github.com/vectorfy-co/valbridge/adapter"
	"github.com/vectorfy-co/valbridge/sourceprofile"
)

const (
	legacyPydanticDecimalPattern = `^(?!^[-+.]*$)[+-]?0*\d*\.?\d*$`
	portableDecimalPattern       = `^[+-]?(?:0*\d+(?:\.\d*)?|0*\.\d+)$`
)

// Result contains normalized schema output plus diagnostics emitted during lowering.
type Result struct {
	Schema      json.RawMessage
	Profile     sourceprofile.Profile
	Diagnostics []adapter.Diagnostic
	Notes       []string
}

// Normalize rewrites source-emitted schema quirks into valbridge's generation-safe subset.
func Normalize(schema json.RawMessage, profile sourceprofile.Profile) (Result, error) {
	if profile == "" {
		profile = sourceprofile.JSONSchema
	}

	var parsed any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return Result{}, fmt.Errorf("failed to parse schema for normalization: %w", err)
	}

	diagnostics := make([]adapter.Diagnostic, 0, 1)
	notes := make([]string, 0, 1)

	normalizeNode(parsed, profile, true, &diagnostics, &notes)

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal normalized schema: %w", err)
	}

	return Result{
		Schema:      encoded,
		Profile:     profile,
		Diagnostics: diagnostics,
		Notes:       notes,
	}, nil
}

func normalizeNode(
	node any,
	profile sourceprofile.Profile,
	isRoot bool,
	diagnostics *[]adapter.Diagnostic,
	notes *[]string,
) {
	switch value := node.(type) {
	case map[string]any:
		if isRoot {
			ensureRootValbridge(value, profile)
		}

		if profile == sourceprofile.Pydantic {
			rewritePydanticPattern(value, diagnostics, notes)
		}

		for _, child := range value {
			normalizeNode(child, profile, false, diagnostics, notes)
		}
	case []any:
		for _, child := range value {
			normalizeNode(child, profile, false, diagnostics, notes)
		}
	}
}

func ensureRootValbridge(node map[string]any, profile sourceprofile.Profile) {
	extension, ok := node["x-valbridge"].(map[string]any)
	if !ok {
		extension = map[string]any{}
		node["x-valbridge"] = extension
	}

	if _, ok := extension["sourceProfile"].(string); !ok {
		extension["sourceProfile"] = string(profile)
	}
}

func rewritePydanticPattern(
	node map[string]any,
	diagnostics *[]adapter.Diagnostic,
	notes *[]string,
) {
	pattern, ok := node["pattern"].(string)
	if !ok {
		return
	}
	if pattern != legacyPydanticDecimalPattern {
		return
	}

	node["pattern"] = portableDecimalPattern
	*notes = append(*notes, "rewrote portable decimal regex")
	*diagnostics = append(*diagnostics, adapter.Diagnostic{
		Severity: "info",
		Code:     "normalizer.pydantic.decimal_pattern_rewritten",
		Message:  "Rewrote a Pydantic-emitted decimal regex into a portable RE2-compatible form.",
		Path:     "$.pattern",
		Source:   string(sourceprofile.Pydantic),
		Target:   "canonical-json-schema",
	})
}
