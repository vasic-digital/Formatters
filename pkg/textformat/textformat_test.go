// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package textformat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func markdownFormat() TextFormat {
	return TextFormat{
		ID:                IDMarkdown,
		Name:              "Markdown",
		DefaultExtension:  ".md",
		Extensions:        []string{".md", ".markdown"},
		DetectionPatterns: []string{"^#+ "},
	}
}

func plaintextFormat() TextFormat {
	return TextFormat{
		ID:               IDPlaintext,
		Name:             "Plain Text",
		DefaultExtension: ".txt",
		Extensions:       []string{".txt", ".text"},
	}
}

func csvFormat() TextFormat {
	return TextFormat{
		ID:                IDCSV,
		Name:              "CSV",
		DefaultExtension:  ".csv",
		Extensions:        []string{".csv"},
		DetectionPatterns: []string{"^.*,.*,.*$"},
	}
}

// FormatRegistry Tests

func TestFormatRegistry_GetByID(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), plaintextFormat())
	f := r.GetByID(IDMarkdown)
	require.NotNil(t, f)
	assert.Equal(t, "Markdown", f.Name)
	assert.Nil(t, r.GetByID("nonexistent"))
}

func TestFormatRegistry_GetByExtension(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), plaintextFormat())
	f := r.GetByExtension(".md")
	require.NotNil(t, f)
	assert.Equal(t, IDMarkdown, f.ID)

	f2 := r.GetByExtension("md")
	require.NotNil(t, f2)
	assert.Equal(t, IDMarkdown, f2.ID)
}

func TestFormatRegistry_DetectByExtension(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), plaintextFormat())
	f := r.DetectByExtension(".unknown")
	assert.Equal(t, IDPlaintext, f.ID)

	f2 := r.DetectByExtension(".md")
	assert.Equal(t, IDMarkdown, f2.ID)
}

func TestFormatRegistry_DetectByContent(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), csvFormat())
	f := r.DetectByContent("# Title\n\nContent", 10)
	require.NotNil(t, f)
	assert.Equal(t, IDMarkdown, f.ID)

	assert.Nil(t, r.DetectByContent("", 10))
	assert.Nil(t, r.DetectByContent("plain text here", 10))
}

func TestFormatRegistry_DetectByFilename(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), plaintextFormat(), csvFormat())
	assert.Equal(t, IDMarkdown, r.DetectByFilename("README.md").ID)
	assert.Equal(t, IDCSV, r.DetectByFilename("data.csv").ID)
	assert.Equal(t, IDPlaintext, r.DetectByFilename("noextension").ID)
}

func TestFormatRegistry_GetFormatsByExtension(t *testing.T) {
	todotxt := TextFormat{ID: IDTodoTxt, Name: "Todo.txt", DefaultExtension: ".txt", Extensions: []string{".txt"}}
	r := NewFormatRegistry(plaintextFormat(), todotxt)
	formats := r.GetFormatsByExtension(".txt")
	assert.Len(t, formats, 2)
}

func TestFormatRegistry_IsSupported(t *testing.T) {
	r := NewFormatRegistry(markdownFormat())
	assert.True(t, r.IsSupported(IDMarkdown))
	assert.False(t, r.IsSupported("nonexistent"))
}

func TestFormatRegistry_IsExtensionSupported(t *testing.T) {
	r := NewFormatRegistry(markdownFormat())
	assert.True(t, r.IsExtensionSupported(".md"))
	assert.False(t, r.IsExtensionSupported(".xyz"))
}

func TestFormatRegistry_GetFormatNames(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), plaintextFormat())
	names := r.GetFormatNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "Markdown")
}

func TestFormatRegistry_GetAllExtensions(t *testing.T) {
	r := NewFormatRegistry(markdownFormat(), csvFormat())
	exts := r.GetAllExtensions()
	assert.Contains(t, exts, ".md")
	assert.Contains(t, exts, ".csv")
}

func TestFormatRegistry_Register(t *testing.T) {
	r := NewFormatRegistry()
	assert.Len(t, r.Formats(), 0)
	r.Register(markdownFormat())
	assert.Len(t, r.Formats(), 1)
}

func TestFormatRegistry_RegisterAll(t *testing.T) {
	r := NewFormatRegistry()
	r.RegisterAll([]TextFormat{markdownFormat(), plaintextFormat()})
	assert.Len(t, r.Formats(), 2)
}

func TestFormatRegistry_Clear(t *testing.T) {
	r := NewFormatRegistry(markdownFormat())
	r.Clear()
	assert.Len(t, r.Formats(), 0)
}

// ParserRegistry Tests

type mockParser struct {
	format TextFormat
}

func (p *mockParser) SupportedFormat() TextFormat      { return p.format }
func (p *mockParser) CanParse(f TextFormat) bool       { return p.format.ID == f.ID }
func (p *mockParser) Parse(content string, opts map[string]interface{}) *ParsedDocument {
	return &ParsedDocument{
		Format:        p.format,
		RawContent:    content,
		ParsedContent: content,
	}
}
func (p *mockParser) ToHTML(doc *ParsedDocument, light bool) string {
	return "<pre>" + EscapeHTML(doc.RawContent) + "</pre>"
}
func (p *mockParser) Validate(content string) []string { return nil }

func TestParserRegistry_RegisterAndGet(t *testing.T) {
	r := NewParserRegistry()
	parser := &mockParser{format: markdownFormat()}
	err := r.Register(parser)
	require.NoError(t, err)

	found := r.GetParser(markdownFormat())
	require.NotNil(t, found)
	assert.Equal(t, IDMarkdown, found.SupportedFormat().ID)
}

func TestParserRegistry_RegisterDuplicate(t *testing.T) {
	r := NewParserRegistry()
	err := r.Register(&mockParser{format: markdownFormat()})
	require.NoError(t, err)
	err = r.Register(&mockParser{format: markdownFormat()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestParserRegistry_LazyRegistration(t *testing.T) {
	r := NewParserRegistry()
	instantiated := false
	err := r.RegisterLazy(IDMarkdown, func() TextParser {
		instantiated = true
		return &mockParser{format: markdownFormat()}
	})
	require.NoError(t, err)
	assert.False(t, instantiated)
	assert.Equal(t, 1, r.GetPendingParserCount())
	assert.Equal(t, 0, r.GetInstantiatedParserCount())

	found := r.GetParser(markdownFormat())
	require.NotNil(t, found)
	assert.True(t, instantiated)
	assert.Equal(t, 0, r.GetPendingParserCount())
	assert.Equal(t, 1, r.GetInstantiatedParserCount())
}

func TestParserRegistry_HasParser(t *testing.T) {
	r := NewParserRegistry()
	assert.False(t, r.HasParser(markdownFormat()))
	r.Register(&mockParser{format: markdownFormat()})
	assert.True(t, r.HasParser(markdownFormat()))
}

func TestParserRegistry_GetParserNotFound(t *testing.T) {
	r := NewParserRegistry()
	assert.Nil(t, r.GetParser(markdownFormat()))
}

func TestParserRegistry_GetAllParsers(t *testing.T) {
	r := NewParserRegistry()
	r.Register(&mockParser{format: markdownFormat()})
	assert.Len(t, r.GetAllParsers(), 1)
}

func TestParserRegistry_Clear(t *testing.T) {
	r := NewParserRegistry()
	r.Register(&mockParser{format: markdownFormat()})
	r.RegisterLazy(IDPlaintext, func() TextParser { return &mockParser{format: plaintextFormat()} })
	r.Clear()
	assert.Equal(t, 0, r.GetInstantiatedParserCount())
	assert.Equal(t, 0, r.GetPendingParserCount())
}

// EscapeHTML Tests

func TestEscapeHTML(t *testing.T) {
	assert.Equal(t, "&amp;", EscapeHTML("&"))
	assert.Equal(t, "&lt;", EscapeHTML("<"))
	assert.Equal(t, "&gt;", EscapeHTML(">"))
	assert.Equal(t, "&#34;", EscapeHTML("\""))
	assert.Equal(t, "&#39;", EscapeHTML("'"))
	assert.Equal(t, "", EscapeHTML(""))
	assert.Equal(t, "hello world", EscapeHTML("hello world"))
}

// ParseOptions Tests

func TestParseOptions_Build(t *testing.T) {
	opts := NewParseOptions().
		EnableLineNumbers(true).
		EnableHighlighting(true).
		SetBaseURL("https://example.com").
		Set("custom", 42).
		Build()
	assert.Equal(t, true, opts["lineNumbers"])
	assert.Equal(t, true, opts["highlighting"])
	assert.Equal(t, "https://example.com", opts["baseUrl"])
	assert.Equal(t, 42, opts["custom"])
}

func TestParseOptions_Empty(t *testing.T) {
	opts := NewParseOptions().Build()
	assert.Len(t, opts, 0)
}

// Format ID Constants

func TestFormatIDConstants(t *testing.T) {
	assert.Equal(t, "unknown", IDUnknown)
	assert.Equal(t, "plaintext", IDPlaintext)
	assert.Equal(t, "markdown", IDMarkdown)
	assert.Equal(t, "todotxt", IDTodoTxt)
	assert.Equal(t, "csv", IDCSV)
	assert.Equal(t, "latex", IDLaTeX)
	assert.Equal(t, "jupyter", IDJupyter)
	assert.Equal(t, "binary", IDBinary)
}
