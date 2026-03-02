// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package prefixids implements the prefixIds SVGO plugin.
// It prefixes IDs, classes, and references in SVG documents to avoid
// naming collisions when multiple SVGs are inlined on the same page.
package prefixids

import (
	"regexp"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "prefixIds",
		Description: "prefix IDs",
		Fn:          fn,
	})
}

// getBasename extracts the filename from a path.
func getBasename(path string) string {
	i := strings.LastIndexAny(path, "/\\")
	if i >= 0 {
		return path[i+1:]
	}
	return path
}

// escapeIdentifierName replaces dots and spaces with underscores.
func escapeIdentifierName(str string) string {
	return strings.NewReplacer(".", "_", " ", "_").Replace(str)
}

// unquote removes surrounding quotes from a string.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// prefixId prepends the generated prefix to the given body, unless it already
// starts with the prefix.
func prefixId(prefixGenerator func(string) string, body string) string {
	prefix := prefixGenerator(body)
	if strings.HasPrefix(body, prefix) {
		return body
	}
	return prefix + body
}

// prefixReference inserts a prefix into a reference string that starts with "#".
func prefixReference(prefixGenerator func(string) string, reference string) (string, bool) {
	if strings.HasPrefix(reference, "#") {
		return "#" + prefixId(prefixGenerator, reference[1:]), true
	}
	return "", false
}

// generatePrefix produces the prefix string based on params and info.
func generatePrefix(info *plugin.PluginInfo, prefixParam any, delim string) func(string) string {
	return func(_ string) string {
		switch v := prefixParam.(type) {
		case string:
			return v + delim
		case bool:
			if !v {
				return ""
			}
		}

		// Default behavior (prefix is true or nil)
		if info != nil && info.Path != "" {
			return escapeIdentifierName(getBasename(info.Path)) + delim
		}
		return "prefix" + delim
	}
}

// urlRefPattern matches url(#id) references with optional quotes.
// Go RE2 doesn't support backreferences (\1), so we use a broader pattern
// and handle quote matching in the replacement function.
var urlRefPattern = regexp.MustCompile(`\burl\((["']?)(#.+?)(["']?)\)`)

// extractURLContent extracts the content between url( and ).
func extractURLContent(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "url(") || !strings.HasSuffix(s, ")") {
		return s
	}
	return s[4 : len(s)-1]
}

// rewriteCSSWithMinify rewrites CSS text with prefix transformations and
// produces compact output. Uses a two-pass approach: first rewrite identifiers,
// then compact.
func rewriteCSSWithMinify(cssText string, prefixGen func(string) string, prefixIDs, prefixClassNames bool) string {
	// Simple approach: use token-level rewriting that handles selectors properly.
	// We parse through the CSS tokens and track context to know when we're in a
	// selector vs a value.
	return rewriteCSSTokens(cssText, prefixGen, prefixIDs, prefixClassNames)
}

// rewriteCSSTokens performs a context-aware rewrite of CSS text.
// It tracks whether we're in a selector or declaration value to correctly
// handle #hash tokens (IDs vs colors) and .class tokens.
func rewriteCSSTokens(cssText string, prefixGen func(string) string, prefixIDs, prefixClassNames bool) string {
	var w strings.Builder
	p := css.NewParser(parse.NewInput(strings.NewReader(cssText)), false)

	rewriteBlock(&w, p, prefixGen, prefixIDs, prefixClassNames, false)

	return w.String()
}

// rewriteBlock processes a CSS block (top-level or nested within at-rules).
func rewriteBlock(w *strings.Builder, p *css.Parser, prefixGen func(string) string, prefixIDs, prefixClassNames bool, nested bool) {
	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar {
			break
		}
		if nested && (gt == css.EndRulesetGrammar || gt == css.EndAtRuleGrammar) {
			break
		}

		switch gt {
		case css.AtRuleGrammar:
			// Standalone at-rule
			w.WriteString(strings.TrimSpace(string(data)))
			prelude := compactPrelude(p.Values())
			if prelude != "" {
				w.WriteString(" ")
				w.WriteString(prelude)
			}
			w.WriteString(";")

		case css.BeginAtRuleGrammar:
			atName := strings.TrimSpace(string(data))
			values := p.Values()

			if isKeyframesAtRule(atName) {
				rewriteKeyframesAtRule(w, p, atName, values, prefixGen, prefixIDs, prefixClassNames)
			} else {
				// @media, @supports, etc.
				w.WriteString(atName)
				prelude := compactPrelude(values)
				if prelude != "" {
					w.WriteString(" ")
					w.WriteString(prelude)
				}
				w.WriteString("{")
				rewriteBlock(w, p, prefixGen, prefixIDs, prefixClassNames, true)
				w.WriteString("}")
			}

		case css.QualifiedRuleGrammar, css.BeginRulesetGrammar:
			selectorStr := buildSelector(string(data), p.Values())
			if gt == css.BeginRulesetGrammar {
				rewriteRuleset(w, p, selectorStr, prefixGen, prefixIDs, prefixClassNames)
			}

		case css.DeclarationGrammar:
			// Standalone declarations (shouldn't happen at top level, but handle for safety)
			writeDeclaration(w, string(data), p.Values(), prefixGen)
		}
	}
}

// rewriteKeyframesAtRule writes a @keyframes block with url() prefixing in values.
func rewriteKeyframesAtRule(w *strings.Builder, p *css.Parser, name string, values []css.Token, prefixGen func(string) string, _, _ bool) {
	w.WriteString(name)
	prelude := compactPrelude(values)
	if prelude != "" {
		w.WriteString(" ")
		w.WriteString(prelude)
	}
	w.WriteString("{")

	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar || gt == css.EndAtRuleGrammar {
			break
		}
		if gt == css.BeginRulesetGrammar || gt == css.QualifiedRuleGrammar {
			selector := buildSelector(string(data), p.Values())
			w.WriteString(selector)
			w.WriteString("{")
			firstDecl := true
			for {
				dgt, _, ddata := p.Next()
				if dgt == css.ErrorGrammar || dgt == css.EndRulesetGrammar || dgt == css.EndAtRuleGrammar {
					break
				}
				if dgt == css.DeclarationGrammar {
					if !firstDecl {
						w.WriteString(";")
					}
					firstDecl = false
					writeDeclaration(w, string(ddata), p.Values(), prefixGen)
				}
			}
			w.WriteString("}")
		}
	}
	w.WriteString("}")
}

// rewriteRuleset writes a CSS ruleset with prefixed selectors and values.
func rewriteRuleset(w *strings.Builder, p *css.Parser, selectorStr string, prefixGen func(string) string, prefixIDs, prefixClassNames bool) {
	// Read declarations into buffer
	var declBuf strings.Builder
	firstDecl := true
	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar || gt == css.EndRulesetGrammar || gt == css.EndAtRuleGrammar {
			break
		}
		if gt == css.DeclarationGrammar {
			if !firstDecl {
				declBuf.WriteString(";")
			}
			firstDecl = false
			writeDeclaration(&declBuf, string(data), p.Values(), prefixGen)
		}
	}

	// Rewrite selector
	prefixedSelector := rewriteSelector(selectorStr, prefixGen, prefixIDs, prefixClassNames)

	w.WriteString(prefixedSelector)
	w.WriteString("{")
	w.WriteString(declBuf.String())
	w.WriteString("}")
}

// rewriteSelector rewrites a CSS selector string, prefixing ID and class selectors.
func rewriteSelector(selector string, prefixGen func(string) string, prefixIDs, prefixClassNames bool) string {
	var w strings.Builder
	i := 0

	for i < len(selector) {
		ch := selector[i]

		if ch == '#' && prefixIDs {
			// ID selector
			i++ // skip #
			name := readCSSIdent(selector, &i)
			w.WriteString("#")
			w.WriteString(prefixId(prefixGen, name))
			continue
		}

		if ch == '.' && prefixClassNames {
			// Class selector - check it's not a decimal number
			if i+1 < len(selector) && isIdentStart(selector[i+1]) {
				i++ // skip .
				name := readCSSIdent(selector, &i)
				w.WriteString(".")
				w.WriteString(prefixId(prefixGen, name))
				continue
			}
		}

		if ch == '[' {
			// Attribute selector - pass through unchanged
			w.WriteByte(ch)
			i++
			for i < len(selector) && selector[i] != ']' {
				w.WriteByte(selector[i])
				i++
			}
			if i < len(selector) {
				w.WriteByte(selector[i])
				i++
			}
			continue
		}

		if ch == ':' {
			// Pseudo-class/element - pass through
			w.WriteByte(ch)
			i++
			if i < len(selector) && selector[i] == ':' {
				w.WriteByte(selector[i])
				i++
			}
			// Read pseudo name
			for i < len(selector) && isIdentChar(selector[i]) {
				w.WriteByte(selector[i])
				i++
			}
			// Handle function pseudo like :not()
			if i < len(selector) && selector[i] == '(' {
				depth := 1
				w.WriteByte(selector[i])
				i++
				for i < len(selector) && depth > 0 {
					if selector[i] == '(' {
						depth++
					}
					if selector[i] == ')' {
						depth--
					}
					w.WriteByte(selector[i])
					i++
				}
			}
			continue
		}

		// Whitespace compaction
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			// Collapse whitespace
			for i < len(selector) && (selector[i] == ' ' || selector[i] == '\t' || selector[i] == '\n' || selector[i] == '\r') {
				i++
			}
			if w.Len() > 0 && i < len(selector) {
				w.WriteByte(' ')
			}
			continue
		}

		w.WriteByte(ch)
		i++
	}

	return w.String()
}

// readCSSIdent reads a CSS identifier starting at position i.
func readCSSIdent(s string, i *int) string {
	start := *i
	for *i < len(s) {
		ch := s[*i]
		if isIdentChar(ch) || ch == '\\' {
			if ch == '\\' {
				(*i)++ // skip backslash
				if *i < len(s) {
					(*i)++ // skip escaped char
				}
			} else {
				(*i)++
			}
		} else {
			break
		}
	}
	return s[start:*i]
}

// isIdentStart checks if a character can start a CSS identifier.
func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '-' || ch >= 0x80
}

// isIdentChar checks if a character can be part of a CSS identifier.
func isIdentChar(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

// buildSelector builds a compact selector string from parser data and values.
func buildSelector(data string, values []css.Token) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(data))
	for _, v := range values {
		sb.WriteString(string(v.Data))
	}
	return strings.TrimSpace(sb.String())
}

// compactPrelude compacts an at-rule prelude.
func compactPrelude(values []css.Token) string {
	var sb strings.Builder
	prevWS := true
	for _, v := range values {
		if v.TokenType == css.WhitespaceToken {
			if !prevWS {
				sb.WriteByte(' ')
				prevWS = true
			}
			continue
		}
		prevWS = false
		sb.WriteString(string(v.Data))
	}
	return strings.TrimSpace(sb.String())
}

// writeDeclaration writes a compact CSS declaration, prefixing url() references.
func writeDeclaration(w *strings.Builder, name string, values []css.Token, prefixGen func(string) string) {
	w.WriteString(strings.TrimSpace(name))
	w.WriteString(":")

	important := false
	var parts []string
	for _, v := range values {
		s := string(v.Data)
		if v.TokenType == css.DelimToken && s == "!" {
			continue
		}
		if v.TokenType == css.IdentToken && strings.EqualFold(s, "important") {
			important = true
			continue
		}
		// Handle url() tokens - prefix references
		if v.TokenType == css.URLToken {
			inner := extractURLContent(s)
			unquoted := unquote(inner)
			if prefixed, ok := prefixReference(prefixGen, unquoted); ok {
				s = "url(" + prefixed + ")"
			} else {
				s = "url(" + unquoted + ")"
			}
		}
		if v.TokenType == css.WhitespaceToken {
			if len(parts) > 0 {
				parts = append(parts, " ")
			}
			continue
		}
		parts = append(parts, s)
	}

	// Join parts
	var result strings.Builder
	prevWS := true
	for _, part := range parts {
		if part == " " {
			if !prevWS {
				result.WriteByte(' ')
				prevWS = true
			}
			continue
		}
		prevWS = false
		result.WriteString(part)
	}

	w.WriteString(strings.TrimSpace(result.String()))
	if important {
		w.WriteString("!important")
	}
}

// isKeyframesAtRule checks if an at-rule name is a keyframes rule.
func isKeyframesAtRule(name string) bool {
	n := strings.TrimPrefix(strings.ToLower(name), "@")
	switch n {
	case "keyframes", "-webkit-keyframes", "-o-keyframes", "-moz-keyframes":
		return true
	}
	return false
}

func fn(_ *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	// Parse params
	delim := "__"
	if d, ok := params["delim"]; ok {
		if ds, ok := d.(string); ok {
			delim = ds
		}
	}

	var prefixParam any
	if p, ok := params["prefix"]; ok {
		prefixParam = p
	}

	prefixIDs := true
	if p, ok := params["prefixIds"]; ok {
		if pb, ok := p.(bool); ok {
			prefixIDs = pb
		}
	}

	prefixClassNames := true
	if p, ok := params["prefixClassNames"]; ok {
		if pb, ok := p.(bool); ok {
			prefixClassNames = pb
		}
	}

	prefixGen := generatePrefix(info, prefixParam, delim)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Prefix id/class selectors and url() references in <style> elements
				if elem.Name == "style" {
					if len(elem.Children) == 0 {
						return nil
					}

					for _, child := range elem.Children {
						switch c := child.(type) {
						case *svgast.Text:
							c.Value = rewriteCSSWithMinify(c.Value, prefixGen, prefixIDs, prefixClassNames)
						case *svgast.Cdata:
							c.Value = rewriteCSSWithMinify(c.Value, prefixGen, prefixIDs, prefixClassNames)
						}
					}
				}

				// Prefix id attribute
				if prefixIDs {
					if id, ok := elem.Attributes.Get("id"); ok && id != "" {
						elem.Attributes.Set("id", prefixId(prefixGen, id))
					}
				}

				// Prefix class attribute
				if prefixClassNames {
					if class, ok := elem.Attributes.Get("class"); ok && class != "" {
						classes := strings.Fields(class)
						prefixed := make([]string, 0, len(classes))
						for _, c := range classes {
							if c != "" {
								prefixed = append(prefixed, prefixId(prefixGen, c))
							}
						}
						elem.Attributes.Set("class", strings.Join(prefixed, " "))
					}
				}

				// Prefix href and xlink:href
				for _, name := range []string{"href", "xlink:href"} {
					if href, ok := elem.Attributes.Get(name); ok && href != "" {
						if prefixed, ok := prefixReference(prefixGen, href); ok {
							elem.Attributes.Set(name, prefixed)
						}
					}
				}

				// Prefix url() in reference properties
				for prop := range collections.ReferencesProps {
					if val, ok := elem.Attributes.Get(prop); ok && val != "" {
						newVal := urlRefPattern.ReplaceAllStringFunc(val, func(match string) string {
							submatch := urlRefPattern.FindStringSubmatch(match)
							if len(submatch) < 4 {
								return match
							}
							// Verify quotes match (submatch[1] = open quote, submatch[3] = close quote)
							if submatch[1] != submatch[3] {
								return match
							}
							url := submatch[2]
							if prefixed, ok := prefixReference(prefixGen, url); ok {
								return "url(" + prefixed + ")"
							}
							return match
						})
						if newVal != val {
							elem.Attributes.Set(prop, newVal)
						}
					}
				}

				// Prefix begin/end attribute element.event references
				for _, name := range []string{"begin", "end"} {
					if val, ok := elem.Attributes.Get(name); ok && val != "" {
						parts := splitBeginEnd(val)
						newParts := make([]string, len(parts))
						for i, part := range parts {
							part = strings.TrimSpace(part)
							if strings.HasSuffix(part, ".end") || strings.HasSuffix(part, ".start") {
								dotIdx := strings.IndexByte(part, '.')
								if dotIdx > 0 {
									id := part[:dotIdx]
									postfix := part[dotIdx+1:]
									newParts[i] = prefixId(prefixGen, id) + "." + postfix
								} else {
									newParts[i] = part
								}
							} else {
								newParts[i] = part
							}
						}
						elem.Attributes.Set(name, strings.Join(newParts, "; "))
					}
				}

				// Prefix url() in style attribute (not in collections.ReferencesProps["style"])
				if style, ok := elem.Attributes.Get("style"); ok && style != "" {
					newStyle := urlRefPattern.ReplaceAllStringFunc(style, func(match string) string {
						submatch := urlRefPattern.FindStringSubmatch(match)
						if len(submatch) < 4 {
							return match
						}
						if submatch[1] != submatch[3] {
							return match
						}
						url := submatch[2]
						if prefixed, ok := prefixReference(prefixGen, url); ok {
							return "url(" + prefixed + ")"
						}
						return match
					})
					if newStyle != style {
						elem.Attributes.Set("style", newStyle)
					}
				}

				return nil
			},
		},
	}
}

// splitBeginEnd splits a begin/end attribute value by semicolons,
// matching SVGO's split(/\s*;\s+/).
func splitBeginEnd(s string) []string {
	// SVGO uses /\s*;\s+/ which splits on semicolons surrounded by optional
	// leading whitespace and required trailing whitespace.
	// We need to be compatible: split on "; " (semicolon + space(s))
	re := regexp.MustCompile(`\s*;\s+`)
	return re.Split(s, -1)
}
