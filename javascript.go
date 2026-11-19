package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

// javascriptOutlineOptions extracts top-level declarations and class members.
func javascriptOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(javascript.Language())); err != nil {
		return "", fmt.Errorf("set JavaScript language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse JavaScript source: no syntax tree")
	}
	defer tree.Close()

	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	write := func(n, doc *sitter.Node, indent, header string) {
		if doc != nil && !opts.NoDocs {
			d := text(doc)
			if !opts.FullDocs {
				var lines []string
				for _, line := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(d, "/**"), "*/"), "\n") {
					if line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*")); line != "" {
						lines = append(lines, line)
					}
				}
				d = compactDoc(strings.Join(lines, " | "))
			}
			if d != "" {
				fmt.Fprintf(&out, "%s%s\n", indent, d)
			}
		}
		location := ""
		if !opts.NoLines {
			start, end := n.StartPosition().Row+1, n.EndPosition().Row+1
			location = fmt.Sprintf("L%d ", start)
			if start != end {
				location = fmt.Sprintf("L%d-%d ", start, end)
			}
		}
		fmt.Fprintf(&out, "%s%s%s\n", indent, location, strings.TrimSpace(header))
	}
	var declarations func(*sitter.Node, string)
	var declaration func(*sitter.Node, *sitter.Node, string, string)
	declaration = func(n, doc *sitter.Node, indent, prefix string) {
		switch n.Kind() {
		case "export_statement":
			child := n.ChildByFieldName("declaration")
			if child == nil {
				child = n.ChildByFieldName("value")
			}
			if child != nil {
				declaration(child, doc, indent, text(n)[:child.StartByte()-n.StartByte()])
			}
		case "function_declaration", "generator_function_declaration", "function_expression", "generator_function", "method_definition":
			name := n.ChildByFieldName("name")
			if name != nil && name.Kind() == "private_property_identifier" && !opts.IncludeNonPublic {
				return
			}
			if body := n.ChildByFieldName("body"); body != nil {
				write(n, doc, indent, prefix+string(source[n.StartByte():body.StartByte()]))
			}
		case "class_declaration", "class":
			if body := n.ChildByFieldName("body"); body != nil {
				write(n, doc, indent, prefix+string(source[n.StartByte():body.StartByte()]))
				declarations(body, indent+"  ")
			}
		case "field_definition":
			name := n.ChildByFieldName("property")
			if name != nil && name.Kind() == "private_property_identifier" && !opts.IncludeNonPublic {
				return
			}
			end := n.EndByte()
			if value := n.ChildByFieldName("value"); value != nil {
				end = value.StartByte()
				write(n, doc, indent, prefix+strings.TrimSpace(string(source[n.StartByte():end]))+" …")
			} else {
				write(n, doc, indent, prefix+text(n))
			}
		case "lexical_declaration", "variable_declaration":
			kind := "var"
			if k := n.ChildByFieldName("kind"); k != nil {
				kind = text(k)
			}
			for i := uint(0); i < n.NamedChildCount(); i++ {
				v := n.NamedChild(i)
				if v.Kind() != "variable_declarator" || v.ChildByFieldName("name") == nil {
					continue
				}
				header := prefix + kind + " " + text(v.ChildByFieldName("name"))
				if value := v.ChildByFieldName("value"); value != nil {
					switch value.Kind() {
					case "arrow_function", "function_expression", "generator_function":
						if body := value.ChildByFieldName("body"); body != nil {
							header += " = " + strings.TrimSpace(string(source[value.StartByte():body.StartByte()])) + " …"
						}
					default:
						header += " = …"
					}
				}
				write(v, doc, indent, header)
				doc = nil
			}
		}
	}
	declarations = func(parent *sitter.Node, indent string) {
		var doc *sitter.Node
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			if n.Kind() == "comment" {
				if strings.HasPrefix(text(n), "/**") {
					doc = n
				} else {
					doc = nil
				}
				continue
			}
			declaration(n, doc, indent, "")
			doc = nil
		}
	}
	declarations(tree.RootNode(), "")
	return out.String(), nil
}
