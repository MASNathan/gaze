// Package gaze extracts concise source contracts without implementation bodies.
package gaze

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Options controls which declarations appear in an outline.
type Options struct {
	// IncludeNonPublic includes private and protected members.
	IncludeNonPublic bool
	// FullDocs preserves documentation comments in their original format.
	FullDocs bool
	// NoDocs omits documentation output while retaining compact PHPDoc type inference.
	NoDocs bool
	// NoLines omits declaration line ranges.
	NoLines bool
}

// Outline extracts a contract from source. The file extension selects the language;
// See README for supported languages. Documentation is compact by default; FullDocs
// preserves original docblocks. Compact output merges PHPDoc and native types,
// without distinguishing whether PHP enforces them, and omits @inheritDoc.
// Output is not a stable machine-readable format. Syntax errors may
// produce a partial outline; parsing is tolerant of incomplete source.
func Outline(filename string, source []byte, opts Options) (string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".php":
		return phpOutlineOptions(source, opts)
	case ".go":
		return goOutlineOptions(source, opts)
	case ".js", ".mjs", ".cjs":
		return javascriptOutlineOptions(source, opts)
	case ".py":
		return pythonOutlineOptions(source, opts)
	case ".html", ".htm":
		return htmlOutlineOptions(source, opts)
	case ".css":
		return cssOutlineOptions(source, opts)
	case ".lua":
		return luaOutlineOptions(source, opts)
	case ".r":
		return rOutlineOptions(source, opts)
	case ".zig":
		return zigOutlineOptions(source, opts)
	case ".scala", ".sc":
		return scalaOutlineOptions(source, opts)
	case ".ex", ".exs":
		return elixirOutlineOptions(source, opts)
	case ".dart":
		return dartOutlineOptions(source, opts)
	case ".m", ".mm":
		return objcOutlineOptions(source, opts)
	case ".ts", ".mts", ".cts":
		return typescriptOutlineOptions(source, opts, false)
	case ".tsx":
		return typescriptOutlineOptions(source, opts, true)
	case ".rs":
		return rustOutlineOptions(source, opts)
	case ".java":
		return javaOutlineOptions(source, opts)
	case ".kt", ".kts":
		return kotlinOutlineOptions(source, opts)
	case ".cs":
		return csharpOutlineOptions(source, opts)
	case ".c", ".h":
		return cOutlineOptions(source, opts)
	case ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx":
		return cppOutlineOptions(source, opts)
	case ".rb":
		return rubyOutlineOptions(source, opts)
	case ".sh", ".bash":
		return shellOutlineOptions(source, opts)
	default:
		return "", fmt.Errorf("unsupported file type %q", filepath.Ext(filename))
	}
}
