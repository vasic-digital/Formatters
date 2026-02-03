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

func TestServiceFormatter_Format_MarshalError(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	cfg := DefaultConfig("http://localhost:9999")
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	// Channels cannot be marshaled to JSON
	req := &formatter.FormatRequest{
		Content:  "test content",
		Language: "test",
		Config:   map[string]interface{}{"unmarshalable": make(chan int)},
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestServiceFormatter_Format_InvalidURL(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	// Control character in URL makes it invalid for request creation
	cfg := DefaultConfig("http://localhost\x00:9999")
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "test content",
		Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid control character")
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestServiceFormatter_Format_ReadBodyError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
			// Write less data than Content-Length promises,
			// causing io.ReadAll to fail
			_, _ = w.Write([]byte("partial"))
			// Force close the connection to trigger read error
			if hijacker, ok := w.(http.Hijacker); ok {
				conn, _, _ := hijacker.Hijack()
				if conn != nil {
					_ = conn.Close()
				}
			}
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
		Content:  "test content",
		Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	// Error could be read error or parse error depending on timing
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestServiceFormatter_FormatBatch_WithErrors(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var response string
			if callCount == 2 {
				// Second request fails
				response = `{
					"success": false,
					"error": "format error on second request",
					"formatter": "test"
				}`
			} else {
				response = `{
					"success": true,
					"content": "formatted",
					"changed": true,
					"formatter": "test"
				}`
			}
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

	// First request succeeds
	assert.True(t, results[0].Success)
	assert.Equal(t, "formatted", results[0].Content)

	// Second request fails
	assert.False(t, results[1].Success)
	assert.NotNil(t, results[1].Error)
	assert.Contains(t, results[1].Error.Error(), "format error")

	// Third request succeeds
	assert.True(t, results[2].Success)
	assert.Equal(t, "formatted", results[2].Content)
}

func TestServiceFormatter_HealthCheck_InvalidURL(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	// Control character in URL makes it invalid for request creation
	cfg := DefaultConfig("http://localhost\x00:9999")
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	err := f.HealthCheck(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create health check request")
}

func TestServiceFormatter_HealthCheck_ReadBodyError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
			// Write partial data then close connection
			_, _ = w.Write([]byte("partial"))
			if hijacker, ok := w.(http.Hijacker); ok {
				conn, _, _ := hijacker.Hijack()
				if conn != nil {
					_ = conn.Close()
				}
			}
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
	require.Error(t, err)
	// Error could be read error or parse error depending on timing
}

func TestServiceFormatter_HealthCheck_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Invalid JSON response
			_, _ = w.Write([]byte("not valid json {{{"))
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse health response")
}

func TestServiceFormatter_HealthCheck_UnhealthyStatus(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := `{
				"status": "unhealthy",
				"formatter": "test",
				"version": "1.0",
				"error": "database connection failed"
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service unhealthy")
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestServiceFormatter_HealthCheck_ConnectionError(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	// Non-existent server
	cfg := DefaultConfig("http://localhost:59999")
	cfg.Timeout = 100 * time.Millisecond
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	err := f.HealthCheck(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestServiceFormatter_Format_ConnectionError(t *testing.T) {
	metadata := &formatter.FormatterMetadata{
		Name:      "test",
		Type:      formatter.FormatterTypeService,
		Languages: []string{"test"},
	}
	// Non-existent server
	cfg := DefaultConfig("http://localhost:59999")
	cfg.Timeout = 100 * time.Millisecond
	f := NewServiceFormatter(metadata, cfg)

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "test content",
		Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestServiceFormatter_Format_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			time.Sleep(500 * time.Millisecond)
			response := `{"success": true, "content": "formatted", "changed": true}`
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

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &formatter.FormatRequest{
		Content:  "test content",
		Language: "test",
	}

	result, err := f.Format(ctx, req)
	require.Error(t, err)
	assert.False(t, result.Success)
}
