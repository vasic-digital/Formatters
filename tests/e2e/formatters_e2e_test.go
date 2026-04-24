package e2e

import (
	"context"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/cache"
	"digital.vasic.formatters/pkg/executor"
	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFormatter is an end-to-end test formatter.
type testFormatter struct {
	*formatter.BaseFormatter
}

func newTestFormatter(name string, languages []string) *testFormatter {
	return &testFormatter{
		BaseFormatter: formatter.NewBaseFormatter(&formatter.FormatterMetadata{
			Name:            name,
			Type:            formatter.FormatterTypeNative,
			Version:         "2.0.0",
			Languages:       languages,
			SupportsStdin:   true,
			SupportsInPlace: true,
			SupportsCheck:   true,
			SupportsConfig:  true,
		}),
	}
}

func (f *testFormatter) Format(
	_ context.Context,
	req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	if req.CheckOnly {
		return &formatter.FormatResult{
			Content: req.Content,
			Changed: false,
			Success: true,
		}, nil
	}
	return &formatter.FormatResult{
		Content:          req.Content + "\n",
		Changed:          true,
		FormatterName:    f.Name(),
		FormatterVersion: f.Version(),
		Success:          true,
		Stats: &formatter.FormatStats{
			LinesTotal:   1,
			LinesChanged: 1,
		},
	}, nil
}

func (f *testFormatter) FormatBatch(
	ctx context.Context,
	reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	results := make([]*formatter.FormatResult, len(reqs))
	for i, req := range reqs {
		res, err := f.Format(ctx, req)
		if err != nil {
			return nil, err
		}
		results[i] = res
	}
	return results, nil
}

func (f *testFormatter) HealthCheck(_ context.Context) error {
	return nil
}

func TestFullFormattingWorkflowE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	goFmt := newTestFormatter("gofmt", []string{"go"})
	pyFmt := newTestFormatter("black", []string{"python"})
	jsFmt := newTestFormatter("prettier", []string{"javascript", "typescript"})

	require.NoError(t, reg.Register(goFmt))
	require.NoError(t, reg.Register(pyFmt))
	require.NoError(t, reg.Register(jsFmt))

	assert.Equal(t, 3, reg.Count())

	exec := executor.New(reg, executor.Config{
		DefaultTimeout: 10 * time.Second,
		MaxRetries:     2,
		MaxConcurrent:  5,
	})
	exec.Use(executor.ValidationMiddleware())
	exec.Use(executor.TimeoutMiddleware(5 * time.Second))

	ctx := context.Background()
	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "gofmt", result.FormatterName)
}

func TestLanguageDetectionAndFormattingE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newTestFormatter("gofmt", []string{"go"})))
	require.NoError(t, reg.Register(newTestFormatter("black", []string{"python"})))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "package main",
		FilePath: "main.go",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "gofmt", result.FormatterName)

	result, err = exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "def hello(): pass",
		FilePath: "hello.py",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "black", result.FormatterName)
}

func TestCheckOnlyModeE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newTestFormatter("gofmt", []string{"go"})))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:   "package main",
		Language:  "go",
		CheckOnly: true,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.Changed)
	assert.Equal(t, "package main", result.Content)
}

func TestCacheHitAndMissE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	c := cache.NewInMemoryCache(cache.Config{
		MaxEntries:  50,
		TTL:         5 * time.Second,
		CleanupFreq: 1 * time.Second,
	})
	defer c.Stop()

	req := &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	}
	result := &formatter.FormatResult{
		Content: "package main\n",
		Changed: true,
		Success: true,
	}

	_, found := c.Get(req)
	assert.False(t, found)

	c.Set(req, result)

	cached, found := c.Get(req)
	assert.True(t, found)
	assert.Equal(t, result.Content, cached.Content)

	c.Invalidate(req)

	_, found = c.Get(req)
	assert.False(t, found)
}

func TestRegistryMetadataWorkflowE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	goFmt := newTestFormatter("gofmt", []string{"go"})
	metadata := &formatter.FormatterMetadata{
		Name:         "gofmt",
		Type:         formatter.FormatterTypeNative,
		Version:      "2.0.0",
		Languages:    []string{"go"},
		Architecture: "binary",
		Performance:  "very_fast",
	}

	require.NoError(t, reg.RegisterWithMetadata(goFmt, metadata))

	retrieved, err := reg.GetMetadata("gofmt")
	require.NoError(t, err)
	assert.Equal(t, "binary", retrieved.Architecture)
	assert.Equal(t, "very_fast", retrieved.Performance)

	nativeFormatters := reg.ListByType(formatter.FormatterTypeNative)
	assert.Contains(t, nativeFormatters, "gofmt")
}

func TestRegistryRemoveAndReAddE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	f := newTestFormatter("gofmt", []string{"go"})
	require.NoError(t, reg.Register(f))
	assert.Equal(t, 1, reg.Count())

	require.NoError(t, reg.Remove("gofmt"))
	assert.Equal(t, 0, reg.Count())

	_, err := reg.Get("gofmt")
	assert.Error(t, err)

	require.NoError(t, reg.Register(f))
	assert.Equal(t, 1, reg.Count())

	retrieved, err := reg.Get("gofmt")
	require.NoError(t, err)
	assert.Equal(t, "gofmt", retrieved.Name())
}
