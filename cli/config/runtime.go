package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	EnvWorkspaceRoot             = "VALBRIDGE_WORKSPACE_ROOT"
	EnvPreferWorkspace          = "VALBRIDGE_PREFER_WORKSPACE"
	EnvZodExtractorPackage      = "VALBRIDGE_ZOD_EXTRACTOR_PACKAGE"
	EnvPydanticExtractorPackage = "VALBRIDGE_PYDANTIC_EXTRACTOR_PACKAGE"
)

func WorkspaceRoot() string {
	root := strings.TrimSpace(os.Getenv(EnvWorkspaceRoot))
	if root == "" {
		return ""
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return filepath.Clean(absRoot)
}

func PreferWorkspace() bool {
	raw := strings.TrimSpace(os.Getenv(EnvPreferWorkspace))
	if raw == "" {
		return false
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return value
}

func PublishedPackageRef(envKey string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return fallback
	}
	return value
}
