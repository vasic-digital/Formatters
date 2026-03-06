// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package textformat provides types and interfaces mirroring Formatters-KMP
// for cross-platform text format detection, parsing, and registry.
package textformat

import (
	"html"
	"path"
	"regexp"
	"strings"
)

// Standard format ID constants.
const (
	IDUnknown           = "unknown"
	IDPlaintext         = "plaintext"
	IDMarkdown          = "markdown"
	IDTodoTxt           = "todotxt"
	IDCSV               = "csv"
	IDWikiText          = "wikitext"
	IDKeyValue          = "keyvalue"
	IDAsciiDoc          = "asciidoc"
	IDOrgMode           = "orgmode"
	IDLaTeX             = "latex"
	IDReStructuredText  = "restructuredtext"
	IDTaskPaper         = "taskpaper"
	IDTextile           = "textile"
	IDCreole            = "creole"
	IDTiddlyWiki        = "tiddlywiki"
	IDJupyter           = "jupyter"
	IDRMarkdown         = "rmarkdown"
	IDBinary            = "binary"
)

// TextFormat represents a text format with metadata for detection.
type TextFormat struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	DefaultExtension  string   `json:"defaultExtension"`
	Extensions        []string `json:"extensions"`
	DetectionPatterns []string `json:"detectionPatterns"`
}

// ParsedDocument represents a parsed document with structured content.
type ParsedDocument struct {
	Format        TextFormat        `json:"format"`
	RawContent    string            `json:"rawContent"`
	ParsedContent string            `json:"parsedContent"`
	Metadata      map[string]string `json:"metadata"`
	Errors        []string          `json:"errors"`
}

// TextParser defines the interface for format-specific parsers.
type TextParser interface {
	SupportedFormat() TextFormat
	CanParse(format TextFormat) bool
	Parse(content string, options map[string]interface{}) *ParsedDocument
	ToHTML(document *ParsedDocument, lightMode bool) string
	Validate(content string) []string
}

// FormatRegistry provides configurable format detection and lookup.
type FormatRegistry struct {
	formats []TextFormat
}

// NewFormatRegistry creates a new registry with the given formats.
func NewFormatRegistry(formats ...TextFormat) *FormatRegistry {
	return &FormatRegistry{formats: append([]TextFormat{}, formats...)}
}

// Register adds a format to the registry.
func (r *FormatRegistry) Register(format TextFormat) {
	r.formats = append(r.formats, format)
}

// RegisterAll adds multiple formats.
func (r *FormatRegistry) RegisterAll(formats []TextFormat) {
	r.formats = append(r.formats, formats...)
}

// Formats returns all registered formats.
func (r *FormatRegistry) Formats() []TextFormat {
	result := make([]TextFormat, len(r.formats))
	copy(result, r.formats)
	return result
}

// GetByID returns the format with the given ID, or nil.
func (r *FormatRegistry) GetByID(id string) *TextFormat {
	for i := range r.formats {
		if r.formats[i].ID == id {
			return &r.formats[i]
		}
	}
	return nil
}

// GetByExtension returns the first format matching the extension.
func (r *FormatRegistry) GetByExtension(extension string) *TextFormat {
	clean := cleanExtension(extension)
	for i := range r.formats {
		for _, ext := range r.formats[i].Extensions {
			if strings.EqualFold(ext, clean) {
				return &r.formats[i]
			}
		}
	}
	return nil
}

// DetectByExtension returns the format for the extension, or a fallback.
func (r *FormatRegistry) DetectByExtension(extension string) TextFormat {
	if f := r.GetByExtension(extension); f != nil {
		return *f
	}
	if f := r.GetByID(IDPlaintext); f != nil {
		return *f
	}
	return TextFormat{ID: IDPlaintext, Name: "Plain Text", DefaultExtension: ".txt"}
}

// DetectByContent analyzes content and returns the matching format.
func (r *FormatRegistry) DetectByContent(content string, maxLines int) *TextFormat {
	if content == "" {
		return nil
	}
	if maxLines <= 0 {
		maxLines = 10
	}
	lines := strings.SplitN(content, "\n", maxLines+1)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	sample := strings.Join(lines, "\n")

	for i := range r.formats {
		for _, pattern := range r.formats[i].DetectionPatterns {
			re, err := regexp.Compile("(?m)" + pattern)
			if err != nil {
				continue
			}
			if re.MatchString(sample) {
				return &r.formats[i]
			}
		}
	}
	return nil
}

// DetectByFilename returns the format for a filename.
func (r *FormatRegistry) DetectByFilename(filename string) TextFormat {
	ext := path.Ext(filename)
	if ext != "" {
		return r.DetectByExtension(ext)
	}
	if f := r.GetByID(IDPlaintext); f != nil {
		return *f
	}
	return TextFormat{ID: IDPlaintext, Name: "Plain Text", DefaultExtension: ".txt"}
}

// GetFormatsByExtension returns all formats supporting the extension.
func (r *FormatRegistry) GetFormatsByExtension(extension string) []TextFormat {
	clean := cleanExtension(extension)
	var result []TextFormat
	for _, f := range r.formats {
		for _, ext := range f.Extensions {
			if strings.EqualFold(ext, clean) {
				result = append(result, f)
				break
			}
		}
	}
	return result
}

// IsSupported checks if a format ID is registered.
func (r *FormatRegistry) IsSupported(formatID string) bool {
	return r.GetByID(formatID) != nil
}

// IsExtensionSupported checks if an extension is supported.
func (r *FormatRegistry) IsExtensionSupported(extension string) bool {
	return r.GetByExtension(extension) != nil
}

// GetFormatNames returns all format names.
func (r *FormatRegistry) GetFormatNames() []string {
	names := make([]string, len(r.formats))
	for i, f := range r.formats {
		names[i] = f.Name
	}
	return names
}

// GetAllExtensions returns all unique extensions.
func (r *FormatRegistry) GetAllExtensions() []string {
	seen := map[string]bool{}
	var exts []string
	for _, f := range r.formats {
		for _, ext := range f.Extensions {
			if !seen[ext] {
				seen[ext] = true
				exts = append(exts, ext)
			}
		}
	}
	return exts
}

// Clear removes all registered formats.
func (r *FormatRegistry) Clear() {
	r.formats = nil
}

// ParserRegistry manages parser instances with lazy loading.
type ParserRegistry struct {
	parsers   map[string]TextParser
	factories map[string]func() TextParser
}

// NewParserRegistry creates a new empty parser registry.
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers:   make(map[string]TextParser),
		factories: make(map[string]func() TextParser),
	}
}

// Register adds a parser (eager).
func (r *ParserRegistry) Register(parser TextParser) error {
	id := parser.SupportedFormat().ID
	if _, ok := r.parsers[id]; ok {
		return &DuplicateParserError{FormatID: id}
	}
	if _, ok := r.factories[id]; ok {
		return &DuplicateParserError{FormatID: id}
	}
	r.parsers[id] = parser
	return nil
}

// RegisterLazy adds a parser factory (lazy).
func (r *ParserRegistry) RegisterLazy(formatID string, factory func() TextParser) error {
	if _, ok := r.parsers[formatID]; ok {
		return &DuplicateParserError{FormatID: formatID}
	}
	if _, ok := r.factories[formatID]; ok {
		return &DuplicateParserError{FormatID: formatID}
	}
	r.factories[formatID] = factory
	return nil
}

// GetParser returns the parser for the format.
func (r *ParserRegistry) GetParser(format TextFormat) TextParser {
	id := format.ID
	if p, ok := r.parsers[id]; ok {
		return p
	}
	if factory, ok := r.factories[id]; ok {
		p := factory()
		r.parsers[id] = p
		delete(r.factories, id)
		return p
	}
	for _, p := range r.parsers {
		if p.CanParse(format) {
			return p
		}
	}
	return nil
}

// HasParser checks if a parser exists for the format.
func (r *ParserRegistry) HasParser(format TextFormat) bool {
	id := format.ID
	if _, ok := r.parsers[id]; ok {
		return true
	}
	if _, ok := r.factories[id]; ok {
		return true
	}
	for _, p := range r.parsers {
		if p.CanParse(format) {
			return true
		}
	}
	return false
}

// GetAllParsers returns all instantiated parsers.
func (r *ParserRegistry) GetAllParsers() []TextParser {
	result := make([]TextParser, 0, len(r.parsers))
	for _, p := range r.parsers {
		result = append(result, p)
	}
	return result
}

// GetPendingParserCount returns the number of lazy factories.
func (r *ParserRegistry) GetPendingParserCount() int {
	return len(r.factories)
}

// GetInstantiatedParserCount returns the number of instantiated parsers.
func (r *ParserRegistry) GetInstantiatedParserCount() int {
	return len(r.parsers)
}

// Clear removes all parsers and factories.
func (r *ParserRegistry) Clear() {
	r.parsers = make(map[string]TextParser)
	r.factories = make(map[string]func() TextParser)
}

// DuplicateParserError is returned when registering a duplicate parser.
type DuplicateParserError struct {
	FormatID string
}

func (e *DuplicateParserError) Error() string {
	return "parser for format '" + e.FormatID + "' is already registered"
}

// EscapeHTML escapes HTML special characters.
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// ParseOptions provides a builder for parsing options.
type ParseOptions struct {
	options map[string]interface{}
}

// NewParseOptions creates a new options builder.
func NewParseOptions() *ParseOptions {
	return &ParseOptions{options: make(map[string]interface{})}
}

// EnableLineNumbers toggles line numbers.
func (o *ParseOptions) EnableLineNumbers(enable bool) *ParseOptions {
	o.options["lineNumbers"] = enable
	return o
}

// EnableHighlighting toggles syntax highlighting.
func (o *ParseOptions) EnableHighlighting(enable bool) *ParseOptions {
	o.options["highlighting"] = enable
	return o
}

// SetBaseURL sets the base URL.
func (o *ParseOptions) SetBaseURL(url string) *ParseOptions {
	o.options["baseUrl"] = url
	return o
}

// Set adds a custom option.
func (o *ParseOptions) Set(key string, value interface{}) *ParseOptions {
	o.options[key] = value
	return o
}

// Build returns the options map.
func (o *ParseOptions) Build() map[string]interface{} {
	result := make(map[string]interface{}, len(o.options))
	for k, v := range o.options {
		result[k] = v
	}
	return result
}

func cleanExtension(ext string) string {
	e := strings.TrimSpace(strings.ToLower(ext))
	if !strings.HasPrefix(e, ".") {
		e = "." + e
	}
	return e
}
