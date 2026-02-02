package native

import (
	"context"
	"testing"

	"digital.vasic.formatters/pkg/formatter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNativeFormatter(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:            "test-formatter",
		Type:            formatter.FormatterTypeNative,
		Version:         "1.0.0",
		Languages:       []string{"testlang"},
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   true,
		SupportsConfig:  true,
	}

	f := NewNativeFormatter(
		metadata, "fake-binary", []string{"--quiet"}, true,
	)
	assert.NotNil(t, f)
	assert.Equal(t, "test-formatter", f.Name())
	assert.Equal(t, "1.0.0", f.Version())
	assert.Contains(t, f.Languages(), "testlang")
	assert.True(t, f.SupportsStdin())
	assert.True(t, f.SupportsInPlace())
	assert.True(t, f.SupportsCheck())
	assert.True(t, f.SupportsConfig())
}

func TestNativeFormatter_BuildArgs(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		stdinFlag     bool
		checkOnly     bool
		supportsCheck bool
		expected      []string
	}{
		{
			name:      "basic args",
			args:      []string{"--quiet"},
			stdinFlag: false,
			expected:  []string{"--quiet"},
		},
		{
			name:      "with stdin flag",
			args:      []string{"--quiet"},
			stdinFlag: true,
			expected:  []string{"--quiet", "-"},
		},
		{
			name:          "check only with support",
			args:          []string{"--quiet"},
			stdinFlag:     false,
			checkOnly:     true,
			supportsCheck: true,
			expected:      []string{"--quiet", "--check"},
		},
		{
			name:          "check only without support",
			args:          []string{"--quiet"},
			stdinFlag:     false,
			checkOnly:     true,
			supportsCheck: false,
			expected:      []string{"--quiet"},
		},
		{
			name:          "stdin and check",
			args:          []string{"--quiet"},
			stdinFlag:     true,
			checkOnly:     true,
			supportsCheck: true,
			expected:      []string{"--quiet", "-", "--check"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata := &formatter.FormatterMetadata{
				Name:          "test",
				Type:          formatter.FormatterTypeNative,
				Languages:     []string{"test"},
				SupportsCheck: tc.supportsCheck,
			}
			f := &NativeFormatter{
				BaseFormatter: formatter.NewBaseFormatter(metadata),
				binaryPath:    "fake",
				args:          tc.args,
				stdinFlag:     tc.stdinFlag,
			}

			req := &formatter.FormatRequest{
				CheckOnly: tc.checkOnly,
			}
			args := f.buildArgs(req)
			assert.Equal(t, tc.expected, args)
		})
	}
}

func TestNativeFormatter_HealthCheck_BinaryMissing(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}

	f := NewNativeFormatter(
		metadata,
		"/nonexistent/binary/that/does/not/exist",
		nil, false,
	)
	ctx := context.Background()
	err := f.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "formatter binary not available")
}

func TestNativeFormatter_Format_MissingBinary(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}

	f := NewNativeFormatter(
		metadata,
		"/nonexistent/binary/that/does/not/exist",
		[]string{"--quiet"}, true,
	)
	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "test",
		Language: "test",
	}
	result, err := f.Format(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t,
		result.Error.Error(), "formatter execution failed",
	)
}

func TestNativeFormatter_FormatBatch(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}

	f := NewNativeFormatter(
		metadata, "fake-binary", []string{"--quiet"}, true,
	)
	ctx := context.Background()
	reqs := []*formatter.FormatRequest{
		{Content: "content1", Language: "test"},
		{Content: "content2", Language: "test"},
	}

	results, err := f.FormatBatch(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		assert.False(t, result.Success)
		assert.Contains(t,
			result.Error.Error(), "formatter execution failed",
		)
	}
}

func TestComputeLineChanges(t *testing.T) {
	tests := []struct {
		name      string
		original  string
		formatted string
		expected  int
	}{
		{
			name:      "no changes",
			original:  "line1\nline2",
			formatted: "line1\nline2",
			expected:  0,
		},
		{
			name:      "one line changed",
			original:  "line1\nline2",
			formatted: "line1\nLINE2",
			expected:  1,
		},
		{
			name:      "all lines changed",
			original:  "a\nb\nc",
			formatted: "x\ny\nz",
			expected:  3,
		},
		{
			name:      "extra lines in formatted",
			original:  "a",
			formatted: "a\nb",
			expected:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := computeLineChanges(tc.original, tc.formatted)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewGoFormatter(t *testing.T) {
	f := NewGoFormatter()
	assert.Equal(t, "gofmt", f.Name())
	assert.Contains(t, f.Languages(), "go")
	assert.True(t, f.SupportsStdin())
}

func TestNewPythonFormatter(t *testing.T) {
	f := NewPythonFormatter()
	assert.Equal(t, "black", f.Name())
	assert.Contains(t, f.Languages(), "python")
	assert.True(t, f.SupportsCheck())
}

func TestNewJSFormatter(t *testing.T) {
	f := NewJSFormatter()
	assert.Equal(t, "prettier", f.Name())
	assert.Contains(t, f.Languages(), "javascript")
	assert.Contains(t, f.Languages(), "typescript")
}

func TestNewRustFormatter(t *testing.T) {
	f := NewRustFormatter()
	assert.Equal(t, "rustfmt", f.Name())
	assert.Contains(t, f.Languages(), "rust")
	assert.True(t, f.SupportsCheck())
}

func TestNewSQLFormatter(t *testing.T) {
	f := NewSQLFormatter()
	assert.Equal(t, "sqlformat", f.Name())
	assert.Contains(t, f.Languages(), "sql")
	assert.True(t, f.SupportsStdin())
}
