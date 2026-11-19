package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// managedOutline extracts declarations from Java, Kotlin, and C# syntax trees.
func managedOutline(source []byte, opts Options, language *sitter.Language, name string) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		return "", fmt.Errorf("set %s language: %w", name, err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse %s source: no syntax tree", name)
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
			if kind == "comment" || kind == "block_comment" || kind == "line_comment" {
				if strings.HasPrefix(text(n), "/**") || strings.HasPrefix(text(n), "///") {
					doc = n
				} else {
					doc = nil
				}
				continue
			}
			container := false
			declaration := true
			switch kind {
			case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "struct_declaration", "annotation_type_declaration", "object_declaration", "namespace_declaration":
				container = true
			case "method_declaration", "constructor_declaration", "compact_constructor_declaration", "function_declaration", "secondary_constructor", "field_declaration", "property_declaration", "constant_declaration", "annotation_type_element_declaration", "delegate_declaration", "event_declaration", "event_field_declaration", "indexer_declaration", "operator_declaration", "conversion_operator_declaration", "destructor_declaration", "enum_member_declaration":
			default:
				declaration = false
			}
			if !declaration {
				doc = nil
				continue
			}
			body := n.ChildByFieldName("body")
			if body == nil {
				for j := uint(0); j < n.NamedChildCount(); j++ {
					c := n.NamedChild(j)
					switch c.Kind() {
					case "class_body", "enum_class_body", "function_body", "declaration_list", "enum_member_declaration_list":
						body = c
					}
				}
			}
			if kind == "property_declaration" {
				if accessors := n.ChildByFieldName("accessors"); accessors != nil {
					body = accessors
				}
			}
			end := n.EndByte()
			if body != nil {
				end = body.StartByte()
			}
			if value := n.ChildByFieldName("value"); value != nil && value.StartByte() < end {
				end = value.StartByte()
			}
			header := strings.TrimSpace(string(source[n.StartByte():end]))
			if !opts.IncludeNonPublic && (strings.Contains(" "+header+" ", " private ") || strings.Contains(" "+header+" ", " protected ")) {
				doc = nil
				continue
			}
			// ponytail: Initializers are omitted wholesale; retain expressions when contract inference is needed.
			if kind == "field_declaration" || kind == "property_declaration" || kind == "constant_declaration" || kind == "event_field_declaration" {
				if pos := strings.Index(header, " = "); pos >= 0 {
					header = header[:pos] + " = …"
				}
			}
			if doc != nil && !opts.NoDocs {
				d := text(doc)
				if !opts.FullDocs {
					var lines []string
					for _, line := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(d, "/**"), "*/"), "\n") {
						line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(line), "///"), "*"))
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
			location := ""
			if !opts.NoLines {
				location = fmt.Sprintf("L%d ", n.StartPosition().Row+1)
			}
			fmt.Fprintf(&out, "%s%s%s\n", indent, location, strings.TrimSpace(header))
			if container && body != nil {
				walk(body, indent+"  ")
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
