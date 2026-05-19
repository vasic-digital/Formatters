# Formatters

Generic, reusable code-formatting module for Go applications. Pluggable
formatter registry + execution engine + middleware + native-binary shims
+ result cache, all built around a single `formatter.Formatter` interface.

This module is **project-not-aware**: it makes no assumption about the
consuming project (HelixCode, HelixQA, third-party). Every dependency on
host state (binaries, network, filesystem) is opt-in via constructor
parameters or `FormatRequest` fields.

```
module digital.vasic.formatters
go     1.24+
```

## Architecture at a glance

```
┌────────────────────────┐         ┌────────────────────────┐
│   pkg/formatter        │◄────────│   pkg/native           │
│   (core contracts)     │         │  NativeFormatter +     │
│   Formatter, Request,  │         │  Go/Python/JS/Rust/SQL │
│   Result, Metadata     │         └────────────────────────┘
└──────────┬─────────────┘                    ▲
           │ implements                       │ register
           ▼                                  │
┌────────────────────────┐         ┌─────────┴──────────────┐
│   pkg/registry         │◄────────│   pkg/executor         │
│  Registry, Default(),  │         │  Executor, Middleware, │
│  language detection    │         │  Pipeline, BatchFormat │
└────────────────────────┘         └─────────┬──────────────┘
                                             │ optional
                                             ▼
                                   ┌────────────────────────┐
                                   │   pkg/cache            │
                                   │  InMemoryCache (TTL +  │
                                   │  size-bounded LRU)     │
                                   └────────────────────────┘
```

## Packages

| Package         | Description |
|-----------------|-------------|
| `pkg/formatter` | Core `Formatter` interface, `FormatRequest`, `FormatResult`, `FormatStats`, `FormatterMetadata`, `BaseFormatter` helper. |
| `pkg/registry`  | Thread-safe registry; `New`, `Register`, `Get`, `List`, `GetByLanguage`, `DetectFormatter`, `DetectLanguageFromPath`, `HealthCheckAll`, package-level `Default()` singleton. |
| `pkg/native`    | `NativeFormatter` wraps an external binary via `os/exec`; ships `New{Go,Python,JS,Rust,SQL}Formatter` for gofmt / black / prettier / rustfmt / sqlformat. |
| `pkg/service`   | HTTP/RPC service-based formatters for containerised tools. |
| `pkg/executor`  | `Executor` (lang + filepath dispatch), `Pipeline` (compose steps), `BatchFormat` (concurrent fan-out), `{Timeout,Retry,Validation}Middleware`. |
| `pkg/cache`     | `InMemoryCache` implementing the `FormatCache` interface with TTL, size cap, background cleanup. |
| `pkg/textformat`| Pure-Go text-formatting transforms (no external binaries). |

## Usage

```go
import (
    "context"
    "time"

    "digital.vasic.formatters/pkg/executor"
    "digital.vasic.formatters/pkg/formatter"
    "digital.vasic.formatters/pkg/native"
    "digital.vasic.formatters/pkg/registry"
)

reg := registry.New()
_ = reg.Register(native.NewGoFormatter())
_ = reg.Register(native.NewPythonFormatter())

exec := executor.New(reg, executor.DefaultExecutorConfig())
exec.Use(executor.TimeoutMiddleware(30 * time.Second))
exec.Use(executor.ValidationMiddleware())

result, err := exec.Execute(context.Background(), &formatter.FormatRequest{
    Content:  "package main\nfunc main(){}\n",
    Language: "go",
})
if err != nil {
    // real error from real formatter execution — surface, do not swallow
}
_ = result.Content
```

## Testing

```bash
go test -race -count=1 ./pkg/...                   # all unit tests
go run ./challenges/runner -lang=en                # English Challenge run
go run ./challenges/runner -lang=sr                # Serbian Challenge run
./challenges/formatters_describe_challenge.sh      # baseline + 8 paired mutations
```

A green `go test -race ./pkg/...` plus exit-0 from
`formatters_describe_challenge.sh` is the Definition-of-Done for this
module (CONST-035 / Rule 8 — output evidence, not adjectives).

See `docs/test-coverage.md` for the symbol→test ledger.

## Anti-bluff guarantees (CONST-035 / Article XI §11.9)

> "We had been in position that all tests do execute with success and all
> Challenges as well, but in reality the most of the features does not
> work and can't be used! This MUST NOT be the case…" — 2026-05-19 mandate

This module commits to the following invariants. Every one is enforced by
an executable check captured in `challenges/runner/main.go`:

1. **Registry is real.** `Register` + `Get` + `List` + `GetByLanguage` round-trip
   real `Formatter` instances; no constructor-only PASS (check1; mutation
   `skip_registry`).
2. **Executor preserves bytes.** Bilingual fixtures (English + Serbian Latin
   across Go / Python / SQL) round-trip byte-exact through
   `executor.Execute` against the real registry (check2; mutation
   `corrupt_roundtrip`).
3. **Middleware actually fires.** Timeout + Validation middleware is composed
   and the underlying `Format` is invoked exactly once (check3; mutation
   `no_format_call`).
4. **Pipeline composes.** `executor.NewPipeline(a, b, c)` runs all three
   steps in order; each step's call counter is exactly 1 (check4; mutation
   `skip_steps`).
5. **Cache stores + retrieves.** `cache.InMemoryCache.Set` followed by `Get`
   returns the stored `FormatResult` with content preserved (check5;
   mutation `cache_miss`).
6. **Native constructors are honest.** All five `native.New*Formatter`
   helpers produce `Formatter`s whose `Name()` + `Languages()` match the
   metadata they advertise (check6; mutation `constructor_drift`).
7. **Language detection is correct.** `registry.DetectLanguageFromPath`
   resolves 5 representative extensions correctly (check7; mutation
   `detect_drift`).
8. **Negative-leg is real.** `executor.Execute` rejects an ambiguous
   `FormatRequest` (no Language, no FilePath) with a non-nil error (check8;
   mutation `accept_ambiguous`).

The paired-mutation gate (`formatters_describe_challenge.sh`) flips each
invariant in turn via `MUTATE=<name>`; the runner exits 99, the gate
records the expected failure, and the gate exits 0 only when every
mutation flipped its target. A check that cannot be broken on demand
cannot be trusted to detect a real regression (§1.1).

## Bilingual fixtures (CONST-046 — no hardcoded English-only output)

The runner is locale-aware (`-lang=en|sr|all`) and ships fixtures in two
locales so the byte-exact round-trip test cannot silently regress for
non-English content:

| Fixture                   | Language | Locale         |
|---------------------------|----------|----------------|
| `go_hello_en.go`          | Go       | English        |
| `go_hello_sr.go`          | Go       | Serbian Latin  |
| `python_hello_en.py`      | Python   | English        |
| `python_hello_sr.py`      | Python   | Serbian Latin  |
| `sql_query_en.sql`        | SQL      | English        |

Adding a third locale = drop one file per language into
`challenges/fixtures/` and append a `fixture{}` entry in
`challenges/runner/main.go`. The labels map under `labels[locale]` covers
every user-facing string the runner emits.

## Governance

This repository inherits the HelixCode constitution at every level:

- `CLAUDE.md`, `AGENTS.md`, `CONSTITUTION.md` — full text cascaded from
  the constitution submodule (CONST-047 / CONST-059).
- CONST-035 anti-bluff posture, CONST-046 no-hardcoded-strings, CONST-048
  full-automation-coverage, CONST-050 no-fakes-beyond-unit-tests,
  CONST-051 equal-codebase + decoupling, CONST-053 .gitignore mandate,
  CONST-060 fetch-before-edit — all binding here.
- Round-255 §11.4 deliverable: see `docs/test-coverage.md` for the
  symbol→test ledger and `challenges/runner/main.go` for the captured
  evidence runner.

No CI/CD pipelines (Rule 1): `make test`, `./challenges/...` scripts, and
the runner are invoked manually or by the parent project's test runner.
