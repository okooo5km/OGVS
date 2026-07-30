// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package minifystyles minifies <style> elements and style attributes using CSS minification.
package minifystyles

import (
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "minifyStyles",
		Description: "minifies styles and removes unused styles",
		Fn:          fn,
	})
}

// usageConfig controls which types of unused selectors to remove.
type usageConfig struct {
	force   bool // override deoptimization (remove unused even with scripts)
	ids     bool // remove unused ID selectors
	classes bool // remove unused class selectors
	tags    bool // remove unused tag selectors
}

func fn(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	type styleEntry struct {
		node   *svgast.Element
		parent svgast.Parent
	}

	var styleElements []styleEntry
	var elementsWithStyleAttr []*svgast.Element
	deoptimized := false

	// Parse usage config
	usageEnabled := true // whether to remove unused selectors at all
	usage := usageConfig{
		force:   false,
		ids:     true,
		classes: true,
		tags:    true,
	}
	if v, ok := params["usage"].(bool); ok {
		usageEnabled = v
	} else if v, ok := params["usage"].(map[string]any); ok {
		if f, ok := v["force"].(bool); ok {
			usage.force = f
		}
		if f, ok := v["ids"].(bool); ok {
			usage.ids = f
		}
		if f, ok := v["classes"].(bool); ok {
			usage.classes = f
		}
		if f, ok := v["tags"].(bool); ok {
			usage.tags = f
		}
	}

	// Collect used tags, classes, IDs
	usedTags := make(map[string]bool)
	usedClasses := make(map[string]bool)
	usedIDs := make(map[string]bool)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem, ok := node.(*svgast.Element)
				if !ok {
					return nil
				}

				// detect scripts that deoptimize processing
				if tools.HasScripts(elem) {
					deoptimized = true
				}

				// collect usage data for every element, style elements
				// included. Tag names are lowercased because csso matches
				// type selectors case-insensitively.
				usedTags[strings.ToLower(elem.Name)] = true
				if classAttr, has := elem.Attributes.Get("class"); has {
					for _, cls := range strings.Fields(classAttr) {
						usedClasses[cls] = true
					}
				}
				if idAttr, has := elem.Attributes.Get("id"); has && idAttr != "" {
					usedIDs[idAttr] = true
				}

				// collect style elements or elements with a style attribute
				if elem.Name == "style" && len(elem.Children) > 0 {
					styleElements = append(styleElements, styleEntry{node: elem, parent: parent})
				} else if elem.Attributes.Has("style") {
					elementsWithStyleAttr = append(elementsWithStyleAttr, elem)
				}

				return nil
			},
		},
		Root: &svgast.VisitorCallbacks{
			Exit: func(node svgast.Node, parent svgast.Parent) {
				m := minify.New()
				m.AddFunc("text/css", css.Minify)

				// Determine if we should remove unused selectors
				removeUnused := usageEnabled && (!deoptimized || usage.force)
				idx := &usageIndex{
					tags:    usedTags,
					classes: usedClasses,
					ids:     usedIDs,
					cfg:     usage,
				}

				// minify style elements
				for _, entry := range styleElements {
					child := entry.node.Children[0]
					var cssText string
					var isCdata bool

					switch c := child.(type) {
					case *svgast.Text:
						cssText = c.Value
					case *svgast.Cdata:
						cssText = c.Value
						isCdata = true
					default:
						continue
					}

					// First minify to normalize
					minified, err := m.String("text/css", cssText)
					if err != nil {
						continue
					}

					// Collapse longhand properties into shorthand
					minified = collapseLonghandsInCSS(minified)

					// Then remove unused selectors
					if removeUnused {
						minified = removeUnusedRules(minified, idx)
					}

					minified = spaceAfterAtKeyword(minified)
					if forms := attrFlagForms(cssText); len(forms) > 0 {
						minified = separateAttrFlags(minified, forms)
					}
					minified = requoteFontFamilyKeywords(minified, cssText)

					if minified == "" {
						svgast.DetachNodeFromParent(entry.node, entry.parent)
						continue
					}

					// preserve cdata if CSS contains < or >
					if strings.ContainsAny(minified, "<>") {
						entry.node.Children[0] = &svgast.Cdata{Value: minified}
					} else if isCdata {
						entry.node.Children[0] = &svgast.Text{Value: minified}
					} else {
						entry.node.Children[0] = &svgast.Text{Value: minified}
					}
				}

				// minify style attributes
				for _, elem := range elementsWithStyleAttr {
					styleVal, _ := elem.Attributes.Get("style")
					minified, err := m.String("text/css", "{"+styleVal+"}")
					if err != nil {
						continue
					}
					// Remove the surrounding braces we added
					minified = strings.TrimPrefix(minified, "{")
					minified = strings.TrimSuffix(minified, "}")
					// Collapse longhand properties into shorthand
					minified = collapseDeclarations(minified)
					minified = requoteFontFamilyKeywords(minified, styleVal)
					elem.Attributes.Set("style", minified)
				}
			},
		},
	}
}

// spaceAfterAtKeyword restores the space csso keeps between an at-rule name and
// a parenthesised condition that follows it directly, as in "@media (min-width:1px)".
// The tdewolff minifier drops that space. Quoted strings are stepped over so a
// literal '@' inside one is left alone.
func spaceAfterAtKeyword(cssText string) string {
	var sb strings.Builder
	i := 0
	for i < len(cssText) {
		ch := cssText[i]
		switch {
		case ch == '"' || ch == '\'':
			quote := ch
			sb.WriteByte(ch)
			i++
			for i < len(cssText) {
				if cssText[i] == '\\' && i+1 < len(cssText) {
					sb.WriteString(cssText[i : i+2])
					i += 2
					continue
				}
				sb.WriteByte(cssText[i])
				i++
				if cssText[i-1] == quote {
					break
				}
			}
		case ch == '@':
			start := i
			i++
			for i < len(cssText) && isAtKeywordChar(cssText[i]) {
				i++
			}
			sb.WriteString(cssText[start:i])
			if i > start+1 && i < len(cssText) && cssText[i] == '(' {
				sb.WriteByte(' ')
			}
		default:
			sb.WriteByte(ch)
			i++
		}
	}
	return sb.String()
}

// reservedFontFamilies are the identifiers that name something other than a font
// family: the CSS-wide keywords and the generic family names. A family whose name
// is spelled like one of them is only reachable while it stays quoted.
var reservedFontFamilies = map[string]bool{
	"inherit": true, "initial": true, "unset": true, "revert": true,
	"revert-layer": true, "default": true, "none": true,
	"serif": true, "sans-serif": true, "monospace": true, "cursive": true,
	"fantasy": true, "system-ui": true, "math": true, "emoji": true,
	"fangsong": true, "ui-serif": true, "ui-sans-serif": true,
	"ui-monospace": true, "ui-rounded": true,
}

// requoteFontFamilyKeywords puts back the quotes on a font-family value that
// names a family after a CSS-wide keyword or a generic family. The tdewolff CSS
// minifier unquotes and lowercases such a value, which turns "the font called
// inherit" into the inherit keyword. original is the CSS as it was before
// minification; a keyword the source also uses bare is left untouched, since
// there the bare form really is the keyword.
func requoteFontFamilyKeywords(minified, original string) string {
	quoted := make(map[string]string)
	bare := make(map[string]bool)
	forEachFontFamilyValue(original, func(value string) {
		if inner, ok := wholeQuotedString(value); ok {
			lower := strings.ToLower(inner)
			if reservedFontFamilies[lower] {
				quoted[lower] = `"` + inner + `"`
			}
			return
		}
		if reservedFontFamilies[strings.ToLower(value)] {
			bare[strings.ToLower(value)] = true
		}
	})
	for name := range bare {
		delete(quoted, name)
	}
	if len(quoted) == 0 {
		return minified
	}

	var sb strings.Builder
	prev := 0
	forEachFontFamilyValueAt(minified, func(start, end int) {
		replacement, ok := quoted[strings.ToLower(minified[start:end])]
		if !ok {
			return
		}
		sb.WriteString(minified[prev:start])
		sb.WriteString(replacement)
		prev = end
	})
	if prev == 0 {
		return minified
	}
	sb.WriteString(minified[prev:])
	return sb.String()
}

// wholeQuotedString reports the content of value when value is nothing but a
// single quoted string.
func wholeQuotedString(value string) (string, bool) {
	if len(value) < 2 {
		return "", false
	}
	quote := value[0]
	if quote != '"' && quote != '\'' {
		return "", false
	}
	if skipCSSString(value, 0) != len(value) || value[len(value)-1] != quote {
		return "", false
	}
	return value[1 : len(value)-1], true
}

func forEachFontFamilyValue(cssText string, fn func(value string)) {
	forEachFontFamilyValueAt(cssText, func(start, end int) {
		fn(cssText[start:end])
	})
}

// forEachFontFamilyValueAt reports the bounds of the value of every font-family
// declaration in cssText, with surrounding whitespace trimmed off.
func forEachFontFamilyValueAt(cssText string, fn func(start, end int)) {
	const prop = "font-family"
	for i := 0; i+len(prop) < len(cssText); i++ {
		if !strings.EqualFold(cssText[i:i+len(prop)], prop) {
			continue
		}
		if i > 0 && isAtKeywordChar(cssText[i-1]) {
			continue
		}
		j := i + len(prop)
		for j < len(cssText) && isCSSSpaceByte(cssText[j]) {
			j++
		}
		if j >= len(cssText) || cssText[j] != ':' {
			continue
		}
		j++
		for j < len(cssText) && isCSSSpaceByte(cssText[j]) {
			j++
		}
		start := j
		for j < len(cssText) && cssText[j] != ';' && cssText[j] != '}' {
			if cssText[j] == '"' || cssText[j] == '\'' {
				j = skipCSSString(cssText, j)
				continue
			}
			j++
		}
		end := j
		for end > start && isCSSSpaceByte(cssText[end-1]) {
			end--
		}
		if end > start {
			fn(start, end)
		}
		i = j - 1
	}
}

// attrFlagForms collects the "<name><op><value><flag>" spellings, with the
// separating whitespace removed, of every attribute selector in cssText that
// carries an explicit 's' or 'S' case-sensitivity flag. Both the quoted and the
// unquoted form of the value are recorded, because the minifier drops redundant
// quotes. An empty result means the sheet uses no such flag.
func attrFlagForms(cssText string) map[string]bool {
	var forms map[string]bool
	forEachAttrSelector(cssText, func(start, end int) {
		content := strings.TrimRight(cssText[start:end], " \t\n\r\f")
		if len(content) < 2 {
			return
		}
		flag := content[len(content)-1]
		if flag != 's' && flag != 'S' {
			return
		}
		value := strings.TrimRight(content[:len(content)-1], " \t\n\r\f")
		if len(value) == len(content)-1 {
			// Nothing separated the trailing letter from the value, so it is
			// part of the value rather than a flag.
			return
		}
		if forms == nil {
			forms = make(map[string]bool)
		}
		forms[value+string(flag)] = true
		if unquoted, ok := unquoteAttrValue(value); ok {
			forms[unquoted+string(flag)] = true
		}
	})
	return forms
}

// separateAttrFlags reinstates the space between an attribute selector's value
// and its case-sensitivity flag. The tdewolff CSS minifier recognises only the
// 'i' flag, so an 's' flag comes out glued to the value and the selector then
// tests for a different value. Only the spellings in forms are touched.
func separateAttrFlags(cssText string, forms map[string]bool) string {
	var positions []int
	forEachAttrSelector(cssText, func(start, end int) {
		content := cssText[start:end]
		if len(content) < 2 || isCSSSpaceByte(content[len(content)-2]) {
			return
		}
		if forms[content] {
			positions = append(positions, end-1)
		}
	})
	if len(positions) == 0 {
		return cssText
	}
	var sb strings.Builder
	prev := 0
	for _, pos := range positions {
		sb.WriteString(cssText[prev:pos])
		sb.WriteByte(' ')
		prev = pos
	}
	sb.WriteString(cssText[prev:])
	return sb.String()
}

// unquoteAttrValue strips the quotes from a "<name><op>" prefix whose value is a
// quoted string, reporting whether the input ended in one.
func unquoteAttrValue(s string) (string, bool) {
	if len(s) < 2 {
		return s, false
	}
	quote := s[len(s)-1]
	if quote != '"' && quote != '\'' {
		return s, false
	}
	open := strings.LastIndexByte(s[:len(s)-1], quote)
	if open < 0 {
		return s, false
	}
	return s[:open] + s[open+1:len(s)-1], true
}

// forEachAttrSelector reports the bounds of the content of every attribute
// selector in cssText, stepping over quoted strings so brackets inside one are
// not mistaken for selector delimiters.
func forEachAttrSelector(cssText string, fn func(start, end int)) {
	i := 0
	for i < len(cssText) {
		switch cssText[i] {
		case '"', '\'':
			i = skipCSSString(cssText, i)
		case '[':
			start := i + 1
			j := start
			for j < len(cssText) && cssText[j] != ']' {
				if cssText[j] == '"' || cssText[j] == '\'' {
					j = skipCSSString(cssText, j)
					continue
				}
				j++
			}
			if j >= len(cssText) {
				return
			}
			fn(start, j)
			i = j + 1
		default:
			i++
		}
	}
}

// skipCSSString returns the offset just past the quoted string starting at s[i].
func skipCSSString(s string, i int) int {
	quote := s[i]
	i++
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if s[i] == quote {
			return i + 1
		}
		i++
	}
	return i
}

func isCSSSpaceByte(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f'
}

func isAtKeywordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch >= 0x80
}

// shorthandGroup defines a CSS shorthand property and its longhands in top/right/bottom/left order.
type shorthandGroup struct {
	shorthand string
	longhands [4]string
}

var boxShorthands = []shorthandGroup{
	{"padding", [4]string{"padding-top", "padding-right", "padding-bottom", "padding-left"}},
	{"margin", [4]string{"margin-top", "margin-right", "margin-bottom", "margin-left"}},
	{"border-width", [4]string{"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"}},
	{"border-style", [4]string{"border-top-style", "border-right-style", "border-bottom-style", "border-left-style"}},
	{"border-color", [4]string{"border-top-color", "border-right-color", "border-bottom-color", "border-left-color"}},
}

// collapseLonghandsInCSS collapses longhand properties into shorthand
// within declaration blocks in minified CSS text.
func collapseLonghandsInCSS(css string) string {
	var result strings.Builder
	i := 0
	for i < len(css) {
		braceIdx := strings.IndexByte(css[i:], '{')
		if braceIdx < 0 {
			result.WriteString(css[i:])
			break
		}
		// Write everything up to and including '{'
		result.WriteString(css[i : i+braceIdx+1])
		i = i + braceIdx + 1

		// Find matching '}' tracking depth
		start := i
		depth := 1
		for i < len(css) && depth > 0 {
			if css[i] == '{' {
				depth++
			} else if css[i] == '}' {
				depth--
			}
			if depth > 0 {
				i++
			}
		}

		content := css[start:i]

		// Check if content has nested braces (it's a container like @media, not declarations)
		if strings.ContainsAny(content, "{}") {
			// Recursively process nested content
			result.WriteString(collapseLonghandsInCSS(content))
		} else {
			// It's a declaration block, collapse longhands
			result.WriteString(collapseDeclarations(content))
		}

		// Write the closing '}'
		if i < len(css) {
			result.WriteByte('}')
			i++
		}
	}
	return result.String()
}

// collapseDeclarations collapses longhand properties into shorthand
// within a semicolon-separated declaration list.
func collapseDeclarations(decls string) string {
	type decl struct {
		prop string
		val  string
	}

	// Parse declarations preserving order
	parts := strings.Split(decls, ";")
	var declList []decl
	declIdx := make(map[string]int) // prop → index in declList

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colonIdx := strings.IndexByte(part, ':')
		if colonIdx < 0 {
			declList = append(declList, decl{part, ""})
			continue
		}
		prop := part[:colonIdx]
		val := part[colonIdx+1:]
		declIdx[prop] = len(declList)
		declList = append(declList, decl{prop, val})
	}

	// Try collapsing each shorthand group
	for _, group := range boxShorthands {
		values := [4]string{}
		allPresent := true
		for i, longhand := range group.longhands {
			if idx, ok := declIdx[longhand]; ok {
				values[i] = declList[idx].val
			} else {
				allPresent = false
				break
			}
		}
		if !allPresent {
			continue
		}

		// Don't collapse if any value contains !important (complex case)
		hasImportant := false
		for _, v := range values {
			if strings.Contains(v, "!important") {
				hasImportant = true
				break
			}
		}
		if hasImportant {
			continue
		}

		// Determine shorthand value
		var shortVal string
		if values[0] == values[1] && values[1] == values[2] && values[2] == values[3] {
			shortVal = values[0]
		} else if values[0] == values[2] && values[1] == values[3] {
			shortVal = values[0] + " " + values[1]
		} else if values[1] == values[3] {
			shortVal = values[0] + " " + values[1] + " " + values[2]
		} else {
			shortVal = values[0] + " " + values[1] + " " + values[2] + " " + values[3]
		}

		// Replace first longhand with shorthand, mark rest for removal
		firstIdx := -1
		removeSet := make(map[int]bool)
		for _, longhand := range group.longhands {
			idx := declIdx[longhand]
			if firstIdx < 0 {
				firstIdx = idx
			} else {
				removeSet[idx] = true
			}
			delete(declIdx, longhand)
		}
		declList[firstIdx] = decl{group.shorthand, shortVal}
		declIdx[group.shorthand] = firstIdx

		// Rebuild list without removed entries
		var newList []decl
		newIdx := make(map[string]int)
		for i, d := range declList {
			if removeSet[i] {
				continue
			}
			newIdx[d.prop] = len(newList)
			newList = append(newList, d)
		}
		declList = newList
		declIdx = newIdx
	}

	// Rebuild declaration string
	var result []string
	for _, d := range declList {
		if d.val == "" {
			result = append(result, d.prop)
		} else {
			result = append(result, d.prop+":"+d.val)
		}
	}
	return strings.Join(result, ";")
}
