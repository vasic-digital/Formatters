package native

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"digital.vasic.formatters/pkg/formatter"
)

// FormatFunc is the signature for the format function.
// This allows injection for testing.
type FormatFunc func(
	ctx context.Context,
	req *formatter.FormatRequest,
) (*formatter.FormatResult, error)

// NativeFormatter implements a formatter using a native binary.
type NativeFormatter struct {
	*formatter.BaseFormatter
	binaryPath string
	args       []string
	stdinFlag  bool
	formatFunc FormatFunc // allows injection for testing
}

// NewNativeFormatter creates a new native binary formatter.
func NewNativeFormatter(
	metadata *formatter.FormatterMetadata,
	binaryPath string,
	args []string,
	stdinFlag bool,
) *NativeFormatter {
	return &NativeFormatter{
		BaseFormatter: formatter.NewBaseFormatter(metadata),
		binaryPath:    binaryPath,
		args:          args,
		stdinFlag:     stdinFlag,
	}
}

// Format formats code using the native binary.
func (n *NativeFormatter) Format(
	ctx context.Context, req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	start := time.Now()

	cmdArgs := n.buildArgs(req)
	cmd := exec.CommandContext(ctx, n.binaryPath, cmdArgs...)

	if n.stdinFlag {
		cmd.Stdin = strings.NewReader(req.Content)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		return &formatter.FormatResult{
			Success: false,
			Error: fmt.Errorf(
				"formatter execution failed: %w (stderr: %s)",
				err, stderr.String(),
			),
			FormatterName:    n.Name(),
			FormatterVersion: n.Version(),
			Duration:         duration,
		}, nil
	}

	formattedContent := stdout.String()
	changed := formattedContent != req.Content

	stats := &formatter.FormatStats{
		LinesTotal:   strings.Count(req.Content, "\n") + 1,
		LinesChanged: computeLineChanges(req.Content, formattedContent),
		BytesTotal:   len(req.Content),
		BytesChanged: len(formattedContent) - len(req.Content),
	}

	return &formatter.FormatResult{
		Content:          formattedContent,
		Changed:          changed,
		FormatterName:    n.Name(),
		FormatterVersion: n.Version(),
		Duration:         duration,
		Success:          true,
		Stats:            stats,
	}, nil
}

// SetFormatFuncForTest allows injecting a custom format function for testing.
// This should only be used in tests.
func (n *NativeFormatter) SetFormatFuncForTest(fn FormatFunc) {
	n.formatFunc = fn
}

// FormatBatch formats multiple requests sequentially.
func (n *NativeFormatter) FormatBatch(
	ctx context.Context, reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	results := make([]*formatter.FormatResult, len(reqs))

	formatFn := n.Format
	if n.formatFunc != nil {
		formatFn = n.formatFunc
	}

	for i, req := range reqs {
		result, err := formatFn(ctx, req)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

// HealthCheck checks if the formatter binary is available.
func (n *NativeFormatter) HealthCheck(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, n.binaryPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"formatter binary not available: %w", err,
		)
	}
	return nil
}

// buildArgs builds command arguments based on the request.
func (n *NativeFormatter) buildArgs(
	req *formatter.FormatRequest,
) []string {
	args := make([]string, len(n.args))
	copy(args, n.args)

	if n.stdinFlag {
		args = append(args, "-")
	}

	if req.CheckOnly && n.SupportsCheck() {
		args = append(args, "--check")
	}

	return args
}

// computeLineChanges calculates the number of lines changed.
func computeLineChanges(original, formatted string) int {
	if original == formatted {
		return 0
	}

	origLines := strings.Split(original, "\n")
	formattedLines := strings.Split(formatted, "\n")

	changed := 0
	maxLen := len(origLines)
	if len(formattedLines) > maxLen {
		maxLen = len(formattedLines)
	}

	for i := 0; i < maxLen; i++ {
		var origLine, formattedLine string
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(formattedLines) {
			formattedLine = formattedLines[i]
		}
		if origLine != formattedLine {
			changed++
		}
	}

	return changed
}

// --- Specific Native Formatters ---

// NewGoFormatter creates a gofmt Go formatter.
func NewGoFormatter() *NativeFormatter {
	metadata := &formatter.FormatterMetadata{
		Name:            "gofmt",
		Type:            formatter.FormatterTypeBuiltin,
		Architecture:    "binary",
		GitHubURL:       "https://github.com/golang/go",
		Version:         "go1.24",
		Languages:       []string{"go"},
		License:         "BSD-3-Clause",
		InstallMethod:   "builtin",
		BinaryPath:      "gofmt",
		ConfigFormat:    "none",
		Performance:     "fast",
		Complexity:      "easy",
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   false,
		SupportsConfig:  false,
	}

	return NewNativeFormatter(metadata, "gofmt", []string{}, true)
}

// NewPythonFormatter creates a Black Python formatter.
func NewPythonFormatter() *NativeFormatter {
	metadata := &formatter.FormatterMetadata{
		Name:            "black",
		Type:            formatter.FormatterTypeNative,
		Architecture:    "python",
		GitHubURL:       "https://github.com/psf/black",
		Version:         "26.1a1",
		Languages:       []string{"python"},
		License:         "MIT",
		InstallMethod:   "pip",
		BinaryPath:      "black",
		ConfigFormat:    "toml",
		Performance:     "medium",
		Complexity:      "easy",
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   true,
		SupportsConfig:  true,
	}

	return NewNativeFormatter(
		metadata, "black", []string{"--quiet"}, true,
	)
}

// NewJSFormatter creates a Prettier JS/TS formatter.
func NewJSFormatter() *NativeFormatter {
	metadata := &formatter.FormatterMetadata{
		Name:         "prettier",
		Type:         formatter.FormatterTypeUnified,
		Architecture: "node",
		GitHubURL:    "https://github.com/prettier/prettier",
		Version:      "3.4.2",
		Languages: []string{
			"javascript", "typescript", "json", "html",
			"css", "scss", "markdown", "yaml", "graphql",
		},
		License:         "MIT",
		InstallMethod:   "npm",
		BinaryPath:      "prettier",
		ConfigFormat:    "json",
		Performance:     "medium",
		Complexity:      "easy",
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   true,
		SupportsConfig:  true,
	}

	return NewNativeFormatter(
		metadata, "prettier",
		[]string{"--stdin-filepath", "temp.js"}, true,
	)
}

// NewRustFormatter creates a rustfmt Rust formatter.
func NewRustFormatter() *NativeFormatter {
	metadata := &formatter.FormatterMetadata{
		Name:            "rustfmt",
		Type:            formatter.FormatterTypeNative,
		Architecture:    "binary",
		GitHubURL:       "https://github.com/rust-lang/rustfmt",
		Version:         "1.8.1",
		Languages:       []string{"rust"},
		License:         "Apache 2.0",
		InstallMethod:   "cargo",
		BinaryPath:      "rustfmt",
		ConfigFormat:    "toml",
		Performance:     "fast",
		Complexity:      "easy",
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   true,
		SupportsConfig:  true,
	}

	return NewNativeFormatter(
		metadata, "rustfmt", []string{"--edition=2024"}, true,
	)
}

// NewSQLFormatter creates a basic SQL formatter using sqlformat.
func NewSQLFormatter() *NativeFormatter {
	metadata := &formatter.FormatterMetadata{
		Name:            "sqlformat",
		Type:            formatter.FormatterTypeNative,
		Architecture:    "python",
		GitHubURL:       "https://github.com/andialbrecht/sqlparse",
		Version:         "0.5.3",
		Languages:       []string{"sql"},
		License:         "BSD-3-Clause",
		InstallMethod:   "pip",
		BinaryPath:      "sqlformat",
		ConfigFormat:    "none",
		Performance:     "fast",
		Complexity:      "easy",
		SupportsStdin:   true,
		SupportsInPlace: false,
		SupportsCheck:   false,
		SupportsConfig:  false,
	}

	return NewNativeFormatter(
		metadata, "sqlformat",
		[]string{"--reindent", "--keywords", "upper"}, true,
	)
}
