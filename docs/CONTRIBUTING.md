# Contributing

Guidelines for contributing to the `digital.vasic.formatters` module.

## Prerequisites

- Go 1.24 or later
- `gofmt` and `goimports` installed
- `golangci-lint` installed (for linting)
- `testify` is the only test dependency and is already in `go.mod`

## Getting Started

1. Clone the repository (SSH only):
   ```bash
   git clone <ssh-url>
   cd Formatters
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run tests to verify your setup:
   ```bash
   go test ./... -count=1 -race
   ```

## Development Workflow

### Branch Naming

Use conventional branch prefixes:
- `feat/` -- New formatter, middleware, or feature
- `fix/` -- Bug fix
- `refactor/` -- Code restructuring without behavior change
- `test/` -- Test additions or improvements
- `docs/` -- Documentation changes
- `chore/` -- Maintenance tasks

Examples: `feat/add-ruby-formatter`, `fix/cache-ttl-expiry`, `test/executor-batch-coverage`.

### Commit Messages

Follow Conventional Commits:
```
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`.

Scopes: `formatter`, `registry`, `native`, `service`, `executor`, `cache`.

Examples:
```
feat(native): add Ruby formatter using rufo
fix(cache): correct TTL expiry check for zero timestamps
test(executor): add batch processing edge case tests
refactor(registry): simplify language detection logic
docs(api): update FormatResult field descriptions
```

### Code Style

- Standard Go conventions per [Effective Go](https://go.dev/doc/effective_go).
- Format with `gofmt`. Use `goimports` for import ordering.
- Imports grouped in three blocks separated by blank lines: stdlib, third-party, internal.
- Line length: 100 characters maximum.
- Naming conventions:
  - `camelCase` for unexported identifiers.
  - `PascalCase` for exported identifiers.
  - `UPPER_SNAKE_CASE` for constants.
  - Acronyms in all caps: `HTTP`, `URL`, `ID`, `TTL`, `JSON`.
- Receiver names: 1-2 letters (e.g., `r` for Registry, `c` for cache, `n` for NativeFormatter).
- Always check errors. Wrap with `fmt.Errorf("...: %w", err)`.
- Use `defer` for cleanup (closing resources, releasing locks).
- Interfaces: small and focused. Accept interfaces, return structs.
- Concurrency: always use `context.Context`. Protect shared state with `sync.Mutex` or `sync.RWMutex`.

### Before Committing

Run the full quality check:
```bash
gofmt -l ./...          # Check formatting
go vet ./...            # Static analysis
go test ./... -count=1 -race  # Tests with race detection
```

All three must pass with zero issues before committing.

## Adding a New Formatter

### Native Formatter

1. Add a constructor function in `pkg/native/native.go`:
   ```go
   func NewRubyFormatter() *NativeFormatter {
       metadata := &formatter.FormatterMetadata{
           Name:          "rufo",
           Type:          formatter.FormatterTypeNative,
           Version:       "0.18.0",
           Languages:     []string{"ruby"},
           SupportsStdin: true,
           // ... fill all fields
       }
       return NewNativeFormatter(metadata, "rufo", []string{}, true)
   }
   ```

2. Add tests in `pkg/native/native_test.go` covering:
   - Formatter creation and metadata verification.
   - Format with valid input (requires the binary on PATH).
   - Format with empty input.
   - HealthCheck when binary is available.
   - HealthCheck when binary is missing.

### Service Formatter

1. Create the formatter using `NewServiceFormatter`:
   ```go
   svcFmt := service.NewServiceFormatter(metadata, config)
   ```

2. Add tests using `net/http/httptest` to mock the HTTP service.

### Custom Formatter

1. Implement the `formatter.Formatter` interface.
2. Embed `formatter.BaseFormatter` for common methods.
3. Implement at minimum: `Format`, `FormatBatch`, `HealthCheck`.
4. Register with the registry.
5. Add comprehensive tests.

## Adding New Middleware

1. Implement the `executor.Middleware` signature:
   ```go
   func MyMiddleware(param Type) executor.Middleware {
       return func(next executor.ExecuteFunc) executor.ExecuteFunc {
           return func(ctx context.Context, f formatter.Formatter, req *formatter.FormatRequest) (*formatter.FormatResult, error) {
               // Pre-processing
               result, err := next(ctx, f, req)
               // Post-processing
               return result, err
           }
       }
   }
   ```

2. Always call `next` unless you intend to short-circuit.
3. Add tests verifying the middleware in isolation and composed with other middleware.

## Testing Guidelines

### Test Naming

Use the pattern `Test<Struct>_<Method>_<Scenario>`:
```go
func TestRegistry_Register_DuplicateName(t *testing.T) { ... }
func TestNativeFormatter_Format_EmptyContent(t *testing.T) { ... }
func TestInMemoryCache_Get_ExpiredEntry(t *testing.T) { ... }
```

### Table-Driven Tests

Prefer table-driven tests:
```go
func TestDetectLanguageFromPath(t *testing.T) {
    tests := []struct {
        name     string
        path     string
        expected string
    }{
        {"go file", "main.go", "go"},
        {"python file", "app.py", "python"},
        {"unknown", "Makefile", ""},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            result := registry.DetectLanguageFromPath(tc.path)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### Test Dependencies

- Use `testify/assert` for assertions and `testify/require` for fatal checks.
- Use `net/http/httptest` for service formatter tests.
- Use mock formatters (implementing `formatter.Formatter`) in executor and registry tests.
- Mocks are permitted in unit tests only.

### Running Tests

```bash
# All tests
go test ./... -count=1 -race

# Specific package
go test ./pkg/cache/... -v

# Specific test
go test -v -run TestInMemoryCache_Get_ExpiredEntry ./pkg/cache/...

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Implementing the FormatCache Interface

To add a new cache backend (e.g., Redis), implement the `FormatCache` interface:

```go
type FormatCache interface {
    Get(req *formatter.FormatRequest) (*formatter.FormatResult, bool)
    Set(req *formatter.FormatRequest, result *formatter.FormatResult)
    Invalidate(req *formatter.FormatRequest)
}
```

Place the implementation in `pkg/cache/` or a new package if it introduces external dependencies.

## Pull Request Checklist

- [ ] Code follows the style guidelines.
- [ ] `gofmt` produces no changes.
- [ ] `go vet` reports no issues.
- [ ] All tests pass with `-race`.
- [ ] New code has corresponding tests.
- [ ] Test names follow `Test<Struct>_<Method>_<Scenario>`.
- [ ] Commit messages follow Conventional Commits.
- [ ] No new external dependencies added without discussion.
- [ ] Documentation updated if public API changed.
