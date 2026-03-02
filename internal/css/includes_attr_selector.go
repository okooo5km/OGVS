// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import "strings"

// IncludesAttrSelector determines if any CSS rule's selector includes or
// traverses the given attribute name. Optionally checks for a specific value.
//
// Classes and IDs are generated as attribute selectors, so you can check for
// if a .class or #id is included by passing name="class" or name="id"
// respectively.
//
// This is ported from SVGO's includesAttrSelector in lib/style.js.
// In the SVGO codebase, this is called as:
//
//	includesAttrSelector(rule.selector, name)
//
// with traversed=false and value=null by default.
func IncludesAttrSelector(rules []StylesheetRule, name string, value *string) bool {
	for _, rule := range rules {
		if selectorIncludesAttr(rule.Selector, name, value) {
			return true
		}
	}
	return false
}

// selectorIncludesAttr checks if a single selector string contains an
// attribute selector that matches the given name and optional value.
//
// This parses the selector and looks for:
//   - [name] - attribute presence
//   - [name=value] - attribute value match
//   - [name~=value], [name|=value], etc. - other attribute selectors
//
// The traversed parameter from SVGO is not needed here since
// removeUnknownsAndDefaults always calls with traversed=false.
func selectorIncludesAttr(selector string, name string, value *string) bool {
	// Split comma-separated selectors
	selectors := splitSelectors(selector)
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		if subselectorIncludesAttr(sel, name, value) {
			return true
		}
	}
	return false
}

// subselectorIncludesAttr checks a single (non-comma-separated) selector
// for attribute selectors matching the given name.
func subselectorIncludesAttr(sel string, name string, value *string) bool {
	i := 0
	for i < len(sel) {
		ch := sel[i]

		switch ch {
		case '[':
			// Parse attribute selector
			i++ // skip [
			attrName, attrVal, hasVal := parseAttrSelector(sel, &i)
			if attrName == name {
				if value == nil {
					return true
				}
				if hasVal && attrVal == *value {
					return true
				}
			}

		case '\'', '"':
			// Skip quoted strings
			quote := ch
			i++
			for i < len(sel) && sel[i] != quote {
				if sel[i] == '\\' && i+1 < len(sel) {
					i++ // skip escaped char
				}
				i++
			}
			if i < len(sel) {
				i++ // skip closing quote
			}

		case '(':
			// Skip parenthesized content (but still recurse into it for :not() etc.)
			depth := 1
			i++
			start := i
			for i < len(sel) && depth > 0 {
				if sel[i] == '(' {
					depth++
				} else if sel[i] == ')' {
					depth--
				}
				if depth > 0 {
					i++
				}
			}
			// Check inner content for attribute selectors too
			inner := sel[start:i]
			if subselectorIncludesAttr(inner, name, value) {
				return true
			}
			if i < len(sel) {
				i++ // skip closing )
			}

		default:
			i++
		}
	}
	return false
}
