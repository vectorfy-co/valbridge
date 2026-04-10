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
