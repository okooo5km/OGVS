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
	var raw rawTokenState

	for {
		gt, tt, data := p.Next()
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

		case css.CustomPropertyGrammar:
			if !firstDecl {
				w.WriteString(";")
			}
			firstDecl = false
			compactCustomProperty(w, string(data), p.Values())

		case css.TokenGrammar:
			// At-rules the parser has no model for (@container, @scope,
			// @starting-style) surface their whole body as a raw token stream.
			raw.write(w, tt, data)
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
				switch dgt {
				case css.DeclarationGrammar:
					if !firstDecl {
						w.WriteString(";")
					}
					firstDecl = false
					compactDeclaration(w, string(ddata), p.Values())
				case css.CustomPropertyGrammar:
					if !firstDecl {
						w.WriteString(";")
					}
					firstDecl = false
					compactCustomProperty(w, string(ddata), p.Values())
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
		switch gt {
		case css.DeclarationGrammar:
			if !firstDecl {
				w.WriteString(";")
			}
			firstDecl = false
			compactDeclaration(w, string(data), p.Values())
		case css.CustomPropertyGrammar:
			if !firstDecl {
				w.WriteString(";")
			}
			firstDecl = false
			compactCustomProperty(w, string(data), p.Values())
		case css.TokenGrammar:
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
		switch gt {
		case css.DeclarationGrammar:
			if !firstDecl {
				declBuf.WriteString(";")
			}
			firstDecl = false
			compactDeclaration(&declBuf, string(data), p.Values())
		case css.CustomPropertyGrammar:
			if !firstDecl {
				declBuf.WriteString(";")
			}
			firstDecl = false
			compactCustomProperty(&declBuf, string(data), p.Values())
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
// The tokenizer discards the whitespace inside an attribute selector, so the
// space that separates a value from a following case-sensitivity flag is put
// back: without it the flag would read as part of the value.
func compactSelector(data string, values []css.Token) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(data))

	inAttr := false
	valueSeen := false
	for _, v := range values {
		switch v.TokenType {
		case css.LeftBracketToken:
			inAttr = true
			valueSeen = false
		case css.RightBracketToken:
			inAttr = false
			valueSeen = false
		case css.IdentToken, css.StringToken, css.NumberToken, css.DimensionToken:
			if inAttr {
				if valueSeen {
					sb.WriteByte(' ')
				}
				valueSeen = true
			}
		default:
			if inAttr {
				valueSeen = false
			}
		}
		sb.WriteString(string(v.Data))
	}
	return strings.TrimSpace(sb.String())
}

// compactSelectorString compacts a single selector by collapsing each run of
// whitespace to at most one space. A run is dropped only when one of the
// characters it sits between already separates two selector components: a
// combinator, a comma, an opening bracket on the left, a closing bracket on the
// right, or an attribute-selector '='. Every other run is a descendant
// combinator (or an attribute-selector modifier such as the 'i'/'s' flag) and
// must survive. Quoted strings are copied verbatim.
func compactSelectorString(s string) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == '"' || ch == '\'' {
			i = copyQuotedString(&sb, s, i)
			continue
		}
		if isCSSSpace(ch) {
			for i < len(s) && isCSSSpace(s[i]) {
				i++
			}
			// Don't add leading or trailing space
			if sb.Len() == 0 || i >= len(s) {
				continue
			}
			buf := sb.String()
			prev := buf[len(buf)-1]
			next := s[i]
			if !separatesAfter(prev) && !separatesBefore(next) {
				sb.WriteByte(' ')
			}
			continue
		}
		sb.WriteByte(ch)
		i++
	}
	return strings.TrimSpace(sb.String())
}

// copyQuotedString copies the quoted string starting at s[i] to sb verbatim and
// returns the index just past its closing quote.
func copyQuotedString(sb *strings.Builder, s string, i int) int {
	quote := s[i]
	sb.WriteByte(quote)
	i++
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			sb.WriteString(s[i : i+2])
			i += 2
			continue
		}
		sb.WriteByte(s[i])
		i++
		if s[i-1] == quote {
			break
		}
	}
	return i
}

func isCSSSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f'
}

// separatesAfter reports whether whitespace following ch is redundant.
func separatesAfter(ch byte) bool {
	switch ch {
	case '>', '+', '~', '/', ',', '(', '[', '=', '{', '}':
		return true
	}
	return false
}

// separatesBefore reports whether whitespace preceding ch is redundant.
func separatesBefore(ch byte) bool {
	switch ch {
	case '>', '+', '~', '/', ',', ')', ']', '=', '{':
		return true
	}
	return false
}

// compactCustomProperty writes a compact custom property declaration. tdewolff
// reports these through CustomPropertyGrammar with a single raw value token that
// still carries its surrounding whitespace.
func compactCustomProperty(w *strings.Builder, name string, values []css.Token) {
	w.WriteString(strings.TrimSpace(name))
	w.WriteString(":")
	var raw strings.Builder
	for _, v := range values {
		raw.WriteString(string(v.Data))
	}
	w.WriteString(strings.TrimSpace(raw.String()))
}

// rawTokenState reassembles the raw token stream tdewolff emits for at-rules it
// has no grammar for (@container, @scope, @starting-style) into compact CSS.
// Whitespace runs collapse to a single space and are dropped where they can
// never be significant. Brace depth tells selector context (depth 0) from
// declaration context (depth above 0), which decides whether a colon or a
// combinator character is a separator.
type rawTokenState struct {
	pendingWS bool
	prev      byte
	depth     int
}

func (r *rawTokenState) write(w *strings.Builder, tt css.TokenType, data []byte) {
	if tt == css.CommentToken {
		return
	}
	if tt == css.WhitespaceToken {
		r.pendingWS = true
		return
	}
	if len(data) == 0 {
		return
	}
	if r.pendingWS {
		r.pendingWS = false
		if r.prev != 0 && !r.separatesAfter(r.prev) && !r.separatesBefore(data[0]) {
			w.WriteByte(' ')
		}
	}
	w.Write(data)
	r.prev = data[len(data)-1]
	for _, ch := range data {
		switch ch {
		case '{':
			r.depth++
		case '}':
			r.depth--
		}
	}
}

// separatesAfter reports whether whitespace following ch is redundant.
func (r *rawTokenState) separatesAfter(ch byte) bool {
	switch ch {
	case '{', '}', ';', ',', '(', '[', '=':
		return true
	case ':':
		return r.depth > 0
	case '>', '+', '~':
		return r.depth == 0
	}
	return false
}

// separatesBefore reports whether whitespace preceding ch is redundant.
func (r *rawTokenState) separatesBefore(ch byte) bool {
	switch ch {
	case '{', '}', ';', ',', ')', ']', '=', '!':
		return true
	case ':':
		return r.depth > 0
	case '>', '+', '~':
		return r.depth == 0
	}
	return false
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
	// tdewolff can emit an unterminated "url(" (len 4) at EOF; require a
	// closing ')' so the slice below never goes out of range.
	if !strings.HasPrefix(s, "url(") || !strings.HasSuffix(s, ")") || len(s) < 5 {
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
