package gaze

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

func javaOutlineOptions(source []byte, opts Options) (string, error) {
	return managedOutline(source, opts, sitter.NewLanguage(java.Language()), "Java")
}
