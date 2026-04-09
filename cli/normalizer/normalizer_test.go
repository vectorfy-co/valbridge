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

	rootExtension := parsed["x-valbridge"].(map[string]any)
	if got := rootExtension["sourceProfile"]; got != string(sourceprofile.Pydantic) {
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

	rootExtension := parsed["x-valbridge"].(map[string]any)
	if got := rootExtension["sourceProfile"]; got != string(sourceprofile.Zod) {
		t.Fatalf("expected sourceProfile %q, got %v", sourceprofile.Zod, got)
	}
}
