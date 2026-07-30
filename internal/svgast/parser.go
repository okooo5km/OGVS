// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// entityDeclaration matches ENTITY declarations in DOCTYPE internal subset.
var entityDeclaration = regexp.MustCompile(`<!ENTITY\s+(\S+)\s+(?:'([^']+)'|"([^"]+)")\s*>`)

// MaxNestingDepth caps how deeply elements may nest. Deep nesting is a
// denial-of-service vector: several parts of the pipeline (notably
// css.ComputeStyle's ancestor walk, which mirrors SVGO's) cost O(depth) per
// element, so a spine of N nested elements costs O(N^2).
//
// The bound is chosen far above anything legitimate: across 31809 SVG files on
// a developer machine the deepest was 11 levels, and SVGO's own 363 plugin
// fixtures top out at 6. Node SVGO itself throws
// "RangeError: Maximum call stack size exceeded" from its stringifier at around
// 2300 levels, so this limit never rejects a file the reference implementation
// would have optimized.
const MaxNestingDepth = 1024

// MaxNodeCount caps the number of elements in a document. Every plugin visits
// every node, so the element count is a direct multiplier on the whole
// pipeline. The largest real-world SVG found in the same 31809-file scan had
// 14730 elements, so this leaves roughly 70x headroom.
const MaxNodeCount = 1 << 20

// ParserError represents an SVG parsing error with location information.
type ParserError struct {
	Message string
	Reason  string
	Line    int
	Column  int
	Source  string
	File    string
}

func (e *ParserError) Error() string {
	return e.Message
}

// FormatError formats the error with source code context,
// matching SVGO's SvgoParserError.toString() output.
func (e *ParserError) FormatError() string {
	file := e.File
	if file == "" {
		file = "<input>"
	}
	header := fmt.Sprintf("SvgoParserError: %s:%d:%d: %s", file, e.Line, e.Column, e.Reason)

	lines := strings.Split(e.Source, "\n")
	startLine := max(e.Line-3, 0)
	endLine := min(e.Line+2, len(lines))

	lineNumWidth := len(fmt.Sprintf("%d", endLine))

	var code strings.Builder
	for i := startLine; i < endLine; i++ {
		num := i + 1
		gutter := fmt.Sprintf(" %*d | ", lineNumWidth, num)
		line := lines[i]

		if num == e.Line {
			code.WriteString(">" + gutter + line + "\n")
			spacing := strings.Repeat(" ", len(gutter)+1)
			col := min(e.Column-1, len(line))
			lineSpacing := strings.Repeat(" ", col)
			code.WriteString(" " + spacing + lineSpacing + "^")
		} else {
			code.WriteString(" " + gutter + line)
		}
		if i < endLine-1 {
			code.WriteString("\n")
		}
	}

	return header + "\n\n" + code.String() + "\n"
}

// ParseSvg converts an SVG XML string to an XAST tree.
//
// This implementation uses encoding/xml's Decoder in RawToken mode,
// which preserves original namespace prefixes without URI expansion.
// This matches SVGO's SAX-based parser behavior:
// - Element and attribute names preserve original case
// - Comments are trimmed
// - Text in non-textElems is trimmed (empty text nodes are dropped)
// - DOCTYPE with ENTITY declarations are handled
// - Processing instructions are preserved
func ParseSvg(data string, from string) (*Root, error) {
	data = toValidUTF8(data)

	root := &Root{}
	var current Parent = root
	stack := []Parent{root}
	nodeCount := 0

	// Entity map for custom entities from DOCTYPE
	entities := make(map[string]string)

	// Pre-scan for DOCTYPE to extract entity declarations.
	// encoding/xml handles standard entities but not custom ones.
	doctypeNode := extractDoctype(data, entities)

	// If there's a DOCTYPE, add it to root first
	if doctypeNode != nil {
		root.Children = append(root.Children, doctypeNode)
	}

	// Replace custom entities in the data before parsing
	processedData, err := replaceEntities(data, entities)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(strings.NewReader(processedData))
	decoder.Strict = true
	// Allow custom entity handling
	decoder.Entity = entities
	// Preserve original attribute values
	decoder.AutoClose = nil
	// Support non-UTF-8 encodings (e.g. UTF-16)
	decoder.CharsetReader = charsetReader

	for {
		tok, err := decoder.RawToken()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			// Parse error - try to extract location
			offset := decoder.InputOffset()
			line, col := offsetToLineCol(data, int(offset))
			return nil, &ParserError{
				Message: fmt.Sprintf("%s:%d:%d: %s", fileOrInput(from), line, col, err.Error()),
				Reason:  err.Error(),
				Line:    line,
				Column:  col,
				Source:  data,
				File:    from,
			}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			t = xml.CopyToken(t).(xml.StartElement)

			// Build element name from raw prefix + local
			elemName := t.Name.Local
			if t.Name.Space != "" {
				elemName = t.Name.Space + ":" + t.Name.Local
			}

			elem := &Element{
				Name:       elemName,
				Attributes: NewOrderedAttrs(),
				Children:   nil,
			}

			// Add attributes in order, preserving original prefixes
			for _, attr := range t.Attr {
				name := rawAttrName(attr.Name)
				elem.Attributes.Set(name, attr.Value)
			}

			current.SetChildren(append(current.GetChildren(), elem))
			current = elem
			stack = append(stack, elem)
			if len(stack)-1 > MaxNestingDepth {
				return nil, fmt.Errorf("element nesting exceeds %d levels (possible denial-of-service input)", MaxNestingDepth)
			}
			nodeCount++
			if nodeCount > MaxNodeCount {
				return nil, fmt.Errorf("document exceeds %d nodes (possible denial-of-service input)", MaxNodeCount)
			}

		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
				current = stack[len(stack)-1]
			}

		case xml.CharData:
			text := string(t)

			// Detect CDATA sections by checking the raw input.
			// encoding/xml merges CDATA into CharData tokens, losing the distinction.
			// After consuming <![CDATA[content]]>, InputOffset() points right after ]]>.
			// In valid XML, ]]> can only appear at the end of a CDATA section.
			isCdata := false
			offset := int(decoder.InputOffset())
			if offset >= 3 && offset <= len(processedData) && processedData[offset-3:offset] == "]]>" {
				isCdata = true
			}

			// Text outside the root element is not well-formed. SVGO's SAX
			// parser rejects it; without this check a document with no markup
			// at all parses into an empty tree and stringifies to nothing,
			// which in the CLI's default in-place mode destroys the file.
			if _, inElement := current.(*Element); !inElement {
				if strings.TrimSpace(text) != "" {
					reason := "Text data outside of root node."
					if len(root.Children) == 0 || !hasElementChild(root) {
						reason = "Non-whitespace before first tag."
					}
					start := int(decoder.InputOffset()) - len(text)
					line, col := offsetToLineCol(data, max(start, 0))
					return nil, &ParserError{
						Message: fmt.Sprintf("%s:%d:%d: %s", fileOrInput(from), line, col, reason),
						Reason:  reason,
						Line:    line,
						Column:  col,
						Source:  data,
						File:    from,
					}
				}
			}

			// Check if current element is a textElem
			if elem, ok := current.(*Element); ok {
				if IsTextElem(elem.Name) {
					// Preserve whitespace in text elements
					var node Node
					if isCdata {
						node = &Cdata{Value: text}
					} else {
						node = &Text{Value: text}
					}
					current.SetChildren(append(current.GetChildren(), node))
				} else if isCdata {
					// CDATA content is always preserved as-is (no trimming)
					node := &Cdata{Value: text}
					current.SetChildren(append(current.GetChildren(), node))
				} else {
					// Trim text, skip if empty
					trimmed := strings.TrimSpace(text)
					if trimmed != "" {
						node := &Text{Value: trimmed}
						current.SetChildren(append(current.GetChildren(), node))
					}
				}
			}

		case xml.Comment:
			node := &Comment{Value: strings.TrimSpace(string(t))}
			current.SetChildren(append(current.GetChildren(), node))

		case xml.ProcInst:
			node := &Instruction{
				Name:  t.Target,
				Value: string(t.Inst),
			}
			current.SetChildren(append(current.GetChildren(), node))

		case xml.Directive:
			// Directives include DOCTYPE
			// We already pre-scanned for DOCTYPE, so skip duplicate
			dir := string(t)
			if !strings.HasPrefix(strings.TrimSpace(strings.ToUpper(dir)), "DOCTYPE") {
				// Non-DOCTYPE directive, preserve as-is
				_ = dir
			}
		}
	}

	return root, nil
}

// hasElementChild reports whether the root already holds an element, i.e.
// whether any markup has been seen.
func hasElementChild(root *Root) bool {
	for _, child := range root.Children {
		if _, ok := child.(*Element); ok {
			return true
		}
	}
	return false
}

// toValidUTF8 replaces invalid UTF-8 bytes with U+FFFD. SVGO reads its input
// through Node's UTF-8 decoder, which substitutes the replacement character
// and carries on; Go's encoding/xml instead rejects the whole document, so
// legacy Latin-1 files and stray high bytes have to be folded first.
func toValidUTF8(data string) string {
	if utf8.ValidString(data) {
		return data
	}
	return strings.ToValidUTF8(data, "�")
}

// rawAttrName builds the attribute name from the raw xml.Name,
// preserving the original namespace prefix.
func rawAttrName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

// extractDoctype pre-scans the data for a DOCTYPE declaration and extracts
// entity declarations from the internal subset.
func extractDoctype(data string, entities map[string]string) *Doctype {
	// Find DOCTYPE in the raw text
	upper := strings.ToUpper(data)
	idx := strings.Index(upper, "<!DOCTYPE")
	if idx < 0 {
		return nil
	}

	// Find the end of the DOCTYPE declaration
	// Handle nested [...] subset
	start := idx + len("<!DOCTYPE")
	depth := 0
	end := -1
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '[':
			depth++
		case ']':
			depth--
		case '>':
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil
	}

	doctype := data[start:end]

	// Extract entity declarations from internal subset
	subsetStart := strings.Index(doctype, "[")
	if subsetStart >= 0 {
		matches := entityDeclaration.FindAllStringSubmatch(doctype[subsetStart:], -1)
		for _, m := range matches {
			name := m[1]
			value := m[2]
			if value == "" {
				value = m[3]
			}
			entities[name] = value
		}
	}

	return &Doctype{
		Name: "svg",
		Data: DoctypeData{
			Doctype: doctype,
		},
	}
}

// maxEntityExpansion caps the size of the entity-expanded document. Internal
// DTD entities can be crafted to expand exponentially (billion laughs) or
// quadratically (one large value referenced many times), exhausting memory.
// A legitimate SVG never needs this much expansion; anything larger is rejected.
const maxEntityExpansion = 32 * 1024 * 1024 // 32 MB

// errEntityExpansion is returned once expansion outgrows maxEntityExpansion.
var errEntityExpansion = fmt.Errorf("entity expansion exceeds %d bytes (possible entity-expansion attack)", maxEntityExpansion)

// entityResolver expands entity references to a fixed point.
//
// SVGO configures sax with unparsedEntities, which feeds an entity's value
// back through the parser, so an entity whose value references another entity
// resolves all the way down. Values are therefore expanded before they are
// substituted into the document, and both the per-value and the cumulative
// sizes are bounded so that recursive or exponentially nested declarations
// cannot exhaust memory.
type entityResolver struct {
	entities map[string]string
	resolved map[string]string
	visiting map[string]bool
	total    int
	err      error
}

func newEntityResolver(entities map[string]string) *entityResolver {
	return &entityResolver{
		entities: entities,
		resolved: make(map[string]string, len(entities)),
		visiting: make(map[string]bool, len(entities)),
	}
}

// isEntityNameByte reports whether c may appear inside an entity name. Names
// come from the DOCTYPE internal subset, where they are matched as a run of
// non-whitespace, so only the characters that end or invalidate a reference
// are excluded.
func isEntityNameByte(c byte) bool {
	switch c {
	case ';', '&', '<', '>', ' ', '\t', '\r', '\n':
		return false
	}
	return true
}

// lookup returns the fully expanded value of a declared entity.
func (r *entityResolver) lookup(name string) (string, bool) {
	raw, declared := r.entities[name]
	if !declared {
		return "", false
	}
	if value, ok := r.resolved[name]; ok {
		return value, true
	}
	if r.visiting[name] {
		// Self- or mutually-recursive declaration: keep the reference literal
		// so expansion terminates.
		return "&" + name + ";", true
	}

	r.visiting[name] = true
	value, err := r.expand(raw)
	delete(r.visiting, name)
	if err != nil {
		r.err = err
		return "", false
	}

	r.total += len(value)
	if r.total > maxEntityExpansion {
		r.err = errEntityExpansion
		return "", false
	}
	r.resolved[name] = value
	return value, true
}

// expand rewrites every declared entity reference in s with its expanded
// value, scanning left to right so the result never depends on map order.
func (r *entityResolver) expand(s string) (string, error) {
	if !strings.ContainsRune(s, '&') {
		return s, nil
	}

	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i + 1
		for j < len(s) && isEntityNameByte(s[j]) {
			j++
		}
		if j < len(s) && s[j] == ';' && j > i+1 {
			if value, ok := r.lookup(s[i+1 : j]); ok {
				b.WriteString(value)
				if b.Len() > maxEntityExpansion {
					return "", errEntityExpansion
				}
				i = j + 1
				continue
			}
			if r.err != nil {
				return "", r.err
			}
		}
		b.WriteByte('&')
		i++
	}
	return b.String(), nil
}

// replaceEntities replaces custom entity references in the data. It bounds the
// total expanded size to prevent entity-expansion denial-of-service attacks.
func replaceEntities(data string, entities map[string]string) (string, error) {
	if len(entities) == 0 {
		return data, nil
	}

	r := newEntityResolver(entities)

	// Expand every declared value up front, in name order, so that a cycle is
	// always broken at the same reference regardless of where the document
	// happens to enter it.
	names := make([]string, 0, len(entities))
	for name := range entities {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, ok := r.lookup(name); !ok && r.err != nil {
			return "", r.err
		}
	}

	result, err := r.expand(data)
	if err != nil {
		return "", err
	}
	if r.err != nil {
		return "", r.err
	}
	return result, nil
}

// offsetToLineCol converts a byte offset to line and column numbers.
func offsetToLineCol(data string, offset int) (int, int) {
	offset = min(offset, len(data))
	line := 1
	col := 1
	for i := 0; i < offset; i++ {
		if data[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func fileOrInput(from string) string {
	if from == "" {
		return "<input>"
	}
	return from
}

// charsetReader returns a reader for the named charset.
// SVGO's SAX parser ignores encoding declarations and always treats input as
// UTF-8. Many SVG files in the wild declare encoding="utf-16" but are actually
// UTF-8, so we match SVGO behavior by returning the input reader unchanged.
func charsetReader(_ string, input io.Reader) (io.Reader, error) {
	return input, nil
}
