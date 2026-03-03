package stress

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/cache"
	"digital.vasic.formatters/pkg/executor"
	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stressFormatter struct {
	*formatter.BaseFormatter
}

func newStressFormatter(name string, languages []string) *stressFormatter {
	return &stressFormatter{
		BaseFormatter: formatter.NewBaseFormatter(&formatter.FormatterMetadata{
			Name:          name,
			Type:          formatter.FormatterTypeNative,
			Version:       "1.0.0",
			Languages:     languages,
			SupportsStdin: true,
		}),
	}
}

func (f *stressFormatter) Format(
	_ context.Context,
	req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	return &formatter.FormatResult{
		Content:          req.Content + "\n",
		Changed:          true,
		FormatterName:    f.Name(),
		FormatterVersion: f.Version(),
		Success:          true,
	}, nil
}

func (f *stressFormatter) FormatBatch(
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

func (f *stressFormatter) HealthCheck(_ context.Context) error {
	return nil
}

func TestConcurrentRegistryAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	reg := registry.New()
	const numFormatters = 50

	for i := 0; i < numFormatters; i++ {
		name := fmt.Sprintf("fmt-%d", i)
		require.NoError(t, reg.Register(
			newStressFormatter(name, []string{fmt.Sprintf("lang-%d", i)}),
		))
	}

	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("fmt-%d", id%numFormatters)
			f, err := reg.Get(name)
			assert.NoError(t, err)
			assert.NotNil(t, f)
			_ = reg.List()
			_ = reg.Count()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, numFormatters, reg.Count())
}

func TestConcurrentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	reg := registry.New()
	require.NoError(t, reg.Register(
		newStressFormatter("gofmt", []string{"go"}),
	))

	exec := executor.New(reg, executor.Config{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     1,
		MaxConcurrent:  50,
	})

	var wg sync.WaitGroup
	const goroutines = 80
	errors := make([]error, goroutines)

	ctx := context.Background()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := exec.Execute(ctx, &formatter.FormatRequest{
				Content:  fmt.Sprintf("package p%d", id),
				Language: "go",
			})
			errors[id] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d failed", i)
	}
}

func TestConcurrentCacheOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	c := cache.NewInMemoryCache(cache.Config{
		MaxEntries:  1000,
		TTL:         5 * time.Minute,
		CleanupFreq: 1 * time.Minute,
	})
	defer c.Stop()

	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &formatter.FormatRequest{
				Content:  fmt.Sprintf("content-%d", id),
				Language: "go",
			}
			result := &formatter.FormatResult{
				Content: fmt.Sprintf("formatted-%d", id),
				Success: true,
			}

			c.Set(req, result)
			_, _ = c.Get(req)
			_ = c.Size()
		}(i)
	}

	wg.Wait()
	assert.Greater(t, c.Size(), 0)
}

func TestConcurrentHealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	reg := registry.New()
	for i := 0; i < 20; i++ {
		require.NoError(t, reg.Register(
			newStressFormatter(
				fmt.Sprintf("fmt-%d", i),
				[]string{fmt.Sprintf("lang-%d", i)},
			),
		))
	}

	var wg sync.WaitGroup
	const goroutines = 50

	ctx := context.Background()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := reg.HealthCheckAll(ctx)
			for _, err := range results {
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentBatchFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	reg := registry.New()
	require.NoError(t, reg.Register(
		newStressFormatter("gofmt", []string{"go"}),
	))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	var wg sync.WaitGroup
	const batchCount = 50

	for i := 0; i < batchCount; i++ {
		wg.Add(1)
		go func(batchID int) {
			defer wg.Done()
			reqs := make([]*formatter.FormatRequest, 5)
			for j := 0; j < 5; j++ {
				reqs[j] = &formatter.FormatRequest{
					Content:  fmt.Sprintf("package p%d_%d", batchID, j),
					Language: "go",
				}
			}
			results, err := executor.BatchFormat(ctx, exec, reqs, 3)
			assert.NoError(t, err)
			assert.Len(t, results, 5)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentRegistryModification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	reg := registry.New()
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent-fmt-%d", id)
			f := newStressFormatter(name, []string{"go"})
			_ = reg.Register(f)
			_ = reg.List()
			_ = reg.Count()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, goroutines, reg.Count())
}
