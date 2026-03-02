// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package collections provides SVG specification constants
// ported from SVGO's _collections.js.
//
// Based on https://www.w3.org/TR/SVG11/intro.html#Definitions
package collections

// StringSet is a set of strings implemented as a map.
type StringSet = map[string]bool

// Element groups based on SVG spec.

var AnimationElems = StringSet{
	"animate": true, "animateColor": true, "animateMotion": true,
	"animateTransform": true, "set": true,
}

var DescriptiveElems = StringSet{
	"desc": true, "metadata": true, "title": true,
}

var ShapeElems = StringSet{
	"circle": true, "ellipse": true, "line": true, "path": true,
	"polygon": true, "polyline": true, "rect": true,
}

var StructuralElems = StringSet{
	"defs": true, "g": true, "svg": true, "symbol": true, "use": true,
}

var PaintServerElems = StringSet{
	"hatch": true, "linearGradient": true, "meshGradient": true,
	"pattern": true, "radialGradient": true, "solidColor": true,
}

var NonRenderingElems = StringSet{
	"clipPath": true, "filter": true, "linearGradient": true,
	"marker": true, "mask": true, "pattern": true,
	"radialGradient": true, "solidColor": true, "symbol": true,
}

var ContainerElems = StringSet{
	"a": true, "defs": true, "foreignObject": true, "g": true,
	"marker": true, "mask": true, "missing-glyph": true, "pattern": true,
	"svg": true, "switch": true, "symbol": true,
}

var TextContentElems = StringSet{
	"a": true, "altGlyph": true, "altGlyphDef": true, "altGlyphItem": true,
	"glyph": true, "glyphRef": true, "text": true, "textPath": true,
	"tref": true, "tspan": true,
}

var TextContentChildElems = StringSet{
	"altGlyph": true, "textPath": true, "tref": true, "tspan": true,
}

var LightSourceElems = StringSet{
	"feDiffuseLighting": true, "feDistantLight": true, "fePointLight": true,
	"feSpecularLighting": true, "feSpotLight": true,
}

var FilterPrimitiveElems = StringSet{
	"feBlend": true, "feColorMatrix": true, "feComponentTransfer": true,
	"feComposite": true, "feConvolveMatrix": true, "feDiffuseLighting": true,
	"feDisplacementMap": true, "feDropShadow": true, "feFlood": true,
	"feFuncA": true, "feFuncB": true, "feFuncG": true, "feFuncR": true,
	"feGaussianBlur": true, "feImage": true, "feMerge": true,
	"feMergeNode": true, "feMorphology": true, "feOffset": true,
	"feSpecularLighting": true, "feTile": true, "feTurbulence": true,
}

// ElemsGroups maps group names to their element sets.
// Matches SVGO's elemsGroups export.
var ElemsGroups = map[string]StringSet{
	"animation":        AnimationElems,
	"descriptive":      DescriptiveElems,
	"shape":            ShapeElems,
	"structural":       StructuralElems,
	"paintServer":      PaintServerElems,
	"nonRendering":     NonRenderingElems,
	"container":        ContainerElems,
	"textContent":      TextContentElems,
	"textContentChild": TextContentChildElems,
	"lightSource":      LightSourceElems,
	"filterPrimitive":  FilterPrimitiveElems,
}

// TextElems are elements where whitespace may affect rendering.
// Includes all textContent elements plus 'pre' and 'title'.
var TextElems = StringSet{
	"a": true, "altGlyph": true, "altGlyphDef": true, "altGlyphItem": true,
	"glyph": true, "glyphRef": true, "text": true, "textPath": true,
	"tref": true, "tspan": true, "pre": true, "title": true,
}

// PathElems are elements that contain path data.
var PathElems = StringSet{
	"glyph": true, "missing-glyph": true, "path": true,
}
