package gaze

import (
	kotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func kotlinOutlineOptions(source []byte, opts Options) (string, error) {
	return managedOutline(source, opts, sitter.NewLanguage(kotlin.Language()), "Kotlin")
}
