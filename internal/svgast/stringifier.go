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
	indentCache string   // lazily grown repetitions of indent
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

	var b strings.Builder
	stringifyChildren(&b, root, opts, st)
	svg := b.String()

	if opts.FinalNewline && len(svg) > 0 && !strings.HasSuffix(svg, "\n") {
		svg += eol
	}

	return svg
}

// stringifyChildren appends all children of a parent node to b.
func stringifyChildren(b *strings.Builder, parent Parent, opts *StringifyOptions, st *stringifyState) {
	st.indentLevel++
	for _, child := range parent.GetChildren() {
		switch n := child.(type) {
		case *Element:
			stringifyElement(b, n, opts, st)
		case *Text:
			stringifyText(b, n, opts, st)
		case *Doctype:
			stringifyDoctype(b, n, st)
		case *Instruction:
			stringifyInstruction(b, n, st)
		case *Comment:
			stringifyComment(b, n, st)
		case *Cdata:
			stringifyCdata(b, n, opts, st)
		}
	}
	st.indentLevel--
}

// writeIndent appends the indentation for the current level to b. The repeated
// indent is grown once and sliced, instead of a strings.Repeat per node.
func writeIndent(b *strings.Builder, opts *StringifyOptions, st *stringifyState) {
	if !opts.Pretty || st.textContext != nil {
		return
	}
	n := st.indentLevel - 1
	if n <= 0 {
		return
	}
	need := n * len(st.indent)
	for len(st.indentCache) < need {
		st.indentCache += st.indent
	}
	b.WriteString(st.indentCache[:need])
}

func stringifyDoctype(b *strings.Builder, n *Doctype, st *stringifyState) {
	b.WriteString("<!DOCTYPE")
	b.WriteString(n.Data.Doctype)
	b.WriteString(st.doctypeEnd)
}

func stringifyInstruction(b *strings.Builder, n *Instruction, st *stringifyState) {
	b.WriteString("<?")
	b.WriteString(n.Name)
	b.WriteByte(' ')
	b.WriteString(n.Value)
	b.WriteString(st.procInstEnd)
}

func stringifyComment(b *strings.Builder, n *Comment, st *stringifyState) {
	b.WriteString("<!--")
	b.WriteString(n.Value)
	b.WriteString(st.commentEnd)
}

func stringifyCdata(b *strings.Builder, n *Cdata, opts *StringifyOptions, st *stringifyState) {
	writeIndent(b, opts, st)
	b.WriteString("<![CDATA[")
	b.WriteString(n.Value)
	b.WriteString(st.cdataEnd)
}

func stringifyElement(b *strings.Builder, n *Element, opts *StringifyOptions, st *stringifyState) {
	// Empty element
	if len(n.Children) == 0 {
		writeIndent(b, opts, st)
		b.WriteByte('<')
		b.WriteString(n.Name)
		stringifyAttributes(b, n)
		if opts.UseShortTags {
			b.WriteString(st.tagShortEnd)
			return
		}
		b.WriteString(st.tagOpenEnd)
		b.WriteString("</")
		b.WriteString(n.Name)
		b.WriteString(st.tagCloseEnd)
		return
	}

	// Non-empty element
	tagOpenEnd := st.tagOpenEnd
	tagCloseEnd := st.tagCloseEnd
	indentOpen := true
	indentClose := true
	enterTextContext := false

	if st.textContext != nil {
		// Inside a text element — use bare tags, no indentation
		tagOpenEnd = ">"
		tagCloseEnd = ">"
		indentOpen = false
	} else if IsTextElem(n.Name) {
		// This IS a text element — no EOL after open/before close
		tagOpenEnd = ">"
		indentClose = false
		enterTextContext = true
	}

	// The open tag is still indented at the outer level, so the indent must be
	// emitted before this element becomes the text context.
	if indentOpen {
		writeIndent(b, opts, st)
	}
	if enterTextContext {
		st.textContext = n
	}
	b.WriteByte('<')
	b.WriteString(n.Name)
	stringifyAttributes(b, n)
	b.WriteString(tagOpenEnd)

	stringifyChildren(b, n, opts, st)

	if st.textContext == n {
		st.textContext = nil
	}

	if indentClose {
		writeIndent(b, opts, st)
	}
	b.WriteString("</")
	b.WriteString(n.Name)
	b.WriteString(tagCloseEnd)
}

func stringifyAttributes(b *strings.Builder, n *Element) {
	for _, attr := range n.Attributes.Entries() {
		b.WriteByte(' ')
		b.WriteString(attr.Name)
		if attr.Value != UndefinedAttrValue {
			b.WriteString("=\"")
			writeEncodedAttrValue(b, attr.Value)
			b.WriteByte('"')
		}
	}
}

// writeEncodedTextValue appends text content with entities encoded: & ' " > <
func writeEncodedTextValue(b *strings.Builder, s string) {
	for i := range len(s) {
		if ent, ok := textEntities[s[i]]; ok {
			b.WriteString(ent)
		} else {
			b.WriteByte(s[i])
		}
	}
}

// writeEncodedAttrValue appends an attribute value with entities encoded: & " > <
func writeEncodedAttrValue(b *strings.Builder, s string) {
	for i := range len(s) {
		if ent, ok := attrEntities[s[i]]; ok {
			b.WriteString(ent)
		} else {
			b.WriteByte(s[i])
		}
	}
}

func stringifyText(b *strings.Builder, n *Text, opts *StringifyOptions, st *stringifyState) {
	writeIndent(b, opts, st)
	writeEncodedTextValue(b, n.Value)
	if st.textContext == nil {
		b.WriteString(st.textEnd)
	}
}
