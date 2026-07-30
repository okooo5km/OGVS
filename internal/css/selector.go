// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"strings"
	"unicode/utf8"

	"github.com/okooo5km/ogvs/internal/svgast"
)

// matchCtx carries matcher state across a single selector evaluation.
// err is set when css-select would have thrown, which makes the whole selector
// unusable; callers that care use the *Err variants.
type matchCtx struct {
	parents map[svgast.Node]svgast.Parent
	err     bool
}

// Matches checks if an element matches a CSS selector string.
// parents provides the parent map for traversal.
func Matches(node *svgast.Element, selector string, parents map[svgast.Node]svgast.Parent) bool {
	matched, _ := MatchesErr(node, selector, parents)
	return matched
}

// MatchesErr is Matches, additionally reporting whether the selector could be
// fully evaluated. When ok is false the selector contains something css-select
// throws on, and SVGO skips the rule entirely.
func MatchesErr(node *svgast.Element, selector string, parents map[svgast.Node]svgast.Parent) (matched, ok bool) {
	ctx := &matchCtx{parents: parents}
	matched = matchesSelectorList(node, selector, ctx)
	return matched, !ctx.err
}

// matchesCompound matches a single complex selector (no commas).
func matchesCompound(node *svgast.Element, selector string, ctx *matchCtx) bool {
	// Parse the selector into parts split by combinators
	parts, combinators := parseSelectorParts(selector)
	if len(parts) == 0 {
		return false
	}

	// Match from right to left
	if !matchesSimple(node, parts[len(parts)-1], ctx) {
		return false
	}

	if len(parts) == 1 {
		return true
	}

	// Walk up the selector parts with combinators
	currentNode := svgast.Node(node)
	for i := len(parts) - 2; i >= 0; i-- {
		combinator := combinators[i]
		part := parts[i]

		switch combinator {
		case ' ': // descendant
			found := false
			parent := ctx.parents[currentNode]
			for parent != nil {
				if elem, ok := parent.(*svgast.Element); ok {
					if matchesSimple(elem, part, ctx) {
						currentNode = elem
						found = true
						break
					}
				}
				parent = ctx.parents[parent]
			}
			if !found {
				return false
			}

		case '>': // child
			parent := ctx.parents[currentNode]
			if parent == nil {
				return false
			}
			elem, ok := parent.(*svgast.Element)
			if !ok || !matchesSimple(elem, part, ctx) {
				return false
			}
			currentNode = elem

		case '+': // adjacent sibling
			sibling := getPreviousSibling(currentNode, ctx.parents)
			if sibling == nil || !matchesSimple(sibling, part, ctx) {
				return false
			}
			currentNode = sibling

		case '~': // general sibling
			found := false
			sibling := getPreviousSibling(currentNode, ctx.parents)
			for sibling != nil {
				if matchesSimple(sibling, part, ctx) {
					currentNode = sibling
					found = true
					break
				}
				sibling = getPreviousSibling(sibling, ctx.parents)
			}
			if !found {
				return false
			}

		default:
			return false
		}
	}

	return true
}

// getPreviousSibling returns the previous element sibling.
func getPreviousSibling(node svgast.Node, parents map[svgast.Node]svgast.Parent) *svgast.Element {
	parent := parents[node]
	if parent == nil {
		return nil
	}
	var children []svgast.Node
	switch p := parent.(type) {
	case *svgast.Element:
		children = p.Children
	case *svgast.Root:
		children = p.Children
	default:
		return nil
	}

	for i, child := range children {
		if child == node {
			// Walk backwards to find previous element sibling
			for j := i - 1; j >= 0; j-- {
				if elem, ok := children[j].(*svgast.Element); ok {
					return elem
				}
			}
			return nil
		}
	}
	return nil
}

// parseSelectorParts splits a compound selector into simple selector parts
// and combinators. Returns parts and combinators (one fewer than parts).
func parseSelectorParts(selector string) (parts []string, combinators []byte) {
	var current strings.Builder
	i := 0

	for i < len(selector) {
		ch := selector[i]

		// Check for combinators (space, >, +, ~)
		if ch == '>' || ch == '+' || ch == '~' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			combinators = append(combinators, ch)
			i++
			// Skip whitespace after combinator
			for i < len(selector) && selector[i] == ' ' {
				i++
			}
			continue
		}

		if ch == ' ' {
			// Could be a descendant combinator or just whitespace before >+~
			j := i + 1
			for j < len(selector) && selector[j] == ' ' {
				j++
			}
			if j < len(selector) && (selector[j] == '>' || selector[j] == '+' || selector[j] == '~') {
				i = j
				continue
			}
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
				combinators = append(combinators, ' ')
			}
			i = j
			continue
		}

		// Skip content inside brackets []
		if ch == '[' {
			current.WriteByte(ch)
			i++
			for i < len(selector) && selector[i] != ']' {
				current.WriteByte(selector[i])
				i++
			}
			if i < len(selector) {
				current.WriteByte(selector[i])
				i++
			}
			continue
		}

		// Skip content inside parentheses ()
		if ch == '(' {
			depth := 1
			current.WriteByte(ch)
			i++
			for i < len(selector) && depth > 0 {
				if selector[i] == '(' {
					depth++
				}
				if selector[i] == ')' {
					depth--
				}
				current.WriteByte(selector[i])
				i++
			}
			continue
		}

		current.WriteByte(ch)
		i++
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return
}

// matchesSimple matches an element against a simple selector (no combinators).
// Supports: element, .class, #id, [attr], [attr=val], *
func matchesSimple(elem *svgast.Element, selector string, ctx *matchCtx) bool {
	if selector == "*" {
		return true
	}

	// Parse simple selector into conditions
	conditions := parseSimpleSelector(selector)
	for _, cond := range conditions {
		if !matchCondition(elem, cond, ctx) {
			return false
		}
	}

	return len(conditions) > 0
}

type selectorCondition struct {
	condType   string // "element", "class", "id", "attr", "attr-eq", "pseudo"
	name       string
	value      string
	op         string // attribute operator: "=", "~=", "|=", "^=", "$=", "*=", "!="
	ignoreCase bool   // the `i` flag on an attribute selector
	invalid    bool   // css-select rejects this selector outright
	arg        string // functional pseudo-class argument, e.g. "2n+1" or ".a, .b"
	nth        nthExpr
	nthOK      bool // false when the An+B argument failed to parse
}

// parseSimpleSelector parses a simple selector into conditions.
func parseSimpleSelector(selector string) []selectorCondition {
	var conditions []selectorCondition
	i := 0

	for i < len(selector) {
		switch selector[i] {
		case '#': // ID selector
			i++
			name := readIdent(selector, &i)
			conditions = append(conditions, selectorCondition{
				condType: "id", name: name,
			})

		case '.': // Class selector
			i++
			name := readIdent(selector, &i)
			conditions = append(conditions, selectorCondition{
				condType: "class", name: name,
			})

		case '[': // Attribute selector
			i++ // skip [
			as := parseAttrSelector(selector, &i)
			cond := selectorCondition{
				condType:   "attr",
				name:       as.name,
				value:      as.value,
				op:         as.op,
				ignoreCase: as.ignoreCase,
				invalid:    as.invalid,
			}
			if as.op != "" {
				cond.condType = "attr-eq"
			}
			conditions = append(conditions, cond)

		case ':': // Pseudo-class/element
			i++
			if i < len(selector) && selector[i] == ':' {
				// Pseudo-element: css-select rejects these outright and SVGO
				// never strips them, so keep ignoring them.
				i++
				readIdent(selector, &i)
				continue
			}
			name := strings.ToLower(readIdent(selector, &i))
			cond := selectorCondition{condType: "pseudo", name: name}
			if i < len(selector) && selector[i] == '(' {
				cond.arg = readParenGroup(selector, &i)
			}
			switch name {
			case "nth-child", "nth-last-child", "nth-of-type", "nth-last-of-type":
				cond.nth, cond.nthOK = parseNth(cond.arg)
			}
			conditions = append(conditions, cond)

		case '*':
			i++
			// Universal selector, matches anything

		default: // Element type selector
			before := i
			name := readIdent(selector, &i)
			if name != "" {
				conditions = append(conditions, selectorCondition{
					condType: "element", name: name,
				})
			} else if i == before {
				// readIdent made no progress on an unexpected char
				// (e.g. a stray ')' or '('); skip it to avoid an infinite loop.
				i++
			}
		}
	}

	return conditions
}

// readIdent reads a CSS identifier from position i.
func readIdent(s string, i *int) string {
	start := *i
	for *i < len(s) {
		ch := s[*i]
		if ch == '.' || ch == '#' || ch == '[' || ch == ']' ||
			ch == ':' || ch == ' ' || ch == '>' || ch == '+' ||
			ch == '~' || ch == '(' || ch == ')' || ch == ',' {
			break
		}
		(*i)++
	}
	return s[start:*i]
}

// attrSelector is a parsed `[...]` attribute selector.
//
// op is "" for a bare presence test, otherwise one of "=", "~=", "|=", "^=",
// "$=", "*=" or "!=" ("!=" is a css-select extension, not standard CSS).
// invalid marks selectors that css-select rejects outright — namespaced
// attributes and malformed syntax.
type attrSelector struct {
	name       string
	op         string
	value      string
	ignoreCase bool
	invalid    bool
}

// parseAttrSelector parses an attribute selector body starting just after the
// opening '[', e.g. `attr]`, `attr=value]` or `attr~="value" i]`, and advances
// *i past the closing ']'.
//
// This mirrors css-what's attribute branch (lib/commonjs/parse.js), which is
// the parser css-select — and therefore SVGO — uses.
func parseAttrSelector(s string, i *int) attrSelector {
	var as attrSelector

	skipSelectorSpace(s, i)

	// Namespace prefix. `[|attr]` means "no namespace" and is equivalent to
	// `[attr]`. Anything else (`[ns|attr]`, `[*|attr]`) makes css-select throw
	// "Namespaced attributes are not yet supported".
	if *i < len(s) && s[*i] == '|' {
		(*i)++
	} else if strings.HasPrefix(s[*i:], "*|") {
		*i += 2
		as.invalid = true
	}

	as.name = readAttrName(s, i)
	if *i < len(s) && s[*i] == '|' && (*i+1 >= len(s) || s[*i+1] != '=') {
		(*i)++
		as.name = readAttrName(s, i)
		as.invalid = true
	}

	skipSelectorSpace(s, i)

	// Comparison operator. An operator character not followed by '=' makes
	// css-what throw "Expected `=`".
	if *i < len(s) {
		switch c := s[*i]; c {
		case '~', '|', '^', '$', '*', '!':
			if *i+1 < len(s) && s[*i+1] == '=' {
				as.op = string(c) + "="
				*i += 2
			} else {
				as.invalid = true
			}
		case '=':
			as.op = "="
			(*i)++
		}
	}

	if as.op != "" {
		skipSelectorSpace(s, i)
		as.value = readAttrValue(s, i)
		skipSelectorSpace(s, i)

		// Optional case-sensitivity flag. `i` forces case-insensitive
		// matching, `s` forces case-sensitive — which is already the default,
		// because SVGO drives css-select with xmlMode: true and that disables
		// the HTML case-insensitive attribute list. Both flags are themselves
		// matched case-insensitively.
		if *i < len(s) {
			switch s[*i] {
			case 'i', 'I':
				as.ignoreCase = true
				(*i)++
				skipSelectorSpace(s, i)
			case 's', 'S':
				(*i)++
				skipSelectorSpace(s, i)
			}
		}
	}

	if *i < len(s) && s[*i] == ']' {
		(*i)++
	} else {
		// css-what: "Attribute selector didn't terminate".
		as.invalid = true
		for *i < len(s) && s[*i] != ']' {
			(*i)++
		}
		if *i < len(s) {
			(*i)++
		}
	}

	return as
}

// readAttrName reads an attribute name inside an attribute selector, resolving
// CSS escapes.
func readAttrName(s string, i *int) string {
	start := *i
	for *i < len(s) {
		c := s[*i]
		if c == '\\' && *i+1 < len(s) {
			*i += 2
			continue
		}
		if isSelectorSpace(c) || c == '=' || c == ']' || c == '~' ||
			c == '|' || c == '^' || c == '$' || c == '*' || c == '!' {
			break
		}
		(*i)++
	}
	return unescapeCSSIdentifier(s[start:*i])
}

// readAttrValue reads a quoted or bare attribute selector value, resolving CSS
// escapes. A bare value ends at whitespace or ']'.
func readAttrValue(s string, i *int) string {
	if *i >= len(s) {
		return ""
	}

	if s[*i] == '\'' || s[*i] == '"' {
		quote := s[*i]
		(*i)++
		start := *i
		for *i < len(s) && s[*i] != quote {
			if s[*i] == '\\' && *i+1 < len(s) {
				(*i)++
			}
			(*i)++
		}
		val := unescapeCSSIdentifier(s[start:*i])
		if *i < len(s) {
			(*i)++ // skip closing quote
		}
		return val
	}

	start := *i
	for *i < len(s) {
		c := s[*i]
		if c == '\\' && *i+1 < len(s) {
			*i += 2
			continue
		}
		if c == ']' || isSelectorSpace(c) {
			break
		}
		(*i)++
	}
	return unescapeCSSIdentifier(s[start:*i])
}

// isSelectorSpace reports whether c is whitespace inside selector syntax,
// matching css-what's isWhitespace.
func isSelectorSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}

// skipSelectorSpace advances *i past selector whitespace.
func skipSelectorSpace(s string, i *int) {
	for *i < len(s) && isSelectorSpace(s[*i]) {
		(*i)++
	}
}

// readParenGroup reads a parenthesised group starting at s[*i] == '(' and
// returns its contents, leaving *i just past the closing parenthesis.
func readParenGroup(s string, i *int) string {
	(*i)++ // skip (
	start := *i
	depth := 1
	for *i < len(s) && depth > 0 {
		switch s[*i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth > 0 {
			(*i)++
		}
	}
	arg := s[start:*i]
	if *i < len(s) {
		(*i)++ // skip )
	}
	return arg
}

// matchCondition checks if an element matches a single condition.
func matchCondition(elem *svgast.Element, cond selectorCondition, ctx *matchCtx) bool {
	if cond.invalid {
		// css-select throws here, which aborts the whole selector.
		ctx.err = true
		return false
	}
	switch cond.condType {
	case "element":
		return elem.Name == cond.name
	case "id":
		id, _ := elem.Attributes.Get("id")
		return id == cond.name
	case "class":
		classAttr, _ := elem.Attributes.Get("class")
		classes := strings.Fields(classAttr)
		for _, c := range classes {
			if c == cond.name {
				return true
			}
		}
		return false
	case "attr":
		return elem.Attributes.Has(cond.name)
	case "attr-eq":
		return matchAttrOp(elem, cond)
	case "pseudo":
		return matchPseudo(elem, cond, ctx)
	}
	return false
}

// matchAttrOp evaluates an attribute selector that carries a comparison
// operator, mirroring css-select's attributeRules (lib/attributes.js).
//
// SVGO drives css-select with xmlMode: true (lib/xast.js), which disables
// css-select's HTML case-insensitive attribute list, so matching is always
// case-sensitive unless the selector carries an explicit `i` flag.
func matchAttrOp(elem *svgast.Element, cond selectorCondition) bool {
	attr, ok := elem.Attributes.Get(cond.name)
	value := cond.value

	if cond.ignoreCase {
		attr = strings.ToLower(attr)
		value = strings.ToLower(value)
	}

	switch cond.op {
	case "=": // exact match
		return ok && attr == value

	case "~=": // membership in a whitespace-separated word list
		// css-select compiles this to `(?:^|\s)value(?:$|\s)` and short-
		// circuits to "never matches" when the value itself has whitespace.
		// Note it does NOT reject an empty value: `[a~=""]` matches an empty
		// or all-whitespace attribute.
		if !ok || strings.ContainsFunc(value, isCSSSpaceRune) {
			return false
		}
		return containsWord(attr, value)

	case "|=": // exactly value, or value followed by "-"
		return ok && strings.HasPrefix(attr, value) &&
			(len(attr) == len(value) || attr[len(value)] == '-')

	case "^=": // prefix; an empty value never matches
		return ok && value != "" && strings.HasPrefix(attr, value)

	case "$=": // suffix; an empty value never matches
		return ok && value != "" && strings.HasSuffix(attr, value)

	case "*=": // substring; an empty value never matches
		return ok && value != "" && strings.Contains(attr, value)

	case "!=": // css-select extension: everything except an exact match
		if value == "" {
			return attr != ""
		}
		return attr != value
	}

	return false
}

// containsWord reports whether value occurs in attr delimited by whitespace or
// string boundaries. It reproduces css-select's `(?:^|\s)value(?:$|\s)` regex,
// including the empty-value case, where an empty or all-whitespace attr
// matches.
func containsWord(attr, value string) bool {
	if len(attr) < len(value) {
		return false
	}
	for p := 0; p+len(value) <= len(attr); p++ {
		if attr[p:p+len(value)] != value {
			continue
		}
		if p > 0 {
			r, _ := utf8.DecodeLastRuneInString(attr[:p])
			if !isCSSSpaceRune(r) {
				continue
			}
		}
		if end := p + len(value); end < len(attr) {
			r, _ := utf8.DecodeRuneInString(attr[end:])
			if !isCSSSpaceRune(r) {
				continue
			}
		}
		return true
	}
	return false
}

// isCSSSpaceRune reports whether r is whitespace as JavaScript's `\s` character
// class defines it, which is what css-select's `~=` regex matches against.
func isCSSSpaceRune(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ',
		0x00a0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// QuerySelectorAll finds all descendant elements matching a selector.
// Selectors css-select would throw on yield no matches.
func QuerySelectorAll(node svgast.Node, selector string, parents map[svgast.Node]svgast.Parent) []*svgast.Element {
	results, _ := QuerySelectorAllErr(node, selector, parents)
	return results
}

// QuerySelectorAllErr is QuerySelectorAll, additionally reporting whether the
// selector could be fully evaluated. SVGO wraps its querySelectorAll call in a
// try/catch and skips the rule when css-select throws; ok mirrors that signal.
func QuerySelectorAllErr(node svgast.Node, selector string, parents map[svgast.Node]svgast.Parent) (results []*svgast.Element, ok bool) {
	ctx := &matchCtx{parents: parents}
	walkElements(node, func(elem *svgast.Element) {
		if matchesSelectorList(elem, selector, ctx) {
			results = append(results, elem)
		}
	})
	if ctx.err {
		return nil, false
	}
	return results, true
}

// walkElements walks all element nodes under a node.
func walkElements(node svgast.Node, fn func(*svgast.Element)) {
	switch n := node.(type) {
	case *svgast.Root:
		for _, child := range n.Children {
			walkElements(child, fn)
		}
	case *svgast.Element:
		fn(n)
		for _, child := range n.Children {
			walkElements(child, fn)
		}
	}
}
