package normalizer

import (
	"encoding/json"
	"testing"

	"github.com/vectorfy-co/valbridge/sourceprofile"
)

func TestNormalize_PydanticDecimalPatternRewrite(t *testing.T) {
	input := json.RawMessage(`{
		"type": "string",
		"pattern": "^(?!^[-+.]*$)[+-]?0*\\d*\\.?\\d*$"
	}`)

	result, err := Normalize(input, sourceprofile.Pydantic)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result.Schema, &parsed); err != nil {
		t.Fatalf("failed to parse normalized schema: %v", err)
	}

	if got := parsed["pattern"]; got != portableDecimalPattern {
		t.Fatalf("expected portable pattern, got %v", got)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %d", len(result.Diagnostics))
	}
	if got := result.Diagnostics[0].Path; got != "$.pattern" {
		t.Fatalf("expected root diagnostic path, got %q", got)
	}

	rootExtensionValue, ok := parsed["x-valbridge"]
	if !ok {
		t.Fatal("expected x-valbridge extension to be present")
	}
	rootExtension, ok := rootExtensionValue.(map[string]any)
	if !ok {
		t.Fatalf("expected x-valbridge extension to be an object, got %T", rootExtensionValue)
	}
	sourceProfileValue, ok := rootExtension["sourceProfile"]
	if !ok {
		t.Fatal("expected sourceProfile to be present")
	}
	sourceProfile, ok := sourceProfileValue.(string)
	if !ok {
		t.Fatalf("expected sourceProfile to be a string, got %T", sourceProfileValue)
	}
	if got := sourceProfile; got != string(sourceprofile.Pydantic) {
		t.Fatalf("expected sourceProfile %q, got %v", sourceprofile.Pydantic, got)
	}
}

func TestNormalize_ZodStampsSourceProfile(t *testing.T) {
	input := json.RawMessage(`{"type":"string"}`)

	result, err := Normalize(input, sourceprofile.Zod)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result.Schema, &parsed); err != nil {
		t.Fatalf("failed to parse normalized schema: %v", err)
	}

	rootExtensionValue, ok := parsed["x-valbridge"]
	if !ok {
		t.Fatal("expected x-valbridge extension to be present")
	}
	rootExtension, ok := rootExtensionValue.(map[string]any)
	if !ok {
		t.Fatalf("expected x-valbridge extension to be an object, got %T", rootExtensionValue)
	}
	sourceProfileValue, ok := rootExtension["sourceProfile"]
	if !ok {
		t.Fatal("expected sourceProfile to be present")
	}
	sourceProfile, ok := sourceProfileValue.(string)
	if !ok {
		t.Fatalf("expected sourceProfile to be a string, got %T", sourceProfileValue)
	}
	if got := sourceProfile; got != string(sourceprofile.Zod) {
		t.Fatalf("expected sourceProfile %q, got %v", sourceprofile.Zod, got)
	}
}

func TestNormalize_PydanticDecimalPatternRewriteTracksNestedPath(t *testing.T) {
	input := json.RawMessage(`{
		"type": "object",
		"properties": {
			"amount": {
				"type": "string",
				"pattern": "^(?!^[-+.]*$)[+-]?0*\\d*\\.?\\d*$"
			}
		}
	}`)

	result, err := Normalize(input, sourceprofile.Pydantic)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %d", len(result.Diagnostics))
	}
	if got := result.Diagnostics[0].Path; got != "$.properties.amount.pattern" {
		t.Fatalf("expected nested diagnostic path, got %q", got)
	}
}

func TestNormalize_PydanticPatternRewriteSkipsNonSchemaMetadata(t *testing.T) {
	input := json.RawMessage(`{
		"type": "object",
		"default": {
			"pattern": "^(?!^[-+.]*$)[+-]?0*\\d*\\.?\\d*$"
		},
		"examples": [
			{
				"pattern": "^(?!^[-+.]*$)[+-]?0*\\d*\\.?\\d*$"
			}
		],
		"x-valbridge": {
			"pattern": "^(?!^[-+.]*$)[+-]?0*\\d*\\.?\\d*$"
		},
		"properties": {
			"amount": {
				"type": "string",
				"pattern": "^(?!^[-+.]*$)[+-]?0*\\d*\\.?\\d*$"
			}
		}
	}`)

	result, err := Normalize(input, sourceprofile.Pydantic)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result.Schema, &parsed); err != nil {
		t.Fatalf("failed to parse normalized schema: %v", err)
	}

	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected only schema pattern rewrite diagnostic, got %#v", result.Diagnostics)
	}
	if got := result.Diagnostics[0].Path; got != "$.properties.amount.pattern" {
		t.Fatalf("expected schema pattern diagnostic path, got %q", got)
	}

	defaultValue, ok := parsed["default"].(map[string]any)
	if !ok {
		t.Fatalf("expected default object, got %T", parsed["default"])
	}
	if got := defaultValue["pattern"]; got != legacyPydanticDecimalPattern {
		t.Fatalf("expected default pattern to remain unchanged, got %v", got)
	}

	examplesValue, ok := parsed["examples"].([]any)
	if !ok || len(examplesValue) != 1 {
		t.Fatalf("expected one example object, got %#v", parsed["examples"])
	}
	exampleObject, ok := examplesValue[0].(map[string]any)
	if !ok {
		t.Fatalf("expected example object, got %T", examplesValue[0])
	}
	if got := exampleObject["pattern"]; got != legacyPydanticDecimalPattern {
		t.Fatalf("expected example pattern to remain unchanged, got %v", got)
	}

	rootExtension, ok := parsed["x-valbridge"].(map[string]any)
	if !ok {
		t.Fatalf("expected x-valbridge object, got %T", parsed["x-valbridge"])
	}
	if got := rootExtension["pattern"]; got != legacyPydanticDecimalPattern {
		t.Fatalf("expected x-valbridge pattern to remain unchanged, got %v", got)
	}

	properties, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object, got %T", parsed["properties"])
	}
	amount, ok := properties["amount"].(map[string]any)
	if !ok {
		t.Fatalf("expected amount schema object, got %T", properties["amount"])
	}
	if got := amount["pattern"]; got != portableDecimalPattern {
		t.Fatalf("expected schema pattern rewrite, got %v", got)
	}
}
