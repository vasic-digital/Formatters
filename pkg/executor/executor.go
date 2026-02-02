package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/registry"
)

// Middleware wraps formatter execution.
type Middleware func(next ExecuteFunc) ExecuteFunc

// ExecuteFunc is the execution function signature.
type ExecuteFunc func(
	ctx context.Context,
	f formatter.Formatter,
	req *formatter.FormatRequest,
) (*formatter.FormatResult, error)

// Config configures the executor.
type Config struct {
	DefaultTimeout time.Duration
	MaxRetries     int
	MaxConcurrent  int
}

// DefaultExecutorConfig returns a default executor configuration.
func DefaultExecutorConfig() Config {
	return Config{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		MaxConcurrent:  10,
	}
}

// Executor manages concurrent formatting execution.
type Executor struct {
	registry   *registry.Registry
	middleware []Middleware
	config     Config
}

// New creates a new formatter executor.
func New(reg *registry.Registry, config Config) *Executor {
	return &Executor{
		registry:   reg,
		config:     config,
		middleware: make([]Middleware, 0),
	}
}

// Execute executes a formatting request.
func (e *Executor) Execute(
	ctx context.Context, req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	start := time.Now()

	var f formatter.Formatter
	var err error

	if req.Language != "" {
		formatters := e.registry.GetByLanguage(req.Language)
		if len(formatters) == 0 {
			return nil, fmt.Errorf(
				"no formatters available for language: %s",
				req.Language,
			)
		}
		f = formatters[0]
	} else if req.FilePath != "" {
		f, err = e.registry.DetectFormatter(req.FilePath)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to detect formatter: %w", err,
			)
		}
	} else {
		return nil, fmt.Errorf(
			"either language or file_path must be specified",
		)
	}

	executeFunc := e.buildChain(f)

	result, err := executeFunc(ctx, f, req)
	if err != nil {
		return nil, err
	}

	result.Duration = time.Since(start)

	return result, nil
}

// ExecuteBatch executes multiple formatting requests concurrently.
func (e *Executor) ExecuteBatch(
	ctx context.Context, reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	results := make([]*formatter.FormatResult, len(reqs))
	errors := make([]error, len(reqs))

	type resultPair struct {
		index  int
		result *formatter.FormatResult
		err    error
	}

	resultChan := make(chan resultPair, len(reqs))

	for i, req := range reqs {
		go func(index int, request *formatter.FormatRequest) {
			result, err := e.Execute(ctx, request)
			resultChan <- resultPair{
				index:  index,
				result: result,
				err:    err,
			}
		}(i, req)
	}

	for i := 0; i < len(reqs); i++ {
		pair := <-resultChan
		results[pair.index] = pair.result
		errors[pair.index] = pair.err
	}

	var firstError error
	for _, err := range errors {
		if err != nil {
			firstError = err
			break
		}
	}

	return results, firstError
}

// Use adds middleware to the execution chain.
func (e *Executor) Use(middleware ...Middleware) {
	e.middleware = append(e.middleware, middleware...)
}

// buildChain builds the execution chain with middleware.
func (e *Executor) buildChain(
	f formatter.Formatter,
) ExecuteFunc {
	base := func(
		ctx context.Context,
		f formatter.Formatter,
		req *formatter.FormatRequest,
	) (*formatter.FormatResult, error) {
		return f.Format(ctx, req)
	}

	for i := len(e.middleware) - 1; i >= 0; i-- {
		base = e.middleware[i](base)
	}

	return base
}

// --- Pipeline ---

// Pipeline chains multiple formatters in sequence.
type Pipeline struct {
	steps []formatter.Formatter
}

// NewPipeline creates a new formatting pipeline.
func NewPipeline(steps ...formatter.Formatter) *Pipeline {
	return &Pipeline{steps: steps}
}

// Execute runs the pipeline, passing output of each step as input
// to the next.
func (p *Pipeline) Execute(
	ctx context.Context, req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	current := req.Content
	var lastResult *formatter.FormatResult

	for _, step := range p.steps {
		stepReq := &formatter.FormatRequest{
			Content:    current,
			FilePath:   req.FilePath,
			Language:   req.Language,
			Config:     req.Config,
			LineLength: req.LineLength,
			IndentSize: req.IndentSize,
			UseTabs:    req.UseTabs,
			CheckOnly:  req.CheckOnly,
			Timeout:    req.Timeout,
			RequestID:  req.RequestID,
		}

		result, err := step.Format(ctx, stepReq)
		if err != nil {
			return nil, fmt.Errorf(
				"pipeline step %s failed: %w",
				step.Name(), err,
			)
		}

		if !result.Success {
			return result, nil
		}

		current = result.Content
		lastResult = result
	}

	if lastResult == nil {
		return &formatter.FormatResult{
			Content: req.Content,
			Changed: false,
			Success: true,
		}, nil
	}

	lastResult.Changed = lastResult.Content != req.Content
	return lastResult, nil
}

// --- Batch Format ---

// BatchFormat formats multiple files concurrently with rate limiting.
func BatchFormat(
	ctx context.Context,
	exec *Executor,
	reqs []*formatter.FormatRequest,
	maxConcurrent int,
) ([]*formatter.FormatResult, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	results := make([]*formatter.FormatResult, len(reqs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)

	var firstErr error

	for i, req := range reqs {
		wg.Add(1)
		go func(index int, request *formatter.FormatRequest) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := exec.Execute(ctx, request)

			mu.Lock()
			defer mu.Unlock()

			if err != nil && firstErr == nil {
				firstErr = err
			}
			results[index] = result
		}(i, req)
	}

	wg.Wait()

	return results, firstErr
}

// --- Middleware Implementations ---

// TimeoutMiddleware adds timeout handling.
func TimeoutMiddleware(defaultTimeout time.Duration) Middleware {
	return func(next ExecuteFunc) ExecuteFunc {
		return func(
			ctx context.Context,
			f formatter.Formatter,
			req *formatter.FormatRequest,
		) (*formatter.FormatResult, error) {
			timeout := req.Timeout
			if timeout == 0 {
				timeout = defaultTimeout
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			resultChan := make(chan *formatter.FormatResult, 1)
			errorChan := make(chan error, 1)

			go func() {
				result, err := next(ctx, f, req)
				if err != nil {
					errorChan <- err
				} else {
					resultChan <- result
				}
			}()

			select {
			case result := <-resultChan:
				return result, nil
			case err := <-errorChan:
				return nil, err
			case <-ctx.Done():
				return nil, fmt.Errorf(
					"formatter execution timed out after %v",
					timeout,
				)
			}
		}
	}
}

// RetryMiddleware adds retry logic with exponential backoff.
func RetryMiddleware(maxRetries int) Middleware {
	if maxRetries > 30 {
		maxRetries = 30
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	return func(next ExecuteFunc) ExecuteFunc {
		return func(
			ctx context.Context,
			f formatter.Formatter,
			req *formatter.FormatRequest,
		) (*formatter.FormatResult, error) {
			var lastErr error

			for attempt := 0; attempt <= maxRetries; attempt++ {
				result, err := next(ctx, f, req)
				if err == nil {
					return result, nil
				}

				lastErr = err

				if ctx.Err() != nil {
					break
				}

				if attempt < maxRetries {
					waitTime := time.Duration(
						1<<uint(attempt&0x3f),
					) * time.Second
					time.Sleep(waitTime)
				}
			}

			return nil, fmt.Errorf(
				"formatter execution failed after %d retries: %w",
				maxRetries, lastErr,
			)
		}
	}
}

// ValidationMiddleware adds pre/post validation.
func ValidationMiddleware() Middleware {
	return func(next ExecuteFunc) ExecuteFunc {
		return func(
			ctx context.Context,
			f formatter.Formatter,
			req *formatter.FormatRequest,
		) (*formatter.FormatResult, error) {
			if req.Content == "" {
				return nil, fmt.Errorf("empty content provided")
			}

			result, err := next(ctx, f, req)
			if err != nil {
				return nil, err
			}

			if result.Success && result.Content == "" {
				return nil, fmt.Errorf(
					"formatter returned empty content",
				)
			}

			return result, nil
		}
	}
}
