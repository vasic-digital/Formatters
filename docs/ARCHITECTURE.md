# Architecture

This document describes the design decisions, patterns, and structural choices in the `digital.vasic.formatters` module.

## Design Goals

1. **Generic and reusable**: No dependency on any specific application. The module can be embedded in any Go project.
2. **Extensible**: New formatters, middleware, and cache implementations can be added without modifying existing code.
3. **Minimal dependencies**: Only `stretchr/testify` for testing. Zero runtime dependencies beyond the Go standard library.
4. **Thread-safe**: All shared state is protected by mutexes. Concurrent access is safe by default.
5. **Composable**: Formatters, middleware, pipelines, and caches can be combined freely.

## Package Dependency Graph

```
formatter  (core, no internal deps)
    ^
    |
    +--- registry  (depends on formatter)
    |       ^
    |       |
    |       +--- executor  (depends on formatter, registry)
    |
    +--- native    (depends on formatter)
    |
    +--- service   (depends on formatter)
    |
    +--- cache     (depends on formatter)
```

The dependency direction is strictly top-down. `formatter` is the foundation. `registry` builds on it. `executor` builds on both. `native`, `service`, and `cache` depend only on `formatter`. There are no circular dependencies.

## Design Patterns

### Strategy Pattern (Formatter Interface)

The `Formatter` interface defines a strategy for formatting code. Each implementation (NativeFormatter, ServiceFormatter, or any custom type) provides a different algorithm for the same operation.

```
<<interface>>
Formatter
    +Format(ctx, req) -> (result, error)
    +FormatBatch(ctx, reqs) -> (results, error)
    +HealthCheck(ctx) -> error
    +Name() -> string
    +Version() -> string
    +Languages() -> []string
    ...
```

Consumers program against the interface, never against concrete types. The executor, registry, pipeline, and cache all operate on `Formatter` without knowledge of the implementation.

**Why Strategy**: Different languages require fundamentally different formatting approaches -- system binaries, HTTP services, built-in parsers. The Strategy pattern lets us swap implementations transparently.

### Registry Pattern (Formatter Registry)

The `Registry` provides centralized formatter management with multiple lookup strategies:

- **By name**: Direct O(1) lookup via map.
- **By language**: O(1) lookup via pre-built language-to-formatter index.
- **By file path**: Extension detection followed by language lookup.
- **By type**: Metadata-based filtering for native, service, builtin, or unified formatters.

The registry also provides a default singleton via `Default()` for simple use cases where dependency injection is unnecessary.

**Why Registry**: Applications may register dozens of formatters at startup. The registry centralizes discovery and ensures consistent formatter resolution across the codebase. It also enables health checking all formatters in one call.

### Factory Pattern (Formatter Constructors)

The `native` package provides factory functions for common formatters:

- `NewGoFormatter()` -- creates a gofmt formatter
- `NewPythonFormatter()` -- creates a black formatter
- `NewJSFormatter()` -- creates a prettier formatter
- `NewRustFormatter()` -- creates a rustfmt formatter
- `NewSQLFormatter()` -- creates a sqlformat formatter

Each factory encapsulates the metadata, binary path, default arguments, and stdin configuration. The `NewNativeFormatter()` function is the general-purpose factory for custom native formatters.

Similarly, `service.NewServiceFormatter()` creates HTTP service formatters with configurable endpoints.

**Why Factory**: Formatter construction requires significant metadata (name, version, languages, capabilities, binary paths, arguments). Factories encapsulate this complexity and provide sensible defaults.

### Chain of Responsibility (Middleware)

The executor implements middleware as a chain of responsibility. Each middleware wraps the next handler in the chain:

```
Request -> TimeoutMiddleware -> RetryMiddleware -> ValidationMiddleware -> Format()
```

The middleware signature is:

```go
type Middleware func(next ExecuteFunc) ExecuteFunc
```

Each middleware can:
1. Inspect or modify the request before passing it down.
2. Call `next(ctx, f, req)` to invoke the next handler.
3. Inspect or modify the result after it returns.
4. Short-circuit the chain by returning early (e.g., validation failure).

Built-in middleware:
- **TimeoutMiddleware**: Wraps execution in a context with deadline.
- **RetryMiddleware**: Retries with exponential backoff on failure.
- **ValidationMiddleware**: Validates non-empty input and non-empty output.

**Why Chain of Responsibility**: Cross-cutting concerns (timeouts, retries, validation, logging, metrics) should not be embedded in formatter implementations. Middleware provides clean separation and composability.

### Pipeline Pattern (Sequential Execution)

The `Pipeline` chains multiple formatters in sequence, feeding the output of each step as input to the next:

```
Input -> Formatter A -> Formatter B -> Formatter C -> Output
```

This enables multi-pass formatting. For example, a pipeline might first format syntax, then fix lint issues, then apply style rules.

If any step fails (`Success: false`), the pipeline short-circuits and returns that result.

**Why Pipeline**: Some formatting tasks require multiple tools in sequence. The pipeline provides a clean abstraction for this without requiring callers to manage intermediate results.

## Concurrency Model

### Registry

- Protected by `sync.RWMutex`.
- Reads (`Get`, `GetByLanguage`, `List`, `Count`) acquire read locks.
- Writes (`Register`, `Remove`) acquire write locks.
- `HealthCheckAll` uses a semaphore (buffered channel) to limit concurrent health checks to 10.

### Executor

- `Execute` is safe for concurrent use (no shared mutable state).
- `ExecuteBatch` launches goroutines per request and collects results via a channel.
- `BatchFormat` uses a semaphore for rate-limited concurrency.

### Cache

- Protected by `sync.RWMutex`.
- `Get` acquires a read lock.
- `Set`, `Invalidate`, `Clear` acquire write locks.
- A background goroutine runs periodic cleanup, stoppable via `Stop()`.

## Cache Key Strategy

Cache keys are SHA-256 hashes computed from three fields:

```go
h := sha256.New()
h.Write([]byte(req.Content))
h.Write([]byte(req.Language))
h.Write([]byte(req.FilePath))
return hex.EncodeToString(h.Sum(nil))
```

This means the same content formatted with different languages or file paths produces different cache entries. Configuration options (`Config`, `IndentSize`, `UseTabs`, `LineLength`) are intentionally excluded from the key to keep the caching model simple. If your application needs config-aware caching, implement a custom `FormatCache`.

## Eviction Strategy

The cache uses a simple oldest-first eviction when `MaxEntries` is reached. It iterates all entries to find the one with the earliest timestamp. This is O(n) but acceptable for typical cache sizes (10,000 entries by default). For larger caches, consider implementing a custom `FormatCache` with an LRU data structure.

## Error Handling Strategy

The module uses a two-tier error model:

1. **Go errors** (returned as `error`): Used for execution failures, missing formatters, network errors, timeouts, and other infrastructure problems. These indicate the formatting operation could not be attempted or completed.

2. **Result errors** (`FormatResult.Success` and `FormatResult.Error`): Used when the formatter ran but could not produce valid output. For example, a syntax error in the input that the formatter cannot handle. The native formatter returns `Success: false` with the stderr output when the binary exits with a non-zero code, but does not return a Go error.

This separation allows callers to distinguish between "could not format" and "formatted but with issues."

## Native Formatter Execution Model

Native formatters execute system binaries as child processes:

1. Build command arguments from defaults and request options.
2. Create an `exec.CommandContext` with the provided context for cancellation.
3. Pipe input via stdin (if `stdinFlag` is true).
4. Capture stdout (formatted output) and stderr (error messages).
5. Measure execution duration.
6. Compute line-level diff statistics.

Health checks execute the binary with `--version` to verify it is installed and accessible.

## Service Formatter HTTP Protocol

Service formatters use a simple JSON-over-HTTP protocol:

- **Format**: `POST {endpoint}{formatPath}` with `Content-Type: application/json`.
- **Health**: `GET {endpoint}{healthPath}` expecting `{"status": "healthy"}`.

The protocol is intentionally minimal to support any containerized formatter service. The `ServiceFormatRequest` and `ServiceFormatResponse` structs define the wire format.

## BaseFormatter Embedding

The `BaseFormatter` struct provides default implementations for identity methods (`Name`, `Version`, `Languages`), capability methods (`SupportsStdin`, `SupportsInPlace`, `SupportsCheck`, `SupportsConfig`), and configuration methods (`DefaultConfig`, `ValidateConfig`). Concrete formatters embed `BaseFormatter` and override only the methods that require custom behavior (typically `Format`, `FormatBatch`, and `HealthCheck`).

This reduces boilerplate and ensures consistent behavior across formatter implementations.

## Thread Safety Summary

| Component | Mechanism | Scope |
|-----------|-----------|-------|
| Registry | `sync.RWMutex` | All public methods |
| InMemoryCache | `sync.RWMutex` | All public methods |
| Executor | Stateless execution | Per-call goroutines |
| BatchFormat | Semaphore (buffered channel) | Rate limiting |
| HealthCheckAll | Semaphore (buffered channel) | Max 10 concurrent |
| Pipeline | Sequential | No concurrency |
