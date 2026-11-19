package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	scala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
)

func scalaOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(scala.Language())); err != nil {
		return "", fmt.Errorf("set Scala language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Scala source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		var doc *sitter.Node
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			kind := n.Kind()
			if kind == "comment" || kind == "block_comment" {
				if strings.HasPrefix(text(n), "/**") {
					doc = n
				} else {
					doc = nil
				}
				continue
			}
			container := false
			switch kind {
			case "class_definition", "object_definition", "trait_definition", "enum_definition", "package_clause":
				container = true
			case "function_definition", "function_declaration", "val_definition", "val_declaration", "var_definition", "var_declaration", "type_definition", "type_declaration":
			default:
				doc = nil
				continue
			}
			body := n.ChildByFieldName("body")
			end := n.EndByte()
			if body != nil {
				end = body.StartByte()
			}
			if value := n.ChildByFieldName("value"); value != nil && value.StartByte() < end {
				end = value.StartByte()
			}
			header := strings.TrimSpace(string(source[n.StartByte():end]))
			header = strings.TrimSpace(strings.TrimSuffix(header, "="))
			if !opts.IncludeNonPublic && (strings.HasPrefix(header, "private ") || strings.HasPrefix(header, "protected ") || strings.HasPrefix(header, "private[") || strings.HasPrefix(header, "protected[") || strings.Contains(" "+header, " private ") || strings.Contains(" "+header, " protected ")) {
				doc = nil
				continue
			}
			if doc != nil && !opts.NoDocs {
				d := text(doc)
				if !opts.FullDocs {
					var lines []string
					for _, line := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(d, "/**"), "*/"), "\n") {
						line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
						if line != "" {
							lines = append(lines, line)
						}
					}
					d = compactDoc(strings.Join(lines, " | "))
				}
				if d != "" {
					fmt.Fprintf(&out, "%s%s\n", indent, d)
				}
			}
			doc = nil
			loc := ""
			if !opts.NoLines {
				a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
				loc = fmt.Sprintf("L%d ", a)
				if a != b {
					loc = fmt.Sprintf("L%d-%d ", a, b)
				}
			}
			fmt.Fprintf(&out, "%s%s%s\n", indent, loc, header)
			if container && body != nil {
				walk(body, indent+"  ")
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
