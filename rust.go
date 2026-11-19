package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
)

func rustOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(rust.Language())); err != nil {
		return "", fmt.Errorf("set Rust language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Rust source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	write := func(n *sitter.Node, docs []string, indent, header string) {
		if !opts.NoDocs && len(docs) > 0 {
			if opts.FullDocs {
				for _, d := range docs {
					fmt.Fprintf(&out, "%s%s\n", indent, d)
				}
			} else {
				var lines []string
				for _, d := range docs {
					d = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(d, "///"), "/**"), "*/"))
					if d != "" {
						lines = append(lines, d)
					}
				}
				if len(lines) > 0 {
					fmt.Fprintf(&out, "%s%s\n", indent, compactDoc(strings.Join(lines, " | ")))
				}
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
	var declarations func(*sitter.Node, string, bool)
	declarations = func(parent *sitter.Node, indent string, implicitPublic bool) {
		var docs []string
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			if n.Kind() == "line_comment" || n.Kind() == "block_comment" {
				d := text(n)
				if strings.HasPrefix(d, "///") || strings.HasPrefix(d, "/**") {
					docs = append(docs, d)
				} else {
					docs = nil
				}
				continue
			}
			if n.Kind() == "attribute_item" {
				continue
			}
			public := implicitPublic
			for j := uint(0); j < n.NamedChildCount(); j++ {
				if n.NamedChild(j).Kind() == "visibility_modifier" {
					public = true
					break
				}
			}
			if !public && !opts.IncludeNonPublic && n.Kind() != "impl_item" {
				docs = nil
				continue
			}
			switch n.Kind() {
			case "function_item", "function_signature_item":
				end := n.EndByte()
				if body := n.ChildByFieldName("body"); body != nil {
					end = body.StartByte()
				}
				write(n, docs, indent, string(source[n.StartByte():end]))
			case "struct_item", "enum_item", "trait_item", "impl_item", "mod_item", "union_item":
				body := n.ChildByFieldName("body")
				if body == nil {
					write(n, docs, indent, text(n))
					break
				}
				write(n, docs, indent, string(source[n.StartByte():body.StartByte()]))
				declarations(body, indent+"  ", n.Kind() == "trait_item" || n.Kind() == "enum_item" || (n.Kind() == "impl_item" && n.ChildByFieldName("trait") != nil))
			case "field_declaration", "enum_variant":
				end := n.EndByte()
				if value := n.ChildByFieldName("value"); value != nil {
					end = value.StartByte()
					write(n, docs, indent, strings.TrimSpace(string(source[n.StartByte():end]))+" …")
				} else {
					write(n, docs, indent, text(n))
				}
			case "const_item", "static_item":
				end := n.EndByte()
				if value := n.ChildByFieldName("value"); value != nil {
					end = value.StartByte()
					write(n, docs, indent, strings.TrimSpace(string(source[n.StartByte():end]))+" …")
				} else {
					write(n, docs, indent, text(n))
				}
			case "type_item", "associated_type":
				write(n, docs, indent, text(n))
			}
			docs = nil
		}
	}
	declarations(tree.RootNode(), "", false)
	return out.String(), nil
}
