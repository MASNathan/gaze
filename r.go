package gaze

import (
	"fmt"
	"strings"

	r "github.com/r-lib/tree-sitter-r/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func rOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(r.Language())); err != nil {
		return "", fmt.Errorf("set R language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse R source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	var docs []*sitter.Node
	root := tree.RootNode()
	for i := uint(0); i < root.NamedChildCount(); i++ {
		n := root.NamedChild(i)
		if n.Kind() == "comment" {
			docs = append(docs, n)
			continue
		}
		if n.Kind() == "binary_operator" {
			lhs, op, rhs := n.ChildByFieldName("lhs"), n.ChildByFieldName("operator"), n.ChildByFieldName("rhs")
			if lhs != nil && op != nil && rhs != nil {
				operator := text(op)
				name, value := lhs, rhs
				if operator == "->" || operator == "->>" {
					name, value = rhs, lhs
				}
				if (operator == "<-" || operator == "<<-" || operator == "=" || operator == "->" || operator == "->>") && name.Kind() == "identifier" && (opts.IncludeNonPublic || !strings.HasPrefix(text(name), ".")) {
					header := text(name) + " " + operator + " …"
					if value.Kind() == "function_definition" {
						params := value.ChildByFieldName("parameters")
						if params != nil {
							var p strings.Builder
							pos := params.StartByte()
							for j := uint(0); j < params.NamedChildCount(); j++ {
								param := params.NamedChild(j)
								if def := param.ChildByFieldName("default"); def != nil {
									p.Write(source[pos:def.StartByte()])
									p.WriteString("…")
									pos = def.EndByte()
								}
							}
							p.Write(source[pos:params.EndByte()])
							header = text(name) + " <- function" + p.String()
						}
					}
					scriptDocs(&out, source, docs, n, opts, "#")
					scriptLine(&out, n, opts)
					out.WriteString(header + "\n")
				}
			}
		}
		docs = nil
	}
	return out.String(), nil
}
