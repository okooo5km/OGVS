// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// CompactStylesheet generates compact CSS from raw CSS text while optionally
// removing specified selectors. At-rules are preserved in compact form.
// shouldSkip is called for each rule with its selector and media query context
// to determine if the rule should be removed from the output.
// If shouldSkip is nil, all rules are preserved.
func CompactStylesheet(cssText string, shouldSkip func(selector, mediaQuery string) bool) string {
	var w strings.Builder
	p := css.NewParser(parse.NewInput(strings.NewReader(cssText)), false)
	compactBlock(p, &w, shouldSkip, "", false)
	return w.String()
}

// compactBlock processes a CSS block (top-level or inside an at-rule).
func compactBlock(p *css.Parser, w *strings.Builder, shouldSkip func(string, string) bool, mediaQuery string, nested bool) {
	firstDecl := true

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
			// Standalone at-rule: @charset, @import, @namespace
			compactStandaloneAtRule(w, string(data), p.Values())

		case css.BeginAtRuleGrammar:
			atName := strings.TrimSpace(string(data))
			values := p.Values()

			if isKeyframesAtRule(atName) {
				compactKeyframesAtRule(p, w, atName, values)
			} else if isDeclarationAtRule(atName) {
				// @font-face, @viewport, @page — contains declarations
				compactDeclarationAtRule(p, w, atName, values)
			} else {
				// @media, @supports, @document — contains rulesets
				innerMQ := buildAtRuleMediaQuery(atName, values)
				compactRulesetAtRule(p, w, atName, values, shouldSkip, innerMQ)
			}

		case css.QualifiedRuleGrammar, css.BeginRulesetGrammar:
			selectorStr := compactSelector(string(data), p.Values())
			if gt == css.BeginRulesetGrammar {
				compactRuleset(p, w, selectorStr, shouldSkip, mediaQuery)
			}

		case css.DeclarationGrammar:
			// Declarations inside at-rule blocks like @font-face
			if !firstDecl {
				w.WriteString(";")
			}
			firstDecl = false
			compactDeclaration(w, string(data), p.Values())
		}
	}
}

// compactStandaloneAtRule writes a standalone at-rule (e.g., @charset, @import, @namespace).
func compactStandaloneAtRule(w *strings.Builder, name string, values []css.Token) {
	w.WriteString(strings.TrimSpace(name))
	prelude := compactAtRulePrelude(values)
	if prelude != "" {
		w.WriteString(" ")
		w.WriteString(prelude)
	}
	w.WriteString(";")
}

// compactKeyframesAtRule writes a @keyframes block.
func compactKeyframesAtRule(p *css.Parser, w *strings.Builder, name string, values []css.Token) {
	w.WriteString(name)
	prelude := compactAtRulePrelude(values)
	if prelude != "" {
		w.WriteString(" ")
		w.WriteString(prelude)
	}
	w.WriteString("{")

	// Process keyframe blocks
	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar || gt == css.EndAtRuleGrammar {
			break
		}
		if gt == css.BeginRulesetGrammar || gt == css.QualifiedRuleGrammar {
			selector := compactSelector(string(data), p.Values())
			w.WriteString(selector)
			w.WriteString("{")
			// Read declarations
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
					compactDeclaration(w, string(ddata), p.Values())
				}
			}
			w.WriteString("}")
		}
	}
	w.WriteString("}")
}

// compactDeclarationAtRule writes an at-rule that contains declarations (e.g., @font-face, @viewport, @page).
func compactDeclarationAtRule(p *css.Parser, w *strings.Builder, name string, values []css.Token) {
	w.WriteString(name)
	prelude := compactAtRulePrelude(values)
	if prelude != "" {
		w.WriteString(" ")
		w.WriteString(prelude)
	}
	w.WriteString("{")

	firstDecl := true
	var rawTokens []string
	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar || gt == css.EndRulesetGrammar || gt == css.EndAtRuleGrammar {
			break
		}
		if gt == css.DeclarationGrammar {
			if !firstDecl {
				w.WriteString(";")
			}
			firstDecl = false
			compactDeclaration(w, string(data), p.Values())
		} else if gt == css.TokenGrammar {
			// Raw token fallback for at-rules the parser doesn't understand (e.g., @viewport)
			rawTokens = append(rawTokens, string(data))
		}
	}
	// If no declarations were found but raw tokens exist, compact them
	if firstDecl && len(rawTokens) > 0 {
		w.WriteString(compactRawDeclarations(rawTokens))
	}
	w.WriteString("}")
}

// compactRawDeclarations compacts raw CSS declaration tokens into a compact string.
// Used for at-rules that the parser doesn't understand (e.g., @viewport).
func compactRawDeclarations(tokens []string) string {
	var sb strings.Builder
	for _, tok := range tokens {
		t := strings.TrimSpace(tok)
		if t == "" {
			continue
		}
		sb.WriteString(t)
	}
	// Remove trailing semicolons
	result := strings.TrimRight(sb.String(), ";")
	return result
}

// compactRulesetAtRule writes an at-rule that contains rulesets (e.g., @media, @supports, @document).
func compactRulesetAtRule(p *css.Parser, w *strings.Builder, name string, values []css.Token,
	shouldSkip func(string, string) bool, mediaQuery string) {
	w.WriteString(name)
	prelude := compactAtRulePrelude(values)
	if prelude != "" {
		w.WriteString(" ")
		w.WriteString(prelude)
	}
	w.WriteString("{")
	compactBlock(p, w, shouldSkip, mediaQuery, true)
	w.WriteString("}")
}

// compactRuleset writes a CSS ruleset, optionally skipping it based on shouldSkip.
func compactRuleset(p *css.Parser, w *strings.Builder, selectorStr string,
	shouldSkip func(string, string) bool, mediaQuery string) {

	// Read declarations
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
			compactDeclaration(&declBuf, string(data), p.Values())
		}
	}

	// Split comma-separated selectors and filter
	sels := splitSelectors(selectorStr)
	var kept []string
	for _, s := range sels {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if shouldSkip == nil || !shouldSkip(s, mediaQuery) {
			kept = append(kept, compactSelectorString(s))
		}
	}

	if len(kept) > 0 {
		w.WriteString(strings.Join(kept, ","))
		w.WriteString("{")
		w.WriteString(declBuf.String())
		w.WriteString("}")
	}
}

// compactSelector builds a compact selector string from data and values.
func compactSelector(data string, values []css.Token) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(data))
	for _, v := range values {
		sb.WriteString(string(v.Data))
	}
	return strings.TrimSpace(sb.String())
}

// compactSelectorString compacts a single selector by removing unnecessary whitespace.
// Preserves spaces only between two identifier-like tokens (descendant combinator).
// Removes spaces before/after [, ], (, ), >, +, ~, commas, and special tokens.
func compactSelectorString(s string) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			// Collapse whitespace
			for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
				i++
			}
			// Don't add leading or trailing space
			if sb.Len() == 0 || i >= len(s) {
				continue
			}
			// Check adjacent characters: skip space if next or prev is not an
			// identifier character (letter, digit, -, _, *, %, /, .)
			next := s[i]
			buf := sb.String()
			prev := buf[len(buf)-1]
			if isSelectorIdentChar(prev) && isSelectorIdentChar(next) {
				sb.WriteByte(' ')
			}
			continue
		}
		sb.WriteByte(ch)
		i++
	}
	return strings.TrimSpace(sb.String())
}

// isSelectorIdentChar returns true if the character can be part of an identifier-like
// token in a CSS selector. Spaces are only preserved between two such characters.
func isSelectorIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '*' ||
		ch == '%' || ch == ')' || ch == ']' || ch == '.'
}

// compactDeclaration writes a compact CSS declaration.
func compactDeclaration(w *strings.Builder, name string, values []css.Token) {
	w.WriteString(strings.TrimSpace(name))
	w.WriteString(":")
	compactDeclValue(w, values)
}

// compactDeclValue writes compact CSS declaration values.
func compactDeclValue(w *strings.Builder, values []css.Token) {
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
		// Normalize single-quoted strings to double-quoted
		if v.TokenType == css.StringToken && len(s) >= 2 && s[0] == '\'' {
			s = `"` + s[1:len(s)-1] + `"`
		}
		// Normalize URL tokens: url('...') -> url(...)
		if v.TokenType == css.URLToken {
			s = normalizeURLToken(s)
		}
		if v.TokenType == css.WhitespaceToken {
			if len(parts) > 0 {
				parts = append(parts, " ")
			}
			continue
		}
		// Commas: the parser strips whitespace after commas, restore ", "
		if v.TokenType == css.CommaToken {
			parts = append(parts, ", ")
			continue
		}
		parts = append(parts, s)
	}

	// Join parts, collapsing spaces
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

// compactAtRulePrelude compacts an at-rule prelude (the part between @name and {).
func compactAtRulePrelude(values []css.Token) string {
	var sb strings.Builder
	prevWS := true
	parenDepth := 0

	for _, v := range values {
		if v.TokenType == css.WhitespaceToken {
			if !prevWS {
				// Inside parentheses, skip whitespace around ':'
				if parenDepth == 0 {
					sb.WriteByte(' ')
				} else {
					// Only add space between word-like tokens inside parens
					sb.WriteByte(' ')
				}
				prevWS = true
			}
			continue
		}
		prevWS = false
		s := string(v.Data)

		// Track parenthesis depth
		for _, ch := range s {
			if ch == '(' {
				parenDepth++
			} else if ch == ')' {
				parenDepth--
			}
		}

		// Normalize single-quoted strings to double-quoted
		if v.TokenType == css.StringToken && len(s) >= 2 && s[0] == '\'' {
			s = `"` + s[1:len(s)-1] + `"`
		}
		// Normalize URL tokens: url('...') -> url(...)
		if v.TokenType == css.URLToken {
			s = normalizeURLToken(s)
		}
		sb.WriteString(s)
	}

	result := strings.TrimSpace(sb.String())
	// Remove spaces around ':' inside parentheses
	result = removeSpacesAroundColonInParens(result)
	return result
}

// normalizeURLToken removes quotes from url() tokens.
// url('https://example.com') -> url(https://example.com)
func normalizeURLToken(s string) string {
	if !strings.HasPrefix(s, "url(") {
		return s
	}
	inner := s[4 : len(s)-1] // extract content between url( and )
	// Remove quotes if present
	if len(inner) >= 2 && (inner[0] == '\'' || inner[0] == '"') {
		inner = inner[1 : len(inner)-1]
	}
	return "url(" + inner + ")"
}

// removeSpacesAroundColonInParens removes spaces before and after ':' when inside parentheses.
func removeSpacesAroundColonInParens(s string) string {
	var sb strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '(' {
			depth++
			sb.WriteByte(ch)
		} else if ch == ')' {
			depth--
			sb.WriteByte(ch)
		} else if depth > 0 && ch == ' ' {
			// Check if this space is adjacent to ':'
			// Remove space before ':'
			if i+1 < len(s) && s[i+1] == ':' {
				continue
			}
			// Remove space after ':'
			if i > 0 && s[i-1] == ':' {
				continue
			}
			sb.WriteByte(ch)
		} else {
			sb.WriteByte(ch)
		}
	}
	return sb.String()
}

// isDeclarationAtRule returns true for at-rules that contain declarations (not rulesets).
func isDeclarationAtRule(name string) bool {
	n := strings.TrimPrefix(strings.ToLower(name), "@")
	switch n {
	case "font-face", "viewport", "page", "counter-style":
		return true
	}
	return false
}
