package gaze

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	csharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
)

func csharpOutlineOptions(source []byte, opts Options) (string, error) {
	return managedOutline(source, opts, sitter.NewLanguage(csharp.Language()), "C#")
}
