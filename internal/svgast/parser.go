// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// entityDeclaration matches ENTITY declarations in DOCTYPE internal subset.
var entityDeclaration = regexp.MustCompile(`<!ENTITY\s+(\S+)\s+(?:'([^']+)'|"([^"]+)")\s*>`)

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
	root := &Root{}
	var current Parent = root
	stack := []Parent{root}

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
	processedData := replaceEntities(data, entities)

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

// replaceEntities replaces custom entity references in the data.
func replaceEntities(data string, entities map[string]string) string {
	if len(entities) == 0 {
		return data
	}
	result := data
	for name, value := range entities {
		result = strings.ReplaceAll(result, "&"+name+";", value)
	}
	return result
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
