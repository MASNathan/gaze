package gaze

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

// goOutlineOptions extracts Go declarations without executable bodies or initializers.
func goOutlineOptions(source []byte, opts Options) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments|parser.AllErrors)
	if file == nil {
		return "", fmt.Errorf("parse Go source: %w", err)
	}
	var out strings.Builder
	if !opts.NoLines {
		fmt.Fprintf(&out, "L%d ", fset.Position(file.Package).Line)
	}
	fmt.Fprintf(&out, "package %s\n", file.Name.Name)
	write := func(node ast.Node, doc *ast.CommentGroup, value any) error {
		if doc != nil && !opts.NoDocs {
			if opts.FullDocs {
				for _, c := range doc.List {
					out.WriteString(c.Text + "\n")
				}
			} else if text := strings.TrimSpace(doc.Text()); text != "" {
				out.WriteString(compactDoc(strings.Join(strings.Fields(text), " ")) + "\n")
			}
		}
		if !opts.NoLines {
			start, end := fset.Position(node.Pos()).Line, fset.Position(node.End()).Line
			if start == end {
				fmt.Fprintf(&out, "L%d ", start)
			} else {
				fmt.Fprintf(&out, "L%d-%d ", start, end)
			}
		}
		var b bytes.Buffer
		if err := printer.Fprint(&b, fset, value); err != nil {
			return fmt.Errorf("format Go declaration: %w", err)
		}
		out.WriteString(b.String() + "\n")
		return nil
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !opts.IncludeNonPublic && !d.Name.IsExported() {
				continue
			}
			copy := *d
			copy.Body = nil
			copy.Doc = nil
			if err := write(d, d.Doc, &copy); err != nil {
				return "", err
			}
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if !opts.IncludeNonPublic && !s.Name.IsExported() {
						continue
					}
					doc := s.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					original := &ast.BadDecl{From: s.Pos(), To: s.End()}
					// Remove field comments and unexported fields from the parsed tree.
					ast.Inspect(s.Type, func(n ast.Node) bool {
						var fields *ast.FieldList
						switch t := n.(type) {
						case *ast.StructType:
							fields = t.Fields
						case *ast.InterfaceType:
							fields = t.Methods
						}
						if fields == nil {
							return true
						}
						var kept []*ast.Field
						for _, field := range fields.List {
							field.Doc, field.Comment = nil, nil
							if !opts.IncludeNonPublic && len(field.Names) > 0 {
								var names []*ast.Ident
								for _, name := range field.Names {
									if name.IsExported() {
										names = append(names, name)
									}
								}
								if len(names) == 0 {
									continue
								}
								field.Names = names
							} else if !opts.IncludeNonPublic && !goEmbeddedExported(field.Type) {
								continue
							}
							kept = append(kept, field)
						}
						fields.List = kept
						return true
					})
					copy := *s
					copy.Doc, copy.Comment = nil, nil
					if err := write(original, doc, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&copy}}); err != nil {
						return "", err
					}
				case *ast.ValueSpec:
					var names []*ast.Ident
					for _, name := range s.Names {
						if opts.IncludeNonPublic || name.IsExported() {
							names = append(names, name)
						}
					}
					if len(names) == 0 {
						continue
					}
					doc := s.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					copy := *s
					copy.Names, copy.Values, copy.Doc, copy.Comment = names, nil, nil, nil
					if err := write(s, doc, &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{&copy}}); err != nil {
						return "", err
					}
				}
			}
		}
	}
	return out.String(), nil
}

func goEmbeddedExported(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.IsExported()
	case *ast.SelectorExpr:
		return e.Sel.IsExported()
	case *ast.StarExpr:
		return goEmbeddedExported(e.X)
	case *ast.IndexExpr:
		return goEmbeddedExported(e.X)
	case *ast.IndexListExpr:
		return goEmbeddedExported(e.X)
	default:
		// Interface type terms are constraints, not unexported embedded fields.
		return true
	}
}
