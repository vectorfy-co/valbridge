package generator

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/vectorfy-co/valbridge/adapter"
	_ "github.com/vectorfy-co/valbridge/language/langs"
	"github.com/vectorfy-co/valbridge/processor"
	"github.com/vectorfy-co/valbridge/sourceprofile"
)

func TestConvertResultKey(t *testing.T) {
	o := adapter.ConvertResult{
		Namespace: "user",
		ID:        "TestUrl",
	}

	if o.Key() != "user:TestUrl" {
		t.Errorf("expected key 'user:TestUrl', got %q", o.Key())
	}
}

func TestGenerateAdapterNotFound(t *testing.T) {
	originalDir, _ := os.Getwd()
	os.Chdir("testdata/typescript")
	defer os.Chdir(originalDir)

	adapterRef := "@vectorfyco/nonexistent-adapter"
	input := GenerateBatchInput{
		Schemas: []processor.ProcessedSchema{
			{Namespace: "test", ID: "Test", Schema: json.RawMessage(`{"type": "string"}`), Adapter: adapterRef},
		},
		AdapterRef:  adapterRef,
		Language:    "typescript",
		ProjectRoot: ".",
	}

	_, err := Generate(context.Background(), input)
	if err == nil {
		t.Error("expected error for non-existent adapter")
	}
}

func TestGenerateLegacyAdapterRefMigrationGuidance(t *testing.T) {
	input := GenerateBatchInput{
		Schemas: []processor.ProcessedSchema{
			{Namespace: "test", ID: "Test", Schema: json.RawMessage(`{"type": "string"}`), Adapter: "zod"},
		},
		AdapterRef:  "zod",
		Language:    "typescript",
		ProjectRoot: ".",
	}

	_, err := Generate(context.Background(), input)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "migration") || !strings.Contains(err.Error(), "@vectorfyco/valbridge-zod") {
		t.Fatalf("expected migration guidance for legacy adapter ref, got: %v", err)
	}
}

func TestGenerateUnsupportedLanguage(t *testing.T) {
	input := GenerateBatchInput{
		Schemas: []processor.ProcessedSchema{
			{Namespace: "test", ID: "Test", Schema: json.RawMessage(`{"type": "string"}`), Adapter: "zod"},
		},
		AdapterRef:  "zod",
		Language:    "unsupported-lang",
		ProjectRoot: ".",
	}

	_, err := Generate(context.Background(), input)
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestGenerateContextCancellation(t *testing.T) {
	originalDir, _ := os.Getwd()
	os.Chdir("testdata/typescript")
	defer os.Chdir(originalDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	adapterRef := "@vectorfyco/valbridge-zod"
	input := GenerateBatchInput{
		Schemas: []processor.ProcessedSchema{
			{Namespace: "test", ID: "Test", Schema: json.RawMessage(`{"type": "string"}`), Adapter: adapterRef},
		},
		AdapterRef:  adapterRef,
		Language:    "typescript",
		ProjectRoot: ".",
	}

	_, err := Generate(ctx, input)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestGenerateAllEmptySchemas(t *testing.T) {
	outputs, err := GenerateAll(context.Background(), []processor.ProcessedSchema{}, "typescript", ".")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	if len(outputs) != 0 {
		t.Errorf("expected 0 outputs, got %d", len(outputs))
	}
}

func TestConvertInputJSON(t *testing.T) {
	input := adapter.ConvertInput{
		Namespace:     "user",
		ID:            "Test",
		VarName:       "user_Test",
		Schema:        json.RawMessage(`{"type": "string"}`),
		SourceProfile: adapter.SourceProfile(sourceprofile.Pydantic),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded adapter.ConvertInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Namespace != "user" || decoded.ID != "Test" || decoded.VarName != "user_Test" {
		t.Errorf("round-trip failed: %+v", decoded)
	}
	if decoded.SourceProfile != adapter.SourceProfile(sourceprofile.Pydantic) {
		t.Fatalf("expected source profile to round-trip, got %q", decoded.SourceProfile)
	}
}

func TestGenerateRejectsUnsupportedAdapterSourceProfilePair(t *testing.T) {
	err := adapter.ValidateCapabilities("@vectorfyco/valbridge-zod", sourceprofile.Pydantic)
	if err != nil {
		t.Fatalf("expected zod adapter to accept canonical pydantic-origin schemas, got %v", err)
	}
}

func TestValidateSchemaCapabilitiesAcceptsSupportedZodFeatures(t *testing.T) {
	err := adapter.ValidateSchemaCapabilities(
		"@vectorfyco/valbridge-zod",
		sourceprofile.Pydantic,
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"x-valbridge": { "coercionMode": "coerce", "transforms": ["trim"] }
				},
				"count": {
					"type": "number",
					"x-valbridge": { "coercionMode": "coerce" }
				},
				"enabled": {
					"type": "boolean",
					"x-valbridge": { "coercionMode": "coerce" }
				}
			},
			"x-valbridge": { "extraMode": "forbid" }
		}`),
	)
	if err != nil {
		t.Fatalf("expected capability validation to accept supported zod features, got %v", err)
	}
}

func TestValidateSchemaCapabilitiesRejectsUnsupportedZodFeatures(t *testing.T) {
	err := adapter.ValidateSchemaCapabilities(
		"@vectorfyco/valbridge-zod",
		sourceprofile.Pydantic,
		json.RawMessage(`{
			"type": "string",
			"x-valbridge": {
				"aliasInfo": { "validationAlias": "displayName" },
				"codeStubs": [{ "kind": "preprocess", "name": "preprocess" }]
			}
		}`),
	)
	if err == nil {
		t.Fatal("expected unsupported feature rejection")
	}
	if !strings.Contains(err.Error(), "alias.validation") || !strings.Contains(err.Error(), "codeStub.preprocess") {
		t.Fatalf("expected unsupported features in error, got %v", err)
	}
}

func TestValidateSchemaCapabilitiesRejectsCompatibleFeatures(t *testing.T) {
	err := adapter.ValidateSchemaCapabilities(
		"@vectorfyco/valbridge-zod",
		sourceprofile.Pydantic,
		json.RawMessage(`{
			"anyOf": [{ "type": "string" }, { "type": "number" }],
			"x-valbridge": { "resolution": "leftToRight" }
		}`),
	)
	if err == nil {
		t.Fatal("expected compatible feature rejection")
	}
	if !strings.Contains(err.Error(), "union.resolution.leftToRight") {
		t.Fatalf("expected compatible feature in error, got %v", err)
	}
}

func TestValidateSchemaCapabilitiesRejectsNonExactPydanticScalarCoercion(t *testing.T) {
	err := adapter.ValidateSchemaCapabilities(
		"vectorfyco/valbridge-pydantic",
		sourceprofile.Zod,
		json.RawMessage(`{
			"type": "number",
			"x-valbridge": { "coercionMode": "coerce" }
		}`),
	)
	if err == nil {
		t.Fatal("expected non-exact scalar coercion rejection")
	}
	if !strings.Contains(err.Error(), "coercion.number") {
		t.Fatalf("expected coercion.number in error, got %v", err)
	}
}

func TestValidateOutputs(t *testing.T) {
	tests := []struct {
		name    string
		outputs []adapter.ConvertResult
		wantErr bool
	}{
		{
			name:    "empty outputs",
			outputs: []adapter.ConvertResult{},
			wantErr: false,
		},
		{
			name: "valid with schema only",
			outputs: []adapter.ConvertResult{
				{Namespace: "test", ID: "Test", VarName: "test_Test", Schema: "z.string()", Type: ""},
			},
			wantErr: false,
		},
		{
			name: "valid with type only",
			outputs: []adapter.ConvertResult{
				{Namespace: "test", ID: "Test", VarName: "test_Test", Schema: "", Type: "string"},
			},
			wantErr: false,
		},
		{
			name: "valid with both",
			outputs: []adapter.ConvertResult{
				{Namespace: "test", ID: "Test", VarName: "test_Test", Schema: "z.string()", Type: "string"},
			},
			wantErr: false,
		},
		{
			name: "invalid with neither",
			outputs: []adapter.ConvertResult{
				{Namespace: "test", ID: "Test", VarName: "test_Test", Schema: "", Type: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputs(tt.outputs, "test-adapter")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOutputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
