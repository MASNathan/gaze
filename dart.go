package gaze

import (
	"fmt"
	"strings"

	dart "github.com/UserNobody14/tree-sitter-dart/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func dartOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(dart.Language())); err != nil {
		return "", fmt.Errorf("set Dart language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Dart source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		var docs []string
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			kind := n.Kind()
			if kind == "documentation_comment" {
				docs = append(docs, text(n))
				continue
			}
			container := kind == "class_definition" || kind == "enum_declaration" || kind == "extension_declaration" || kind == "mixin_declaration" || kind == "extension_type_declaration"
			if !container && kind != "method_signature" && kind != "function_signature" && kind != "declaration" && kind != "type_alias" && kind != "enum_constant" {
				docs = nil
				continue
			}
			var name func(*sitter.Node) *sitter.Node
			name = func(node *sitter.Node) *sitter.Node {
				if found := node.ChildByFieldName("name"); found != nil {
					return found
				}
				for j := uint(0); j < node.NamedChildCount(); j++ {
					child := node.NamedChild(j)
					if child.Kind() == "identifier" {
						return child
					}
					if found := name(child); found != nil {
						return found
					}
				}
				return nil
			}
			if n := name(n); n != nil && strings.HasPrefix(text(n), "_") && !opts.IncludeNonPublic {
				docs = nil
				continue
			}
			body := n.ChildByFieldName("body")
			end := n.EndByte()
			if container && body != nil {
				end = body.StartByte()
			}
			header := strings.TrimSpace(string(source[n.StartByte():end]))
			if kind == "declaration" {
				// ponytail: Omit initializer expressions; preserve individual values when contract inference needs them.
				if pos := strings.Index(header, "="); pos >= 0 {
					header = strings.TrimSpace(header[:pos]) + " = …"
				}
			}
			if !opts.NoDocs && len(docs) > 0 {
				if opts.FullDocs {
					for _, doc := range docs {
						fmt.Fprintf(&out, "%s%s\n", indent, doc)
					}
				} else {
					var lines []string
					for _, doc := range docs {
						for _, line := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(doc, "/**"), "*/"), "\n") {
							line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(line), "///"), "*"))
							if line != "" {
								lines = append(lines, line)
							}
						}
					}
					if len(lines) > 0 {
						fmt.Fprintf(&out, "%s%s\n", indent, compactDoc(strings.Join(lines, " | ")))
					}
				}
			}
			docs = nil
			location := ""
			if !opts.NoLines {
				location = fmt.Sprintf("L%d ", n.StartPosition().Row+1)
			}
			fmt.Fprintf(&out, "%s%s%s\n", indent, location, header)
			if container && body != nil {
				walk(body, indent+"  ")
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
