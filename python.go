package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

// pythonOutlineOptions extracts declarations without executing Python source.
func pythonOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(python.Language())); err != nil {
		return "", fmt.Errorf("set Python language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Python source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	var clean func(*sitter.Node, uint) string
	clean = func(n *sitter.Node, end uint) string {
		var b strings.Builder
		pos := n.StartByte()
		for i := uint(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.StartByte() >= end || c.EndByte() > end {
				break
			}
			b.Write(source[pos:c.StartByte()])
			if (n.Kind() == "default_parameter" || n.Kind() == "typed_default_parameter") && n.ChildByFieldName("value") != nil && c.Id() == n.ChildByFieldName("value").Id() {
				b.WriteString("…")
			} else {
				b.WriteString(clean(c, c.EndByte()))
			}
			pos = c.EndByte()
		}
		b.Write(source[pos:end])
		return b.String()
	}
	var declarations func(*sitter.Node, string)
	declarations = func(parent *sitter.Node, indent string) {
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			if n.Kind() == "decorated_definition" {
				n = n.ChildByFieldName("definition")
				if n == nil {
					continue
				}
			}
			kind := n.Kind()
			if kind == "expression_statement" && n.NamedChildCount() == 1 && n.NamedChild(0).Kind() == "assignment" {
				n = n.NamedChild(0)
				kind = n.Kind()
			}
			if kind != "class_definition" && kind != "function_definition" && kind != "assignment" {
				continue
			}
			name := n.ChildByFieldName("name")
			if kind == "assignment" {
				name = n.ChildByFieldName("left")
			}
			if name == nil || name.Kind() != "identifier" {
				continue
			}
			id := text(name)
			if !opts.IncludeNonPublic && strings.HasPrefix(id, "_") && !(strings.HasPrefix(id, "__") && strings.HasSuffix(id, "__")) {
				continue
			}
			body := n.ChildByFieldName("body")
			if body != nil && !opts.NoDocs && body.NamedChildCount() > 0 {
				doc := body.NamedChild(0)
				if doc.Kind() == "expression_statement" && doc.NamedChildCount() == 1 && doc.NamedChild(0).Kind() == "string" {
					raw := text(doc.NamedChild(0))
					if opts.FullDocs {
						fmt.Fprintf(&out, "%s%s\n", indent, raw)
					} else {
						// ponytail: Compact plain docstrings only; add Python string decoding when needed.
						quote := strings.IndexAny(raw, "\"'")
						if quote >= 0 {
							q := raw[quote : quote+1]
							if strings.HasPrefix(raw[quote:], strings.Repeat(q, 3)) {
								q = strings.Repeat(q, 3)
							}
							value := strings.TrimSuffix(raw[quote+len(q):], q)
							var lines []string
							for _, line := range strings.Split(value, "\n") {
								if line = strings.TrimSpace(line); line != "" {
									lines = append(lines, line)
								}
							}
							if len(lines) > 0 {
								fmt.Fprintf(&out, "%s%s\n", indent, compactDoc(strings.Join(lines, " | ")))
							}
						}
					}
				}
			}
			header := ""
			if kind == "assignment" {
				header = text(name)
				if typ := n.ChildByFieldName("type"); typ != nil {
					header += ": " + text(typ)
				}
				if n.ChildByFieldName("right") != nil {
					header += " = …"
				}
			} else if body != nil {
				header = strings.TrimSpace(clean(n, body.StartByte()))
			} else {
				header = strings.TrimSpace(clean(n, n.EndByte()))
			}
			location := ""
			if !opts.NoLines {
				start, end := n.StartPosition().Row+1, n.EndPosition().Row+1
				location = fmt.Sprintf("L%d ", start)
				if start != end {
					location = fmt.Sprintf("L%d-%d ", start, end)
				}
			}
			fmt.Fprintf(&out, "%s%s%s\n", indent, location, header)
			if kind == "class_definition" && body != nil {
				declarations(body, indent+"  ")
			}
		}
	}
	declarations(tree.RootNode(), "")
	return out.String(), nil
}
