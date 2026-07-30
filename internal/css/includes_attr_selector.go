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
		if selectorIncludesAttr(rule.Selector, name, value, false) {
			return true
		}
	}
	return false
}

// SelectorIncludesAttr reports whether a single selector string references the
// given attribute name, optionally with a specific value.
//
// When traversed is true only references that are followed by a combinator
// count, mirroring the csswhat.isTraversal check SVGO performs on the segment
// after the matched one.
func SelectorIncludesAttr(selector string, name string, value *string, traversed bool) bool {
	return selectorIncludesAttr(selector, name, value, traversed)
}

// selectorIncludesAttr checks if a selector string contains an attribute
// selector that matches the given name and optional value.
func selectorIncludesAttr(selector string, name string, value *string, traversed bool) bool {
	for _, sel := range splitSelectors(selector) {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		if subselectorIncludesAttr(sel, name, value, traversed) {
			return true
		}
	}
	return false
}

// subselectorIncludesAttr checks a single (non-comma-separated) selector
// for attribute selectors matching the given name.
func subselectorIncludesAttr(sel string, name string, value *string, traversed bool) bool {
	for _, ref := range selectorAttrRefs(sel) {
		if traversed && !ref.followedByTraversal {
			continue
		}
		if ref.name != name {
			continue
		}
		if value == nil {
			return true
		}
		if ref.hasValue && ref.value == *value {
			return true
		}
	}
	return false
}

// selectorAttrRef is an attribute reference made by one segment of a selector,
// in the shape css-what reports it: class and ID selectors are attribute
// selectors on `class` and `id`.
type selectorAttrRef struct {
	name  string
	value string
	// hasValue is false for a bare presence test, which never matches a value.
	hasValue bool
	// followedByTraversal reports whether the next segment is a combinator.
	followedByTraversal bool
}

// selectorAttrRefs collects the attribute references of a single complex
// selector, in source order.
func selectorAttrRefs(sel string) []selectorAttrRef {
	parts, _ := parseSelectorParts(sel)

	var refs []selectorAttrRef
	for pi, part := range parts {
		conditions := parseSimpleSelector(part)
		for ci, cond := range conditions {
			ref, ok := attrRefFromCondition(cond)
			if !ok {
				continue
			}
			// A combinator follows only the last simple selector of a
			// compound, and only when another compound comes after it.
			ref.followedByTraversal = pi < len(parts)-1 && ci == len(conditions)-1
			refs = append(refs, ref)
		}
	}
	return refs
}

// attrRefFromCondition maps a parsed selector condition onto the attribute
// reference css-what would report for it.
func attrRefFromCondition(cond selectorCondition) (selectorAttrRef, bool) {
	switch cond.condType {
	case "class":
		return selectorAttrRef{name: "class", value: cond.name, hasValue: true}, true
	case "id":
		return selectorAttrRef{name: "id", value: cond.name, hasValue: true}, true
	case "attr":
		return selectorAttrRef{name: cond.name}, true
	case "attr-eq":
		return selectorAttrRef{name: cond.name, value: cond.value, hasValue: true}, true
	}
	return selectorAttrRef{}, false
}

// SelectorClassNames returns the names of the class selectors that appear
// directly in a selector, in source order. It mirrors the ClassSelector
// children csstree reports for a Selector node, so names nested inside
// functional pseudo-classes such as :not() are excluded.
func SelectorClassNames(selector string) []string {
	parts, _ := parseSelectorParts(selector)

	var names []string
	for _, part := range parts {
		for _, cond := range parseSimpleSelector(part) {
			if cond.condType == "class" {
				names = append(names, cond.name)
			}
		}
	}
	return names
}

// SelectorLeadingID returns the name of a selector's first sub-selector when
// that sub-selector is an ID selector.
func SelectorLeadingID(selector string) (string, bool) {
	parts, _ := parseSelectorParts(selector)
	if len(parts) == 0 {
		return "", false
	}
	conditions := parseSimpleSelector(parts[0])
	if len(conditions) == 0 || conditions[0].condType != "id" {
		return "", false
	}
	return conditions[0].name, true
}
