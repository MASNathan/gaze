package gaze

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	bash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
)

func shellOutlineOptions(source []byte, opts Options) (string, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(bash.Language())); err != nil {
		return "", fmt.Errorf("set shell language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse shell source: no syntax tree")
	}
	defer tree.Close()
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var out strings.Builder
	var docs []*sitter.Node
	root := tree.RootNode()
	for i := uint(0); i < root.NamedChildCount(); i++ {
		n := root.NamedChild(i)
		if n.Kind() == "comment" {
			if strings.HasPrefix(text(n), "#!") || (len(docs) > 0 && n.StartPosition().Row != docs[len(docs)-1].EndPosition().Row+1) {
				docs = nil
			}
			if !strings.HasPrefix(text(n), "#!") {
				docs = append(docs, n)
			}
			continue
		}
		if len(docs) > 0 && n.StartPosition().Row != docs[len(docs)-1].EndPosition().Row+1 {
			docs = nil
		}
		var headers []string
		if n.Kind() == "function_definition" {
			name := n.ChildByFieldName("name")
			if name != nil && (opts.IncludeNonPublic || !strings.HasPrefix(text(name), "_")) {
				end := n.EndByte()
				if body := n.ChildByFieldName("body"); body != nil {
					end = body.StartByte()
				}
				headers = append(headers, strings.TrimSpace(string(source[n.StartByte():end])))
			}
		} else if len(docs) > 0 && (n.Kind() == "variable_assignment" || n.Kind() == "declaration_command") {
			var assignments []*sitter.Node
			if n.Kind() == "variable_assignment" {
				assignments = append(assignments, n)
			} else {
				for j := uint(0); j < n.NamedChildCount(); j++ {
					if c := n.NamedChild(j); c.Kind() == "variable_assignment" {
						assignments = append(assignments, c)
					}
				}
			}
			for _, assignment := range assignments {
				name := assignment.ChildByFieldName("name")
				if name != nil && (opts.IncludeNonPublic || !strings.HasPrefix(text(name), "_")) {
					headers = append(headers, text(name)+"=…")
				}
			}
		}
		if len(headers) > 0 {
			if !opts.NoDocs && len(docs) > 0 {
				var lines []string
				for _, doc := range docs {
					line := text(doc)
					if !opts.FullDocs {
						line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
					}
					if line != "" {
						lines = append(lines, line)
					}
				}
				if opts.FullDocs {
					out.WriteString(strings.Join(lines, "\n"))
				} else {
					out.WriteString(compactDoc(strings.Join(lines, " | ")))
				}
				if len(lines) > 0 {
					out.WriteByte('\n')
				}
			}
			loc := ""
			if !opts.NoLines {
				a, b := n.StartPosition().Row+1, n.EndPosition().Row+1
				loc = fmt.Sprintf("L%d ", a)
				if a != b {
					loc = fmt.Sprintf("L%d-%d ", a, b)
				}
			}
			for _, header := range headers {
				fmt.Fprintf(&out, "%s%s\n", loc, header)
			}
		}
		docs = nil
	}
	return out.String(), nil
}
