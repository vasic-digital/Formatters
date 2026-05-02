package integration

// BLUFF-VIOLATION: R-12 — This integration test uses mockFormatter implementations.
// Mocks are permitted ONLY in Unit tests per Constitution §6 / R-12.
// Remediation: Register real formatters (gofmt, black) via container or local binary.
// Tracked in: docs/research/chapters/MVP/05_Response/anti_bluff_audit_2026-05-02.md

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

// mockFormatter is a test formatter for integration testing.
type mockFormatter struct {
	*formatter.BaseFormatter
}

func newMockFormatter(name string, languages []string) *mockFormatter {
	return &mockFormatter{
		BaseFormatter: formatter.NewBaseFormatter(&formatter.FormatterMetadata{
			Name:          name,
			Type:          formatter.FormatterTypeNative,
			Version:       "1.0.0",
			Languages:     languages,
			SupportsStdin: true,
		}),
	}
}

func (m *mockFormatter) Format(
	_ context.Context,
	req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	return &formatter.FormatResult{
		Content:          req.Content + "\n",
		Changed:          true,
		FormatterName:    m.Name(),
		FormatterVersion: m.Version(),
		Success:          true,
	}, nil
}

func (m *mockFormatter) FormatBatch(
	ctx context.Context,
	reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	results := make([]*formatter.FormatResult, len(reqs))
	for i, req := range reqs {
		res, err := m.Format(ctx, req)
		if err != nil {
			return nil, err
		}
		results[i] = res
	}
	return results, nil
}

func (m *mockFormatter) HealthCheck(_ context.Context) error {
	return nil
}

func TestRegistryExecutorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	goFmt := newMockFormatter("gofmt", []string{"go"})
	pyFmt := newMockFormatter("black", []string{"python"})

	require.NoError(t, reg.Register(goFmt))
	require.NoError(t, reg.Register(pyFmt))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	req := &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	}
	result, err := exec.Execute(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "gofmt", result.FormatterName)
	assert.Equal(t, "package main\n", result.Content)
}

func TestRegistryExecutorWithMiddlewareIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	goFmt := newMockFormatter("gofmt", []string{"go"})
	require.NoError(t, reg.Register(goFmt))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	exec.Use(executor.ValidationMiddleware())
	exec.Use(executor.TimeoutMiddleware(5 * time.Second))

	ctx := context.Background()

	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "func main() {}",
		Language: "go",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Changed)
}

func TestRegistryCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	goFmt := newMockFormatter("gofmt", []string{"go"})
	require.NoError(t, reg.Register(goFmt))

	c := cache.NewInMemoryCache(cache.Config{
		MaxEntries:  100,
		TTL:         1 * time.Minute,
		CleanupFreq: 30 * time.Second,
	})
	defer c.Stop()

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	}

	f, err := reg.Get("gofmt")
	require.NoError(t, err)

	result, err := f.Format(ctx, req)
	require.NoError(t, err)

	c.Set(req, result)

	cached, found := c.Get(req)
	assert.True(t, found)
	assert.Equal(t, result.Content, cached.Content)
	assert.Equal(t, 1, c.Size())
}

func TestPipelineIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	step1 := newMockFormatter("step1", []string{"go"})
	step2 := newMockFormatter("step2", []string{"go"})

	pipeline := executor.NewPipeline(step1, step2)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	}

	result, err := pipeline.Execute(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	// step1 appends \n, step2 appends \n again
	assert.Equal(t, "package main\n\n", result.Content)
	assert.True(t, result.Changed)
}

func TestExecutorBatchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	goFmt := newMockFormatter("gofmt", []string{"go"})
	require.NoError(t, reg.Register(goFmt))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	reqs := []*formatter.FormatRequest{
		{Content: "package a", Language: "go"},
		{Content: "package b", Language: "go"},
		{Content: "package c", Language: "go"},
	}

	results, err := exec.ExecuteBatch(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, results, 3)

	for i, result := range results {
		assert.True(t, result.Success, "result %d should succeed", i)
		assert.True(t, result.Changed, "result %d should be changed", i)
	}
}

func TestHealthCheckAllIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newMockFormatter("fmt1", []string{"go"})))
	require.NoError(t, reg.Register(newMockFormatter("fmt2", []string{"python"})))
	require.NoError(t, reg.Register(newMockFormatter("fmt3", []string{"rust"})))

	ctx := context.Background()
	results := reg.HealthCheckAll(ctx)

	assert.Len(t, results, 3)
	for name, err := range results {
		assert.NoError(t, err, "health check failed for %s", name)
	}
}
