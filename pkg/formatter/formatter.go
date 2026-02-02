package formatter

import (
	"context"
	"time"
)

// Formatter is the universal interface for all code formatters.
type Formatter interface {
	// Identity
	Name() string        // e.g., "clang-format"
	Version() string     // e.g., "19.1.8"
	Languages() []string // e.g., ["c", "cpp", "java"]

	// Capabilities
	SupportsStdin() bool   // Can accept input via stdin
	SupportsInPlace() bool // Can format files in-place
	SupportsCheck() bool   // Can check without formatting (dry-run)
	SupportsConfig() bool  // Accepts configuration files

	// Formatting
	Format(ctx context.Context, req *FormatRequest) (*FormatResult, error)
	FormatBatch(ctx context.Context, reqs []*FormatRequest) ([]*FormatResult, error)

	// Health
	HealthCheck(ctx context.Context) error

	// Configuration
	ValidateConfig(config map[string]interface{}) error
	DefaultConfig() map[string]interface{}
}

// Options represents formatting options.
type Options struct {
	IndentSize int    // Indent size (e.g., 2, 4)
	UseTabs    bool   // Use tabs instead of spaces
	LineWidth  int    // Maximum line width
	Style      string // Style preset (e.g., "google", "pep8")
}

// FormatRequest represents a formatting request.
type FormatRequest struct {
	// Input
	Content  string // Code content
	FilePath string // Optional file path (for extension detection)
	Language string // Language override

	// Configuration
	Config     map[string]interface{} // Formatter-specific config
	LineLength int                    // Max line length (if supported)
	IndentSize int                    // Indent size
	UseTabs    bool                   // Use tabs vs spaces

	// Behavior
	CheckOnly bool          // Dry-run (check if formatted)
	Timeout   time.Duration // Max execution time

	// Context
	RequestID string // Request tracking
}

// FormatResult represents the formatting result.
type FormatResult struct {
	// Output
	Content string // Formatted content
	Changed bool   // Whether content was modified

	// Metadata
	FormatterName    string        // Formatter used
	FormatterVersion string        // Formatter version
	Duration         time.Duration // Execution time

	// Diagnostics
	Success  bool         // Overall success
	Error    error        // Error if failed
	Warnings []string     // Non-fatal warnings
	Stats    *FormatStats // Formatting statistics
}

// FormatStats provides formatting statistics.
type FormatStats struct {
	LinesTotal   int
	LinesChanged int
	BytesTotal   int
	BytesChanged int
	Violations   int // Style violations fixed
}

// Error represents a formatting error with location.
type Error struct {
	Line     int    // Line number (1-based)
	Column   int    // Column number (1-based)
	Message  string // Error description
	Severity string // "error", "warning", "info"
}

// Result is an enriched formatting result with detailed errors.
type Result struct {
	Formatted string  // Formatted code
	Changed   bool    // Whether code was modified
	Errors    []Error // Formatting errors/diagnostics
}

// FormatterType defines the formatter architecture type.
type FormatterType string

const (
	FormatterTypeNative  FormatterType = "native"  // Standalone binary
	FormatterTypeService FormatterType = "service"  // RPC/HTTP service
	FormatterTypeBuiltin FormatterType = "builtin"  // Language built-in
	FormatterTypeUnified FormatterType = "unified"  // Multi-language
)

// FormatterMetadata provides formatter metadata.
type FormatterMetadata struct {
	Name         string
	Type         FormatterType
	Architecture string // "binary", "python", "node", "jvm", etc.
	GitHubURL    string
	Version      string
	Languages    []string
	License      string

	// Installation
	InstallMethod string // "binary", "apt", "brew", "npm", "pip", etc.
	BinaryPath    string // Path to binary
	ServiceURL    string // Service endpoint (if service-based)

	// Configuration
	ConfigFormat  string // "yaml", "json", "toml", "ini", "none"
	DefaultConfig string // Path to default config file

	// Performance
	Performance string // "very_fast", "fast", "medium", "slow"
	Complexity  string // "easy", "medium", "hard"

	// Integration
	SupportsStdin   bool
	SupportsInPlace bool
	SupportsCheck   bool
	SupportsConfig  bool
}

// BaseFormatter provides common formatter functionality.
type BaseFormatter struct {
	name      string
	version   string
	languages []string
	metadata  *FormatterMetadata
}

// NewBaseFormatter creates a new base formatter.
func NewBaseFormatter(metadata *FormatterMetadata) *BaseFormatter {
	return &BaseFormatter{
		name:      metadata.Name,
		version:   metadata.Version,
		languages: metadata.Languages,
		metadata:  metadata,
	}
}

// Name returns the formatter name.
func (b *BaseFormatter) Name() string {
	return b.name
}

// Version returns the formatter version.
func (b *BaseFormatter) Version() string {
	return b.version
}

// Languages returns supported languages.
func (b *BaseFormatter) Languages() []string {
	return b.languages
}

// SupportsStdin returns whether the formatter supports stdin.
func (b *BaseFormatter) SupportsStdin() bool {
	return b.metadata.SupportsStdin
}

// SupportsInPlace returns whether the formatter supports in-place
// formatting.
func (b *BaseFormatter) SupportsInPlace() bool {
	return b.metadata.SupportsInPlace
}

// SupportsCheck returns whether the formatter supports check mode.
func (b *BaseFormatter) SupportsCheck() bool {
	return b.metadata.SupportsCheck
}

// SupportsConfig returns whether the formatter supports configuration.
func (b *BaseFormatter) SupportsConfig() bool {
	return b.metadata.SupportsConfig
}

// DefaultConfig returns the default configuration.
func (b *BaseFormatter) DefaultConfig() map[string]interface{} {
	return make(map[string]interface{})
}

// ValidateConfig validates the configuration.
func (b *BaseFormatter) ValidateConfig(
	config map[string]interface{},
) error {
	return nil
}

// Metadata returns the formatter metadata.
func (b *BaseFormatter) Metadata() *FormatterMetadata {
	return b.metadata
}
