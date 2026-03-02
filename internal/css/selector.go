// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/svgast"
)

// Matches checks if an element matches a CSS selector string.
// parents provides the parent map for traversal.
func Matches(node *svgast.Element, selector string, parents map[svgast.Node]svgast.Parent) bool {
	selectors := splitSelectors(selector)
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		if matchesCompound(node, sel, parents) {
			return true
		}
	}
	return false
}

// matchesCompound matches a single compound selector (no commas).
func matchesCompound(node *svgast.Element, selector string, parents map[svgast.Node]svgast.Parent) bool {
	// Parse the selector into parts split by combinators
	parts, combinators := parseSelectorParts(selector)
	if len(parts) == 0 {
		return false
	}

	// Match from right to left
	if !matchesSimple(node, parts[len(parts)-1]) {
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
			parent := parents[currentNode]
			for parent != nil {
				if elem, ok := parent.(*svgast.Element); ok {
					if matchesSimple(elem, part) {
						currentNode = elem
						found = true
						break
					}
				}
				parent = parents[parent]
			}
			if !found {
				return false
			}

		case '>': // child
			parent := parents[currentNode]
			if parent == nil {
				return false
			}
			elem, ok := parent.(*svgast.Element)
			if !ok || !matchesSimple(elem, part) {
				return false
			}
			currentNode = elem

		case '+': // adjacent sibling
			sibling := getPreviousSibling(currentNode, parents)
			if sibling == nil || !matchesSimple(sibling, part) {
				return false
			}
			currentNode = sibling

		case '~': // general sibling
			found := false
			sibling := getPreviousSibling(currentNode, parents)
			for sibling != nil {
				if matchesSimple(sibling, part) {
					currentNode = sibling
					found = true
					break
				}
				sibling = getPreviousSibling(sibling, parents)
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
func matchesSimple(elem *svgast.Element, selector string) bool {
	if selector == "*" {
		return true
	}

	// Parse simple selector into conditions
	conditions := parseSimpleSelector(selector)
	for _, cond := range conditions {
		if !matchCondition(elem, cond) {
			return false
		}
	}

	return len(conditions) > 0
}

type selectorCondition struct {
	condType string // "element", "class", "id", "attr", "attr-eq", "not"
	name     string
	value    string
	inner    []selectorCondition // for "not" type
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
			attr, val, hasVal := parseAttrSelector(selector, &i)
			if hasVal {
				conditions = append(conditions, selectorCondition{
					condType: "attr-eq", name: attr, value: val,
				})
			} else {
				conditions = append(conditions, selectorCondition{
					condType: "attr", name: attr,
				})
			}

		case ':': // Pseudo-class/element
			i++
			if i < len(selector) && selector[i] == ':' {
				i++ // pseudo-element — skip
				readIdent(selector, &i)
				continue
			}
			// Read pseudo-class name
			nameStart := i
			name := readIdent(selector, &i)

			if name == "not" && i < len(selector) && selector[i] == '(' {
				// :not() — parse inner selector as negation
				i++ // skip (
				innerStart := i
				depth := 1
				for i < len(selector) && depth > 0 {
					if selector[i] == '(' {
						depth++
					}
					if selector[i] == ')' {
						depth--
					}
					if depth > 0 {
						i++
					}
				}
				innerSel := selector[innerStart:i]
				if i < len(selector) {
					i++ // skip )
				}
				innerConds := parseSimpleSelector(innerSel)
				conditions = append(conditions, selectorCondition{
					condType: "not",
					inner:    innerConds,
				})
			} else {
				// Other pseudo-class — skip (already read name)
				_ = nameStart
				if i < len(selector) && selector[i] == '(' {
					depth := 1
					i++
					for i < len(selector) && depth > 0 {
						if selector[i] == '(' {
							depth++
						}
						if selector[i] == ')' {
							depth--
						}
						i++
					}
				}
			}

		case '*':
			i++
			// Universal selector, matches anything

		default: // Element type selector
			name := readIdent(selector, &i)
			if name != "" {
				conditions = append(conditions, selectorCondition{
					condType: "element", name: name,
				})
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

// parseAttrSelector parses [attr] or [attr=value] from position i.
func parseAttrSelector(s string, i *int) (attr, val string, hasVal bool) {
	// Read attribute name
	start := *i
	for *i < len(s) && s[*i] != '=' && s[*i] != ']' && s[*i] != '~' &&
		s[*i] != '|' && s[*i] != '^' && s[*i] != '$' && s[*i] != '*' {
		(*i)++
	}
	attr = strings.TrimSpace(s[start:*i])

	if *i < len(s) && s[*i] == '=' {
		hasVal = true
		(*i)++ // skip =
		val = readAttrValue(s, i)
	} else if *i < len(s) && (s[*i] == '~' || s[*i] == '|' || s[*i] == '^' ||
		s[*i] == '$' || s[*i] == '*') {
		(*i)++ // skip modifier
		if *i < len(s) && s[*i] == '=' {
			hasVal = true
			(*i)++ // skip =
			val = readAttrValue(s, i)
		}
	}

	// Skip to closing ]
	for *i < len(s) && s[*i] != ']' {
		(*i)++
	}
	if *i < len(s) {
		(*i)++ // skip ]
	}

	return
}

// readAttrValue reads a (possibly quoted) attribute value.
func readAttrValue(s string, i *int) string {
	// Skip whitespace
	for *i < len(s) && s[*i] == ' ' {
		(*i)++
	}
	if *i >= len(s) {
		return ""
	}

	if s[*i] == '\'' || s[*i] == '"' {
		quote := s[*i]
		(*i)++
		start := *i
		for *i < len(s) && s[*i] != quote {
			(*i)++
		}
		val := s[start:*i]
		if *i < len(s) {
			(*i)++ // skip closing quote
		}
		return val
	}

	start := *i
	for *i < len(s) && s[*i] != ']' && s[*i] != ' ' {
		(*i)++
	}
	return s[start:*i]
}

// matchCondition checks if an element matches a single condition.
func matchCondition(elem *svgast.Element, cond selectorCondition) bool {
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
		val, ok := elem.Attributes.Get(cond.name)
		return ok && val == cond.value
	case "not":
		// :not() — match if inner conditions do NOT all match
		for _, inner := range cond.inner {
			if !matchCondition(elem, inner) {
				return true
			}
		}
		return false
	}
	return false
}

// QuerySelectorAll finds all descendant elements matching a selector.
func QuerySelectorAll(node svgast.Node, selector string, parents map[svgast.Node]svgast.Parent) []*svgast.Element {
	var results []*svgast.Element
	walkElements(node, func(elem *svgast.Element) {
		if Matches(elem, selector, parents) {
			results = append(results, elem)
		}
	})
	return results
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
