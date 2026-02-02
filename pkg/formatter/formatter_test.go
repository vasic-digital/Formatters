package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBaseFormatter(t *testing.T) {
	metadata := &FormatterMetadata{
		Name:            "gofmt",
		Version:         "1.24.0",
		Languages:       []string{"go"},
		Type:            FormatterTypeBuiltin,
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   true,
		SupportsConfig:  false,
	}

	bf := NewBaseFormatter(metadata)

	assert.Equal(t, "gofmt", bf.Name())
	assert.Equal(t, "1.24.0", bf.Version())
	assert.Equal(t, []string{"go"}, bf.Languages())
	assert.True(t, bf.SupportsStdin())
	assert.True(t, bf.SupportsInPlace())
	assert.True(t, bf.SupportsCheck())
	assert.False(t, bf.SupportsConfig())
}

func TestBaseFormatter_Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "black",
			expected: "black",
		},
		{
			name:     "hyphenated name",
			input:    "clang-format",
			expected: "clang-format",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bf := NewBaseFormatter(&FormatterMetadata{Name: tc.input})
			assert.Equal(t, tc.expected, bf.Name())
		})
	}
}

func TestBaseFormatter_Version(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"semver", "1.2.3", "1.2.3"},
		{"prerelease", "26.1a1", "26.1a1"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bf := NewBaseFormatter(
				&FormatterMetadata{Version: tc.version},
			)
			assert.Equal(t, tc.expected, bf.Version())
		})
	}
}

func TestBaseFormatter_Languages(t *testing.T) {
	tests := []struct {
		name      string
		languages []string
	}{
		{"single language", []string{"python"}},
		{
			"multiple languages",
			[]string{"c", "cpp", "java", "javascript"},
		},
		{"nil languages", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bf := NewBaseFormatter(
				&FormatterMetadata{Languages: tc.languages},
			)
			assert.Equal(t, tc.languages, bf.Languages())
		})
	}
}

func TestBaseFormatter_SupportsStdin(t *testing.T) {
	tests := []struct {
		name     string
		supports bool
	}{
		{"supports stdin", true},
		{"no stdin", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bf := NewBaseFormatter(
				&FormatterMetadata{SupportsStdin: tc.supports},
			)
			assert.Equal(t, tc.supports, bf.SupportsStdin())
		})
	}
}

func TestBaseFormatter_SupportsInPlace(t *testing.T) {
	bf := NewBaseFormatter(
		&FormatterMetadata{SupportsInPlace: true},
	)
	assert.True(t, bf.SupportsInPlace())

	bf2 := NewBaseFormatter(
		&FormatterMetadata{SupportsInPlace: false},
	)
	assert.False(t, bf2.SupportsInPlace())
}

func TestBaseFormatter_SupportsCheck(t *testing.T) {
	bf := NewBaseFormatter(
		&FormatterMetadata{SupportsCheck: true},
	)
	assert.True(t, bf.SupportsCheck())

	bf2 := NewBaseFormatter(
		&FormatterMetadata{SupportsCheck: false},
	)
	assert.False(t, bf2.SupportsCheck())
}

func TestBaseFormatter_SupportsConfig(t *testing.T) {
	bf := NewBaseFormatter(
		&FormatterMetadata{SupportsConfig: true},
	)
	assert.True(t, bf.SupportsConfig())

	bf2 := NewBaseFormatter(
		&FormatterMetadata{SupportsConfig: false},
	)
	assert.False(t, bf2.SupportsConfig())
}

func TestBaseFormatter_DefaultConfig(t *testing.T) {
	bf := NewBaseFormatter(&FormatterMetadata{Name: "test"})
	cfg := bf.DefaultConfig()
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg)
}

func TestBaseFormatter_ValidateConfig(t *testing.T) {
	bf := NewBaseFormatter(&FormatterMetadata{Name: "test"})

	err := bf.ValidateConfig(nil)
	assert.NoError(t, err)

	err = bf.ValidateConfig(
		map[string]interface{}{"key": "value"},
	)
	assert.NoError(t, err)
}

func TestBaseFormatter_Metadata(t *testing.T) {
	metadata := &FormatterMetadata{
		Name:    "test",
		Version: "1.0.0",
		Type:    FormatterTypeNative,
	}
	bf := NewBaseFormatter(metadata)
	assert.Equal(t, metadata, bf.Metadata())
}

func TestFormatterType_Constants(t *testing.T) {
	assert.Equal(t, FormatterType("native"), FormatterTypeNative)
	assert.Equal(t, FormatterType("service"), FormatterTypeService)
	assert.Equal(t, FormatterType("builtin"), FormatterTypeBuiltin)
	assert.Equal(t, FormatterType("unified"), FormatterTypeUnified)
}

func TestOptions_Fields(t *testing.T) {
	opts := Options{
		IndentSize: 4,
		UseTabs:    false,
		LineWidth:  80,
		Style:      "google",
	}
	assert.Equal(t, 4, opts.IndentSize)
	assert.False(t, opts.UseTabs)
	assert.Equal(t, 80, opts.LineWidth)
	assert.Equal(t, "google", opts.Style)
}

func TestFormatRequest_Fields(t *testing.T) {
	req := &FormatRequest{
		Content:    "x = 1",
		FilePath:   "test.py",
		Language:   "python",
		Config:     map[string]interface{}{"indent": 4},
		LineLength: 88,
		IndentSize: 4,
		UseTabs:    false,
		CheckOnly:  true,
		RequestID:  "req-456",
	}

	assert.Equal(t, "x = 1", req.Content)
	assert.Equal(t, "test.py", req.FilePath)
	assert.Equal(t, "python", req.Language)
	assert.Equal(t, 88, req.LineLength)
	assert.Equal(t, 4, req.IndentSize)
	assert.False(t, req.UseTabs)
	assert.True(t, req.CheckOnly)
	assert.Equal(t, "req-456", req.RequestID)
}

func TestFormatResult_Fields(t *testing.T) {
	stats := &FormatStats{
		LinesTotal:   100,
		LinesChanged: 5,
		BytesTotal:   2000,
		BytesChanged: 100,
		Violations:   3,
	}

	result := &FormatResult{
		Content:          "formatted code",
		Changed:          true,
		FormatterName:    "black",
		FormatterVersion: "26.1a1",
		Success:          true,
		Warnings:         []string{"trailing whitespace"},
		Stats:            stats,
	}

	assert.Equal(t, "formatted code", result.Content)
	assert.True(t, result.Changed)
	assert.Equal(t, "black", result.FormatterName)
	assert.True(t, result.Success)
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, 100, result.Stats.LinesTotal)
	assert.Equal(t, 5, result.Stats.LinesChanged)
	assert.Equal(t, 3, result.Stats.Violations)
}

func TestError_Fields(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		line     int
		col      int
		msg      string
		severity string
	}{
		{
			name:     "syntax error",
			err:      Error{Line: 10, Column: 5, Message: "unexpected token", Severity: "error"},
			line:     10,
			col:      5,
			msg:      "unexpected token",
			severity: "error",
		},
		{
			name:     "warning",
			err:      Error{Line: 1, Column: 1, Message: "trailing whitespace", Severity: "warning"},
			line:     1,
			col:      1,
			msg:      "trailing whitespace",
			severity: "warning",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.line, tc.err.Line)
			assert.Equal(t, tc.col, tc.err.Column)
			assert.Equal(t, tc.msg, tc.err.Message)
			assert.Equal(t, tc.severity, tc.err.Severity)
		})
	}
}

func TestResult_Fields(t *testing.T) {
	r := Result{
		Formatted: "formatted code",
		Changed:   true,
		Errors: []Error{
			{Line: 1, Column: 1, Message: "fixed", Severity: "info"},
		},
	}
	assert.Equal(t, "formatted code", r.Formatted)
	assert.True(t, r.Changed)
	assert.Len(t, r.Errors, 1)
}

func TestFormatterMetadata_Fields(t *testing.T) {
	metadata := &FormatterMetadata{
		Name:          "black",
		Type:          FormatterTypeNative,
		Architecture:  "python",
		GitHubURL:     "https://github.com/psf/black",
		Version:       "26.1a1",
		Languages:     []string{"python"},
		License:       "MIT",
		InstallMethod: "pip",
		BinaryPath:    "/usr/bin/black",
		ConfigFormat:  "toml",
		Performance:   "fast",
		Complexity:    "easy",
	}

	assert.Equal(t, "black", metadata.Name)
	assert.Equal(t, FormatterTypeNative, metadata.Type)
	assert.Equal(t, "python", metadata.Architecture)
	assert.Equal(t, "MIT", metadata.License)
	assert.Equal(t, "pip", metadata.InstallMethod)
	assert.Equal(t, "toml", metadata.ConfigFormat)
	assert.Equal(t, "fast", metadata.Performance)
}
