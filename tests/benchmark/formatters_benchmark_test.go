package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.formatters/pkg/cache"
	"digital.vasic.formatters/pkg/executor"
	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/registry"
)

type benchFormatter struct {
	*formatter.BaseFormatter
}

func newBenchFormatter(name string, languages []string) *benchFormatter {
	return &benchFormatter{
		BaseFormatter: formatter.NewBaseFormatter(&formatter.FormatterMetadata{
			Name:          name,
			Type:          formatter.FormatterTypeNative,
			Version:       "1.0.0",
			Languages:     languages,
			SupportsStdin: true,
		}),
	}
}

func (f *benchFormatter) Format(
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

func (f *benchFormatter) FormatBatch(
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

func (f *benchFormatter) HealthCheck(_ context.Context) error {
	return nil
}

func BenchmarkRegistryGet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	reg := registry.New()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("fmt-%d", i)
		_ = reg.Register(newBenchFormatter(name, []string{fmt.Sprintf("lang-%d", i)}))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("fmt-%d", i%100)
		_, _ = reg.Get(name)
	}
}

func BenchmarkRegistryGetByLanguage(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	reg := registry.New()
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("fmt-%d", i)
		_ = reg.Register(newBenchFormatter(name, []string{"go", "python"}))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.GetByLanguage("go")
	}
}

func BenchmarkExecutorExecute(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	reg := registry.New()
	_ = reg.Register(newBenchFormatter("gofmt", []string{"go"}))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	ctx := context.Background()

	req := &formatter.FormatRequest{
		Content:  "package main\nfunc main() {}",
		Language: "go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exec.Execute(ctx, req)
	}
}

func BenchmarkCacheSetAndGet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	c := cache.NewInMemoryCache(cache.Config{
		MaxEntries:  100000,
		TTL:         10 * time.Minute,
		CleanupFreq: 5 * time.Minute,
	})
	defer c.Stop()

	result := &formatter.FormatResult{
		Content: "formatted code",
		Success: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &formatter.FormatRequest{
			Content:  fmt.Sprintf("content-%d", i),
			Language: "go",
		}
		c.Set(req, result)
		_, _ = c.Get(req)
	}
}

func BenchmarkLanguageDetection(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	paths := []string{
		"main.go", "app.py", "index.js", "lib.rs",
		"query.sql", "style.css", "config.yaml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.DetectLanguageFromPath(paths[i%len(paths)])
	}
}

func BenchmarkPipelineExecution(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	step1 := newBenchFormatter("step1", []string{"go"})
	step2 := newBenchFormatter("step2", []string{"go"})
	step3 := newBenchFormatter("step3", []string{"go"})

	pipeline := executor.NewPipeline(step1, step2, step3)
	ctx := context.Background()

	req := &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Execute(ctx, req)
	}
}

func BenchmarkRegistryRegisterAndRemove(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg := registry.New()
		name := fmt.Sprintf("fmt-%d", i)
		f := newBenchFormatter(name, []string{"go"})
		_ = reg.Register(f)
		_ = reg.Remove(name)
	}
}

func BenchmarkExecutorWithMiddleware(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	reg := registry.New()
	_ = reg.Register(newBenchFormatter("gofmt", []string{"go"}))

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	exec.Use(executor.ValidationMiddleware())
	exec.Use(executor.TimeoutMiddleware(10 * time.Second))

	ctx := context.Background()
	req := &formatter.FormatRequest{
		Content:  "package main",
		Language: "go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exec.Execute(ctx, req)
	}
}
