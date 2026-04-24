# CLAUDE.md


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Native Go + Python formatters through the registry/executor/cache pipeline
cd Formatters && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./pkg/...
```
Expect: PASS; exercises `registry.New`, `executor.New` with `TimeoutMiddleware`, and `native.NewGoFormatter` per `Formatters/README.md`. Service formatters additionally need their Docker images built (`docker compose -f docker/formatters/docker-compose.yml up -d`).


This file provides guidance to Claude Code when working with the Formatters module.

## Module Overview

`digital.vasic.formatters` is a standalone, generic Go module providing a reusable code formatting framework. It supports native binary formatters, service-based (HTTP/Docker) formatters, execution pipelines, result caching, and a thread-safe formatter registry.

**Module**: `digital.vasic.formatters` (Go 1.24.0)

## Packages

### pkg/formatter - Core interfaces and types
- `Formatter` interface: Format, FormatBatch, HealthCheck, Name, Version, Languages, capabilities
- `BaseFormatter` embeddable base implementation
- `FormatRequest`, `FormatResult`, `FormatStats`, `Error`, `Result`, `Options`
- `FormatterMetadata`, `FormatterType` constants (native, service, builtin, unified)

### pkg/registry - Formatter registry
- `Registry` with Register, Get, GetByLanguage, List, Remove, ListByType, DetectFormatter
- Thread-safe with RWMutex
- Default singleton via `Default()`
- `DetectLanguageFromPath()` extension-based detection
- `HealthCheckAll()` with bounded concurrency

### pkg/native - Native binary formatters
- `NativeFormatter` executing system binaries via stdin/stdout
- Built-in constructors: NewGoFormatter, NewPythonFormatter, NewJSFormatter, NewRustFormatter, NewSQLFormatter
- Health checks via `--version` flag

### pkg/service - HTTP service formatters
- `ServiceFormatter` calling containerized formatters via REST
- `Config` with endpoint, timeout, health/format paths
- Health check via GET /health, format via POST /format

### pkg/executor - Execution engine
- `Executor` with middleware chain (timeout, retry, validation)
- `Pipeline` chaining multiple formatters sequentially
- `BatchFormat()` with rate-limited concurrency
- Middleware: TimeoutMiddleware, RetryMiddleware, ValidationMiddleware

### pkg/cache - Result caching
- `FormatCache` interface (Get, Set, Invalidate)
- `InMemoryCache` with SHA-256 content hash keys
- TTL expiration, max entries, periodic cleanup

### pkg/textformat - Cross-platform text format types
- Types and interfaces mirroring Formatters-KMP for text format detection, parsing, and registry; used to share format definitions across Go and Kotlin consumers

## Build & Test

```bash
go test ./... -count=1 -race    # All tests with race detection
go test ./pkg/formatter/...     # Formatter package tests
go test ./pkg/registry/...      # Registry tests
go test ./pkg/native/...        # Native formatter tests
go test ./pkg/service/...       # Service formatter tests (httptest)
go test ./pkg/executor/...      # Executor tests
go test ./pkg/cache/...         # Cache tests
```

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports: stdlib, third-party, internal (blank line separated)
- Line length <= 100 chars
- Table-driven tests with testify
- No external dependencies beyond stretchr/testify

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | HelixLLM |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.
