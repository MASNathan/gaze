package gaze

import (
	"fmt"
	"strings"

	elixir "github.com/elixir-lang/tree-sitter-elixir/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func elixirOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(elixir.Language())); err != nil {
		return "", fmt.Errorf("set Elixir language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Elixir source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	var walk func(*sitter.Node, string)
	walk = func(parent *sitter.Node, indent string) {
		var docs []string
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			s := strings.TrimSpace(text(n))
			if strings.HasPrefix(s, "@doc ") || strings.HasPrefix(s, "@moduledoc ") || strings.HasPrefix(s, "@typedoc ") {
				if !opts.NoDocs {
					docs = append(docs, s)
				}
				continue
			}
			if strings.HasPrefix(s, "@spec ") || strings.HasPrefix(s, "@callback ") || strings.HasPrefix(s, "@macrocallback ") || strings.HasPrefix(s, "@type ") || strings.HasPrefix(s, "@opaque ") || (opts.IncludeNonPublic && strings.HasPrefix(s, "@typep ")) {
				loc := ""
				if !opts.NoLines {
					loc = fmt.Sprintf("L%d ", n.StartPosition().Row+1)
				}
				fmt.Fprintf(&out, "%s%s%s\n", indent, loc, s)
				continue
			}
			if n.Kind() != "call" {
				docs = nil
				continue
			}
			target := n.ChildByFieldName("target")
			if target == nil {
				docs = nil
				continue
			}
			name := text(target)
			container := name == "defmodule" || name == "defprotocol" || name == "defimpl"
			if !container && name != "def" && name != "defmacro" && name != "defguard" && (!opts.IncludeNonPublic || (name != "defp" && name != "defmacrop" && name != "defguardp")) {
				docs = nil
				continue
			}
			var body *sitter.Node
			end := n.EndByte()
			for j := uint(0); j < n.NamedChildCount(); j++ {
				c := n.NamedChild(j)
				if c.Kind() == "do_block" {
					body = c
					end = c.StartByte()
					break
				}
				if c.Kind() == "arguments" {
					for k := uint(0); k < c.NamedChildCount(); k++ {
						a := c.NamedChild(k)
						if a.Kind() == "keywords" {
							for m := uint(0); m < a.NamedChildCount(); m++ {
								pair := a.NamedChild(m)
								key := pair.ChildByFieldName("key")
								if key != nil && strings.TrimSpace(text(key)) == "do:" {
									end = pair.StartByte()
									if end > n.StartByte() && source[end-1] == ' ' {
										end--
									}
									if end > n.StartByte() && source[end-1] == ',' {
										end--
									}
									break
								}
							}
						}
					}
				}
			}
			for _, d := range docs {
				if !opts.FullDocs {
					if i := strings.Index(d, " \""); i >= 0 && strings.HasSuffix(d, "\"") && !strings.HasPrefix(d[i+1:], "\"\"\"") {
						d = d[:i+2] + compactDoc(d[i+2:len(d)-1]) + "\""
					}
				}
				fmt.Fprintf(&out, "%s%s\n", indent, d)
			}
			docs = nil
			loc := ""
			if !opts.NoLines {
				a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
				loc = fmt.Sprintf("L%d ", a)
				if a != b {
					loc = fmt.Sprintf("L%d-%d ", a, b)
				}
			}
			fmt.Fprintf(&out, "%s%s%s\n", indent, loc, strings.TrimSpace(string(source[n.StartByte():end])))
			if container && body != nil {
				walk(body, indent+"  ")
			}
		}
	}
	walk(tree.RootNode(), "")
	return out.String(), nil
}
