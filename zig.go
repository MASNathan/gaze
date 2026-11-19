package gaze

import (
	"fmt"
	"strings"

	zig "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func zigOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(zig.Language())); err != nil {
		return "", fmt.Errorf("set Zig language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Zig source: no syntax tree")
	}
	defer tree.Close()
	var out strings.Builder
	var walk func(*sitter.Node, string, bool)
	walk = func(parent *sitter.Node, indent string, fields bool) {
		var docs []string
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			if n.Kind() == "comment" {
				docs = append(docs, string(source[n.StartByte():n.EndByte()]))
				continue
			}
			text := string(source[n.StartByte():n.EndByte()])
			public := strings.HasPrefix(strings.TrimSpace(text), "pub ") || strings.HasPrefix(strings.TrimSpace(text), "export ")
			if !(public || opts.IncludeNonPublic || fields && n.Kind() == "container_field") {
				docs = nil
				continue
			}
			switch n.Kind() {
			case "function_declaration", "variable_declaration", "container_field", "using_namespace_declaration":
				end := n.EndByte()
				if body := n.ChildByFieldName("body"); body != nil {
					end = body.StartByte()
				}
				var container *sitter.Node
				if n.Kind() == "variable_declaration" {
					for j := uint(0); j < n.NamedChildCount(); j++ {
						c := n.NamedChild(j)
						switch c.Kind() {
						case "struct_declaration", "enum_declaration", "union_declaration", "opaque_declaration":
							container = c
						}
					}
					if container != nil {
						end = container.StartByte()
						if brace := strings.IndexByte(string(source[container.StartByte():container.EndByte()]), '{'); brace >= 0 {
							end = container.StartByte() + uint(brace) + 1
						}
					}
				}
				if !opts.NoDocs {
					for _, d := range docs {
						if opts.FullDocs {
							fmt.Fprintf(&out, "%s%s\n", indent, d)
						} else {
							fmt.Fprintf(&out, "%s%s\n", indent, compactDoc(strings.TrimSpace(strings.TrimLeft(d, "/! "))))
						}
					}
				}
				loc := ""
				if !opts.NoLines {
					a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
					loc = fmt.Sprintf("L%d ", a)
					if a != b {
						loc = fmt.Sprintf("L%d-%d ", a, b)
					}
				}
				fmt.Fprintf(&out, "%s%s%s\n", indent, loc, strings.TrimSpace(string(source[n.StartByte():end])))
				if container != nil {
					walk(container, indent+"  ", true)
					fmt.Fprintf(&out, "%s}\n", indent)
				}
			}
			docs = nil
		}
	}
	walk(tree.RootNode(), "", false)
	return out.String(), nil
}
