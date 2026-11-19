package gaze

import (
	"fmt"
	"strings"

	objc "github.com/tree-sitter-grammars/tree-sitter-objc/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func objcOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(objc.Language())); err != nil {
		return "", fmt.Errorf("set Objective-C language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Objective-C source: no syntax tree")
	}
	defer tree.Close()
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		var docs []string
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			if n.Kind() == "comment" {
				docs = append(docs, string(source[n.StartByte():n.EndByte()]))
				continue
			}
			container := n.Kind() == "class_interface" || n.Kind() == "protocol_declaration" || n.Kind() == "implementation_definition"
			switch n.Kind() {
			case "class_interface", "protocol_declaration", "implementation_definition", "method_declaration", "method_definition", "property_declaration", "function_definition", "declaration", "type_definition", "protocol_forward_declaration", "class_declaration":
				if n.Kind() == "implementation_definition" && !opts.IncludeNonPublic {
					docs = nil
					continue
				}
				end := n.EndByte()
				if body := n.ChildByFieldName("body"); body != nil {
					end = body.StartByte()
				}
				if container {
					text := source[n.StartByte():n.EndByte()]
					if nl := strings.IndexByte(string(text), '\n'); nl >= 0 {
						end = n.StartByte() + uint(nl)
					}
				} else if n.Kind() == "method_definition" {
					for j := uint(0); j < n.NamedChildCount(); j++ {
						c := n.NamedChild(j)
						if c.Kind() == "compound_statement" {
							end = c.StartByte()
							break
						}
					}
				}
				if !opts.NoDocs {
					for _, d := range docs {
						if opts.FullDocs {
							fmt.Fprintf(&out, "%s%s\n", indent, d)
						} else {
							fmt.Fprintf(&out, "%s%s\n", indent, compactDoc(strings.TrimSpace(strings.Trim(d, "/* "))))
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
				if container {
					walk(n, indent+"  ")
					fmt.Fprintf(&out, "%s@end\n", indent)
				}
			}
			docs = nil
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
