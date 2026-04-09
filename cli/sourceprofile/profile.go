package sourceprofile

import (
	"fmt"
	"strings"
)

// Profile identifies the source ecosystem a schema originated from.
type Profile string

const (
	JSONSchema Profile = "json-schema"
	Pydantic   Profile = "pydantic"
	Zod        Profile = "zod"
)

// Normalize returns the canonical string form or an empty value when unset.
func Normalize(value string) Profile {
	switch Profile(strings.TrimSpace(strings.ToLower(value))) {
	case JSONSchema:
		return JSONSchema
	case Pydantic:
		return Pydantic
	case Zod:
		return Zod
	default:
		return ""
	}
}

// Parse validates a user-provided profile.
func Parse(value string) (Profile, error) {
	profile := Normalize(value)
	if profile == "" {
		return "", fmt.Errorf("unsupported sourceProfile %q", value)
	}
	return profile, nil
}

// InferFromSourceType returns the default profile for a declaration source type.
func InferFromSourceType(sourceType string) Profile {
	switch strings.TrimSpace(strings.ToLower(sourceType)) {
	case "pydantic":
		return Pydantic
	case "zod":
		return Zod
	default:
		return JSONSchema
	}
}
