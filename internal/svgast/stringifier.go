// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

import (
	"strings"
)

// StringifyOptions configures the SVG stringifier.
type StringifyOptions struct {
	Pretty       bool   // enable pretty-printing with indentation
	Indent       int    // indentation width (negative = use tabs, 0+ = spaces)
	UseShortTags bool   // use self-closing tags for empty elements
	EOL          string // "lf" or "crlf" (default: "lf")
	FinalNewline bool   // add newline at end of output
}

// DefaultStringifyOptions returns the default options matching SVGO defaults.
func DefaultStringifyOptions() *StringifyOptions {
	return &StringifyOptions{
		Pretty:       false,
		Indent:       4,
		UseShortTags: true,
		EOL:          "lf",
		FinalNewline: false,
	}
}

// Entity encoding maps
var (
	textEntities = map[byte]string{
		'&':  "&amp;",
		'\'': "&apos;",
		'"':  "&quot;",
		'>':  "&gt;",
		'<':  "&lt;",
	}
	attrEntities = map[byte]string{
		'&': "&amp;",
		'"': "&quot;",
		'>': "&gt;",
		'<': "&lt;",
	}
)

// stringifyState holds mutable state during stringification.
type stringifyState struct {
	indent      string   // the indentation string per level
	textContext *Element // non-nil when inside a text element
	indentLevel int
	eol         string // resolved EOL string

	// Pretty-mode suffixes (pre-computed with EOL)
	doctypeEnd  string
	procInstEnd string
	commentEnd  string
	cdataEnd    string
	tagShortEnd string
	tagOpenEnd  string
	tagCloseEnd string
	textEnd     string
}

// StringifySvg converts an XAST Root back to an SVG string.
//
// This precisely matches SVGO's stringifier.js behavior:
// - Entity encoding for text content and attribute values
// - Pretty-printing with configurable indentation
// - Short tags for empty elements (when enabled)
// - Text element handling (no indentation inside text/tspan/etc.)
// - EOL handling (lf/crlf)
// - Optional final newline
func StringifySvg(root *Root, opts *StringifyOptions) string {
	if opts == nil {
		opts = DefaultStringifyOptions()
	}

	// Build indent string
	var indent string
	if opts.Indent < 0 {
		indent = "\t"
	} else {
		indent = strings.Repeat(" ", opts.Indent)
	}

	// Resolve EOL
	eol := "\n"
	if opts.EOL == "crlf" {
		eol = "\r\n"
	}

	st := &stringifyState{
		indent:      indent,
		textContext: nil,
		indentLevel: 0,
		eol:         eol,
		// Base suffixes (without EOL)
		doctypeEnd:  ">",
		procInstEnd: "?>",
		commentEnd:  "-->",
		cdataEnd:    "]]>",
		tagShortEnd: "/>",
		tagOpenEnd:  ">",
		tagCloseEnd: ">",
		textEnd:     "",
	}

	// In pretty mode, append EOL to all suffixes
	if opts.Pretty {
		st.doctypeEnd += eol
		st.procInstEnd += eol
		st.commentEnd += eol
		st.cdataEnd += eol
		st.tagShortEnd += eol
		st.tagOpenEnd += eol
		st.tagCloseEnd += eol
		st.textEnd += eol
	}

	svg := stringifyChildren(root, opts, st)

	if opts.FinalNewline && len(svg) > 0 && !strings.HasSuffix(svg, "\n") {
		svg += eol
	}

	return svg
}

// stringifyChildren stringifies all children of a parent node.
func stringifyChildren(parent Parent, opts *StringifyOptions, st *stringifyState) string {
	var b strings.Builder
	st.indentLevel++
	for _, child := range parent.GetChildren() {
		switch n := child.(type) {
		case *Element:
			b.WriteString(stringifyElement(n, opts, st))
		case *Text:
			b.WriteString(stringifyText(n, opts, st))
		case *Doctype:
			b.WriteString(stringifyDoctype(n, st))
		case *Instruction:
			b.WriteString(stringifyInstruction(n, st))
		case *Comment:
			b.WriteString(stringifyComment(n, st))
		case *Cdata:
			b.WriteString(stringifyCdata(n, opts, st))
		}
	}
	st.indentLevel--
	return b.String()
}

// createIndent returns the indentation string for the current level.
func createIndent(opts *StringifyOptions, st *stringifyState) string {
	if opts.Pretty && st.textContext == nil {
		return strings.Repeat(st.indent, st.indentLevel-1)
	}
	return ""
}

func stringifyDoctype(n *Doctype, st *stringifyState) string {
	return "<!DOCTYPE" + n.Data.Doctype + st.doctypeEnd
}

func stringifyInstruction(n *Instruction, st *stringifyState) string {
	return "<?" + n.Name + " " + n.Value + st.procInstEnd
}

func stringifyComment(n *Comment, st *stringifyState) string {
	return "<!--" + n.Value + st.commentEnd
}

func stringifyCdata(n *Cdata, opts *StringifyOptions, st *stringifyState) string {
	return createIndent(opts, st) + "<![CDATA[" + n.Value + st.cdataEnd
}

func stringifyElement(n *Element, opts *StringifyOptions, st *stringifyState) string {
	// Empty element
	if len(n.Children) == 0 {
		if opts.UseShortTags {
			return createIndent(opts, st) +
				"<" + n.Name + stringifyAttributes(n) +
				st.tagShortEnd
		}
		return createIndent(opts, st) +
			"<" + n.Name + stringifyAttributes(n) +
			st.tagOpenEnd +
			"</" + n.Name + st.tagCloseEnd
	}

	// Non-empty element
	tagOpenEnd := st.tagOpenEnd
	tagCloseEnd := st.tagCloseEnd
	openIndent := createIndent(opts, st)
	closeIndent := createIndent(opts, st)

	if st.textContext != nil {
		// Inside a text element — use bare tags, no indentation
		tagOpenEnd = ">"
		tagCloseEnd = ">"
		openIndent = ""
	} else if IsTextElem(n.Name) {
		// This IS a text element — no EOL after open/before close
		tagOpenEnd = ">"
		closeIndent = ""
		st.textContext = n
	}

	children := stringifyChildren(n, opts, st)

	if st.textContext == n {
		st.textContext = nil
	}

	// Use the base tagCloseStart/End for close tag
	return openIndent +
		"<" + n.Name + stringifyAttributes(n) + tagOpenEnd +
		children +
		closeIndent + "</" + n.Name + tagCloseEnd
}

func stringifyAttributes(n *Element) string {
	if n.Attributes.Len() == 0 {
		return ""
	}
	var b strings.Builder
	for _, attr := range n.Attributes.Entries() {
		b.WriteByte(' ')
		b.WriteString(attr.Name)
		if attr.Value != UndefinedAttrValue {
			b.WriteString("=\"")
			b.WriteString(encodeAttrValue(attr.Value))
			b.WriteByte('"')
		}
	}
	return b.String()
}

// encodeTextValue encodes text content entities: & ' " > <
func encodeTextValue(s string) string {
	var b strings.Builder
	for i := range len(s) {
		if ent, ok := textEntities[s[i]]; ok {
			b.WriteString(ent)
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// encodeAttrValue encodes attribute value entities: & " > <
func encodeAttrValue(s string) string {
	var b strings.Builder
	for i := range len(s) {
		if ent, ok := attrEntities[s[i]]; ok {
			b.WriteString(ent)
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func stringifyText(n *Text, opts *StringifyOptions, st *stringifyState) string {
	indent := createIndent(opts, st)
	textEnd := ""
	if st.textContext == nil {
		textEnd = st.textEnd
	}
	return indent + encodeTextValue(n.Value) + textEnd
}
