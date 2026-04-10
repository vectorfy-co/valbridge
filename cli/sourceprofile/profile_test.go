package sourceprofile

import "testing"

func TestInferFromSourceType(t *testing.T) {
	tests := []struct {
		sourceType string
		want       Profile
	}{
		{sourceType: "pydantic", want: Pydantic},
		{sourceType: "zod", want: Zod},
		{sourceType: "file", want: JSONSchema},
		{sourceType: "url", want: JSONSchema},
		{sourceType: "json", want: JSONSchema},
	}

	for _, tt := range tests {
		if got := InferFromSourceType(tt.sourceType); got != tt.want {
			t.Fatalf("InferFromSourceType(%q) = %q, want %q", tt.sourceType, got, tt.want)
		}
	}
}
