# Changelog

All notable changes to the `digital.vasic.formatters` module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-03

### Added

- **pkg/formatter**: Core `Formatter` interface with 11 methods covering identity, capabilities, formatting, health, and configuration.
- **pkg/formatter**: `BaseFormatter` embeddable struct providing default implementations for identity and capability methods.
- **pkg/formatter**: `FormatRequest` and `FormatResult` types for formatting input/output.
- **pkg/formatter**: `FormatStats` type for line/byte change statistics.
- **pkg/formatter**: `Error` and `Result` types for detailed diagnostics.
- **pkg/formatter**: `FormatterMetadata` type with 18 fields covering name, type, architecture, installation, performance, and capabilities.
- **pkg/formatter**: `FormatterType` constants: `native`, `service`, `builtin`, `unified`.
- **pkg/formatter**: `Options` type for indent size, tabs, line width, and style.
- **pkg/registry**: Thread-safe `Registry` with `Register`, `Get`, `GetByLanguage`, `List`, `Remove`, `Count`, `GetMetadata`, `ListByType`, `DetectFormatter`, and `HealthCheckAll`.
- **pkg/registry**: `RegisterWithMetadata` for associating `FormatterMetadata` with registered formatters.
- **pkg/registry**: `DetectLanguageFromPath` function supporting 40+ file extensions.
- **pkg/registry**: Default singleton via `Default()`, `RegisterDefault()`, `GetDefault()`.
- **pkg/registry**: Concurrent health checks with semaphore limiting (max 10 parallel).
- **pkg/native**: `NativeFormatter` executing system binaries via stdin/stdout with `exec.CommandContext`.
- **pkg/native**: `NewGoFormatter` (gofmt), `NewPythonFormatter` (black), `NewJSFormatter` (prettier), `NewRustFormatter` (rustfmt), `NewSQLFormatter` (sqlformat).
- **pkg/native**: Line-level change computation via `computeLineChanges`.
- **pkg/native**: Health checks via `--version` flag.
- **pkg/service**: `ServiceFormatter` communicating with containerized formatters over HTTP (JSON protocol).
- **pkg/service**: `Config` with endpoint, timeout, health path, and format path.
- **pkg/service**: `DefaultConfig` factory function.
- **pkg/service**: `ServiceFormatRequest`, `ServiceFormatResponse`, `ServiceHealthResponse` wire types.
- **pkg/executor**: `Executor` with middleware chain of responsibility pattern.
- **pkg/executor**: `Middleware` and `ExecuteFunc` types.
- **pkg/executor**: `Execute` method resolving formatters by language or file path via registry.
- **pkg/executor**: `ExecuteBatch` for concurrent multi-request execution.
- **pkg/executor**: `Pipeline` for sequential multi-formatter chaining.
- **pkg/executor**: `BatchFormat` function with semaphore-based rate limiting.
- **pkg/executor**: `TimeoutMiddleware` with per-request and default timeouts.
- **pkg/executor**: `RetryMiddleware` with exponential backoff (capped at 30 retries).
- **pkg/executor**: `ValidationMiddleware` for empty input/output rejection.
- **pkg/executor**: `DefaultExecutorConfig` with 30s timeout, 3 retries, 10 concurrent.
- **pkg/cache**: `FormatCache` interface with `Get`, `Set`, `Invalidate`.
- **pkg/cache**: `InMemoryCache` with SHA-256 content hash keys.
- **pkg/cache**: TTL-based expiration with lazy check on `Get`.
- **pkg/cache**: Periodic background cleanup via goroutine.
- **pkg/cache**: Oldest-first eviction when `MaxEntries` is reached.
- **pkg/cache**: `CacheStats` for size, capacity, and TTL reporting.
- **pkg/cache**: `DefaultCacheConfig` with 10000 entries, 1h TTL, 5min cleanup.
- Unit tests for all 6 packages using `testify`.
