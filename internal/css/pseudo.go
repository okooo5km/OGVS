// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/svgast"
)

// treeStructuralPseudoClasses mirrors SVGO's pseudoClasses.treeStructural
// (plugins/_collections.js). They are positional and fully evaluatable at
// optimization time.
var treeStructuralPseudoClasses = map[string]bool{
	"empty":            true,
	"first-child":      true,
	"first-of-type":    true,
	"last-child":       true,
	"last-of-type":     true,
	"nth-child":        true,
	"nth-last-child":   true,
	"nth-last-of-type": true,
	"nth-of-type":      true,
	"only-child":       true,
	"only-of-type":     true,
	"root":             true,
}

// functionalPseudoClasses mirrors SVGO's pseudoClasses.functional.
var functionalPseudoClasses = map[string]bool{
	"is": true, "not": true, "where": true, "has": true,
}

// nthExpr is a parsed An+B micro-syntax expression.
type nthExpr struct {
	a, b int
}

// matches reports whether the 1-based position i satisfies a*n+b == i for some
// integer n >= 0. This is equivalent to nth-check's compile() output.
func (e nthExpr) matches(i int) bool {
	if e.a == 0 {
		return i == e.b
	}
	n := i - e.b
	if n%e.a != 0 {
		return false
	}
	return n/e.a >= 0
}

// matchesNone mirrors nth-check's boolbase.falseFunc shortcut.
func (e nthExpr) matchesNone() bool { return e.b-1 < 0 && e.a <= 0 }

// matchesAll mirrors nth-check's boolbase.trueFunc shortcut.
func (e nthExpr) matchesAll() bool { return e.a == 1 && e.b-1 < 0 }

// parseNth parses an An+B expression, mirroring nth-check's parse()
// (node_modules/nth-check/lib/parse.js). Reports false where nth-check throws.
func parseNth(formula string) (nthExpr, bool) {
	f := strings.ToLower(strings.TrimSpace(formula))
	switch f {
	case "even":
		return nthExpr{a: 2, b: 0}, true
	case "odd":
		return nthExpr{a: 2, b: 1}, true
	}

	idx := 0
	readSign := func() int {
		if idx < len(f) && f[idx] == '-' {
			idx++
			return -1
		}
		if idx < len(f) && f[idx] == '+' {
			idx++
		}
		return 1
	}
	readNumber := func() (int, bool) {
		start := idx
		value := 0
		for idx < len(f) && f[idx] >= '0' && f[idx] <= '9' {
			value = value*10 + int(f[idx]-'0')
			idx++
		}
		return value, idx != start
	}
	skipWS := func() {
		for idx < len(f) {
			switch f[idx] {
			case ' ', '\t', '\n', '\f', '\r':
				idx++
			default:
				return
			}
		}
	}

	a := 0
	sign := readSign()
	number, hasNumber := readNumber()

	if idx < len(f) && f[idx] == 'n' {
		idx++
		if hasNumber {
			a = sign * number
		} else {
			a = sign
		}
		skipWS()
		if idx < len(f) {
			sign = readSign()
			skipWS()
			number, hasNumber = readNumber()
		} else {
			sign, number, hasNumber = 0, 0, true
		}
	}

	if !hasNumber || idx < len(f) {
		return nthExpr{}, false
	}
	return nthExpr{a: a, b: sign * number}, true
}

// siblingNodes returns the raw child list of node's parent. css-select's
// positional pseudo-classes walk the raw list and skip non-elements inline, so
// text, comment and cdata nodes never occupy a position.
func siblingNodes(node svgast.Node, parents map[svgast.Node]svgast.Parent) []svgast.Node {
	parent := parents[node]
	if parent == nil {
		return nil
	}
	return parent.GetChildren()
}

// matchPseudo evaluates a tree-structural or functional pseudo-class against
// elem, mirroring css-select's pseudo-selectors.
func matchPseudo(elem *svgast.Element, cond selectorCondition, ctx *matchCtx) bool {
	self := svgast.Node(elem)

	switch cond.name {
	case "root":
		// css-select: parent == null || !isTag(parent).
		_, parentIsElem := ctx.parents[elem].(*svgast.Element)
		return !parentIsElem

	case "empty":
		if len(elem.Children) == 0 {
			return true
		}
		if _, ok := elem.Children[0].(*svgast.Element); !ok {
			// SVGO's css-select adapter dereferences node.children[0] inside
			// getText(), which throws for text/comment/cdata children and
			// aborts evaluation of the whole selector.
			ctx.err = true
		}
		return false

	case "first-child":
		for _, sib := range siblingNodes(elem, ctx.parents) {
			if _, ok := sib.(*svgast.Element); ok {
				return sib == self
			}
		}
		return false

	case "last-child":
		sibs := siblingNodes(elem, ctx.parents)
		for i := len(sibs) - 1; i >= 0; i-- {
			if sibs[i] == self {
				return true
			}
			if _, ok := sibs[i].(*svgast.Element); ok {
				return false
			}
		}
		return false

	case "first-of-type":
		for _, sib := range siblingNodes(elem, ctx.parents) {
			if sib == self {
				return true
			}
			if e, ok := sib.(*svgast.Element); ok && e.Name == elem.Name {
				return false
			}
		}
		return false

	case "last-of-type":
		sibs := siblingNodes(elem, ctx.parents)
		for i := len(sibs) - 1; i >= 0; i-- {
			if sibs[i] == self {
				return true
			}
			if e, ok := sibs[i].(*svgast.Element); ok && e.Name == elem.Name {
				return false
			}
		}
		return false

	case "only-child":
		for _, sib := range siblingNodes(elem, ctx.parents) {
			if sib == self {
				continue
			}
			if _, ok := sib.(*svgast.Element); ok {
				return false
			}
		}
		return true

	case "only-of-type":
		for _, sib := range siblingNodes(elem, ctx.parents) {
			if sib == self {
				continue
			}
			if e, ok := sib.(*svgast.Element); ok && e.Name == elem.Name {
				return false
			}
		}
		return true

	case "nth-child", "nth-last-child", "nth-of-type", "nth-last-of-type":
		if !cond.nthOK {
			// nth-check throws on unparseable formulas.
			ctx.err = true
			return false
		}
		if cond.nth.matchesNone() {
			return false
		}
		if cond.nth.matchesAll() {
			// css-select routes always-true formulas through getChildFunc,
			// which additionally requires the parent to be an element.
			_, parentIsElem := ctx.parents[elem].(*svgast.Element)
			return parentIsElem
		}
		sameType := cond.name == "nth-of-type" || cond.name == "nth-last-of-type"
		fromEnd := cond.name == "nth-last-child" || cond.name == "nth-last-of-type"
		sibs := siblingNodes(elem, ctx.parents)
		pos := 0
		if fromEnd {
			for i := len(sibs) - 1; i >= 0; i-- {
				if sibs[i] == self {
					break
				}
				if e, ok := sibs[i].(*svgast.Element); ok && (!sameType || e.Name == elem.Name) {
					pos++
				}
			}
		} else {
			for i := 0; i < len(sibs); i++ {
				if sibs[i] == self {
					break
				}
				if e, ok := sibs[i].(*svgast.Element); ok && (!sameType || e.Name == elem.Name) {
					pos++
				}
			}
		}
		return cond.nth.matches(pos + 1)

	case "not":
		return !matchesSelectorList(elem, cond.arg, ctx)

	case "is", "matches", "where":
		return matchesSelectorList(elem, cond.arg, ctx)

	case "has":
		return matchHas(elem, cond.arg, ctx)

	case "hover", "visited", "active":
		// css-select compiles these to boolbase.falseFunc because SVGO's
		// adapter provides no isHovered/isVisited/isActive hooks.
		return false
	}

	// Anything else makes css-select throw ("Unknown pseudo-class").
	ctx.err = true
	return false
}

// matchesSelectorList reports whether elem matches any selector in a
// comma-separated list, sharing the caller's match context.
func matchesSelectorList(elem *svgast.Element, list string, ctx *matchCtx) bool {
	matched := false
	any := false
	for _, sel := range splitSelectors(list) {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		any = true
		if matchesCompound(elem, sel, ctx) {
			matched = true
		}
	}
	if !any {
		// css-tree and css-what both reject an empty functional argument.
		ctx.err = true
	}
	return matched
}

// matchHas evaluates :has(). The argument is a relative selector list; a
// leading combinator anchors it at elem, otherwise it is a descendant query.
func matchHas(elem *svgast.Element, arg string, ctx *matchCtx) bool {
	for _, rel := range splitSelectors(arg) {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		lead := byte(' ')
		if rel[0] == '>' || rel[0] == '+' || rel[0] == '~' {
			lead = rel[0]
			rel = strings.TrimSpace(rel[1:])
		}
		if rel == "" {
			ctx.err = true
			return false
		}

		switch lead {
		case '>':
			for _, child := range elem.Children {
				if e, ok := child.(*svgast.Element); ok && matchesCompound(e, rel, ctx) {
					return true
				}
			}
		case '+', '~':
			after := false
			for _, sib := range siblingNodes(elem, ctx.parents) {
				if sib == svgast.Node(elem) {
					after = true
					continue
				}
				if !after {
					continue
				}
				e, ok := sib.(*svgast.Element)
				if !ok {
					continue
				}
				if matchesCompound(e, rel, ctx) {
					return true
				}
				if lead == '+' {
					break
				}
			}
		default:
			found := false
			for _, child := range elem.Children {
				walkElements(child, func(d *svgast.Element) {
					if !found && matchesCompound(d, rel, ctx) {
						found = true
					}
				})
				if found {
					return true
				}
			}
		}
	}
	return false
}
