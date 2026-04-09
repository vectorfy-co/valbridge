package adapter

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/vectorfy-co/valbridge/sourceprofile"
)

type SupportLevel string

const (
	SupportExact        SupportLevel = "exact"
	SupportCompatible   SupportLevel = "compatible"
	SupportMetadataOnly SupportLevel = "metadata-only"
	SupportUnsupported  SupportLevel = "unsupported"
)

type FeatureSupport struct {
	Level SupportLevel
}

// Capabilities describes the contract classes an adapter supports.
type Capabilities struct {
	AdapterRef              string
	SupportsCanonicalIR     bool
	SupportedSourceProfiles map[sourceprofile.Profile]bool
	FeatureSupport          map[string]FeatureSupport
}

var knownCapabilities = map[string]Capabilities{
	"@vectorfyco/valbridge-zod": {
		AdapterRef:          "@vectorfyco/valbridge-zod",
		SupportsCanonicalIR: true,
		SupportedSourceProfiles: map[sourceprofile.Profile]bool{
			sourceprofile.JSONSchema: true,
			sourceprofile.Pydantic:   true,
			sourceprofile.Zod:        true,
		},
		FeatureSupport: map[string]FeatureSupport{
			"coercion.string":              {Level: SupportExact},
			"coercion.number":              {Level: SupportExact},
			"coercion.boolean":             {Level: SupportExact},
			"extra.allow":                  {Level: SupportExact},
			"extra.ignore":                 {Level: SupportExact},
			"extra.forbid":                 {Level: SupportExact},
			"union.discriminator":          {Level: SupportExact},
			"union.resolution.leftToRight": {Level: SupportCompatible},
			"union.resolution.smart":       {Level: SupportUnsupported},
			"union.resolution.allErrors":   {Level: SupportUnsupported},
			"transform.trim":               {Level: SupportExact},
			"transform.toLowerCase":        {Level: SupportExact},
			"transform.toUpperCase":        {Level: SupportExact},
			"default.default":              {Level: SupportExact},
			"default.prefault":             {Level: SupportExact},
			"default.factory":              {Level: SupportUnsupported},
			"alias.validation":             {Level: SupportUnsupported},
			"alias.serialization":          {Level: SupportUnsupported},
			"alias.path":                   {Level: SupportUnsupported},
			"alias.choices":                {Level: SupportUnsupported},
			"codeStub.transform":           {Level: SupportUnsupported},
			"codeStub.preprocess":          {Level: SupportUnsupported},
			"codeStub.codec":               {Level: SupportUnsupported},
			"codeStub.validator":           {Level: SupportUnsupported},
			"codeStub.serializer":          {Level: SupportUnsupported},
			"codeStub.modelValidator":      {Level: SupportUnsupported},
		},
	},
	"vectorfyco/valbridge-pydantic": {
		AdapterRef:          "vectorfyco/valbridge-pydantic",
		SupportsCanonicalIR: true,
		SupportedSourceProfiles: map[sourceprofile.Profile]bool{
			sourceprofile.JSONSchema: true,
			sourceprofile.Pydantic:   true,
			sourceprofile.Zod:        true,
		},
		FeatureSupport: map[string]FeatureSupport{
			"coercion.string":              {Level: SupportCompatible},
			"coercion.number":              {Level: SupportCompatible},
			"coercion.boolean":             {Level: SupportExact},
			"extra.allow":                  {Level: SupportExact},
			"extra.ignore":                 {Level: SupportExact},
			"extra.forbid":                 {Level: SupportExact},
			"union.discriminator":          {Level: SupportExact},
			"union.resolution.leftToRight": {Level: SupportExact},
			"union.resolution.smart":       {Level: SupportCompatible},
			"union.resolution.allErrors":   {Level: SupportUnsupported},
			"transform.trim":               {Level: SupportExact},
			"transform.toLowerCase":        {Level: SupportExact},
			"transform.toUpperCase":        {Level: SupportExact},
			"default.default":              {Level: SupportExact},
			"default.prefault":             {Level: SupportUnsupported},
			"default.factory":              {Level: SupportUnsupported},
			"alias.validation":             {Level: SupportExact},
			"alias.serialization":          {Level: SupportExact},
			"alias.path":                   {Level: SupportUnsupported},
			"alias.choices":                {Level: SupportUnsupported},
			"codeStub.transform":           {Level: SupportUnsupported},
			"codeStub.preprocess":          {Level: SupportUnsupported},
			"codeStub.codec":               {Level: SupportUnsupported},
			"codeStub.validator":           {Level: SupportUnsupported},
			"codeStub.serializer":          {Level: SupportUnsupported},
			"codeStub.modelValidator":      {Level: SupportUnsupported},
		},
	},
}

func LookupCapabilities(adapterRef string) (Capabilities, bool) {
	caps, ok := knownCapabilities[adapterRef]
	return caps, ok
}

func ValidateCapabilities(adapterRef string, profile sourceprofile.Profile) error {
	caps, ok := LookupCapabilities(adapterRef)
	if !ok {
		return nil
	}
	if !caps.SupportsCanonicalIR {
		return fmt.Errorf("adapter %s does not support canonical valbridge IR", adapterRef)
	}
	if profile != "" && !caps.SupportedSourceProfiles[profile] {
		return fmt.Errorf("adapter %s does not support source profile %q", adapterRef, profile)
	}
	return nil
}

func ValidateSchemaCapabilities(adapterRef string, profile sourceprofile.Profile, rawSchema json.RawMessage) error {
	if err := ValidateCapabilities(adapterRef, profile); err != nil {
		return err
	}

	caps, ok := LookupCapabilities(adapterRef)
	if !ok || len(rawSchema) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(rawSchema, &decoded); err != nil {
		return fmt.Errorf("invalid schema payload for capability analysis: %w", err)
	}

	required := collectRequiredFeatures(decoded)
	if len(required) == 0 {
		return nil
	}

	nonExact := make([]string, 0)
	for _, feature := range required {
		support, ok := caps.FeatureSupport[feature]
		if !ok || support.Level != SupportExact {
			nonExact = append(nonExact, feature)
		}
	}

	if len(nonExact) == 0 {
		return nil
	}

	sort.Strings(nonExact)
	return fmt.Errorf(
		"adapter %s cannot preserve schema features exactly: %v",
		adapterRef,
		nonExact,
	)
}

func collectRequiredFeatures(schema any) []string {
	features := map[string]struct{}{}
	walkSchema(schema, features)
	result := make([]string, 0, len(features))
	for feature := range features {
		result = append(result, feature)
	}
	sort.Strings(result)
	return result
}

func walkSchema(schema any, features map[string]struct{}) {
	switch value := schema.(type) {
	case map[string]any:
		collectExtensionFeatures(value, features)

		for _, key := range []string{
			"$defs", "definitions", "properties", "patternProperties",
			"dependentSchemas", "dependencies",
		} {
			walkSchemaMap(value[key], features)
		}
		for _, key := range []string{
			"additionalProperties", "propertyNames", "items", "additionalItems",
			"contains", "not", "if", "then", "else", "unevaluatedItems",
			"unevaluatedProperties",
		} {
			walkSchema(value[key], features)
		}
		for _, key := range []string{"anyOf", "oneOf", "allOf", "prefixItems"} {
			walkSchema(value[key], features)
		}
	case []any:
		for _, item := range value {
			walkSchema(item, features)
		}
	}
}

func walkSchemaMap(value any, features map[string]struct{}) {
	mapping, ok := value.(map[string]any)
	if !ok {
		return
	}
	for _, child := range mapping {
		walkSchema(child, features)
	}
}

func collectExtensionFeatures(schema map[string]any, features map[string]struct{}) {
	extension, ok := schema["x-valbridge"].(map[string]any)
	if !ok {
		return
	}

	if coercionMode, ok := extension["coercionMode"].(string); ok && coercionMode == "coerce" {
		switch schemaType(schema) {
		case "string":
			features["coercion.string"] = struct{}{}
		case "number", "integer":
			features["coercion.number"] = struct{}{}
		case "boolean":
			features["coercion.boolean"] = struct{}{}
		}
	}

	if extraMode, ok := extension["extraMode"].(string); ok {
		features["extra."+extraMode] = struct{}{}
	}
	if _, ok := extension["discriminator"].(string); ok {
		features["union.discriminator"] = struct{}{}
	}
	if resolution, ok := extension["resolution"].(string); ok {
		features["union.resolution."+resolution] = struct{}{}
	}
	if defaultBehavior, ok := extension["defaultBehavior"].(map[string]any); ok {
		if kind, ok := defaultBehavior["kind"].(string); ok {
			features["default."+kind] = struct{}{}
		}
	}
	if aliasInfo, ok := extension["aliasInfo"].(map[string]any); ok {
		if _, ok := aliasInfo["validationAlias"].(string); ok {
			features["alias.validation"] = struct{}{}
		}
		if _, ok := aliasInfo["serializationAlias"].(string); ok {
			features["alias.serialization"] = struct{}{}
		}
		if aliasPath, ok := aliasInfo["aliasPath"].([]any); ok && len(aliasPath) > 0 {
			features["alias.path"] = struct{}{}
		}
	}
	if transforms, ok := extension["transforms"].([]any); ok {
		for _, transform := range transforms {
			if kind, ok := extractKind(transform); ok {
				features["transform."+kind] = struct{}{}
			}
		}
	}
	if codeStubs, ok := extension["codeStubs"].([]any); ok {
		for _, stub := range codeStubs {
			if kind, ok := extractKind(stub); ok {
				features["codeStub."+kind] = struct{}{}
			}
		}
	}
	if registryMeta, ok := extension["registryMeta"].(map[string]any); ok {
		if _, ok := registryMeta["validationAliasChoices"]; ok {
			features["alias.choices"] = struct{}{}
		}
	}
}

func schemaType(schema map[string]any) string {
	switch value := schema["type"].(type) {
	case string:
		return value
	case []any:
		for _, candidate := range value {
			if kind, ok := candidate.(string); ok && kind != "null" {
				return kind
			}
		}
	}
	return ""
}

func extractKind(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case map[string]any:
		kind, ok := typed["kind"].(string)
		return kind, ok
	default:
		return "", false
	}
}
