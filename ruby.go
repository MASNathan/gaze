package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
)

func rubyOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(ruby.Language())); err != nil {
		return "", fmt.Errorf("set Ruby language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Ruby source: no syntax tree")
	}
	defer tree.Close()
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		private := false
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			kind := n.Kind()
			if kind == "call" || kind == "identifier" {
				s := strings.TrimSpace(string(source[n.StartByte():n.EndByte()]))
				if s == "private" || s == "protected" {
					private = true
				}
				if s == "public" {
					private = false
				}
				continue
			}
			if kind != "class" && kind != "module" && kind != "method" && kind != "singleton_method" {
				continue
			}
			if private && !opts.IncludeNonPublic && (kind == "method" || kind == "singleton_method") {
				continue
			}
			body := n.ChildByFieldName("body")
			end := n.EndByte()
			if body != nil {
				end = body.StartByte()
			}
			header := strings.TrimSpace(string(source[n.StartByte():end]))
			header = strings.TrimSuffix(header, "end")
			header = strings.TrimSpace(header)
			loc := ""
			if !opts.NoLines {
				a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
				loc = fmt.Sprintf("L%d ", a)
				if a != b {
					loc = fmt.Sprintf("L%d-%d ", a, b)
				}
			}
			fmt.Fprintf(&out, "%s%s%s\n", indent, loc, header)
			if body != nil && (kind == "class" || kind == "module") {
				walk(body, indent+"  ")
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
