# Formatters

Generic, reusable code formatting module for Go applications.

## Module

```
digital.vasic.formatters
```

## Packages

| Package | Description |
|---------|-------------|
| `pkg/formatter` | Core interfaces: Formatter, FormatRequest, FormatResult, Options |
| `pkg/registry` | Thread-safe formatter registry with language detection |
| `pkg/native` | Native binary formatters (gofmt, black, prettier, rustfmt, sqlformat) |
| `pkg/service` | HTTP service-based formatters for containerized tools |
| `pkg/executor` | Execution engine with middleware, pipelines, and batch processing |
| `pkg/cache` | In-memory format result caching with TTL and eviction |

## Usage

```go
import (
    "digital.vasic.formatters/pkg/formatter"
    "digital.vasic.formatters/pkg/registry"
    "digital.vasic.formatters/pkg/native"
    "digital.vasic.formatters/pkg/executor"
)

// Create registry and register formatters
reg := registry.New()
reg.Register(native.NewGoFormatter())
reg.Register(native.NewPythonFormatter())

// Create executor with middleware
exec := executor.New(reg, executor.DefaultExecutorConfig())
exec.Use(executor.TimeoutMiddleware(30 * time.Second))
exec.Use(executor.ValidationMiddleware())

// Format code
result, err := exec.Execute(ctx, &formatter.FormatRequest{
    Content:  "package main\nfunc main(){}\n",
    Language: "go",
})
```

## Testing

```bash
go test ./... -count=1 -race
```
