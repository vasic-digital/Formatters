# API Reference

Complete reference for all exported types, functions, and methods in `digital.vasic.formatters`.

---

## Package `formatter`

Import: `digital.vasic.formatters/pkg/formatter`

The core package defining all interfaces, types, and the embeddable base formatter.

### Interfaces

#### Formatter

The universal interface for all code formatters.

```go
type Formatter interface {
    // Identity
    Name() string
    Version() string
    Languages() []string

    // Capabilities
    SupportsStdin() bool
    SupportsInPlace() bool
    SupportsCheck() bool
    SupportsConfig() bool

    // Formatting
    Format(ctx context.Context, req *FormatRequest) (*FormatResult, error)
    FormatBatch(ctx context.Context, reqs []*FormatRequest) ([]*FormatResult, error)

    // Health
    HealthCheck(ctx context.Context) error

    // Configuration
    ValidateConfig(config map[string]interface{}) error
    DefaultConfig() map[string]interface{}
}
```

| Method | Description |
|--------|-------------|
| `Name()` | Returns the formatter name (e.g., "clang-format"). |
| `Version()` | Returns the formatter version (e.g., "19.1.8"). |
| `Languages()` | Returns the list of supported languages (e.g., ["c", "cpp"]). |
| `SupportsStdin()` | Returns true if the formatter accepts input via stdin. |
| `SupportsInPlace()` | Returns true if the formatter can format files in-place. |
| `SupportsCheck()` | Returns true if the formatter supports dry-run/check mode. |
| `SupportsConfig()` | Returns true if the formatter accepts configuration files. |
| `Format(ctx, req)` | Formats a single code input. Returns the result or an error. |
| `FormatBatch(ctx, reqs)` | Formats multiple code inputs. Returns results or an error. |
| `HealthCheck(ctx)` | Verifies the formatter is operational. Returns nil if healthy. |
| `ValidateConfig(config)` | Validates a configuration map. Returns nil if valid. |
| `DefaultConfig()` | Returns the default configuration map. |

### Types

#### Options

```go
type Options struct {
    IndentSize int
    UseTabs    bool
    LineWidth  int
    Style      string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `IndentSize` | `int` | Indent size (e.g., 2, 4). |
| `UseTabs` | `bool` | Use tabs instead of spaces. |
| `LineWidth` | `int` | Maximum line width. |
| `Style` | `string` | Style preset (e.g., "google", "pep8"). |

#### FormatRequest

```go
type FormatRequest struct {
    Content    string
    FilePath   string
    Language   string
    Config     map[string]interface{}
    LineLength int
    IndentSize int
    UseTabs    bool
    CheckOnly  bool
    Timeout    time.Duration
    RequestID  string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Content` | `string` | Code content to format. |
| `FilePath` | `string` | Optional file path for extension-based detection. |
| `Language` | `string` | Language override (e.g., "python"). |
| `Config` | `map[string]interface{}` | Formatter-specific configuration. |
| `LineLength` | `int` | Maximum line length (if supported). |
| `IndentSize` | `int` | Indent size. |
| `UseTabs` | `bool` | Use tabs instead of spaces. |
| `CheckOnly` | `bool` | Dry-run mode (check without formatting). |
| `Timeout` | `time.Duration` | Maximum execution time. |
| `RequestID` | `string` | Request tracking identifier. |

#### FormatResult

```go
type FormatResult struct {
    Content          string
    Changed          bool
    FormatterName    string
    FormatterVersion string
    Duration         time.Duration
    Success          bool
    Error            error
    Warnings         []string
    Stats            *FormatStats
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Content` | `string` | Formatted code output. |
| `Changed` | `bool` | Whether the content was modified. |
| `FormatterName` | `string` | Name of the formatter used. |
| `FormatterVersion` | `string` | Version of the formatter used. |
| `Duration` | `time.Duration` | Execution time. |
| `Success` | `bool` | Whether formatting succeeded. |
| `Error` | `error` | Error if formatting failed. |
| `Warnings` | `[]string` | Non-fatal warnings. |
| `Stats` | `*FormatStats` | Formatting statistics (may be nil). |

#### FormatStats

```go
type FormatStats struct {
    LinesTotal   int
    LinesChanged int
    BytesTotal   int
    BytesChanged int
    Violations   int
}
```

| Field | Type | Description |
|-------|------|-------------|
| `LinesTotal` | `int` | Total lines in input. |
| `LinesChanged` | `int` | Lines that were modified. |
| `BytesTotal` | `int` | Total bytes in input. |
| `BytesChanged` | `int` | Byte difference (output - input). |
| `Violations` | `int` | Number of style violations fixed. |

#### Error

```go
type Error struct {
    Line     int
    Column   int
    Message  string
    Severity string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Line` | `int` | Line number (1-based). |
| `Column` | `int` | Column number (1-based). |
| `Message` | `string` | Error description. |
| `Severity` | `string` | One of "error", "warning", "info". |

#### Result

```go
type Result struct {
    Formatted string
    Changed   bool
    Errors    []Error
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Formatted` | `string` | Formatted code. |
| `Changed` | `bool` | Whether code was modified. |
| `Errors` | `[]Error` | Formatting errors and diagnostics. |

#### FormatterType

```go
type FormatterType string

const (
    FormatterTypeNative  FormatterType = "native"
    FormatterTypeService FormatterType = "service"
    FormatterTypeBuiltin FormatterType = "builtin"
    FormatterTypeUnified FormatterType = "unified"
)
```

#### FormatterMetadata

```go
type FormatterMetadata struct {
    Name            string
    Type            FormatterType
    Architecture    string
    GitHubURL       string
    Version         string
    Languages       []string
    License         string
    InstallMethod   string
    BinaryPath      string
    ServiceURL      string
    ConfigFormat    string
    DefaultConfig   string
    Performance     string
    Complexity      string
    SupportsStdin   bool
    SupportsInPlace bool
    SupportsCheck   bool
    SupportsConfig  bool
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Formatter name. |
| `Type` | `FormatterType` | Architecture type (native, service, builtin, unified). |
| `Architecture` | `string` | Runtime architecture ("binary", "python", "node", "jvm"). |
| `GitHubURL` | `string` | GitHub repository URL. |
| `Version` | `string` | Formatter version. |
| `Languages` | `[]string` | Supported languages. |
| `License` | `string` | License identifier. |
| `InstallMethod` | `string` | Installation method ("binary", "apt", "brew", "npm", "pip"). |
| `BinaryPath` | `string` | Path to the binary. |
| `ServiceURL` | `string` | Service endpoint URL (if service-based). |
| `ConfigFormat` | `string` | Config file format ("yaml", "json", "toml", "ini", "none"). |
| `DefaultConfig` | `string` | Path to default config file. |
| `Performance` | `string` | Performance tier ("very_fast", "fast", "medium", "slow"). |
| `Complexity` | `string` | Setup complexity ("easy", "medium", "hard"). |
| `SupportsStdin` | `bool` | Whether stdin input is supported. |
| `SupportsInPlace` | `bool` | Whether in-place formatting is supported. |
| `SupportsCheck` | `bool` | Whether check/dry-run mode is supported. |
| `SupportsConfig` | `bool` | Whether configuration files are supported. |

#### BaseFormatter

```go
type BaseFormatter struct { /* unexported fields */ }
```

##### Functions

```go
func NewBaseFormatter(metadata *FormatterMetadata) *BaseFormatter
```

Creates a new `BaseFormatter` from the provided metadata.

##### Methods

| Method | Return | Description |
|--------|--------|-------------|
| `Name()` | `string` | Returns the formatter name from metadata. |
| `Version()` | `string` | Returns the formatter version from metadata. |
| `Languages()` | `[]string` | Returns supported languages from metadata. |
| `SupportsStdin()` | `bool` | Returns stdin support from metadata. |
| `SupportsInPlace()` | `bool` | Returns in-place support from metadata. |
| `SupportsCheck()` | `bool` | Returns check mode support from metadata. |
| `SupportsConfig()` | `bool` | Returns config support from metadata. |
| `DefaultConfig()` | `map[string]interface{}` | Returns an empty map. |
| `ValidateConfig(config)` | `error` | Returns nil (accepts all configs). |
| `Metadata()` | `*FormatterMetadata` | Returns the full metadata struct. |

---

## Package `registry`

Import: `digital.vasic.formatters/pkg/registry`

Thread-safe formatter registry with language detection and health checking.

### Constants

```go
const maxConcurrentHealthChecks = 10
```

Maximum number of parallel health checks in `HealthCheckAll`.

### Types

#### Registry

```go
type Registry struct { /* unexported fields */ }
```

##### Functions

```go
func New() *Registry
```

Creates a new empty registry.

```go
func Default() *Registry
```

Returns the package-level default registry singleton.

```go
func RegisterDefault(f formatter.Formatter) error
```

Registers a formatter in the default registry. Returns an error if a formatter with the same name is already registered.

```go
func GetDefault(name string) (formatter.Formatter, error)
```

Retrieves a formatter from the default registry by name.

```go
func DetectLanguageFromPath(filePath string) string
```

Detects the programming language from a file extension. Returns an empty string if the extension is unknown. Supports 40+ file extensions.

##### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Register` | `(f formatter.Formatter) error` | Registers a formatter. Returns error if name already exists. |
| `RegisterWithMetadata` | `(f formatter.Formatter, metadata *FormatterMetadata) error` | Registers with associated metadata. |
| `Get` | `(name string) (formatter.Formatter, error)` | Retrieves a formatter by name. |
| `GetByLanguage` | `(language string) []formatter.Formatter` | Returns all formatters for a language (case-insensitive). |
| `List` | `() []string` | Returns all registered formatter names. |
| `Remove` | `(name string) error` | Removes a formatter by name. Returns error if not found. |
| `Count` | `() int` | Returns the number of registered formatters. |
| `GetMetadata` | `(name string) (*FormatterMetadata, error)` | Returns metadata for a formatter. |
| `ListByType` | `(ftype FormatterType) []string` | Returns formatter names matching the given type. |
| `DetectFormatter` | `(filePath string) (formatter.Formatter, error)` | Detects and returns a formatter for a file path. |
| `HealthCheckAll` | `(ctx context.Context) map[string]error` | Health checks all formatters concurrently. Returns a map of name to error (nil = healthy). |

---

## Package `native`

Import: `digital.vasic.formatters/pkg/native`

Native binary formatters that execute system commands via stdin/stdout.

### Types

#### NativeFormatter

```go
type NativeFormatter struct {
    *formatter.BaseFormatter
    // unexported fields
}
```

##### Functions

```go
func NewNativeFormatter(
    metadata *formatter.FormatterMetadata,
    binaryPath string,
    args []string,
    stdinFlag bool,
) *NativeFormatter
```

Creates a new native formatter. Parameters:
- `metadata`: Formatter metadata (name, version, languages, capabilities).
- `binaryPath`: Path to the system binary (e.g., "gofmt", "/usr/bin/black").
- `args`: Default command-line arguments.
- `stdinFlag`: If true, input is piped via stdin; a `-` argument is appended.

```go
func NewGoFormatter() *NativeFormatter
```

Creates a gofmt formatter for Go. Binary: `gofmt`. Languages: `["go"]`.

```go
func NewPythonFormatter() *NativeFormatter
```

Creates a Black formatter for Python. Binary: `black`. Args: `["--quiet"]`. Languages: `["python"]`.

```go
func NewJSFormatter() *NativeFormatter
```

Creates a Prettier formatter. Binary: `prettier`. Args: `["--stdin-filepath", "temp.js"]`. Languages: `["javascript", "typescript", "json", "html", "css", "scss", "markdown", "yaml", "graphql"]`.

```go
func NewRustFormatter() *NativeFormatter
```

Creates a rustfmt formatter for Rust. Binary: `rustfmt`. Args: `["--edition=2024"]`. Languages: `["rust"]`.

```go
func NewSQLFormatter() *NativeFormatter
```

Creates a sqlformat formatter for SQL. Binary: `sqlformat`. Args: `["--reindent", "--keywords", "upper"]`. Languages: `["sql"]`.

##### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Format` | `(ctx context.Context, req *FormatRequest) (*FormatResult, error)` | Formats code by executing the binary with stdin/stdout. Returns result with stats. |
| `FormatBatch` | `(ctx context.Context, reqs []*FormatRequest) ([]*FormatResult, error)` | Formats multiple requests sequentially. |
| `HealthCheck` | `(ctx context.Context) error` | Runs binary with `--version` to verify availability. |

Inherited from `BaseFormatter`: `Name`, `Version`, `Languages`, `SupportsStdin`, `SupportsInPlace`, `SupportsCheck`, `SupportsConfig`, `DefaultConfig`, `ValidateConfig`, `Metadata`.

---

## Package `service`

Import: `digital.vasic.formatters/pkg/service`

HTTP service-based formatters for containerized formatting tools.

### Types

#### Config

```go
type Config struct {
    Endpoint   string
    Timeout    time.Duration
    HealthPath string
    FormatPath string
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Endpoint` | `string` | (required) | Base URL (e.g., "http://localhost:9210"). |
| `Timeout` | `time.Duration` | 30s | HTTP request timeout. |
| `HealthPath` | `string` | "/health" | Health check endpoint path. |
| `FormatPath` | `string` | "/format" | Format endpoint path. |

##### Functions

```go
func DefaultConfig(endpoint string) Config
```

Returns a config with the given endpoint and default values (30s timeout, `/health`, `/format`).

#### ServiceFormatter

```go
type ServiceFormatter struct { /* unexported fields */ }
```

##### Functions

```go
func NewServiceFormatter(
    metadata *formatter.FormatterMetadata,
    config Config,
) *ServiceFormatter
```

Creates a new service formatter. Applies defaults for zero-value `Timeout`, empty `HealthPath`, and empty `FormatPath`.

##### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Name` | `() string` | Returns the formatter name from metadata. |
| `Version` | `() string` | Returns the formatter version from metadata. |
| `Languages` | `() []string` | Returns supported languages from metadata. |
| `SupportsStdin` | `() bool` | Returns stdin support from metadata. |
| `SupportsInPlace` | `() bool` | Returns in-place support from metadata. |
| `SupportsCheck` | `() bool` | Returns check mode support from metadata. |
| `SupportsConfig` | `() bool` | Returns config support from metadata. |
| `Format` | `(ctx context.Context, req *FormatRequest) (*FormatResult, error)` | Sends a POST request to the format endpoint. |
| `FormatBatch` | `(ctx context.Context, reqs []*FormatRequest) ([]*FormatResult, error)` | Formats requests sequentially. Continues on individual errors. |
| `HealthCheck` | `(ctx context.Context) error` | Sends GET to health endpoint. Expects status 200 and `{"status": "healthy"}`. |
| `ValidateConfig` | `(config map[string]interface{}) error` | Returns nil (no validation). |
| `DefaultConfig` | `() map[string]interface{}` | Returns an empty map. |
| `GetMetadata` | `() *FormatterMetadata` | Returns the formatter metadata. |

#### ServiceFormatRequest

```go
type ServiceFormatRequest struct {
    Content string                 `json:"content"`
    Options map[string]interface{} `json:"options"`
}
```

#### ServiceFormatResponse

```go
type ServiceFormatResponse struct {
    Success   bool   `json:"success"`
    Content   string `json:"content"`
    Changed   bool   `json:"changed"`
    Formatter string `json:"formatter"`
    Error     string `json:"error,omitempty"`
}
```

#### ServiceHealthResponse

```go
type ServiceHealthResponse struct {
    Status    string `json:"status"`
    Formatter string `json:"formatter"`
    Version   string `json:"version"`
    Error     string `json:"error,omitempty"`
}
```

---

## Package `executor`

Import: `digital.vasic.formatters/pkg/executor`

Execution engine with middleware chains, pipelines, and batch processing.

### Types

#### ExecuteFunc

```go
type ExecuteFunc func(
    ctx context.Context,
    f formatter.Formatter,
    req *formatter.FormatRequest,
) (*formatter.FormatResult, error)
```

The function signature for format execution, used by middleware.

#### Middleware

```go
type Middleware func(next ExecuteFunc) ExecuteFunc
```

A function that wraps an `ExecuteFunc` to add behavior.

#### Config

```go
type Config struct {
    DefaultTimeout time.Duration
    MaxRetries     int
    MaxConcurrent  int
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `DefaultTimeout` | `time.Duration` | 30s | Default execution timeout. |
| `MaxRetries` | `int` | 3 | Maximum retry attempts. |
| `MaxConcurrent` | `int` | 10 | Maximum concurrent batch operations. |

##### Functions

```go
func DefaultExecutorConfig() Config
```

Returns a config with 30s timeout, 3 retries, 10 concurrent.

#### Executor

```go
type Executor struct { /* unexported fields */ }
```

##### Functions

```go
func New(reg *registry.Registry, config Config) *Executor
```

Creates a new executor backed by the given registry.

##### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Execute` | `(ctx context.Context, req *FormatRequest) (*FormatResult, error)` | Resolves formatter from language or file path, runs middleware chain, returns result. |
| `ExecuteBatch` | `(ctx context.Context, reqs []*FormatRequest) ([]*FormatResult, error)` | Executes multiple requests concurrently. Returns results and first error. |
| `Use` | `(middleware ...Middleware)` | Appends middleware to the execution chain. |

#### Pipeline

```go
type Pipeline struct { /* unexported fields */ }
```

##### Functions

```go
func NewPipeline(steps ...formatter.Formatter) *Pipeline
```

Creates a pipeline that chains the given formatters sequentially.

##### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Execute` | `(ctx context.Context, req *FormatRequest) (*FormatResult, error)` | Runs each step, passing output as input to the next. Stops on first failure. |

#### BatchFormat

```go
func BatchFormat(
    ctx context.Context,
    exec *Executor,
    reqs []*formatter.FormatRequest,
    maxConcurrent int,
) ([]*formatter.FormatResult, error)
```

Formats multiple requests concurrently with a semaphore for rate limiting. If `maxConcurrent <= 0`, defaults to 10. Returns results and the first error encountered.

### Middleware Functions

```go
func TimeoutMiddleware(defaultTimeout time.Duration) Middleware
```

Creates middleware that enforces a timeout. Uses `req.Timeout` if set, otherwise `defaultTimeout`. Returns a timeout error if the deadline is exceeded.

```go
func RetryMiddleware(maxRetries int) Middleware
```

Creates middleware that retries on failure with exponential backoff (1s, 2s, 4s, ...). Capped at 30 retries. Respects context cancellation.

```go
func ValidationMiddleware() Middleware
```

Creates middleware that rejects empty input content (before execution) and rejects empty output content on successful results (after execution).

---

## Package `cache`

Import: `digital.vasic.formatters/pkg/cache`

In-memory format result caching with TTL expiration and periodic cleanup.

### Interfaces

#### FormatCache

```go
type FormatCache interface {
    Get(req *formatter.FormatRequest) (*formatter.FormatResult, bool)
    Set(req *formatter.FormatRequest, result *formatter.FormatResult)
    Invalidate(req *formatter.FormatRequest)
}
```

| Method | Description |
|--------|-------------|
| `Get(req)` | Returns the cached result and true if found and not expired. |
| `Set(req, result)` | Stores a result. Evicts the oldest entry if at capacity. |
| `Invalidate(req)` | Removes the cached entry for the given request. |

### Types

#### Config

```go
type Config struct {
    MaxEntries  int
    TTL         time.Duration
    CleanupFreq time.Duration
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxEntries` | `int` | 10000 | Maximum number of cache entries. |
| `TTL` | `time.Duration` | 1 hour | Time to live for entries. |
| `CleanupFreq` | `time.Duration` | 5 minutes | Frequency of expired entry cleanup. |

##### Functions

```go
func DefaultCacheConfig() Config
```

Returns a config with 10000 entries, 1 hour TTL, 5 minute cleanup.

#### CacheStats

```go
type CacheStats struct {
    Size       int
    MaxEntries int
    TTL        time.Duration
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Size` | `int` | Current number of cached entries. |
| `MaxEntries` | `int` | Maximum configured entries. |
| `TTL` | `time.Duration` | Configured TTL. |

#### InMemoryCache

```go
type InMemoryCache struct { /* unexported fields */ }
```

##### Functions

```go
func NewInMemoryCache(config Config) *InMemoryCache
```

Creates a new in-memory cache and starts the background cleanup goroutine.

##### Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get` | `(req *FormatRequest) (*FormatResult, bool)` | Returns cached result if found and not expired. |
| `Set` | `(req *FormatRequest, result *FormatResult)` | Stores a result. Evicts oldest entry if at capacity. |
| `Invalidate` | `(req *FormatRequest)` | Removes a specific cached entry. |
| `Clear` | `()` | Removes all cached entries. |
| `Size` | `() int` | Returns the current number of entries. |
| `Stop` | `()` | Stops the background cleanup goroutine. Must be called to prevent leaks. |
| `Stats` | `() CacheStats` | Returns current cache statistics. |
