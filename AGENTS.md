# AGENTS.md

Multi-agent coordination guide for the Formatters module (`digital.vasic.formatters`).

## Module Overview

The Formatters module is a standalone, generic Go library that provides a reusable code formatting framework. It is designed to be consumed by any Go application that needs to format source code across multiple languages. The module supports native binary formatters, HTTP service-based formatters, execution pipelines with middleware, and in-memory result caching.

**Module path**: `digital.vasic.formatters`
**Go version**: 1.24.0
**External dependencies**: `github.com/stretchr/testify` (test only)

## Package Responsibilities

### pkg/formatter (Core)
- **Owner**: Foundation team
- **Responsibility**: Defines all core interfaces and types. Every other package depends on this one.
- **Key types**: `Formatter` interface, `FormatRequest`, `FormatResult`, `FormatStats`, `Options`, `Error`, `Result`, `FormatterMetadata`, `FormatterType`, `BaseFormatter`
- **Rule**: Changes here affect all downstream packages. Coordinate with all agents before modifying interfaces.

### pkg/registry (Discovery)
- **Owner**: Registry team
- **Responsibility**: Thread-safe formatter registration, lookup by name or language, language detection from file paths, health checking all formatters concurrently.
- **Key types**: `Registry`
- **Key functions**: `New`, `Default`, `RegisterDefault`, `GetDefault`, `DetectLanguageFromPath`
- **Rule**: The registry is the central coordination point. Agents adding new formatters must register through this package.

### pkg/native (Native Formatters)
- **Owner**: Formatter providers team
- **Responsibility**: Implements formatters that invoke system binaries (gofmt, black, prettier, rustfmt, sqlformat) via stdin/stdout.
- **Key types**: `NativeFormatter`
- **Key constructors**: `NewGoFormatter`, `NewPythonFormatter`, `NewJSFormatter`, `NewRustFormatter`, `NewSQLFormatter`
- **Rule**: Each new native formatter must provide metadata, a health check via `--version`, and support stdin-based formatting.

### pkg/service (Service Formatters)
- **Owner**: Infrastructure team
- **Responsibility**: Implements formatters that communicate with containerized formatting services over HTTP.
- **Key types**: `ServiceFormatter`, `Config`, `ServiceFormatRequest`, `ServiceFormatResponse`, `ServiceHealthResponse`
- **Rule**: Service formatters require a running HTTP endpoint. Health checks use `GET /health`, formatting uses `POST /format`.

### pkg/executor (Execution Engine)
- **Owner**: Pipeline team
- **Responsibility**: Orchestrates formatting execution with middleware chains, pipelines, and concurrent batch processing.
- **Key types**: `Executor`, `Config`, `Pipeline`, `Middleware`, `ExecuteFunc`
- **Key functions**: `New`, `DefaultExecutorConfig`, `NewPipeline`, `BatchFormat`, `TimeoutMiddleware`, `RetryMiddleware`, `ValidationMiddleware`
- **Rule**: Middleware must follow the chain-of-responsibility pattern. New middleware must be composable with existing middleware.

### pkg/cache (Caching)
- **Owner**: Performance team
- **Responsibility**: In-memory caching of format results with TTL expiration, max entry limits, and periodic cleanup.
- **Key types**: `FormatCache` interface, `InMemoryCache`, `Config`, `CacheStats`
- **Key functions**: `NewInMemoryCache`, `DefaultCacheConfig`
- **Rule**: Cache keys are SHA-256 hashes of content + language + file path. Always call `Stop()` to clean up the background goroutine.

## Agent Coordination Rules

1. **Interface changes require full coordination**: Any modification to `Formatter`, `FormatCache`, `Middleware`, or `ExecuteFunc` requires all agents to review and update their implementations.

2. **Registry is the single source of truth**: All formatter instances must be registered before use. The executor resolves formatters through the registry.

3. **Dependency direction is strict**: `formatter` has no internal dependencies. `registry` depends on `formatter`. `native` and `service` depend on `formatter`. `executor` depends on `formatter` and `registry`. `cache` depends on `formatter`. Never introduce circular dependencies.

4. **Testing independence**: Each package has self-contained tests. Unit tests use mocks/stubs. Integration tests may require system binaries (for native) or HTTP servers (for service).

5. **Concurrency safety**: The registry uses `sync.RWMutex`. The cache uses `sync.RWMutex`. The executor uses channels and goroutines with semaphores. Agents must maintain thread safety when modifying shared state.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/formatter/formatter.go` | Core interfaces, types, and BaseFormatter |
| `pkg/formatter/formatter_test.go` | Core type tests |
| `pkg/registry/registry.go` | Registry with language detection |
| `pkg/registry/registry_test.go` | Registry tests |
| `pkg/native/native.go` | Native binary formatter + 5 constructors |
| `pkg/native/native_test.go` | Native formatter tests |
| `pkg/service/service.go` | HTTP service formatter |
| `pkg/service/service_test.go` | Service formatter tests |
| `pkg/executor/executor.go` | Executor, Pipeline, BatchFormat, middleware |
| `pkg/executor/executor_test.go` | Executor and middleware tests |
| `pkg/cache/cache.go` | InMemoryCache with TTL |
| `pkg/cache/cache_test.go` | Cache tests |
| `go.mod` | Module definition |
| `CLAUDE.md` | AI assistant guidance |
| `README.md` | Project overview |

## Test Commands

```bash
# All tests with race detection
go test ./... -count=1 -race

# Individual package tests
go test ./pkg/formatter/... -v
go test ./pkg/registry/... -v
go test ./pkg/native/... -v
go test ./pkg/service/... -v
go test ./pkg/executor/... -v
go test ./pkg/cache/... -v

# Run a specific test
go test -v -run TestRegistry_Register ./pkg/registry/...

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Adding a New Formatter

1. Decide the formatter type: native (binary), service (HTTP), or custom.
2. Create the formatter struct implementing `formatter.Formatter`.
3. Embed `formatter.BaseFormatter` for common functionality.
4. Provide `FormatterMetadata` with all required fields.
5. Register the formatter with the registry.
6. Add tests covering Format, FormatBatch, HealthCheck, and edge cases.
7. Update documentation if the formatter introduces new patterns.

## Adding New Middleware

1. Implement the `Middleware` function signature: `func(next ExecuteFunc) ExecuteFunc`.
2. Ensure the middleware calls `next(ctx, f, req)` to continue the chain.
3. Handle errors from `next` appropriately.
4. Add the middleware to the executor via `executor.Use()`.
5. Write tests verifying the middleware in isolation and within a chain.



---

## Universal Mandatory Constraints

> Cascaded from the HelixAgent root `CLAUDE.md` via `/tmp/UNIVERSAL_MANDATORY_RULES.md`.
> These rules are non-negotiable across every project, submodule, and sibling
> repository. Project-specific addenda are welcome but cannot weaken or
> override these.

### Hard Stops (permanent, non-negotiable)

1. **NO CI/CD pipelines.** No `.github/workflows/`, `.gitlab-ci.yml`,
   `Jenkinsfile`, `.travis.yml`, `.circleci/`, or any automated pipeline.
   No Git hooks either. All builds and tests run manually or via
   Makefile/script targets.
2. **NO HTTPS for Git.** SSH URLs only (`git@github.com:…`,
   `git@gitlab.com:…`, etc.) for clones, fetches, pushes, and submodule
   updates. Including for public repos. SSH keys are configured on every
   service.
3. **NO manual container commands.** Container orchestration is owned by
   the project's binary/orchestrator (e.g. `make build` → `./bin/<app>`).
   Direct `docker`/`podman start|stop|rm` and `docker-compose up|down`
   are prohibited as workflows. The orchestrator reads its configured
   `.env` and brings up everything.

### Mandatory Development Standards

1. **100% Test Coverage.** Every component MUST have unit, integration,
   E2E, automation, security/penetration, and benchmark tests. No false
   positives. Mocks/stubs ONLY in unit tests; all other test types use
   real data and live services.
2. **Challenge Coverage.** Every component MUST have Challenge scripts
   (`./challenges/scripts/`) validating real-life use cases. No false
   success — validate actual behavior, not return codes.
3. **Real Data.** Beyond unit tests, all components MUST use actual API
   calls, real databases, live services. No simulated success. Fallback
   chains tested with actual failures.
4. **Health & Observability.** Every service MUST expose health
   endpoints. Circuit breakers for all external dependencies.
   Prometheus / OpenTelemetry integration where applicable.
5. **Documentation & Quality.** Update `CLAUDE.md`, `AGENTS.md`, and
   relevant docs alongside code changes. Pass language-appropriate
   format/lint/security gates. Conventional Commits:
   `<type>(<scope>): <description>`.
6. **Validation Before Release.** Pass the project's full validation
   suite (`make ci-validate-all`-equivalent) plus all challenges
   (`./challenges/scripts/run_all_challenges.sh`).
7. **No Mocks or Stubs in Production.** Mocks, stubs, fakes,
   placeholder classes, TODO implementations are STRICTLY FORBIDDEN in
   production code. All production code is fully functional with real
   integrations. Only unit tests may use mocks/stubs.
8. **Comprehensive Verification.** Every fix MUST be verified from all
   angles: runtime testing (actual HTTP requests / real CLI
   invocations), compile verification, code structure checks,
   dependency existence checks, backward compatibility, and no false
   positives in tests or challenges. Grep-only validation is NEVER
   sufficient.
9. **Resource Limits for Tests & Challenges (CRITICAL).** ALL test and
   challenge execution MUST be strictly limited to 30-40% of host
   system resources. Use `GOMAXPROCS=2`, `nice -n 19`, `ionice -c 3`,
   `-p 1` for `go test`. Container limits required. The host runs
   mission-critical processes — exceeding limits causes system crashes.
10. **Bugfix Documentation.** All bug fixes MUST be documented in
    `docs/issues/fixed/BUGFIXES.md` (or the project's equivalent) with
    root cause analysis, affected files, fix description, and a link to
    the verification test/challenge.
11. **Real Infrastructure for All Non-Unit Tests.** Mocks/fakes/stubs/
    placeholders MAY be used ONLY in unit tests (files ending
    `_test.go` run under `go test -short`, equivalent for other
    languages). ALL other test types — integration, E2E, functional,
    security, stress, chaos, challenge, benchmark, runtime
    verification — MUST execute against the REAL running system with
    REAL containers, REAL databases, REAL services, and REAL HTTP
    calls. Non-unit tests that cannot connect to real services MUST
    skip (not fail).
12. **Reproduction-Before-Fix (CONST-032 — MANDATORY).** Every reported
    error, defect, or unexpected behavior MUST be reproduced by a
    Challenge script BEFORE any fix is attempted. Sequence:
    (1) Write the Challenge first. (2) Run it; confirm fail (it
    reproduces the bug). (3) Then write the fix. (4) Re-run; confirm
    pass. (5) Commit Challenge + fix together. The Challenge becomes
    the regression guard for that bug forever.
13. **Concurrent-Safe Containers (Go-specific, where applicable).** Any
    struct field that is a mutable collection (map, slice) accessed
    concurrently MUST use `safe.Store[K,V]` / `safe.Slice[T]` from
    `digital.vasic.concurrency/pkg/safe` (or the project's equivalent
    primitives). Bare `sync.Mutex + map/slice` combinations are
    prohibited for new code.

### Definition of Done (universal)

A change is NOT done because code compiles and tests pass. "Done"
requires pasted terminal output from a real run, produced in the same
session as the change.

- **No self-certification.** Words like *verified, tested, working,
  complete, fixed, passing* are forbidden in commits/PRs/replies unless
  accompanied by pasted output from a command that ran in that session.
- **Demo before code.** Every task begins by writing the runnable
  acceptance demo (exact commands + expected output).
- **Real system, every time.** Demos run against real artifacts.
- **Skips are loud.** `t.Skip` / `@Ignore` / `xit` / `describe.skip`
  without a trailing `SKIP-OK: #<ticket>` comment break validation.
- **Evidence in the PR.** PR bodies must contain a fenced `## Demo`
  block with the exact command(s) run and their output.
