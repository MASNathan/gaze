package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
)

func cOutlineOptions(source []byte, opts Options) (string, error) {
	return cFamilyOutline(source, opts, sitter.NewLanguage(c.Language()))
}

func cppOutlineOptions(source []byte, opts Options) (string, error) {
	return cFamilyOutline(source, opts, sitter.NewLanguage(cpp.Language()))
}

func cFamilyOutline(source []byte, opts Options, language *sitter.Language) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		return "", fmt.Errorf("set C/C++ language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse C/C++ source: no syntax tree")
	}
	defer tree.Close()

	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	write := func(n, doc *sitter.Node, indent, header string) {
		if doc != nil && !opts.NoDocs {
			d := text(doc)
			if !opts.FullDocs {
				d = strings.TrimPrefix(strings.TrimPrefix(d, "/**"), "///")
				d = strings.TrimSuffix(d, "*/")
				var lines []string
				for _, line := range strings.Split(d, "\n") {
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
	// Remove initializer expressions while preserving declarators and types.
	var clean func(*sitter.Node) string
	clean = func(n *sitter.Node) string {
		var b strings.Builder
		pos := n.StartByte()
		for i := uint(0); i < n.NamedChildCount(); i++ {
			child := n.NamedChild(i)
			b.Write(source[pos:child.StartByte()])
			if value := n.ChildByFieldName("value"); value != nil && child.StartByte() == value.StartByte() && child.EndByte() == value.EndByte() {
				b.WriteString("…")
			} else {
				b.WriteString(clean(child))
			}
			pos = child.EndByte()
		}
		b.Write(source[pos:n.EndByte()])
		return b.String()
	}
	var declarations func(*sitter.Node, string, bool)
	var declaration func(*sitter.Node, *sitter.Node, string)
	declaration = func(n, doc *sitter.Node, indent string) {
		switch n.Kind() {
		case "function_definition":
			if body := n.ChildByFieldName("body"); body != nil {
				write(n, doc, indent, string(source[n.StartByte():body.StartByte()]))
			}
		case "namespace_definition", "class_specifier", "struct_specifier", "union_specifier", "enum_specifier":
			if body := n.ChildByFieldName("body"); body != nil {
				write(n, doc, indent, string(source[n.StartByte():body.StartByte()]))
				declarations(body, indent+"  ", n.Kind() != "class_specifier")
			} else {
				write(n, doc, indent, clean(n))
			}
		case "template_declaration":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				child := n.NamedChild(i)
				if child.Kind() != "template_parameter_list" {
					declaration(child, doc, indent)
				}
			}
		case "declaration", "field_declaration", "type_definition":
			// A definition may wrap a struct/class/enum body.
			for i := uint(0); i < n.NamedChildCount(); i++ {
				child := n.NamedChild(i)
				if body := child.ChildByFieldName("body"); body != nil && (child.Kind() == "struct_specifier" || child.Kind() == "class_specifier" || child.Kind() == "union_specifier" || child.Kind() == "enum_specifier") {
					write(n, doc, indent, string(source[n.StartByte():body.StartByte()]))
					declarations(body, indent+"  ", child.Kind() != "class_specifier")
					return
				}
			}
			write(n, doc, indent, clean(n))
		case "enumerator", "alias_declaration":
			write(n, doc, indent, clean(n))
		}
	}
	declarations = func(parent *sitter.Node, indent string, public bool) {
		var doc *sitter.Node
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			if n.Kind() == "comment" {
				if strings.HasPrefix(text(n), "/**") || strings.HasPrefix(text(n), "///") {
					doc = n
				} else {
					doc = nil
				}
				continue
			}
			if n.Kind() == "access_specifier" {
				public = text(n) == "public"
				doc = nil
				continue
			}
			if public || opts.IncludeNonPublic {
				declaration(n, doc, indent)
			}
			doc = nil
		}
	}
	declarations(tree.RootNode(), "", true)
	return out.String(), nil
}
