package registry

import (
	"context"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/formatter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFormatter is a mock formatter for testing.
type mockFormatter struct {
	formatter.BaseFormatter
	formatFunc func(ctx context.Context, req *formatter.FormatRequest) (*formatter.FormatResult, error)
	healthFunc func(ctx context.Context) error
}

func newMockFormatter(
	name string, version string, languages []string,
) *mockFormatter {
	metadata := &formatter.FormatterMetadata{
		Name:            name,
		Version:         version,
		Languages:       languages,
		Type:            formatter.FormatterTypeNative,
		SupportsStdin:   true,
		SupportsInPlace: true,
		SupportsCheck:   true,
		SupportsConfig:  true,
	}

	return &mockFormatter{
		BaseFormatter: *formatter.NewBaseFormatter(metadata),
	}
}

func (m *mockFormatter) Format(
	ctx context.Context, req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	if m.formatFunc != nil {
		return m.formatFunc(ctx, req)
	}
	return &formatter.FormatResult{
		Content:          req.Content + " formatted",
		Changed:          true,
		FormatterName:    m.Name(),
		FormatterVersion: m.Version(),
		Success:          true,
		Duration:         10 * time.Millisecond,
	}, nil
}

func (m *mockFormatter) FormatBatch(
	ctx context.Context, reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	results := make([]*formatter.FormatResult, len(reqs))
	for i, req := range reqs {
		result, err := m.Format(ctx, req)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *mockFormatter) HealthCheck(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func TestNew(t *testing.T) {
	reg := New()
	assert.NotNil(t, reg)
	assert.Equal(t, 0, reg.Count())
}

func TestRegistry_Register(t *testing.T) {
	reg := New()

	f := newMockFormatter("black", "26.1a1", []string{"python"})

	err := reg.Register(f)
	require.NoError(t, err)

	assert.Equal(t, 1, reg.Count())

	retrieved, err := reg.Get("black")
	require.NoError(t, err)
	assert.Equal(t, "black", retrieved.Name())

	pythonFormatters := reg.GetByLanguage("python")
	assert.Len(t, pythonFormatters, 1)
	assert.Equal(t, "black", pythonFormatters[0].Name())
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	reg := New()

	f := newMockFormatter("black", "26.1a1", []string{"python"})

	err := reg.Register(f)
	require.NoError(t, err)

	err = reg.Register(f)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_RegisterWithMetadata(t *testing.T) {
	reg := New()

	f := newMockFormatter("black", "26.1a1", []string{"python"})
	metadata := &formatter.FormatterMetadata{
		Name:    "black",
		Version: "26.1a1",
		Type:    formatter.FormatterTypeNative,
	}

	err := reg.RegisterWithMetadata(f, metadata)
	require.NoError(t, err)

	got, err := reg.GetMetadata("black")
	require.NoError(t, err)
	assert.Equal(t, "black", got.Name)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	reg := New()

	_, err := reg.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_GetByLanguage(t *testing.T) {
	reg := New()

	formatters := []struct {
		name    string
		version string
	}{
		{"black", "26.1a1"},
		{"ruff", "0.9.6"},
		{"autopep8", "2.0.4"},
	}

	for _, f := range formatters {
		mock := newMockFormatter(
			f.name, f.version, []string{"python"},
		)
		err := reg.Register(mock)
		require.NoError(t, err)
	}

	pythonFormatters := reg.GetByLanguage("python")
	assert.Len(t, pythonFormatters, 3)

	names := make([]string, len(pythonFormatters))
	for i, f := range pythonFormatters {
		names[i] = f.Name()
	}
	assert.Contains(t, names, "black")
	assert.Contains(t, names, "ruff")
	assert.Contains(t, names, "autopep8")
}

func TestRegistry_GetByLanguage_CaseInsensitive(t *testing.T) {
	reg := New()

	f := newMockFormatter("black", "1.0", []string{"Python"})
	err := reg.Register(f)
	require.NoError(t, err)

	result := reg.GetByLanguage("python")
	assert.Len(t, result, 1)

	result = reg.GetByLanguage("PYTHON")
	assert.Len(t, result, 1)
}

func TestRegistry_GetByLanguage_Empty(t *testing.T) {
	reg := New()

	result := reg.GetByLanguage("cobol")
	assert.Empty(t, result)
}

func TestRegistry_Remove(t *testing.T) {
	reg := New()

	f := newMockFormatter("black", "26.1a1", []string{"python"})
	err := reg.Register(f)
	require.NoError(t, err)

	err = reg.Remove("black")
	require.NoError(t, err)

	assert.Equal(t, 0, reg.Count())

	_, err = reg.Get("black")
	assert.Error(t, err)

	pythonFormatters := reg.GetByLanguage("python")
	assert.Empty(t, pythonFormatters)
}

func TestRegistry_Remove_NotFound(t *testing.T) {
	reg := New()

	err := reg.Remove("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_List(t *testing.T) {
	reg := New()

	names := []string{"black", "ruff", "gofmt"}
	for _, name := range names {
		f := newMockFormatter(name, "1.0", []string{"test"})
		err := reg.Register(f)
		require.NoError(t, err)
	}

	listed := reg.List()
	assert.Len(t, listed, 3)
	for _, name := range names {
		assert.Contains(t, listed, name)
	}
}

func TestRegistry_ListByType(t *testing.T) {
	reg := New()

	nativeF := newMockFormatter(
		"black", "26.1a1", []string{"python"},
	)
	err := reg.RegisterWithMetadata(nativeF, &formatter.FormatterMetadata{
		Name: "black",
		Type: formatter.FormatterTypeNative,
	})
	require.NoError(t, err)

	serviceF := newMockFormatter(
		"sqlfluff", "3.4.1", []string{"sql"},
	)
	err = reg.RegisterWithMetadata(serviceF, &formatter.FormatterMetadata{
		Name: "sqlfluff",
		Type: formatter.FormatterTypeService,
	})
	require.NoError(t, err)

	nativeNames := reg.ListByType(formatter.FormatterTypeNative)
	assert.Len(t, nativeNames, 1)
	assert.Contains(t, nativeNames, "black")

	serviceNames := reg.ListByType(formatter.FormatterTypeService)
	assert.Len(t, serviceNames, 1)
	assert.Contains(t, serviceNames, "sqlfluff")
}

func TestDetectLanguageFromPath(t *testing.T) {
	testCases := []struct {
		path     string
		expected string
	}{
		{"main.py", "python"},
		{"script.js", "javascript"},
		{"component.tsx", "typescript"},
		{"main.rs", "rust"},
		{"main.go", "go"},
		{"main.c", "c"},
		{"main.cpp", "cpp"},
		{"Main.java", "java"},
		{"Main.kt", "kotlin"},
		{"Main.scala", "scala"},
		{"script.sh", "bash"},
		{"config.yaml", "yaml"},
		{"data.json", "json"},
		{"config.toml", "toml"},
		{"readme.md", "markdown"},
		{"noextension", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			language := DetectLanguageFromPath(tc.path)
			assert.Equal(t, tc.expected, language)
		})
	}
}

func TestRegistry_DetectFormatter(t *testing.T) {
	reg := New()

	goFmt := newMockFormatter("gofmt", "1.24", []string{"go"})
	err := reg.Register(goFmt)
	require.NoError(t, err)

	f, err := reg.DetectFormatter("main.go")
	require.NoError(t, err)
	assert.Equal(t, "gofmt", f.Name())
}

func TestRegistry_DetectFormatter_UnknownExtension(t *testing.T) {
	reg := New()

	_, err := reg.DetectFormatter("noextension")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to detect language")
}

func TestRegistry_DetectFormatter_NoFormatter(t *testing.T) {
	reg := New()

	_, err := reg.DetectFormatter("main.go")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no formatters available")
}

func TestRegistry_HealthCheckAll(t *testing.T) {
	reg := New()

	healthyF := newMockFormatter(
		"black", "26.1a1", []string{"python"},
	)
	healthyF.healthFunc = func(ctx context.Context) error {
		return nil
	}
	err := reg.Register(healthyF)
	require.NoError(t, err)

	unhealthyF := newMockFormatter(
		"ruff", "0.9.6", []string{"python"},
	)
	unhealthyF.healthFunc = func(ctx context.Context) error {
		return assert.AnError
	}
	err = reg.Register(unhealthyF)
	require.NoError(t, err)

	ctx := context.Background()
	results := reg.HealthCheckAll(ctx)

	assert.Len(t, results, 2)
	assert.NoError(t, results["black"])
	assert.Error(t, results["ruff"])
}

func TestDefault(t *testing.T) {
	reg := Default()
	assert.NotNil(t, reg)
}

func TestRegistry_Count(t *testing.T) {
	reg := New()
	assert.Equal(t, 0, reg.Count())

	f := newMockFormatter("test", "1.0", []string{"go"})
	err := reg.Register(f)
	require.NoError(t, err)
	assert.Equal(t, 1, reg.Count())
}

func TestRegistry_GetMetadata_NotFound(t *testing.T) {
	reg := New()

	_, err := reg.GetMetadata("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
