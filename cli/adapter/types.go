package adapter

import "encoding/json"

type SourceProfile string

// Diagnostic describes a lossy mapping, bridge requirement, or unsupported feature.
type Diagnostic struct {
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	Source     string `json:"source,omitempty"`
	Target     string `json:"target,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ConvertInput is sent to the adapter CLI
type ConvertInput struct {
	Namespace     string          `json:"namespace"`
	ID            string          `json:"id"`
	VarName       string          `json:"varName"`
	Schema        json.RawMessage `json:"schema"`
	SourceProfile SourceProfile   `json:"sourceProfile,omitempty"`
}

// ConvertResult is received from the adapter CLI
type ConvertResult struct {
	Namespace         string       `json:"namespace"`
	ID                string       `json:"id"`
	VarName           string       `json:"varName"`
	Imports           []string     `json:"imports"`
	Schema            string       `json:"schema,omitempty"`
	Type              string       `json:"type,omitempty"`
	Validate          string       `json:"validate,omitempty"`
	ValidationImports []string `json:"validationImports,omitempty"`
	Diagnostics       []Diagnostic `json:"diagnostics,omitempty"`
}

// Key returns the full namespaced key like "namespace:id"
func (r ConvertResult) Key() string {
	return r.Namespace + ":" + r.ID
}
