// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package css provides CSS parsing, cascading, specificity, and
// selector matching for SVG optimization,
// ported from SVGO's lib/style.js.
package css

import "github.com/okooo5km/ogvs/internal/svgast"

// Specificity represents CSS selector specificity as [a, b, c, d].
// a = inline style, b = IDs, c = classes/attributes, d = elements.
type Specificity [4]int

// StylesheetDeclaration represents a single CSS property declaration.
type StylesheetDeclaration struct {
	Name      string
	Value     string
	Important bool
}

// StylesheetRule represents a CSS rule with its selector, specificity, and declarations.
type StylesheetRule struct {
	Dynamic          bool
	Selector         string // cleaned selector (pseudo-classes stripped for matching)
	OriginalSelector string // original selector (with pseudo-classes, for CSS output)
	Specificity      Specificity
	Declarations     []StylesheetDeclaration
	MediaQuery       string // at-rule media query context (e.g., "media screen"), empty for top-level
}

// Stylesheet represents collected CSS rules and parent map.
type Stylesheet struct {
	Rules   []StylesheetRule
	Parents map[svgast.Node]svgast.Parent
}

// ComputedStyleType indicates whether a computed style is static or dynamic.
type ComputedStyleType int

const (
	// StyleStatic is a resolved static CSS value.
	StyleStatic ComputedStyleType = iota
	// StyleDynamic is a value that depends on pseudo-classes or media queries.
	StyleDynamic
)

// ComputedStyle represents a computed style property for an element.
type ComputedStyle struct {
	Type      ComputedStyleType
	Inherited bool
	Value     string // only meaningful when Type == StyleStatic
}

// ComputedStyles maps property names to their computed styles.
type ComputedStyles map[string]*ComputedStyle
