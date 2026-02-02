package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/formatter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceFormatter(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:            "test-service",
		Type:            formatter.FormatterTypeService,
		Version:         "1.0.0",
		Languages:       []string{"testlang"},
		SupportsStdin:   true,
		SupportsInPlace: false,
		SupportsCheck:   true,
		SupportsConfig:  true,
	}

	cfg := DefaultConfig("http://localhost:9999")
	f := NewServiceFormatter(metadata, cfg)
	assert.NotNil(t, f)
	assert.Equal(t, "test-service", f.Name())
	assert.Equal(t, "1.0.0", f.Version())
	assert.Contains(t, f.Languages(), "testlang")
	assert.True(t, f.SupportsStdin())
	assert.False(t, f.SupportsInPlace())
	assert.True(t, f.SupportsCheck())
	assert.True(t, f.SupportsConfig())
}

func TestServiceFormatter_Format_Success(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/format", r.URL.Path)
			require.Equal(t, "POST", r.Method)
			require.Equal(t,
				"application/json", r.Header.Get("Content-Type"),
			)

			response := `{
				"success": true,
				"content": "formatted content",
				"changed": true,
				"formatter": "test-formatter"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
		}),
	)
	defer server.Close()

	metadata := &formatter.FormatterMetadata{
		Name:      "test-formatter",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig(server.URL)
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "original content",
		Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "formatted content", result.Content)
	assert.True(t, result.Changed)
	assert.Equal(t, "test-formatter", result.FormatterName)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestServiceFormatter_Format_ServiceError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := `{
				"success": false,
				"error": "Invalid syntax",
				"formatter": "test"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
		}),
	)
	defer server.Close()

	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig(server.URL)
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content: "test", Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "formatter service error")
	assert.False(t, result.Success)
}

func TestServiceFormatter_Format_HTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		}),
	)
	defer server.Close()

	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig(server.URL)
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content: "test", Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
	assert.False(t, result.Success)
}

func TestServiceFormatter_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/health", r.URL.Path)
			response := `{
				"status": "healthy",
				"formatter": "test",
				"version": "1.0"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
		}),
	)
	defer server.Close()

	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig(server.URL)
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	err := f.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestServiceFormatter_HealthCheck_Unhealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
	)
	defer server.Close()

	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig(server.URL)
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	err := f.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unhealthy status code")
}

func TestServiceFormatter_FormatBatch(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			response := `{
				"success": true,
				"content": "formatted",
				"changed": true,
				"formatter": "test"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
		}),
	)
	defer server.Close()

	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig(server.URL)
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	reqs := []*formatter.FormatRequest{
		{Content: "content1", Language: "test"},
		{Content: "content2", Language: "test"},
		{Content: "content3", Language: "test"},
	}

	results, err := f.FormatBatch(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, 3, requestCount)
	for _, result := range results {
		assert.True(t, result.Success)
		assert.Equal(t, "formatted", result.Content)
	}
}

func TestServiceFormatter_ValidateConfig(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig("http://localhost")
	f := NewServiceFormatter(metadata, cfg)

	err := f.ValidateConfig(nil)
	assert.NoError(t, err)

	err = f.ValidateConfig(map[string]interface{}{"key": "value"})
	assert.NoError(t, err)
}

func TestServiceFormatter_DefaultConfigMethod(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig("http://localhost")
	f := NewServiceFormatter(metadata, cfg)

	defCfg := f.DefaultConfig()
	assert.NotNil(t, defCfg)
	assert.Empty(t, defCfg)
}

func TestServiceFormatter_GetMetadata(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:    "test",
		Type:    formatter.FormatterTypeService,
		Version: "2.0",
	}
	cfg := DefaultConfig("http://localhost")
	f := NewServiceFormatter(metadata, cfg)

	got := f.GetMetadata()
	assert.Equal(t, "test", got.Name)
	assert.Equal(t, "2.0", got.Version)
}

func TestDefaultServiceConfig(t *testing.T) {
	cfg := DefaultConfig("http://localhost:9210")
	assert.Equal(t, "http://localhost:9210", cfg.Endpoint)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, "/health", cfg.HealthPath)
	assert.Equal(t, "/format", cfg.FormatPath)
}

func TestServiceFormatter_DefaultPaths(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := Config{Endpoint: "http://localhost"}
	f := NewServiceFormatter(metadata, cfg)

	assert.Equal(t, "/health", f.config.HealthPath)
	assert.Equal(t, "/format", f.config.FormatPath)
}
