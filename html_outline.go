package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	html "github.com/tree-sitter/tree-sitter-html/bindings/go"
)

func htmlOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(html.Language())); err != nil {
		return "", fmt.Errorf("set HTML language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse HTML source: no syntax tree")
	}
	defer tree.Close()
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			switch n.Kind() {
			case "element", "script_element", "style_element":
				var tag *sitter.Node
				for j := uint(0); j < n.NamedChildCount(); j++ {
					c := n.NamedChild(j)
					if c.Kind() == "start_tag" || c.Kind() == "self_closing_tag" {
						for k := uint(0); k < c.NamedChildCount(); k++ {
							if name := c.NamedChild(k); name.Kind() == "tag_name" {
								tag = name
								break
							}
						}
						break
					}
				}
				if tag == nil {
					continue
				}
				loc := ""
				if !opts.NoLines {
					a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
					loc = fmt.Sprintf("L%d ", a)
					if a != b {
						loc = fmt.Sprintf("L%d-%d ", a, b)
					}
				}
				fmt.Fprintf(&out, "%s%s<%s>\n", indent, loc, source[tag.StartByte():tag.EndByte()])
				if n.Kind() == "element" {
					walk(n, indent+"  ")
				}
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
