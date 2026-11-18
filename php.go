package gaze

import (
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	php "github.com/tree-sitter/tree-sitter-php/bindings/go"
)

// phpOutline omits imports, bodies, attributes and initializer expressions; names and types remain.
func phpOutline(source []byte, all bool, fullDocs bool) (string, error) {
	return phpOutlineOptions(source, Options{IncludeNonPublic: all, FullDocs: fullDocs})
}

func phpOutlineOptions(source []byte, opts Options) (string, error) {
	all, fullDocs := opts.IncludeNonPublic, opts.FullDocs
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(php.LanguagePHP())); err != nil {
		return "", fmt.Errorf("set PHP language: %w", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return "", fmt.Errorf("parse PHP source: no syntax tree")
	}
	defer tree.Close()
	var out strings.Builder
	var declarations func(*sitter.Node, bool)
	text := func(n *sitter.Node) string { return string(source[n.StartByte():n.EndByte()]) }
	var clean func(*sitter.Node, uint) string
	clean = func(n *sitter.Node, end uint) string {
		var b strings.Builder
		var keyword *sitter.Node
		if !fullDocs && (n.Kind() == "function_definition" || n.Kind() == "method_declaration") {
			for i := uint(0); i < n.ChildCount(); i++ {
				if c := n.Child(i); c.Kind() == "function" {
					keyword = c
					break
				}
			}
		}
		raw := func(start, end uint) {
			if keyword != nil && keyword.StartByte() >= start && keyword.EndByte() <= end {
				b.Write(source[start:keyword.StartByte()])
				b.WriteString("fn")
				b.Write(source[keyword.EndByte():end])
				return
			}
			b.Write(source[start:end])
		}
		pos := n.StartByte()
		for i := uint(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.StartByte() >= end {
				break
			}
			if c.EndByte() > end {
				continue
			}
			raw(pos, c.StartByte())
			switch c.Kind() {
			case "attribute_list", "property_hook_list":
				// Attributes and hooks can contain executable expressions.
			case "visibility_modifier":
				if fullDocs || text(c) != "public" || (n.Kind() != "method_declaration" && n.Kind() != "property_declaration" && n.Kind() != "const_declaration") || (n.Kind() == "property_declaration" && n.ChildByFieldName("type") == nil) {
					b.WriteString(text(c))
				}
			case "const_element":
				name := c.NamedChild(0)
				if name == nil {
					b.WriteString("…")
				} else {
					b.Write(source[c.StartByte():name.EndByte()])
					if c.EndByte() > name.EndByte() {
						b.WriteString(" = …")
					}
				}
			default:
				value := c.ChildByFieldName("default_value")
				if c.Kind() == "enum_case" {
					value = c.ChildByFieldName("value")
				}
				if value != nil {
					b.WriteString(clean(c, value.StartByte()))
					literal := strings.TrimSpace(text(value))
					if !fullDocs && (c.Kind() == "simple_parameter" || c.Kind() == "property_promotion_parameter") && phpSimpleDefault(value, literal) {
						if strings.EqualFold(literal, "null") || strings.EqualFold(literal, "true") || strings.EqualFold(literal, "false") {
							literal = strings.ToLower(literal)
						}
						b.WriteString(literal)
					} else {
						b.WriteString("…")
					}
					b.Write(source[value.EndByte():c.EndByte()])
				} else {
					b.WriteString(clean(c, c.EndByte()))
				}
			}
			pos = c.EndByte()
		}
		raw(pos, end)
		return b.String()
	}
	write := func(n *sitter.Node, doc *sitter.Node, indent string, header string) {
		if doc != nil {
			if fullDocs {
				if !opts.NoDocs {
					fmt.Fprintf(&out, "%s%s\n", indent, text(doc))
				}
			} else {
				var lines []string
				rawLines := strings.Split(strings.TrimSuffix(strings.TrimPrefix(text(doc), "/**"), "*/"), "\n")
				counts := map[string]int{}
				for _, raw := range rawLines {
					line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "*"))
					if phpDocHasTag(line, "@return") {
						counts["@return"]++
					} else if phpDocHasTag(line, "@param") {
						if name := phpDocParamName.FindString(line); name != "" {
							counts[name]++
						}
					}
				}
				for i, line := range rawLines {
					line = strings.TrimSpace(line)
					line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
					line = strings.ReplaceAll(line, "{@inheritDoc}", "")
					line = strings.ReplaceAll(line, "{@inheritdoc}", "")
					line = strings.TrimSpace(line)
					if strings.EqualFold(line, "@inheritDoc") || line == "" {
						continue
					}
					// A following continuation may describe this tag; retain it intact.
					continuation := false
					if i+1 < len(rawLines) {
						next := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rawLines[i+1]), "*"))
						continuation = next != "" && !strings.HasPrefix(next, "@")
					}
					key := phpDocParamName.FindString(line)
					if phpDocHasTag(line, "@return") {
						key = "@return"
					}
					if !continuation && counts[key] == 1 && (n.Kind() == "function_definition" || n.Kind() == "method_declaration") {
						line, header = phpDocType(line, n, source, header, tree.RootNode())
					}
					if line != "" {
						lines = append(lines, line)
					}
				}
				if !opts.NoDocs && len(lines) > 0 {
					fmt.Fprintf(&out, "%s%s\n", indent, strings.Join(lines, " | "))
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
		header = strings.TrimSpace(header)
		if !fullDocs {
			header = strings.TrimSpace(strings.TrimSuffix(header, ";"))
		}
		fmt.Fprintf(&out, "%s%s%s\n", indent, location, header)
	}
	declarations = func(parent *sitter.Node, member bool) {
		var doc *sitter.Node
		for i := uint(0); i < parent.NamedChildCount(); i++ {
			n := parent.NamedChild(i)
			kind := n.Kind()
			if kind == "comment" {
				if strings.HasPrefix(text(n), "/**") {
					doc = n
				} else {
					doc = nil
				}
				continue
			}
			if doc != nil && strings.TrimSpace(string(source[doc.EndByte():n.StartByte()])) != "" {
				doc = nil
			}
			if member && !all {
				private := false
				for j := uint(0); j < n.NamedChildCount(); j++ {
					c := n.NamedChild(j)
					if c.Kind() == "visibility_modifier" && text(c) != "public" {
						private = true
					}
				}
				if private {
					doc = nil
					continue
				}
			}
			indent := ""
			if member {
				indent = "  "
			}
			switch kind {
			case "namespace_definition":
				body := n.ChildByFieldName("body")
				if body != nil {
					write(n, doc, indent, clean(n, body.StartByte()))
					declarations(body, false)
				} else {
					write(n, doc, indent, text(n))
				}
			case "class_declaration", "interface_declaration", "trait_declaration", "enum_declaration":
				if member {
					break
				}
				body := n.ChildByFieldName("body")
				if body != nil {
					write(n, doc, indent, clean(n, body.StartByte()))
					declarations(body, true)
				} else {
					write(n, doc, indent, text(n))
				}
			case "function_definition", "method_declaration":
				body := n.ChildByFieldName("body")
				header := clean(n, n.EndByte())
				if body != nil {
					header = clean(n, body.StartByte())
				}
				write(n, doc, indent, header)
			case "property_declaration", "const_declaration", "enum_case", "use_declaration":
				if kind == "enum_case" && n.ChildByFieldName("value") != nil {
					value := n.ChildByFieldName("value")
					write(n, doc, indent, clean(n, value.StartByte())+"…"+string(source[value.EndByte():n.EndByte()]))
				} else {
					write(n, doc, indent, clean(n, n.EndByte()))
				}
			}
			doc = nil
		}
	}
	declarations(tree.RootNode(), false)
	return out.String(), nil
}

var phpDocParamName = regexp.MustCompile(`\$[a-zA-Z_\x80-\xff][a-zA-Z0-9_\x80-\xff]*`)
var phpSignedNumber = regexp.MustCompile(`^[+-](?:[0-9][0-9_]*(?:\.[0-9_]*)?(?:[eE][+-]?[0-9_]+)?|\.[0-9_]+)$`)

func phpSimpleDefault(n *sitter.Node, literal string) bool {
	switch n.Kind() {
	case "integer", "float", "string", "encapsed_string", "boolean", "null":
		return n.Kind() != "encapsed_string" || n.NamedChildCount() == 0
	case "array_creation_expression":
		return n.NamedChildCount() == 0
	case "unary_op_expression":
		return phpSignedNumber.MatchString(literal)
	}
	return false
}

var phpSimpleType = regexp.MustCompile(`^\\?[a-zA-Z_\x80-\xff][a-zA-Z0-9_\x80-\xff]*(?:\\[a-zA-Z_\x80-\xff][a-zA-Z0-9_\x80-\xff]*)*$`)

func phpSameType(doc, native string, source []byte, root *sitter.Node) bool {
	if doc == native {
		return true
	}
	// ponytail: Handle direct imports only; add grouped and aliased imports when needed.
	for i := uint(0); i < root.NamedChildCount(); i++ {
		n := root.NamedChild(i)
		if n.Kind() == "namespace_use_declaration" && strings.EqualFold(strings.TrimSpace(string(source[n.StartByte():n.EndByte()])), "use "+strings.TrimPrefix(doc, `\`)+";") && strings.EqualFold(strings.TrimPrefix(doc, `\`), native) {
			return true
		}
	}
	// ponytail: Without namespace/import resolution, only compare simple names in
	// files without namespace or use declarations. Add AST resolution when needed.
	return phpSimpleType.MatchString(doc) && phpSimpleType.MatchString(native) &&
		strings.TrimPrefix(doc, `\`) == strings.TrimPrefix(native, `\`) &&
		!phpNamespaceOrUse.Match(source)
}

var phpNamespaceOrUse = regexp.MustCompile(`(?i)\b(namespace|use)\b`)

func phpDocHasTag(line, tag string) bool {
	return strings.HasPrefix(line, tag) && len(line) > len(tag) && (line[len(tag)] == ' ' || line[len(tag)] == '\t')
}

// phpDocType only moves unambiguous single-line types; otherwise keep the tag intact.
func phpDocType(line string, n *sitter.Node, source []byte, header string, root *sitter.Node) (string, string) {
	param := phpDocHasTag(line, "@param")
	if !param && !phpDocHasTag(line, "@return") {
		return line, header
	}
	tag := "@return"
	if param {
		tag = "@param"
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, tag))
	depth := 0
	end := len(rest)
	for i, c := range rest {
		switch c {
		case '<', '{', '(', '[':
			depth++
		case '>', '}', ')', ']':
			depth--
		case ' ', '\t':
			if depth == 0 {
				end = i
				goto parsed
			}
		}
		if depth < 0 {
			return line, header
		}
	}
parsed:
	if depth != 0 || end == 0 {
		return line, header
	}
	typ := rest[:end]
	tail := strings.TrimSpace(rest[end:])
	var target *sitter.Node
	if param {
		fields := strings.Fields(tail)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "$") {
			return line, header
		}
		name := fields[0]
		tail = strings.TrimSpace(strings.TrimPrefix(tail, name))
		params := n.ChildByFieldName("parameters")
		if params != nil {
			for i := uint(0); i < params.NamedChildCount(); i++ {
				p := params.NamedChild(i)
				v := p.ChildByFieldName("name")
				if v != nil && string(source[v.StartByte():v.EndByte()]) == name {
					target = p
					break
				}
			}
		}
		if target == nil {
			return line, header
		}
		native := target.ChildByFieldName("type")
		if native != nil {
			if !phpSameType(typ, string(source[native.StartByte():native.EndByte()]), source, root) {
				return line, header
			}
		} else {
			v := target.ChildByFieldName("name")
			old := string(source[v.StartByte():v.EndByte()])
			// Only replace a unique occurrence; ambiguous headers retain the tag.
			if strings.Count(header, old) != 1 || strings.Contains(header, old+": doc<") {
				return line, header
			}
			prefix := string(source[target.StartByte():v.StartByte()])
			insert := old
			for _, marker := range []string{"...", "&"} {
				if before, ok := strings.CutSuffix(strings.TrimSpace(prefix), marker); ok {
					prefix = strings.TrimSpace(before)
					insert = marker + insert
				}
			}
			if strings.Count(header, insert) != 1 {
				return line, header
			}
			header = strings.Replace(header, insert, typ+" "+insert, 1)
			if tail != "" {
				return line, header
			}
			return "", header
		}
		if tail != "" {
			return line, header
		}
		return "", header
	}
	native := n.ChildByFieldName("return_type")
	if native == nil {
		trimmed := strings.TrimRight(header, " \t\r\n")
		if !strings.HasSuffix(trimmed, ")") && !strings.HasSuffix(trimmed, ");") {
			return line, header
		}
		if before, ok := strings.CutSuffix(trimmed, ";"); ok {
			header = before + ": " + typ + ";"
		} else {
			header = trimmed + ": " + typ
		}
		if tail != "" {
			return "@return " + tail, header
		}
		return "", header
	}
	if !phpSameType(typ, string(source[native.StartByte():native.EndByte()]), source, root) {
		return line, header
	}
	if tail != "" {
		return line, header
	}
	return "", header
}
