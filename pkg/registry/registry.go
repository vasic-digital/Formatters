package registry

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"digital.vasic.formatters/pkg/formatter"
)

// Registry manages all registered formatters.
type Registry struct {
	mu         sync.RWMutex
	formatters map[string]formatter.Formatter
	byLanguage map[string][]formatter.Formatter
	metadata   map[string]*formatter.FormatterMetadata
}

// New creates a new formatter registry.
func New() *Registry {
	return &Registry{
		formatters: make(map[string]formatter.Formatter),
		byLanguage: make(map[string][]formatter.Formatter),
		metadata:   make(map[string]*formatter.FormatterMetadata),
	}
}

// Register registers a formatter with optional metadata.
func (r *Registry) Register(f formatter.Formatter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := f.Name()

	if _, exists := r.formatters[name]; exists {
		return fmt.Errorf("formatter %s already registered", name)
	}

	r.formatters[name] = f

	for _, lang := range f.Languages() {
		langLower := strings.ToLower(lang)
		r.byLanguage[langLower] = append(
			r.byLanguage[langLower], f,
		)
	}

	return nil
}

// RegisterWithMetadata registers a formatter with metadata.
func (r *Registry) RegisterWithMetadata(
	f formatter.Formatter,
	metadata *formatter.FormatterMetadata,
) error {
	if err := r.Register(f); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.metadata[f.Name()] = metadata

	return nil
}

// Get retrieves a formatter by name.
func (r *Registry) Get(name string) (formatter.Formatter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, exists := r.formatters[name]
	if !exists {
		return nil, fmt.Errorf("formatter %s not found", name)
	}

	return f, nil
}

// GetByLanguage retrieves all formatters for a language.
func (r *Registry) GetByLanguage(
	language string,
) []formatter.Formatter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	langLower := strings.ToLower(language)
	return r.byLanguage[langLower]
}

// List returns all registered formatter names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.formatters))
	for name := range r.formatters {
		names = append(names, name)
	}

	return names
}

// Remove removes a formatter from the registry.
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, exists := r.formatters[name]
	if !exists {
		return fmt.Errorf("formatter %s not found", name)
	}

	for _, lang := range f.Languages() {
		langLower := strings.ToLower(lang)
		formatters := r.byLanguage[langLower]
		for i, existing := range formatters {
			if existing.Name() == name {
				r.byLanguage[langLower] = append(
					formatters[:i], formatters[i+1:]...,
				)
				break
			}
		}
	}

	delete(r.formatters, name)
	delete(r.metadata, name)

	return nil
}

// Count returns the number of registered formatters.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.formatters)
}

// GetMetadata retrieves formatter metadata.
func (r *Registry) GetMetadata(
	name string,
) (*formatter.FormatterMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[name]
	if !exists {
		return nil, fmt.Errorf("metadata for formatter %s not found", name)
	}

	return metadata, nil
}

// ListByType returns all formatters of a specific type.
func (r *Registry) ListByType(
	ftype formatter.FormatterType,
) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, metadata := range r.metadata {
		if metadata.Type == ftype {
			names = append(names, name)
		}
	}

	return names
}

// DetectLanguageFromPath detects language from file extension.
func DetectLanguageFromPath(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return ""
	}

	ext = strings.TrimPrefix(ext, ".")
	ext = strings.ToLower(ext)

	extensionMap := map[string]string{
		"c": "c", "h": "c",
		"cc": "cpp", "cpp": "cpp", "cxx": "cpp",
		"hpp": "cpp", "hxx": "cpp",
		"rs": "rust", "go": "go",
		"py": "python", "pyw": "python",
		"js": "javascript", "jsx": "javascript",
		"ts": "typescript", "tsx": "typescript",
		"java": "java",
		"kt": "kotlin", "kts": "kotlin",
		"scala": "scala", "sc": "scala",
		"rb": "ruby", "php": "php",
		"swift": "swift", "dart": "dart",
		"sh": "bash", "bash": "bash",
		"lua": "lua",
		"pl": "perl", "pm": "perl",
		"r": "r", "sql": "sql",
		"yaml": "yaml", "yml": "yaml",
		"json": "json", "toml": "toml",
		"xml": "xml",
		"html": "html", "htm": "html",
		"css": "css", "scss": "scss",
		"md": "markdown", "markdown": "markdown",
		"graphql": "graphql", "gql": "graphql",
		"proto": "protobuf",
		"tf": "terraform", "tfvars": "terraform",
		"hs": "haskell",
		"ml": "ocaml", "mli": "ocaml",
		"ex": "elixir", "exs": "elixir",
		"erl": "erlang", "hrl": "erlang",
		"zig": "zig", "nim": "nim",
	}

	return extensionMap[ext]
}

// DetectFormatter detects the appropriate formatter for a file.
func (r *Registry) DetectFormatter(
	filePath string,
) (formatter.Formatter, error) {
	language := DetectLanguageFromPath(filePath)
	if language == "" {
		return nil, fmt.Errorf(
			"unable to detect language from file path: %s", filePath,
		)
	}

	formatters := r.GetByLanguage(language)
	if len(formatters) == 0 {
		return nil, fmt.Errorf(
			"no formatters available for language: %s", language,
		)
	}

	return formatters[0], nil
}

// maxConcurrentHealthChecks limits parallel health checks.
const maxConcurrentHealthChecks = 10

// HealthCheckAll performs health checks on all formatters.
func (r *Registry) HealthCheckAll(
	ctx context.Context,
) map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, maxConcurrentHealthChecks)

	for name, f := range r.formatters {
		wg.Add(1)
		go func(name string, f formatter.Formatter) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			err := f.HealthCheck(ctx)

			mu.Lock()
			results[name] = err
			mu.Unlock()
		}(name, f)
	}

	wg.Wait()

	return results
}

// defaultRegistry is the package-level default registry singleton.
var defaultRegistry = New()

// Default returns the default registry singleton.
func Default() *Registry {
	return defaultRegistry
}

// RegisterDefault registers a formatter in the default registry.
func RegisterDefault(f formatter.Formatter) error {
	return defaultRegistry.Register(f)
}

// GetDefault retrieves a formatter from the default registry.
func GetDefault(name string) (formatter.Formatter, error) {
	return defaultRegistry.Get(name)
}
