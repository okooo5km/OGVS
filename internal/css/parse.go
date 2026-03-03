// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// ParseStyleDeclarations parses an inline style attribute value
// into a slice of StylesheetDeclarations.
// e.g. "fill:red; stroke:blue !important" → [{fill, red, false}, {stroke, blue, true}]
func ParseStyleDeclarations(cssText string) []StylesheetDeclaration {
	var declarations []StylesheetDeclaration

	p := css.NewParser(parse.NewInput(strings.NewReader(cssText)), true)

	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar {
			if p.HasParseError() {
				// Recoverable parse error (invalid declaration), skip and continue
				continue
			}
			break // true EOF
		}

		if gt == css.DeclarationGrammar {
			name := unescapeCSSIdentifier(string(data))
			values := p.Values()
			important := false

			// Check if last token is !important
			// In tdewolff's parser, important is indicated by the grammar type
			var valueParts []string
			for _, val := range values {
				tokenStr := string(val.Data)
				if val.TokenType == css.DelimToken && tokenStr == "!" {
					continue
				}
				if val.TokenType == css.IdentToken && strings.EqualFold(tokenStr, "important") {
					important = true
					continue
				}
				if val.TokenType != css.WhitespaceToken || len(valueParts) > 0 {
					valueParts = append(valueParts, tokenStr)
				}
			}

			value := strings.TrimSpace(strings.Join(valueParts, ""))
			if value != "" || name != "" {
				declarations = append(declarations, StylesheetDeclaration{
					Name:      strings.TrimSpace(name),
					Value:     value,
					Important: important,
				})
			}
		}

		if gt == css.CustomPropertyGrammar {
			decl := parseCustomProperty(string(data), p.Values())
			if decl.Name != "" {
				declarations = append(declarations, decl)
			}
		}
	}

	return declarations
}

// ParseStylesheet parses a CSS stylesheet string into rules.
// dynamic indicates whether the rules come from a media-dependent context.
func ParseStylesheet(cssText string, dynamic bool) []StylesheetRule {
	var rules []StylesheetRule

	p := css.NewParser(parse.NewInput(strings.NewReader(cssText)), false)

	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar {
			break
		}

		switch gt {
		case css.QualifiedRuleGrammar, css.BeginRulesetGrammar:
			selectorStr := strings.TrimSpace(string(data))
			// Collect remaining selector tokens
			values := p.Values()
			for _, v := range values {
				selectorStr += string(v.Data)
			}
			selectorStr = strings.TrimSpace(selectorStr)

			// Parse declarations within the rule
			declarations := parseRuleDeclarations(p)

			// Split comma-separated selectors
			selectors := splitSelectors(selectorStr)
			for _, sel := range selectors {
				sel = strings.TrimSpace(sel)
				if sel == "" {
					continue
				}

				hasPseudo := containsPseudoClass(sel)
				// Strip pseudo-classes from selector for matching
				cleanSel := StripPseudoClasses(sel)

				rules = append(rules, StylesheetRule{
					Specificity:      CalculateSpecificity(sel), // use original for specificity
					Dynamic:          hasPseudo || dynamic,
					Selector:         cleanSel,
					OriginalSelector: sel,
					Declarations:     declarations,
				})
			}

		case css.BeginAtRuleGrammar:
			atRuleName := strings.TrimSpace(string(data))
			// Skip keyframes
			if isKeyframesAtRule(atRuleName) {
				skipBlock(p)
				continue
			}
			// Build media query string: "name prelude"
			mediaQuery := buildAtRuleMediaQuery(atRuleName, p.Values())
			// At-rules like @media make inner rules dynamic
			innerRules := parseAtRuleBlock(p, true)
			for i := range innerRules {
				innerRules[i].MediaQuery = mediaQuery
			}
			rules = append(rules, innerRules...)

		case css.AtRuleGrammar:
			// Skip standalone at-rules
			continue
		}
	}

	return rules
}

// parseRuleDeclarations parses declarations within a ruleset block.
func parseRuleDeclarations(p *css.Parser) []StylesheetDeclaration {
	var declarations []StylesheetDeclaration

	for {
		gt, _, data := p.Next()
		if gt == css.EndRulesetGrammar || gt == css.EndAtRuleGrammar {
			break
		}
		if gt == css.ErrorGrammar {
			if p.HasParseError() {
				continue // recoverable parse error
			}
			break // true EOF
		}

		if gt == css.DeclarationGrammar {
			name := unescapeCSSIdentifier(string(data))
			values := p.Values()
			important := false

			var valueParts []string
			for _, val := range values {
				tokenStr := string(val.Data)
				if val.TokenType == css.DelimToken && tokenStr == "!" {
					continue
				}
				if val.TokenType == css.IdentToken && strings.EqualFold(tokenStr, "important") {
					important = true
					continue
				}
				if val.TokenType != css.WhitespaceToken || len(valueParts) > 0 {
					valueParts = append(valueParts, tokenStr)
				}
			}

			value := strings.TrimSpace(strings.Join(valueParts, ""))
			declarations = append(declarations, StylesheetDeclaration{
				Name:      strings.TrimSpace(name),
				Value:     value,
				Important: important,
			})
		}

		if gt == css.CustomPropertyGrammar {
			decl := parseCustomProperty(string(data), p.Values())
			if decl.Name != "" {
				declarations = append(declarations, decl)
			}
		}
	}

	return declarations
}

// parseAtRuleBlock parses rules inside an at-rule block.
func parseAtRuleBlock(p *css.Parser, dynamic bool) []StylesheetRule {
	var rules []StylesheetRule

	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar || gt == css.EndAtRuleGrammar {
			break
		}

		if gt == css.QualifiedRuleGrammar || gt == css.BeginRulesetGrammar {
			selectorStr := strings.TrimSpace(string(data))
			values := p.Values()
			for _, v := range values {
				selectorStr += string(v.Data)
			}
			selectorStr = strings.TrimSpace(selectorStr)

			declarations := parseRuleDeclarations(p)
			selectors := splitSelectors(selectorStr)

			for _, sel := range selectors {
				sel = strings.TrimSpace(sel)
				if sel == "" {
					continue
				}
				hasPseudo := containsPseudoClass(sel)
				cleanSel := StripPseudoClasses(sel)

				rules = append(rules, StylesheetRule{
					Specificity:      CalculateSpecificity(sel),
					Dynamic:          hasPseudo || dynamic,
					Selector:         cleanSel,
					OriginalSelector: sel,
					Declarations:     declarations,
				})
			}
		}
	}

	return rules
}

// parseCustomProperty parses a CSS custom property (--*) from its name and value tokens.
// Custom properties use CustomPropertyGrammar in tdewolff's parser, which returns
// CustomPropertyValueToken with raw text (including leading whitespace).
func parseCustomProperty(name string, values []css.Token) StylesheetDeclaration {
	var rawValue string
	for _, val := range values {
		rawValue += string(val.Data)
	}
	rawValue = strings.TrimSpace(rawValue)

	important := false
	if idx := strings.LastIndex(strings.ToLower(rawValue), "!important"); idx >= 0 {
		important = true
		rawValue = strings.TrimSpace(rawValue[:idx])
	}

	return StylesheetDeclaration{
		Name:      name,
		Value:     rawValue,
		Important: important,
	}
}

// buildAtRuleMediaQuery builds a media query string from an at-rule name and prelude values.
// Format: "name compactPrelude" (e.g., "media screen", "supports (display:flex)").
// The name has the @ prefix stripped to match SVGO's convention.
func buildAtRuleMediaQuery(name string, values []css.Token) string {
	var sb strings.Builder
	// Strip @ prefix to match SVGO's convention (e.g., "@media" -> "media")
	sb.WriteString(strings.TrimPrefix(strings.TrimSpace(name), "@"))
	prelude := compactTokenValues(values)
	if prelude != "" {
		sb.WriteByte(' ')
		sb.WriteString(prelude)
	}
	return sb.String()
}

// compactTokenValues compacts CSS token values into a string with minimal whitespace.
func compactTokenValues(values []css.Token) string {
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
		s := string(v.Data)
		// Normalize single-quoted strings to double-quoted
		if v.TokenType == css.StringToken && len(s) >= 2 && s[0] == '\'' {
			s = `"` + s[1:len(s)-1] + `"`
		}
		sb.WriteString(s)
	}
	return strings.TrimSpace(sb.String())
}

func skipBlock(p *css.Parser) {
	depth := 1
	for depth > 0 {
		gt, _, _ := p.Next()
		if gt == css.ErrorGrammar {
			break
		}
		switch gt {
		case css.BeginRulesetGrammar, css.BeginAtRuleGrammar:
			depth++
		case css.EndRulesetGrammar, css.EndAtRuleGrammar:
			depth--
		}
	}
}

func isKeyframesAtRule(name string) bool {
	n := strings.TrimPrefix(strings.ToLower(name), "@")
	switch n {
	case "keyframes", "-webkit-keyframes", "-o-keyframes", "-moz-keyframes":
		return true
	}
	return false
}

// splitSelectors splits a comma-separated selector list.
func splitSelectors(s string) []string {
	var result []string
	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				result = append(result, s[start:i])
				start = i + 1
			}
		}
	}
	result = append(result, s[start:])
	return result
}

// unescapeCSSIdentifier unescapes CSS escape sequences in an identifier.
// Per CSS Syntax Level 3:
//   - \XX (1-6 hex digits + optional whitespace) → Unicode code point
//   - \X (any other char) → that character itself
func unescapeCSSIdentifier(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '\\' || i+1 >= len(s) {
			sb.WriteByte(s[i])
			i++
			continue
		}
		i++ // skip backslash
		// Check for hex escape: 1-6 hex digits
		if isHexDigit(s[i]) {
			start := i
			for i < len(s) && i-start < 6 && isHexDigit(s[i]) {
				i++
			}
			// Parse hex value
			var codePoint int
			for _, ch := range s[start:i] {
				codePoint = codePoint*16 + hexVal(byte(ch))
			}
			// Skip optional whitespace after hex escape
			if i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\f') {
				i++
			}
			// Replace null, surrogates, and out-of-range with U+FFFD
			if codePoint == 0 || codePoint > 0x10FFFF ||
				(codePoint >= 0xD800 && codePoint <= 0xDFFF) {
				codePoint = 0xFFFD
			}
			sb.WriteRune(rune(codePoint))
		} else {
			// Any other character: use as-is
			sb.WriteByte(s[i])
			i++
		}
	}
	return sb.String()
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func hexVal(ch byte) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'f':
		return int(ch-'a') + 10
	case ch >= 'A' && ch <= 'F':
		return int(ch-'A') + 10
	}
	return 0
}
