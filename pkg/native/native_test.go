package native

import (
	"context"
	"fmt"
	"os/exec"
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

// --- Tests for success paths (lines 70-88, 100, 116) ---

func TestNativeFormatter_Format_Success(t *testing.T) {
	// Skip if cat is not available
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}

	testCases := []struct {
		name            string
		content         string
		expectedContent string
		expectChanged   bool
	}{
		{
			name:            "content unchanged",
			content:         "hello world\n",
			expectedContent: "hello world\n",
			expectChanged:   false,
		},
		{
			name:            "multiline content",
			content:         "line1\nline2\nline3\n",
			expectedContent: "line1\nline2\nline3\n",
			expectChanged:   false,
		},
		{
			name:            "empty content",
			content:         "",
			expectedContent: "",
			expectChanged:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create formatter directly with stdin enabled
			metadata := &formatter.FormatterMetadata{
				Name:            "cat-formatter",
				Type:            formatter.FormatterTypeBuiltin,
				Version:         "1.0.0",
				Languages:       []string{"any"},
				SupportsStdin:   true,
				SupportsInPlace: false,
				SupportsCheck:   false,
				SupportsConfig:  false,
			}
			f := &NativeFormatter{
				BaseFormatter: formatter.NewBaseFormatter(metadata),
				binaryPath:    "cat",
				args:          []string{},
				stdinFlag:     true, // Pass content via stdin
			}

			ctx := context.Background()
			req := &formatter.FormatRequest{
				Content:  tc.content,
				Language: "any",
			}

			result, err := f.Format(ctx, req)
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.Nil(t, result.Error)
			assert.Equal(t, tc.expectedContent, result.Content)
			assert.Equal(t, tc.expectChanged, result.Changed)
			assert.Equal(t, "cat-formatter", result.FormatterName)
			assert.Equal(t, "1.0.0", result.FormatterVersion)
			assert.NotNil(t, result.Stats)
			assert.Greater(t, result.Stats.LinesTotal, 0)
		})
	}
}

func TestNativeFormatter_Format_Success_WithStats(t *testing.T) {
	// Skip if cat is not available
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}

	// Create a formatter using cat
	metadata := &formatter.FormatterMetadata{
		Name:      "cat-formatter",
		Type:      formatter.FormatterTypeBuiltin,
		Version:   "1.0.0",
		Languages: []string{"any"},
	}
	f := &NativeFormatter{
		BaseFormatter: formatter.NewBaseFormatter(metadata),
		binaryPath:    "cat",
		args:          []string{},
		stdinFlag:     true,
	}
	ctx := context.Background()

	// Content that will be echoed back unchanged
	content := "line1\nline2\nline3\nline4\nline5\n"
	req := &formatter.FormatRequest{
		Content:  content,
		Language: "any",
	}

	result, err := f.Format(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.Changed) // cat echoes unchanged
	assert.NotNil(t, result.Stats)
	assert.Equal(t, 6, result.Stats.LinesTotal) // 5 lines + 1 for trailing newline
	assert.Equal(t, 0, result.Stats.LinesChanged)
	assert.NotZero(t, result.Duration)
}

func TestNativeFormatter_HealthCheck_Success(t *testing.T) {
	// Skip if cat is not available
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}

	// cat --version works on most systems
	metadata := &formatter.FormatterMetadata{
		Name:      "cat-formatter",
		Type:      formatter.FormatterTypeBuiltin,
		Version:   "1.0.0",
		Languages: []string{"any"},
	}
	f := &NativeFormatter{
		BaseFormatter: formatter.NewBaseFormatter(metadata),
		binaryPath:    "cat",
		args:          []string{},
		stdinFlag:     false,
	}
	ctx := context.Background()

	err := f.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestNativeFormatter_HealthCheck_WithTrueBinary(t *testing.T) {
	// /bin/true always exits 0, which is perfect for testing success path
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("true binary not available")
	}

	metadata := &formatter.FormatterMetadata{
		Name:      "true-formatter",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}
	f := NewNativeFormatter(metadata, "true", nil, false)
	ctx := context.Background()

	// true --version exits 0 (just ignores the flag)
	err := f.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestNativeFormatter_FormatBatch_Success(t *testing.T) {
	// Skip if cat is not available
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}

	// Use cat formatter
	metadata := &formatter.FormatterMetadata{
		Name:      "cat-formatter",
		Type:      formatter.FormatterTypeBuiltin,
		Version:   "1.0.0",
		Languages: []string{"any"},
	}
	f := &NativeFormatter{
		BaseFormatter: formatter.NewBaseFormatter(metadata),
		binaryPath:    "cat",
		args:          []string{},
		stdinFlag:     true,
	}
	ctx := context.Background()

	reqs := []*formatter.FormatRequest{
		{
			Content:  "content one\n",
			Language: "any",
		},
		{
			Content:  "content two\n",
			Language: "any",
		},
	}

	results, err := f.FormatBatch(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Both requests unchanged by cat
	assert.True(t, results[0].Success)
	assert.False(t, results[0].Changed)
	assert.Equal(t, "content one\n", results[0].Content)

	assert.True(t, results[1].Success)
	assert.False(t, results[1].Changed)
	assert.Equal(t, "content two\n", results[1].Content)
}

func TestNativeFormatter_FormatBatch_FormatError(t *testing.T) {
	// Test the error path in FormatBatch when Format returns an error
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}

	f := NewNativeFormatter(metadata, "fake-binary", nil, false)

	// Inject a format function that returns an error
	expectedErr := fmt.Errorf("format operation failed")
	f.SetFormatFuncForTest(func(
		ctx context.Context,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return nil, expectedErr
	})

	ctx := context.Background()
	reqs := []*formatter.FormatRequest{
		{Content: "content1", Language: "test"},
		{Content: "content2", Language: "test"},
	}

	results, err := f.FormatBatch(ctx, reqs)
	require.Error(t, err)
	assert.Nil(t, results)
	assert.Equal(t, expectedErr, err)
}

func TestNativeFormatter_FormatBatch_EmptyBatch(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}

	f := NewNativeFormatter(metadata, "fake-binary", nil, false)
	ctx := context.Background()

	results, err := f.FormatBatch(ctx, []*formatter.FormatRequest{})
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestNativeFormatter_Format_WithoutStdin(t *testing.T) {
	// Test formatter without stdin flag (coverage for stdinFlag=false path)
	// This won't actually work for real formatting but tests the code path
	metadata := &formatter.FormatterMetadata{
		Name:      "test-no-stdin",
		Type:      formatter.FormatterTypeNative,
		Languages: []string{"test"},
	}

	f := NewNativeFormatter(
		metadata,
		"/nonexistent/binary",
		[]string{"--some-arg"},
		false, // stdinFlag = false
	)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "test content",
		Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Success)
	// The binary doesn't exist, so it fails
	assert.Contains(t, result.Error.Error(), "formatter execution failed")
}
