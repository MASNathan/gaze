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
// currently only PHP is supported. Documentation is compact by default; FullDocs
// preserves original docblocks. Compact output merges PHPDoc and native types,
// without distinguishing whether PHP enforces them, and omits @inheritDoc.
// Output is not a stable machine-readable format. Syntax errors may
// produce a partial outline; parsing is tolerant of incomplete source.
func Outline(filename string, source []byte, opts Options) (string, error) {
	if !strings.EqualFold(filepath.Ext(filename), ".php") {
		return "", fmt.Errorf("unsupported file type %q", filepath.Ext(filename))
	}
	return phpOutlineOptions(source, opts)
}
