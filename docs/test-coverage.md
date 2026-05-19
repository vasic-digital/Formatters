# Formatters — symbol→test ledger (round 255)

Per CONST-048 § "coverage ledger" + CONST-050(B) "100% test-type coverage",
every exported symbol in `digital.vasic.formatters` MUST be exercised by at
least one captured-evidence test or Challenge check. This document is the
authoritative mapping. Drift between this ledger and the source tree is a
CONST-048 violation of equal severity to a missing test.

## How to verify

```bash
go test -race -count=1 ./pkg/...                          # all unit tests
go run ./challenges/runner -lang=en                       # English Challenge
go run ./challenges/runner -lang=sr                       # Serbian Challenge
./challenges/formatters_describe_challenge.sh             # baseline + 8 mutations
```

A green `make test` + `formatters_describe_challenge.sh` exit 0 is the
documented Definition-of-Done for this module (CONST-035 / Rule 8).

## Coverage table

### `pkg/formatter` — core contracts

| Symbol                              | Unit test                                              | Challenge check                                  |
|-------------------------------------|--------------------------------------------------------|--------------------------------------------------|
| `Formatter` (interface)             | implemented + asserted by every constructor test       | check6 (5 constructors satisfy interface)        |
| `Options`, `FormatRequest`          | `formatter_test.go` field round-trips                  | check2 (bilingual round-trip via real request)   |
| `FormatResult`, `FormatStats`       | `formatter_test.go` populated by `BaseFormatter`       | check3, check5 (chain + cache observe stats)     |
| `Error`, `Result`                   | `formatter_test.go` struct shape                       | check2 negative-leg via exec error               |
| `FormatterType` constants           | `formatter_test.go` literal compares                   | check6 (native uses TypeBuiltin/Native/Unified)  |
| `FormatterMetadata`                 | `formatter_test.go` field-by-field                     | check6 metadata read for 5 formatters            |
| `BaseFormatter.{Name,Version,Languages,SupportsStdin,SupportsInPlace,SupportsCheck,SupportsConfig,DefaultConfig,ValidateConfig,Metadata}` | `formatter_test.go` table-driven | check6 reads `Name()` + `Languages()`           |
| `NewBaseFormatter`                  | `formatter_test.go` constructor                        | check6 5× indirect via native constructors       |

### `pkg/registry` — registry + language detection

| Symbol                              | Unit test                                              | Challenge check                                  |
|-------------------------------------|--------------------------------------------------------|--------------------------------------------------|
| `Registry`, `New`                   | `registry_test.go` `TestNew*`                          | check1 (registry constructed + Register/List)    |
| `Register`, `RegisterWithMetadata`  | `registry_test.go` duplicate + happy paths             | check1, check2, check3, check4 setup             |
| `Get`, `GetByLanguage`, `List`      | `registry_test.go` lookup tables                       | check1 (3 lookups asserted)                      |
| `DetectFormatter`                   | `registry_test.go` extension matrix                    | check2 (when FilePath used)                      |
| `DetectLanguageFromPath`            | `registry_test.go` ~50-extension table                 | check7 (5 path→language pairs)                   |
| `HealthCheckAll`                    | `registry_test.go` concurrent fan-out                  | (covered by unit test — no host-binary dep)      |
| `Default`, `RegisterDefault`, `GetDefault` | `registry_test.go` singleton                    | (unit-only; singleton state unsafe for Challenge)|

### `pkg/native` — built-in / external formatter shims

| Symbol                              | Unit test                                              | Challenge check                                  |
|-------------------------------------|--------------------------------------------------------|--------------------------------------------------|
| `FormatFunc`, `NativeFormatter`     | `native_test.go` with `SetFormatFuncForTest`           | check6 (5 constructors exposed)                  |
| `NewNativeFormatter`                | `native_test.go` happy + edge                          | check6                                           |
| `Format`, `FormatBatch`             | `native_test.go` with injected FormatFunc              | (unit — host gofmt/black/etc. not guaranteed)    |
| `SetFormatFuncForTest`              | `native_test.go` injection                             | (unit-only test seam)                            |
| `HealthCheck`                       | `native_test.go` binary-present + missing branches     | (unit — host binary not guaranteed)              |
| `ValidateConfig`, `DefaultConfig`   | inherited from BaseFormatter — `native_test.go`        | check6 indirect                                  |
| `NewGoFormatter`, `NewPythonFormatter`, `NewJSFormatter`, `NewRustFormatter`, `NewSQLFormatter` | `native_test.go` per-constructor | check6 (all 5 produce expected name+language)   |

### `pkg/executor` — execution engine + middleware

| Symbol                              | Unit test                                              | Challenge check                                  |
|-------------------------------------|--------------------------------------------------------|--------------------------------------------------|
| `Middleware`, `ExecuteFunc`         | `executor_test.go` chain composition                   | check3 (real chain executes)                     |
| `Config`, `DefaultExecutorConfig`   | `executor_test.go` defaults                            | check2, check3, check8                           |
| `Executor`, `New`                   | `executor_test.go` constructor                         | check2, check3, check8                           |
| `Execute`                           | `executor_test.go` lang + filepath paths               | check2 (5 bilingual fixtures), check8 (negative) |
| `ExecuteBatch`                      | `executor_test.go` concurrent fan-out                  | (unit — runner uses single Execute calls)        |
| `Use`                               | `executor_test.go` registration                        | check3 (timeout + validation registered)         |
| `Pipeline`, `NewPipeline`, `Execute`| `executor_test.go` 3-step chain                        | check4 (3-step Pipeline real execution)          |
| `BatchFormat`                       | `executor_test.go` fan-out                             | (unit-covered)                                   |
| `TimeoutMiddleware`                 | `executor_test.go` deadline assertion                  | check3 (registered in chain)                     |
| `RetryMiddleware`                   | `executor_test.go` retry counting                      | (unit-covered; deterministic retries)            |
| `ValidationMiddleware`              | `executor_test.go` request shape                       | check3 (registered in chain)                     |

### `pkg/cache` — in-memory result cache

| Symbol                              | Unit test                                              | Challenge check                                  |
|-------------------------------------|--------------------------------------------------------|--------------------------------------------------|
| `FormatCache` (interface)           | `cache_test.go` via `InMemoryCache`                    | check5 (real Get/Set round-trip)                 |
| `Config`, `DefaultCacheConfig`      | `cache_test.go` defaults                               | check5                                           |
| `InMemoryCache`, `NewInMemoryCache` | `cache_test.go` lifecycle + TTL                        | check5                                           |
| `Get`, `Set`, `Invalidate`, `Clear` | `cache_test.go` table-driven                           | check5 (Set + Get; paired mutation `cache_miss`) |
| `Size`, `Stop`, `Stats`             | `cache_test.go` introspection                          | check5 reads `Size()`                            |
| `CacheStats`                        | `cache_test.go`                                        | check5 reads via `Stats()`                       |

### `pkg/service` / `pkg/textformat`

| Package                             | Status                                                 |
|-------------------------------------|--------------------------------------------------------|
| `pkg/service`                       | unit test PASS (`make test`); no host-side endpoints required by the runner — round 255 extends coverage opportunistically and tracks gap in next round |
| `pkg/textformat`                    | unit test PASS (`make test`); pure-Go transforms — Challenge coverage tracked for round 256 |

## Paired-mutation gate (round 255)

`challenges/formatters_describe_challenge.sh` runs 2 baselines + 8 mutations.
Each mutation flips exactly one check; passing the gate means every check is
falsifiable (not a green-by-construction bluff). Mutation roster:

| Mutation             | Targets check | Failure mode injected                                                |
|----------------------|---------------|----------------------------------------------------------------------|
| `skip_registry`      | check1        | `Register` calls bypassed; `List()` size assertion fails             |
| `corrupt_roundtrip`  | check2        | Result bytes mutated post-format; byte-exact compare fails           |
| `no_format_call`     | check3        | Counter clobbered to 0; "underlying Format invoked" assertion fails  |
| `skip_steps`         | check4        | Step-A counter clobbered to 0; pipeline-step-count assertion fails   |
| `cache_miss`         | check5        | Lookup uses a different key; Get reports miss                        |
| `constructor_drift`  | check6        | Synthetic drift entry forced into bad-list                           |
| `detect_drift`       | check7        | Detected language overridden to `"wrong"`; comparison fails          |
| `accept_ambiguous`   | check8        | Negative-leg `err` cleared; ambiguous-request rejection bluff exposed|

Adding a new check requires adding a paired mutation in the same commit
(§1.1: every assertion must be falsifiable on demand).

## Evidence capture

`make test`, the runner, and the paired-mutation gate ALL print per-check
evidence lines (e.g. `evidence: List=[identity-go identity-py]`). The gate
script tees per-mutation output to `/tmp/formatters_round255_mut_*.log` for
post-run forensic review (CONST-035 / Article XI §11.9).
