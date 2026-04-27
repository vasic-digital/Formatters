package security

import (
	"context"
	"strings"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/cache"
	"digital.vasic.formatters/pkg/executor"
	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// securityFormatter is a formatter for security testing.
type securityFormatter struct {
	*formatter.BaseFormatter
}

func newSecurityFormatter() *securityFormatter {
	return &securityFormatter{
		BaseFormatter: formatter.NewBaseFormatter(&formatter.FormatterMetadata{
			Name:          "secure-fmt",
			Type:          formatter.FormatterTypeNative,
			Version:       "1.0.0",
			Languages:     []string{"go"},
			SupportsStdin: true,
		}),
	}
}

func (f *securityFormatter) Format(
	_ context.Context,
	req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	return &formatter.FormatResult{
		Content: req.Content,
		Changed: false,
		Success: true,
	}, nil
}

func (f *securityFormatter) FormatBatch(
	ctx context.Context,
	reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	results := make([]*formatter.FormatResult, len(reqs))
	for i, req := range reqs {
		r, err := f.Format(ctx, req)
		if err != nil {
			return nil, err
		}
		results[i] = r
	}
	return results, nil
}

func (f *securityFormatter) HealthCheck(_ context.Context) error {
	return nil
}

func TestEmptyContentRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newSecurityFormatter()))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	exec.Use(executor.ValidationMiddleware())

	ctx := context.Background()
	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "",
		Language: "go",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty content")
}

func TestMissingLanguageAndPathRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newSecurityFormatter()))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content: "some code",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either language or file_path")
}

func TestDuplicateRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	f := newSecurityFormatter()
	require.NoError(t, reg.Register(f))

	err := reg.Register(f)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestLargeInputHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newSecurityFormatter()))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	exec.Use(executor.ValidationMiddleware())

	ctx := context.Background()

	largeContent := strings.Repeat("x", 10*1024*1024) // 10MB
	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  largeContent,
		Language: "go",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, len(largeContent), len(result.Content))
}

func TestCacheEvictionSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	c := cache.NewInMemoryCache(cache.Config{
		MaxEntries:  3,
		TTL:         1 * time.Minute,
		CleanupFreq: 30 * time.Second,
	})
	defer c.Stop()

	for i := 0; i < 10; i++ {
		req := &formatter.FormatRequest{
			Content:  strings.Repeat("a", i+1),
			Language: "go",
		}
		c.Set(req, &formatter.FormatResult{
			Content: req.Content,
			Success: true,
		})
	}

	assert.LessOrEqual(t, c.Size(), 3)
}

func TestRegistryRemoveNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	err := reg.Remove("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestContextCancellationHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	reg := registry.New()
	require.NoError(t, reg.Register(newSecurityFormatter()))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	exec.Use(executor.TimeoutMiddleware(1 * time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	})
	// Should either error or succeed quickly; no hang
	_ = err
}

func TestLanguageDetectionUnknownExtension(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	detected := registry.DetectLanguageFromPath("file.unknownext123")
	assert.Empty(t, detected)

	detected = registry.DetectLanguageFromPath("")
	assert.Empty(t, detected)

	detected = registry.DetectLanguageFromPath("noextension")
	assert.Empty(t, detected)
}
