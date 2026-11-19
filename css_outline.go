package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	css "github.com/tree-sitter/tree-sitter-css/bindings/go"
)

func cssOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(css.Language())); err != nil {
		return "", fmt.Errorf("set CSS language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse CSS source: no syntax tree")
	}
	defer tree.Close()
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			loc := ""
			if !opts.NoLines {
				a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
				loc = fmt.Sprintf("L%d ", a)
				if a != b {
					loc = fmt.Sprintf("L%d-%d ", a, b)
				}
			}
			switch n.Kind() {
			case "rule_set":
				for j := uint(0); j < n.NamedChildCount(); j++ {
					c := n.NamedChild(j)
					if c.Kind() == "selectors" {
						fmt.Fprintf(&out, "%s%s%s\n", indent, loc, strings.TrimSpace(string(source[c.StartByte():c.EndByte()])))
					} else if c.Kind() == "block" {
						walk(c, indent+"  ")
					}
				}
			case "declaration":
				for j := uint(0); j < n.NamedChildCount(); j++ {
					c := n.NamedChild(j)
					if c.Kind() == "property_name" {
						name := string(source[c.StartByte():c.EndByte()])
						if strings.HasPrefix(name, "--") {
							fmt.Fprintf(&out, "%s%s%s\n", indent, loc, name)
						}
						break
					}
				}
			default:
				// Preserve nested rules inside at-rules without emitting declaration values.
				walk(n, indent)
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
