package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/registry"

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

func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	return registry.New()
}

func registerMock(
	t *testing.T, reg *registry.Registry,
	name string, langs []string,
) *mockFormatter {
	t.Helper()
	mock := newMockFormatter(name, "1.0.0", langs)
	err := reg.Register(mock)
	require.NoError(t, err)
	return mock
}

func TestNew(t *testing.T) {
	reg := newTestRegistry(t)
	exec := New(reg, DefaultExecutorConfig())
	assert.NotNil(t, exec)
}

func TestExecutor_Execute_ByLanguage(t *testing.T) {
	reg := newTestRegistry(t)
	registerMock(t, reg, "black", []string{"python"})

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "x=1",
		Language: "python",
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "x=1 formatted", result.Content)
	assert.True(t, result.Changed)
}

func TestExecutor_Execute_ByFilePath(t *testing.T) {
	reg := newTestRegistry(t)
	registerMock(t, reg, "gofmt", []string{"go"})

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	result, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "package main",
		FilePath: "main.go",
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content, "formatted")
}

func TestExecutor_Execute_NoLanguageOrPath(t *testing.T) {
	reg := newTestRegistry(t)
	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content: "some code",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(),
		"either language or file_path must be specified",
	)
}

func TestExecutor_Execute_NoFormatterForLanguage(t *testing.T) {
	reg := newTestRegistry(t)
	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "some code",
		Language: "cobol",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(),
		"no formatters available for language",
	)
}

func TestExecutor_Execute_FormatterError(t *testing.T) {
	reg := newTestRegistry(t)
	mock := registerMock(t, reg, "errfmt", []string{"python"})
	mock.formatFunc = func(
		ctx context.Context, req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return nil, fmt.Errorf("format failed")
	}

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "x=1",
		Language: "python",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "format failed")
}

func TestExecutor_Execute_DetectFromUnknownPath(t *testing.T) {
	reg := newTestRegistry(t)
	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content:  "data",
		FilePath: "noextension",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to detect formatter")
}

func TestExecutor_ExecuteBatch(t *testing.T) {
	reg := newTestRegistry(t)
	registerMock(t, reg, "black", []string{"python"})
	registerMock(t, reg, "gofmt", []string{"go"})

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	reqs := []*formatter.FormatRequest{
		{Content: "x=1", Language: "python"},
		{Content: "package main", Language: "go"},
	}

	results, err := exec.ExecuteBatch(ctx, reqs)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	for _, r := range results {
		assert.NotNil(t, r)
		assert.Contains(t, r.Content, "formatted")
	}
}

func TestExecutor_ExecuteBatch_PartialFailure(t *testing.T) {
	reg := newTestRegistry(t)
	mock := registerMock(t, reg, "black", []string{"python"})
	mock.formatFunc = func(
		ctx context.Context, req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return nil, fmt.Errorf("fail")
	}
	registerMock(t, reg, "gofmt", []string{"go"})

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	reqs := []*formatter.FormatRequest{
		{Content: "x=1", Language: "python"},
		{Content: "package main", Language: "go"},
	}

	results, err := exec.ExecuteBatch(ctx, reqs)
	assert.Error(t, err)
	assert.Len(t, results, 2)
}

func TestExecutor_ExecuteBatch_Empty(t *testing.T) {
	reg := newTestRegistry(t)
	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	results, err := exec.ExecuteBatch(ctx, []*formatter.FormatRequest{})
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestExecutor_Use(t *testing.T) {
	reg := newTestRegistry(t)
	exec := New(reg, DefaultExecutorConfig())

	called := false
	mw := func(next ExecuteFunc) ExecuteFunc {
		return func(
			ctx context.Context,
			f formatter.Formatter,
			req *formatter.FormatRequest,
		) (*formatter.FormatResult, error) {
			called = true
			return next(ctx, f, req)
		}
	}

	exec.Use(mw)
	registerMock(t, reg, "black", []string{"python"})

	ctx := context.Background()
	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content: "x=1", Language: "python",
	})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestExecutor_BuildChain_MultipleMiddleware(t *testing.T) {
	reg := newTestRegistry(t)
	registerMock(t, reg, "black", []string{"python"})

	exec := New(reg, DefaultExecutorConfig())

	order := make([]string, 0)

	mw1 := func(next ExecuteFunc) ExecuteFunc {
		return func(
			ctx context.Context,
			f formatter.Formatter,
			req *formatter.FormatRequest,
		) (*formatter.FormatResult, error) {
			order = append(order, "mw1-before")
			result, err := next(ctx, f, req)
			order = append(order, "mw1-after")
			return result, err
		}
	}

	mw2 := func(next ExecuteFunc) ExecuteFunc {
		return func(
			ctx context.Context,
			f formatter.Formatter,
			req *formatter.FormatRequest,
		) (*formatter.FormatResult, error) {
			order = append(order, "mw2-before")
			result, err := next(ctx, f, req)
			order = append(order, "mw2-after")
			return result, err
		}
	}

	exec.Use(mw1, mw2)

	ctx := context.Background()
	_, err := exec.Execute(ctx, &formatter.FormatRequest{
		Content: "x=1", Language: "python",
	})
	assert.NoError(t, err)

	assert.Equal(t, []string{
		"mw1-before", "mw2-before", "mw2-after", "mw1-after",
	}, order)
}

// --- Middleware Tests ---

func TestTimeoutMiddleware_Success(t *testing.T) {
	mw := TimeoutMiddleware(5 * time.Second)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: "done", Success: true,
		}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	result, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.NoError(t, err)
	assert.Equal(t, "done", result.Content)
}

func TestTimeoutMiddleware_Timeout(t *testing.T) {
	mw := TimeoutMiddleware(10 * time.Millisecond)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		time.Sleep(100 * time.Millisecond)
		return &formatter.FormatResult{Content: "done"}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestTimeoutMiddleware_UsesRequestTimeout(t *testing.T) {
	mw := TimeoutMiddleware(5 * time.Second)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		time.Sleep(100 * time.Millisecond)
		return &formatter.FormatResult{Content: "done"}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{
			Content: "code",
			Timeout: 10 * time.Millisecond,
		},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestRetryMiddleware_Success(t *testing.T) {
	mw := RetryMiddleware(3)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: "ok", Success: true,
		}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	result, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.NoError(t, err)
	assert.Equal(t, "ok", result.Content)
}

func TestRetryMiddleware_EventualSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping retry test with sleep in short mode")
	}
	attempts := 0
	mw := RetryMiddleware(3)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		attempts++
		if attempts < 2 {
			return nil, fmt.Errorf("transient error")
		}
		return &formatter.FormatResult{
			Content: "ok", Success: true,
		}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	result, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.NoError(t, err)
	assert.Equal(t, "ok", result.Content)
	assert.Equal(t, 2, attempts)
}

func TestRetryMiddleware_AllFail(t *testing.T) {
	attempts := 0
	mw := RetryMiddleware(0)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		attempts++
		return nil, fmt.Errorf("permanent error")
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permanent error")
	assert.Equal(t, 1, attempts)
}

func TestRetryMiddleware_ContextCancellation(t *testing.T) {
	mw := RetryMiddleware(3)
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		attempts++
		cancel()
		return nil, fmt.Errorf("error")
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		ctx, mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetryMiddleware_MaxRetriesCapped(t *testing.T) {
	mw := RetryMiddleware(100)

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return nil, fmt.Errorf("fail")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		ctx, mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "30 retries")
}

func TestRetryMiddleware_NegativeRetries(t *testing.T) {
	mw := RetryMiddleware(-5)
	attempts := 0

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		attempts++
		return nil, fmt.Errorf("fail")
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}

func TestValidationMiddleware_EmptyContent(t *testing.T) {
	mw := ValidationMiddleware()

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{Content: "ok"}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: ""},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty content provided")
}

func TestValidationMiddleware_EmptyResult(t *testing.T) {
	mw := ValidationMiddleware()

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: "", Success: true,
		}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	_, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "formatter returned empty content")
}

func TestValidationMiddleware_Success(t *testing.T) {
	mw := ValidationMiddleware()

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: "formatted", Success: true,
		}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	result, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.NoError(t, err)
	assert.Equal(t, "formatted", result.Content)
}

func TestValidationMiddleware_FailedResultWithEmptyContent(t *testing.T) {
	mw := ValidationMiddleware()

	base := func(
		ctx context.Context, f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: "", Success: false,
		}, nil
	}

	wrapped := mw(base)
	mock := newMockFormatter("test", "1.0", []string{"go"})
	result, err := wrapped(
		context.Background(), mock,
		&formatter.FormatRequest{Content: "code"},
	)

	assert.NoError(t, err)
	assert.False(t, result.Success)
}

// --- Pipeline Tests ---

func TestPipeline_Execute(t *testing.T) {
	step1 := newMockFormatter("step1", "1.0", []string{"go"})
	step1.formatFunc = func(
		ctx context.Context, req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: req.Content + " [step1]",
			Success: true,
		}, nil
	}

	step2 := newMockFormatter("step2", "1.0", []string{"go"})
	step2.formatFunc = func(
		ctx context.Context, req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Content: req.Content + " [step2]",
			Success: true,
		}, nil
	}

	pipeline := NewPipeline(step1, step2)
	ctx := context.Background()

	result, err := pipeline.Execute(ctx, &formatter.FormatRequest{
		Content: "code",
	})

	require.NoError(t, err)
	assert.Equal(t, "code [step1] [step2]", result.Content)
	assert.True(t, result.Changed)
	assert.True(t, result.Success)
}

func TestPipeline_Execute_Empty(t *testing.T) {
	pipeline := NewPipeline()
	ctx := context.Background()

	result, err := pipeline.Execute(ctx, &formatter.FormatRequest{
		Content: "code",
	})

	require.NoError(t, err)
	assert.Equal(t, "code", result.Content)
	assert.False(t, result.Changed)
}

func TestPipeline_Execute_StepFailure(t *testing.T) {
	step1 := newMockFormatter("step1", "1.0", []string{"go"})
	step1.formatFunc = func(
		ctx context.Context, req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return nil, fmt.Errorf("step1 failed")
	}

	pipeline := NewPipeline(step1)
	ctx := context.Background()

	_, err := pipeline.Execute(ctx, &formatter.FormatRequest{
		Content: "code",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline step step1 failed")
}

func TestPipeline_Execute_StepUnsuccessful(t *testing.T) {
	step1 := newMockFormatter("step1", "1.0", []string{"go"})
	step1.formatFunc = func(
		ctx context.Context, req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return &formatter.FormatResult{
			Success: false,
			Error:   fmt.Errorf("syntax error"),
		}, nil
	}

	pipeline := NewPipeline(step1)
	ctx := context.Background()

	result, err := pipeline.Execute(ctx, &formatter.FormatRequest{
		Content: "code",
	})

	require.NoError(t, err)
	assert.False(t, result.Success)
}

// --- BatchFormat Tests ---

func TestBatchFormat(t *testing.T) {
	reg := newTestRegistry(t)
	registerMock(t, reg, "black", []string{"python"})

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	reqs := []*formatter.FormatRequest{
		{Content: "a=1", Language: "python"},
		{Content: "b=2", Language: "python"},
		{Content: "c=3", Language: "python"},
	}

	results, err := BatchFormat(ctx, exec, reqs, 2)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	for _, r := range results {
		assert.NotNil(t, r)
		assert.Contains(t, r.Content, "formatted")
	}
}

func TestBatchFormat_DefaultConcurrency(t *testing.T) {
	reg := newTestRegistry(t)
	registerMock(t, reg, "black", []string{"python"})

	exec := New(reg, DefaultExecutorConfig())
	ctx := context.Background()

	reqs := []*formatter.FormatRequest{
		{Content: "a=1", Language: "python"},
	}

	results, err := BatchFormat(ctx, exec, reqs, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}
