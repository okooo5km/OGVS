// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package minifystyles

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// usageIndex holds the document usage data used to drop unused CSS rules.
// Tag names are stored lowercased because csso builds its tag map
// case-insensitively; ids and classes are matched case-sensitively.
type usageIndex struct {
	tags    map[string]bool
	classes map[string]bool
	ids     map[string]bool
	cfg     usageConfig
}

// selToken is a CSS token together with its byte range in the source text.
type selToken struct {
	tt         css.TokenType
	start, end int
}

func (t selToken) text(src string) string { return src[t.start:t.end] }

// lexTokens tokenizes s. It returns nil when the tokens do not cover the whole
// input, which lets every caller fall back to leaving the CSS untouched.
func lexTokens(s string) []selToken {
	l := css.NewLexer(parse.NewInput(strings.NewReader(s)))
	toks := make([]selToken, 0, 16)
	pos := 0
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		toks = append(toks, selToken{tt: tt, start: pos, end: pos + len(data)})
		pos += len(data)
	}
	if pos != len(s) {
		return nil
	}
	return toks
}

func isIdentLike(tt css.TokenType) bool {
	return tt == css.IdentToken || tt == css.CustomPropertyNameToken
}

// matchClose returns the token index just past the token closing the group
// opened at toks[i]. When matching parentheses, both FunctionToken and
// LeftParenthesisToken count as openers.
func matchClose(toks []selToken, i, hi int, open, closeT css.TokenType) int {
	depth := 0
	for j := i; j < hi; j++ {
		tt := toks[j].tt
		switch {
		case tt == open || (closeT == css.RightParenthesisToken &&
			(tt == css.FunctionToken || tt == css.LeftParenthesisToken)):
			depth++
		case tt == closeT:
			depth--
			if depth == 0 {
				return j + 1
			}
		}
	}
	return hi
}

// selectorListPseudos are the functional pseudo-classes whose argument csso
// treats as a nested selector list and prunes recursively. It mirrors the
// css-tree 2.2.1 pseudo table vendored by csso, minus ":not" whose argument
// csso explicitly leaves alone.
var selectorListPseudos = map[string]bool{
	"has": true, "matches": true, "is": true,
	"-moz-any": true, "-webkit-any": true, "where": true,
}

// removeUnusedRules removes CSS rules whose selectors are not used by the
// document, mirroring csso's clean/Rule.js. Individual selectors are dropped
// from a selector list and a rule disappears only once its list is empty.
// The stylesheet is walked at the token level so braces inside strings, url()
// and comments never terminate a block.
func removeUnusedRules(cssText string, u *usageIndex) string {
	toks := lexTokens(cssText)
	if toks == nil {
		return cssText
	}
	var sb strings.Builder
	walkRules(cssText, toks, 0, len(toks), u, &sb)
	return sb.String()
}

// walkRules copies toks[lo:hi] to out, dropping unused qualified rules and
// recursing into at-rule blocks other than @keyframes.
func walkRules(src string, toks []selToken, lo, hi int, u *usageIndex, out *strings.Builder) {
	i := lo
	for i < hi {
		if toks[i].tt == css.WhitespaceToken || toks[i].tt == css.CommentToken {
			out.WriteString(toks[i].text(src))
			i++
			continue
		}

		start := i
		depth := 0
		end := hi
		brace := -1
	scan:
		for j := i; j < hi; j++ {
			switch toks[j].tt {
			case css.LeftParenthesisToken, css.LeftBracketToken, css.FunctionToken:
				depth++
			case css.RightParenthesisToken, css.RightBracketToken:
				depth--
			case css.SemicolonToken:
				if depth == 0 {
					end = j + 1
					break scan
				}
			case css.LeftBraceToken:
				if depth == 0 {
					brace = j
					end = matchClose(toks, j, hi, css.LeftBraceToken, css.RightBraceToken)
					break scan
				}
				depth++
			case css.RightBraceToken:
				if depth == 0 {
					end = j
					break scan
				}
				depth--
			}
		}
		if end <= start {
			end = start + 1
		}

		// Declaration, at-statement (@import x;) or trailing text: copy as is.
		if brace < 0 {
			out.WriteString(src[toks[start].start:toks[end-1].end])
			i = end
			continue
		}

		if toks[start].tt == css.AtKeywordToken {
			name := atRuleBasename(strings.TrimPrefix(toks[start].text(src), "@"))
			if name == "keyframes" || toks[end-1].tt != css.RightBraceToken {
				// csso skips usage filtering inside @keyframes.
				out.WriteString(src[toks[start].start:toks[end-1].end])
				i = end
				continue
			}
			var inner strings.Builder
			walkRules(src, toks, brace+1, end-1, u, &inner)
			// csso drops an at-rule once its block holds nothing.
			if strings.TrimSpace(inner.String()) != "" {
				out.WriteString(src[toks[start].start:toks[brace].end])
				out.WriteString(inner.String())
				out.WriteString("}")
			}
			i = end
			continue
		}

		selector := strings.TrimSpace(src[toks[start].start:toks[brace].start])
		if pruned, keep := pruneSelectorList(selector, u); keep {
			out.WriteString(pruned)
			out.WriteString(src[toks[brace].start:toks[end-1].end])
		}
		i = end
	}
}

// atRuleBasename lowercases an at-rule name and strips its vendor prefix,
// matching css-tree's keyword().basename.
func atRuleBasename(name string) string {
	name = strings.ToLower(name)
	if len(name) > 3 && name[0] == '-' && name[1] != '-' {
		if k := strings.Index(name[1:], "-"); k >= 0 {
			return name[k+2:]
		}
	}
	return name
}

// pruneSelectorList drops every unused complex selector from a selector list
// and reports whether anything survived.
func pruneSelectorList(src string, u *usageIndex) (string, bool) {
	toks := lexTokens(src)
	if toks == nil {
		return src, true
	}
	return pruneRange(src, toks, 0, len(toks), u)
}

func pruneRange(src string, toks []selToken, lo, hi int, u *usageIndex) (string, bool) {
	var kept []string
	start := lo
	depth := 0
	for i := lo; i <= hi; i++ {
		if i == hi || (depth == 0 && toks[i].tt == css.CommaToken) {
			var sb strings.Builder
			if scanComplex(src, toks, start, i, u, &sb) {
				if part := strings.TrimSpace(sb.String()); part != "" {
					kept = append(kept, part)
				}
			}
			start = i + 1
			continue
		}
		switch toks[i].tt {
		case css.LeftParenthesisToken, css.LeftBracketToken, css.LeftBraceToken, css.FunctionToken:
			depth++
		case css.RightParenthesisToken, css.RightBracketToken, css.RightBraceToken:
			depth--
		}
	}
	if len(kept) == 0 {
		return "", false
	}
	return strings.Join(kept, ","), true
}

// scanComplex walks one complex selector, writing its (possibly rewritten)
// text to out, and reports whether it survives usage filtering.
func scanComplex(src string, toks []selToken, lo, hi int, u *usageIndex, out *strings.Builder) bool {
	keep := true
	atCompoundStart := true

	for i := lo; i < hi; {
		t := toks[i]
		switch {
		// Attribute selector: opaque, including nested quotes and escapes.
		case t.tt == css.LeftBracketToken:
			j := matchClose(toks, i, hi, css.LeftBracketToken, css.RightBracketToken)
			out.WriteString(src[t.start:toks[j-1].end])
			i = j
			atCompoundStart = false

		// Pseudo-class or pseudo-element.
		case t.tt == css.ColonToken:
			out.WriteString(t.text(src))
			i++
			if i < hi && toks[i].tt == css.ColonToken {
				out.WriteString(toks[i].text(src))
				i++
			}
			if i >= hi {
				atCompoundStart = false
				break
			}
			if toks[i].tt == css.FunctionToken {
				raw := toks[i].text(src)
				name := strings.ToLower(strings.TrimSuffix(raw, "("))
				j := matchClose(toks, i, hi, css.FunctionToken, css.RightParenthesisToken)
				innerLo, innerHi := i+1, j-1
				if innerHi < innerLo {
					innerHi = innerLo
				}
				switch {
				case selectorListPseudos[name]:
					inner, ok := pruneRange(src, toks, innerLo, innerHi, u)
					if !ok {
						keep = false
					}
					out.WriteString(raw)
					out.WriteString(inner)
					out.WriteString(")")
				case name == "slotted":
					// css-tree parses the argument as a single Selector, so
					// csso checks its parts inline without pruning them.
					out.WriteString(raw)
					if !scanComplex(src, toks, innerLo, innerHi, u, out) {
						keep = false
					}
					out.WriteString(")")
				default:
					// ":not()" and every unknown pseudo: argument is opaque.
					out.WriteString(src[toks[i].start:toks[j-1].end])
				}
				i = j
			} else {
				out.WriteString(toks[i].text(src))
				i++
			}
			atCompoundStart = false

		// ID selector.
		case t.tt == css.HashToken:
			name := strings.TrimPrefix(t.text(src), "#")
			if u.cfg.ids && !u.ids[name] {
				keep = false
			}
			out.WriteString(t.text(src))
			i++
			atCompoundStart = false

		// Class selector.
		case t.tt == css.DelimToken && src[t.start] == '.' && i+1 < hi && isIdentLike(toks[i+1].tt):
			name := toks[i+1].text(src)
			if u.cfg.classes && !u.classes[name] {
				keep = false
			}
			out.WriteString(src[t.start:toks[i+1].end])
			i += 2
			atCompoundStart = false

		// Type (or universal) selector: only valid at the start of a compound.
		case atCompoundStart && (isIdentLike(t.tt) ||
			(t.tt == css.DelimToken && (src[t.start] == '*' || src[t.start] == '|'))):
			j, name := readTypeSelector(src, toks, i, hi)
			if !strings.HasSuffix(name, "*") && u.cfg.tags &&
				!u.tags[strings.ToLower(name)] {
				keep = false
			}
			out.WriteString(src[t.start:toks[j-1].end])
			i = j
			atCompoundStart = false

		default:
			out.WriteString(t.text(src))
			if t.tt == css.WhitespaceToken ||
				(t.tt == css.DelimToken && strings.ContainsAny(src[t.start:t.end], ">+~")) {
				atCompoundStart = true
			}
			i++
		}
	}
	return keep
}

// readTypeSelector consumes a type selector at toks[i], including an optional
// namespace prefix, and returns the index just past it plus the name csso sees
// (e.g. "rect", "svg|rect", "*", "svg|*").
func readTypeSelector(src string, toks []selToken, i, hi int) (int, string) {
	adjacent := func(a, b int) bool { return toks[a].end == toks[b].start }
	isName := func(k int) bool {
		return isIdentLike(toks[k].tt) ||
			(toks[k].tt == css.DelimToken && src[toks[k].start] == '*')
	}
	if isName(i) {
		if i+2 < hi && toks[i+1].tt == css.DelimToken && src[toks[i+1].start] == '|' &&
			adjacent(i, i+1) && isName(i+2) && adjacent(i+1, i+2) {
			return i + 3, src[toks[i].start:toks[i+2].end]
		}
		return i + 1, toks[i].text(src)
	}
	if toks[i].tt == css.DelimToken && src[toks[i].start] == '|' && i+1 < hi &&
		isName(i+1) && adjacent(i, i+1) {
		return i + 2, src[toks[i].start:toks[i+1].end]
	}
	return i + 1, toks[i].text(src)
}
