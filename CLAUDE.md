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

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

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
