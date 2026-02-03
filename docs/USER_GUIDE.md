# User Guide

This guide covers how to use the `digital.vasic.formatters` module to format source code in Go applications.

## Installation

```bash
go get digital.vasic.formatters
```

Requires Go 1.24 or later.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "digital.vasic.formatters/pkg/formatter"
    "digital.vasic.formatters/pkg/native"
    "digital.vasic.formatters/pkg/registry"
    "digital.vasic.formatters/pkg/executor"
)

func main() {
    // 1. Create a registry
    reg := registry.New()

    // 2. Register formatters
    reg.Register(native.NewGoFormatter())
    reg.Register(native.NewPythonFormatter())

    // 3. Create an executor
    exec := executor.New(reg, executor.DefaultExecutorConfig())

    // 4. Format code
    ctx := context.Background()
    result, err := exec.Execute(ctx, &formatter.FormatRequest{
        Content:  "package main\nfunc main(){}\n",
        Language: "go",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Content)
    fmt.Printf("Changed: %v, Duration: %v\n", result.Changed, result.Duration)
}
```

## Core Concepts

### FormatRequest

Every formatting operation starts with a `FormatRequest`:

```go
req := &formatter.FormatRequest{
    Content:    "def hello( ):\n  pass",  // Code to format
    Language:   "python",                  // Language identifier
    FilePath:   "main.py",                // Optional, used for detection
    LineLength: 88,                        // Max line length
    IndentSize: 4,                         // Indent width
    UseTabs:    false,                     // Spaces vs tabs
    CheckOnly:  false,                     // Dry-run mode
    Timeout:    10 * time.Second,          // Execution timeout
    RequestID:  "req-001",                 // Tracking identifier
    Config:     map[string]interface{}{    // Formatter-specific config
        "style": "pep8",
    },
}
```

You must provide either `Language` or `FilePath`. If `Language` is empty, the executor will detect it from `FilePath` using extension mapping.

### FormatResult

The result contains the formatted output and metadata:

```go
result, err := exec.Execute(ctx, req)
if err != nil {
    // Execution-level error (no formatter found, etc.)
    log.Fatal(err)
}

if !result.Success {
    // Formatter ran but failed
    fmt.Printf("Formatter error: %v\n", result.Error)
    return
}

fmt.Println(result.Content)          // Formatted code
fmt.Println(result.Changed)          // true if code was modified
fmt.Println(result.FormatterName)    // e.g. "gofmt"
fmt.Println(result.FormatterVersion) // e.g. "go1.24"
fmt.Println(result.Duration)         // Execution time

if result.Stats != nil {
    fmt.Printf("Lines changed: %d/%d\n",
        result.Stats.LinesChanged,
        result.Stats.LinesTotal,
    )
}
```

## Using Native Formatters

Native formatters invoke system binaries. Five formatters are provided out of the box.

### Go (gofmt)

```go
goFmt := native.NewGoFormatter()
result, err := goFmt.Format(ctx, &formatter.FormatRequest{
    Content: "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"hello\")}\n",
})
```

Requires: `gofmt` on PATH (included with Go installation).

### Python (black)

```go
pyFmt := native.NewPythonFormatter()
result, err := pyFmt.Format(ctx, &formatter.FormatRequest{
    Content: "x=1\ny =   2\nz= x+y\n",
})
```

Requires: `pip install black`.

### JavaScript/TypeScript (prettier)

```go
jsFmt := native.NewJSFormatter()
result, err := jsFmt.Format(ctx, &formatter.FormatRequest{
    Content: "const x={a:1,b:2,c:3}\n",
})
```

Requires: `npm install -g prettier`. Supports JavaScript, TypeScript, JSON, HTML, CSS, SCSS, Markdown, YAML, and GraphQL.

### Rust (rustfmt)

```go
rustFmt := native.NewRustFormatter()
result, err := rustFmt.Format(ctx, &formatter.FormatRequest{
    Content: "fn main(){println!(\"hello\");}\n",
})
```

Requires: `rustup component add rustfmt`.

### SQL (sqlformat)

```go
sqlFmt := native.NewSQLFormatter()
result, err := sqlFmt.Format(ctx, &formatter.FormatRequest{
    Content: "select id,name from users where active=1 order by name\n",
})
```

Requires: `pip install sqlparse`.

### Custom Native Formatter

```go
customFmt := native.NewNativeFormatter(
    &formatter.FormatterMetadata{
        Name:          "clang-format",
        Type:          formatter.FormatterTypeNative,
        Version:       "19.1.8",
        Languages:     []string{"c", "cpp"},
        SupportsStdin: true,
    },
    "clang-format",                          // Binary path
    []string{"--style=Google"},              // Default args
    true,                                    // Use stdin
)
```

## Using Service Formatters

Service formatters communicate with containerized formatting tools over HTTP.

### Basic Setup

```go
import "digital.vasic.formatters/pkg/service"

svcFmt := service.NewServiceFormatter(
    &formatter.FormatterMetadata{
        Name:      "rubocop-service",
        Type:      formatter.FormatterTypeService,
        Version:   "1.68.0",
        Languages: []string{"ruby"},
    },
    service.Config{
        Endpoint:   "http://localhost:9210",
        Timeout:    30 * time.Second,
        HealthPath: "/health",
        FormatPath: "/format",
    },
)
```

### Using Default Config

```go
svcFmt := service.NewServiceFormatter(
    metadata,
    service.DefaultConfig("http://localhost:9210"),
)
```

Default config uses 30s timeout, `/health` for health checks, and `/format` for formatting.

### Health Checking

```go
err := svcFmt.HealthCheck(ctx)
if err != nil {
    log.Printf("Service unhealthy: %v", err)
}
```

The health check sends `GET /health` and expects a JSON response with `{"status": "healthy"}`.

### Service HTTP Protocol

**Format request** (`POST /format`):
```json
{
    "content": "code to format",
    "options": {"indent": 4}
}
```

**Format response**:
```json
{
    "success": true,
    "content": "formatted code",
    "changed": true,
    "formatter": "rubocop"
}
```

**Health response** (`GET /health`):
```json
{
    "status": "healthy",
    "formatter": "rubocop",
    "version": "1.68.0"
}
```

## Using the Registry

The registry manages formatter instances and provides lookup by name or language.

### Creating and Populating a Registry

```go
reg := registry.New()

// Register formatters
err := reg.Register(native.NewGoFormatter())
err = reg.Register(native.NewPythonFormatter())
err = reg.Register(native.NewJSFormatter())
```

### Lookup by Name

```go
goFmt, err := reg.Get("gofmt")
if err != nil {
    log.Fatal("gofmt not registered")
}
```

### Lookup by Language

```go
formatters := reg.GetByLanguage("python")
if len(formatters) > 0 {
    result, err := formatters[0].Format(ctx, req)
}
```

### Auto-Detection from File Path

```go
f, err := reg.DetectFormatter("src/main.go")
// Returns the first registered formatter for "go"
```

### Language Detection (Standalone)

```go
lang := registry.DetectLanguageFromPath("app.tsx")
// Returns "typescript"

lang = registry.DetectLanguageFromPath("Makefile")
// Returns "" (unknown)
```

Supported extensions include: `.go`, `.py`, `.js`, `.ts`, `.rs`, `.java`, `.kt`, `.rb`, `.php`, `.c`, `.cpp`, `.sql`, `.html`, `.css`, `.json`, `.yaml`, `.md`, `.sh`, `.lua`, `.r`, `.zig`, `.nim`, `.dart`, `.swift`, `.scala`, `.hs`, `.ml`, `.ex`, `.erl`, `.proto`, `.tf`, `.graphql`, and more.

### Listing and Filtering

```go
// All formatter names
names := reg.List()

// Count
count := reg.Count()

// By type (requires RegisterWithMetadata)
nativeFormatters := reg.ListByType(formatter.FormatterTypeNative)
serviceFormatters := reg.ListByType(formatter.FormatterTypeService)
```

### Registering with Metadata

```go
err := reg.RegisterWithMetadata(myFormatter, &formatter.FormatterMetadata{
    Name:          "my-formatter",
    Type:          formatter.FormatterTypeNative,
    Performance:   "fast",
    InstallMethod: "binary",
})

// Retrieve metadata later
meta, err := reg.GetMetadata("my-formatter")
```

### Removing a Formatter

```go
err := reg.Remove("gofmt")
```

### Health Checking All Formatters

```go
results := reg.HealthCheckAll(ctx)
for name, err := range results {
    if err != nil {
        fmt.Printf("%s: UNHEALTHY (%v)\n", name, err)
    } else {
        fmt.Printf("%s: healthy\n", name)
    }
}
```

Health checks run concurrently with a maximum of 10 parallel checks.

### Default Registry Singleton

```go
// Use the global singleton
registry.RegisterDefault(native.NewGoFormatter())

f, err := registry.GetDefault("gofmt")

// Access the singleton directly
defaultReg := registry.Default()
```

## Using the Executor

The executor provides middleware-based execution, pipelines, and batch processing.

### Basic Execution

```go
exec := executor.New(reg, executor.DefaultExecutorConfig())

result, err := exec.Execute(ctx, &formatter.FormatRequest{
    Content:  "x=1",
    Language: "python",
})
```

The default config uses a 30-second timeout, 3 max retries, and 10 max concurrent operations.

### Adding Middleware

Middleware wraps the execution chain. Order matters: middleware added first runs first (outermost).

```go
exec := executor.New(reg, executor.DefaultExecutorConfig())

// Add timeout enforcement
exec.Use(executor.TimeoutMiddleware(30 * time.Second))

// Add retry logic (exponential backoff, max 30 retries)
exec.Use(executor.RetryMiddleware(3))

// Add input/output validation
exec.Use(executor.ValidationMiddleware())
```

**TimeoutMiddleware**: Enforces a deadline on each formatting call. Uses the request's `Timeout` field if set, otherwise falls back to the provided default.

**RetryMiddleware**: Retries failed formatting with exponential backoff (1s, 2s, 4s, ...). Respects context cancellation. Maximum retry count is capped at 30.

**ValidationMiddleware**: Rejects empty input content before execution and rejects empty output content after execution (when the result reports success).

### Custom Middleware

```go
loggingMiddleware := func(next executor.ExecuteFunc) executor.ExecuteFunc {
    return func(
        ctx context.Context,
        f formatter.Formatter,
        req *formatter.FormatRequest,
    ) (*formatter.FormatResult, error) {
        log.Printf("Formatting %s with %s", req.Language, f.Name())
        result, err := next(ctx, f, req)
        if err != nil {
            log.Printf("Error: %v", err)
        } else {
            log.Printf("Done in %v, changed=%v", result.Duration, result.Changed)
        }
        return result, err
    }
}

exec.Use(loggingMiddleware)
```

### Pipelines

Pipelines chain multiple formatters sequentially, passing the output of each step as input to the next.

```go
pipeline := executor.NewPipeline(
    native.NewJSFormatter(),   // Step 1: format JS
    myLintFormatter,           // Step 2: lint fix
)

result, err := pipeline.Execute(ctx, &formatter.FormatRequest{
    Content:  "const x={a:1}\n",
    Language: "javascript",
})
```

If any step fails (returns `Success: false`), the pipeline stops and returns that result.

### Batch Processing

```go
reqs := []*formatter.FormatRequest{
    {Content: "package a\n", Language: "go"},
    {Content: "x=1\n", Language: "python"},
    {Content: "const x=1\n", Language: "javascript"},
}

// Via executor (concurrent, no rate limiting)
results, err := exec.ExecuteBatch(ctx, reqs)

// Via BatchFormat (concurrent with rate limiting)
results, err = executor.BatchFormat(ctx, exec, reqs, 5) // max 5 concurrent
```

## Using the Cache

The cache stores formatting results keyed by a SHA-256 hash of the request's content, language, and file path.

### Basic Usage

```go
import "digital.vasic.formatters/pkg/cache"

c := cache.NewInMemoryCache(cache.DefaultCacheConfig())
defer c.Stop() // Stop the cleanup goroutine

// Check cache before formatting
if result, hit := c.Get(req); hit {
    fmt.Println("Cache hit!")
    return result, nil
}

// Format and cache the result
result, err := exec.Execute(ctx, req)
if err == nil && result.Success {
    c.Set(req, result)
}
```

### Custom Configuration

```go
c := cache.NewInMemoryCache(cache.Config{
    MaxEntries:  50000,              // Maximum cached entries
    TTL:         30 * time.Minute,   // Entry expiration
    CleanupFreq: 2 * time.Minute,   // Eviction scan interval
})
defer c.Stop()
```

### Cache Operations

```go
// Get a cached result
result, found := c.Get(req)

// Store a result
c.Set(req, result)

// Invalidate a specific entry
c.Invalidate(req)

// Clear everything
c.Clear()

// Check size
size := c.Size()

// Get statistics
stats := c.Stats()
fmt.Printf("Size: %d/%d, TTL: %v\n",
    stats.Size, stats.MaxEntries, stats.TTL,
)
```

### Eviction Behavior

When `MaxEntries` is reached, the oldest entry (by insertion/update time) is evicted. Expired entries are cleaned up periodically based on `CleanupFreq`. Expired entries are also skipped on `Get` (lazy expiration).

## Implementing the Formatter Interface

To create a fully custom formatter, implement the `formatter.Formatter` interface:

```go
type Formatter interface {
    Name() string
    Version() string
    Languages() []string
    SupportsStdin() bool
    SupportsInPlace() bool
    SupportsCheck() bool
    SupportsConfig() bool
    Format(ctx context.Context, req *FormatRequest) (*FormatResult, error)
    FormatBatch(ctx context.Context, reqs []*FormatRequest) ([]*FormatResult, error)
    HealthCheck(ctx context.Context) error
    ValidateConfig(config map[string]interface{}) error
    DefaultConfig() map[string]interface{}
}
```

Use `BaseFormatter` to avoid boilerplate:

```go
type MyFormatter struct {
    *formatter.BaseFormatter
}

func NewMyFormatter() *MyFormatter {
    return &MyFormatter{
        BaseFormatter: formatter.NewBaseFormatter(&formatter.FormatterMetadata{
            Name:          "my-formatter",
            Type:          formatter.FormatterTypeNative,
            Version:       "1.0.0",
            Languages:     []string{"my-lang"},
            SupportsStdin: true,
        }),
    }
}

func (m *MyFormatter) Format(
    ctx context.Context, req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
    // Your formatting logic here
    formatted := doFormat(req.Content)
    return &formatter.FormatResult{
        Content:       formatted,
        Changed:       formatted != req.Content,
        Success:       true,
        FormatterName: m.Name(),
    }, nil
}

func (m *MyFormatter) FormatBatch(
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

func (m *MyFormatter) HealthCheck(ctx context.Context) error {
    // Verify formatter is operational
    return nil
}
```

## Error Handling

The module uses two levels of error reporting:

1. **Go errors** (`error` return values): Indicate execution-level failures such as missing formatters, network errors, or timeouts.
2. **Result errors** (`FormatResult.Error`, `FormatResult.Success`): Indicate formatting failures where the formatter ran but could not produce valid output.

Always check both:

```go
result, err := exec.Execute(ctx, req)
if err != nil {
    // Could not execute at all
    log.Fatal(err)
}
if !result.Success {
    // Formatter ran but failed
    log.Printf("Format failed: %v", result.Error)
}
```
