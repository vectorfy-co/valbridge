package language

import (
	"strings"
)

const (
	// ValbridgeBaseURL is the base URL for github.com/vectorfy-co/valbridge schema files.
	ValbridgeBaseURL = "https://github.com/vectorfy-co/valbridge/schemas/"
)

// SchemaEntry represents a generated schema for template data
type SchemaEntry struct {
	Namespace string // e.g., "user"
	ID        string // e.g., "TestUrl"
	VarName   string // e.g., "user_TestUrl" (safe variable name)
	Code      string // generated schema code
	Type      string // type expression
}

// OutputFileSpec defines a single output file with its template
type OutputFileSpec struct {
	Path     string // relative path within output dir, e.g., "valbridge.gen.ts" or "models/__init__.py"
	Template string // Go text/template for this file's contents
}

// Key returns the full namespaced key like "user:TestUrl"
func (s SchemaEntry) Key() string {
	return s.Namespace + ":" + s.ID
}

type Language struct {
	Name             string
	Extensions       []string // file extensions for source files (for injector)
	SchemaURL        string   // e.g., "https://github.com/vectorfy-co/valbridge/schemas/typescript.jsonc"
	SchemaExt        string   // e.g., "typescript.jsonc" - extracted from SchemaURL
	AdapterBinPrefix string   // e.g., "valbridge-" - prefix for adapter binaries
	DetectRunner     func() (cmd string, args []string, err error)
	AdapterInvoker   AdapterInvoker // resolves adapter refs to a runnable command

	// Client injection (after generation)
	BuildSchemasImport   func(importPath string) string    // build import statement for schemas
	ImportPattern        string                            // regex to find import lines
	InjectSchemasKey     func(configContent string) string // inject "schemas" into config object
	ClientFactoryPattern string                            // regex to find client factory calls e.g. createValbridgeClient({ ... })

	// Output generation
	OutputDir    string                                            // default output directory when --output not specified (e.g., ".valbridge", "_valbridge")
	OutputFile   string                                            // deprecated: use OutputFiles instead. e.g. "valbridge.gen.ts", "__init__.py"
	Template     string                                            // deprecated: use OutputFiles instead. Go text/template for output
	OutputFiles  []OutputFileSpec                                  // multiple output files with separate templates (preferred over OutputFile/Template)
	MergeImports func(imports []string) string                     // dedupe/format imports
	BuildHeader  func(outDir string, schemas []SchemaEntry) string // inserted at top
	BuildFooter  func(outDir string, schemas []SchemaEntry) string // inserted at bottom
	BuildVarName func(namespace, id string) string                 // build variable name from namespace and id

	// Parser (fallback when git not available)
	IgnoreDirs []string // directories to skip when walking

	// Compliance testing
	DetectHarnessRunner func(dir string) (cmd string, args []string, err error) // detect runner for local script files
	GetPackageName      func(string) string                                     // function to get package name from a directory
	AdapterCLIPath      func(adapterPath string) string                         // returns path to adapter CLI binary (e.g., "dist/cli.js" for TS)
	HarnessExtension    string                                                  // file extension for harness files (e.g., ".ts", ".py")
	HarnessTemplate     string                                                  // Go template for generating harness files
}

// BySchemaURL returns the language for a $schema URL like "https://github.com/vectorfy-co/valbridge/schemas/typescript.jsonc"
// Returns nil if URL doesn't match github.com/vectorfy-co/valbridge pattern
func BySchemaURL(url string) *Language {
	return defaultRegistry.BySchemaURL(url)
}

// ByName returns the language config by name
func ByName(name string) *Language {
	return defaultRegistry.ByName(name)
}

// AllIgnoreDirs returns a combined set of all ignore dirs from all languages
// Used when walking directories before language detection
func AllIgnoreDirs() map[string]bool {
	return defaultRegistry.AllIgnoreDirs()
}

// IsValbridgeURL checks if a URL is a github.com/vectorfy-co/valbridge schema URL.
func IsValbridgeURL(url string) bool {
	return strings.HasPrefix(url, ValbridgeBaseURL)
}
