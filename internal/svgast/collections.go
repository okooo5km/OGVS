// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

// textElems is the set of elements that preserve whitespace in text content.
// Matches SVGO: new Set([...elemsGroups.textContent, 'pre', 'title'])
var textElems = map[string]bool{
	// elemsGroups.textContent
	"a":            true,
	"altGlyph":     true,
	"altGlyphDef":  true,
	"altGlyphItem": true,
	"glyph":        true,
	"glyphRef":     true,
	"text":         true,
	"textPath":     true,
	"tref":         true,
	"tspan":        true,
	// additional
	"pre":   true,
	"title": true,
}

// IsTextElem returns true if the given element name preserves whitespace.
func IsTextElem(name string) bool {
	return textElems[name]
}
