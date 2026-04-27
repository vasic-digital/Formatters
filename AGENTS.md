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

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->



<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.
