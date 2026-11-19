package gaze

import (
	"fmt"
	"strings"

	lua "github.com/tree-sitter-grammars/tree-sitter-lua/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func luaOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(lua.Language())); err != nil {
		return "", fmt.Errorf("set Lua language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse Lua source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	root := tree.RootNode()
	var docs []*sitter.Node
	for i := uint(0); i < root.NamedChildCount(); i++ {
		n := root.NamedChild(i)
		if n.Kind() == "comment" {
			docs = append(docs, n)
			continue
		}
		header := ""
		local := false
		switch n.Kind() {
		case "function_declaration":
			name, params := n.ChildByFieldName("name"), n.ChildByFieldName("parameters")
			if name != nil && params != nil {
				local = strings.HasPrefix(strings.TrimSpace(text(n)), "local ")
				header = strings.TrimSpace(string(source[n.StartByte():params.EndByte()]))
			}
		case "variable_declaration", "assignment_statement":
			local = n.Kind() == "variable_declaration"
			a := n
			if local && n.NamedChildCount() > 0 {
				a = n.NamedChild(0)
			}
			if a.Kind() == "assignment_statement" && a.NamedChildCount() >= 2 {
				vars, values := a.NamedChild(0), a.NamedChild(1)
				if vars.NamedChildCount() == 1 && values.NamedChildCount() == 1 {
					v := values.NamedChild(0)
					if v.Kind() == "function_definition" && v.ChildByFieldName("parameters") != nil {
						p := v.ChildByFieldName("parameters")
						header = strings.TrimSpace(string(source[n.StartByte():p.EndByte()]))
					} else {
						header = strings.TrimSpace(text(vars)) + " = …"
						if local {
							header = "local " + header
						}
					}
				}
			}
		}
		if header != "" && (!local || opts.IncludeNonPublic) {
			scriptDocs(&out, source, docs, n, opts, "--")
			scriptLine(&out, n, opts)
			out.WriteString(header + "\n")
		}
		docs = nil
	}
	return out.String(), nil
}

func scriptLine(out *strings.Builder, n *sitter.Node, opts Options) {
	if opts.NoLines {
		return
	}
	start, end := n.StartPosition().Row+1, n.EndPosition().Row+1
	if start == end {
		fmt.Fprintf(out, "L%d ", start)
	} else {
		fmt.Fprintf(out, "L%d-%d ", start, end)
	}
}

func scriptDocs(out *strings.Builder, source []byte, docs []*sitter.Node, n *sitter.Node, opts Options, prefix string) {
	if opts.NoDocs || len(docs) == 0 || docs[len(docs)-1].EndPosition().Row+1 != n.StartPosition().Row {
		return
	}
	// ponytail: Only adjacent line comments are compacted; add structured documentation parsing when needed.
	for _, d := range docs {
		raw := string(source[d.StartByte():d.EndByte()])
		if opts.FullDocs {
			out.WriteString(raw + "\n")
		} else if s := strings.TrimSpace(strings.TrimPrefix(raw, prefix)); s != "" {
			out.WriteString(compactDoc(s) + "\n")
		}
	}
}
