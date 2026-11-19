package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// typescriptOutlineOptions extracts TypeScript or TSX declarations.
func typescriptOutlineOptions(source []byte, opts Options, tsx bool) (string, error) {
	language := typescript.LanguageTypescript()
	if tsx {
		language = typescript.LanguageTSX()
	}
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(language)); err != nil {
		return "", fmt.Errorf("set TypeScript language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse TypeScript source: no syntax tree")
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
		for i := uint(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if !opts.IncludeNonPublic && ((c.Kind() == "accessibility_modifier" && text(c) != "public") || c.Kind() == "private_property_identifier") {
				return
			}
		}
		switch n.Kind() {
		case "export_statement", "ambient_declaration":
			child := n.ChildByFieldName("declaration")
			if child == nil {
				child = n.ChildByFieldName("value")
			}
			if child == nil && n.NamedChildCount() > 0 && n.Kind() == "ambient_declaration" {
				child = n.NamedChild(0)
			}
			if child != nil {
				declaration(child, doc, indent, prefix+string(source[n.StartByte():child.StartByte()]))
			}
		case "function_declaration", "generator_function_declaration", "function_expression", "generator_function", "method_definition":
			if body := n.ChildByFieldName("body"); body != nil {
				write(n, doc, indent, prefix+string(source[n.StartByte():body.StartByte()]))
			}
		case "function_signature", "method_signature", "abstract_method_signature", "property_signature", "call_signature", "construct_signature", "index_signature", "type_alias_declaration":
			write(n, doc, indent, prefix+text(n))
		case "class_declaration", "abstract_class_declaration", "class", "interface_declaration", "enum_declaration", "internal_module":
			if body := n.ChildByFieldName("body"); body != nil {
				write(n, doc, indent, prefix+string(source[n.StartByte():body.StartByte()]))
				declarations(body, indent+"  ")
			}
		case "public_field_definition":
			if value := n.ChildByFieldName("value"); value != nil {
				write(n, doc, indent, prefix+strings.TrimSpace(string(source[n.StartByte():value.StartByte()]))+" …")
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
				end := v.EndByte()
				value := v.ChildByFieldName("value")
				if value != nil {
					end = value.StartByte()
				}
				header := prefix + kind + " " + strings.TrimSpace(string(source[v.StartByte():end]))
				if value != nil {
					if body := value.ChildByFieldName("body"); body != nil && (value.Kind() == "arrow_function" || value.Kind() == "function_expression") {
						header += " " + strings.TrimSpace(string(source[value.StartByte():body.StartByte()])) + " …"
					} else {
						header += " …"
					}
				}
				write(v, doc, indent, header)
				doc = nil
			}
		case "enum_assignment":
			if name := n.ChildByFieldName("name"); name != nil {
				write(n, doc, indent, text(name)+" = …")
			}
		case "property_identifier":
			write(n, doc, indent, text(n))
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
