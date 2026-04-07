package adapter

import (
	"encoding/json"
	"testing"
)

func TestConvertResultJSONRoundTripWithDiagnostics(t *testing.T) {
	original := ConvertResult{
		Namespace: "example",
		ID:        "Event",
		VarName:   "eventSchema",
		Imports:   []string{},
		Schema:    "z.object({})",
		Diagnostics: []Diagnostic{
			{
				Severity:   "warning",
				Code:       "bridge.temporal.bound",
				Path:       "$.properties.createdAt",
				Message:    "Rendered with helper-backed temporal bound validation.",
				Source:     "pydantic",
				Target:     "zod",
				Suggestion: "Install the local zod bridge helper when using this output.",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ConvertResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(decoded.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(decoded.Diagnostics))
	}
	if decoded.Diagnostics[0].Code != "bridge.temporal.bound" {
		t.Fatalf("expected diagnostic code to round-trip, got %q", decoded.Diagnostics[0].Code)
	}
}
