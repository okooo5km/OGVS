// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package collections

// DeprecatedAttrs holds safe and unsafe deprecated attribute sets.
// "safe" attributes can be removed without harm; "unsafe" may change behavior.
type DeprecatedAttrs struct {
	Safe   StringSet
	Unsafe StringSet
}

// ElemConfig describes an SVG element's attribute groups, specific attributes,
// defaults, deprecated attributes, and allowed content.
// Ported from SVGO's _collections.js `elems` object.
type ElemConfig struct {
	AttrsGroups   StringSet
	Attrs         StringSet
	Defaults      map[string]string
	Deprecated    *DeprecatedAttrs
	ContentGroups StringSet
	Content       StringSet
}

// AttrsGroupsDeprecatedFull contains deprecated attributes per attribute group,
// with safe/unsafe distinction. Used by removeDeprecatedAttrs plugin.
// Ported from SVGO's attrsGroupsDeprecated.
var AttrsGroupsDeprecatedFull = map[string]*DeprecatedAttrs{
	"animationAttributeTarget": {Unsafe: StringSet{"attributeType": true}},
	"conditionalProcessing":    {Unsafe: StringSet{"requiredFeatures": true}},
	"core":                     {Unsafe: StringSet{"xml:base": true, "xml:lang": true, "xml:space": true}},
	"presentation": {
		Unsafe: StringSet{
			"clip":                         true,
			"color-profile":                true,
			"enable-background":            true,
			"glyph-orientation-horizontal": true,
			"glyph-orientation-vertical":   true,
			"kerning":                      true,
		},
	},
}

// Elems maps SVG element names to their spec configuration.
// Ported from SVGO's _collections.js `elems` object.
// See https://www.w3.org/TR/SVG11/eltindex.html
var Elems = map[string]*ElemConfig{
	"a": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true,
			"core":                  true,
			"graphicalEvent":        true,
			"presentation":          true,
			"xlink":                 true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"style": true, "target": true, "transform": true,
		},
		Defaults: map[string]string{"target": "_self"},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true, "tspan": true,
		},
	},
	"altGlyph": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "dx": true, "dy": true, "externalResourcesRequired": true,
			"format": true, "glyphRef": true, "rotate": true, "style": true,
			"x": true, "y": true,
		},
	},
	"altGlyphDef": {
		AttrsGroups: StringSet{"core": true},
		Content:     StringSet{"glyphRef": true},
	},
	"altGlyphItem": {
		AttrsGroups: StringSet{"core": true},
		Content:     StringSet{"glyphRef": true, "altGlyphItem": true},
	},
	"animate": {
		AttrsGroups: StringSet{
			"animationAddition": true, "animationAttributeTarget": true,
			"animationEvent": true, "animationTiming": true,
			"animationValue": true, "conditionalProcessing": true,
			"core": true, "presentation": true, "xlink": true,
		},
		Attrs:         StringSet{"externalResourcesRequired": true},
		ContentGroups: StringSet{"descriptive": true},
	},
	"animateColor": {
		AttrsGroups: StringSet{
			"animationAddition": true, "animationAttributeTarget": true,
			"animationEvent": true, "animationTiming": true,
			"animationValue": true, "conditionalProcessing": true,
			"core": true, "presentation": true, "xlink": true,
		},
		Attrs:         StringSet{"externalResourcesRequired": true},
		ContentGroups: StringSet{"descriptive": true},
	},
	"animateMotion": {
		AttrsGroups: StringSet{
			"animationAddition": true, "animationEvent": true,
			"animationTiming": true, "animationValue": true,
			"conditionalProcessing": true, "core": true, "xlink": true,
		},
		Attrs: StringSet{
			"externalResourcesRequired": true, "keyPoints": true,
			"origin": true, "path": true, "rotate": true,
		},
		Defaults:      map[string]string{"rotate": "0"},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"mpath": true},
	},
	"animateTransform": {
		AttrsGroups: StringSet{
			"animationAddition": true, "animationAttributeTarget": true,
			"animationEvent": true, "animationTiming": true,
			"animationValue": true, "conditionalProcessing": true,
			"core": true, "xlink": true,
		},
		Attrs:         StringSet{"externalResourcesRequired": true, "type": true},
		ContentGroups: StringSet{"descriptive": true},
	},
	"circle": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "cx": true, "cy": true,
			"externalResourcesRequired": true, "r": true,
			"style": true, "transform": true,
		},
		Defaults:      map[string]string{"cx": "0", "cy": "0"},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"clipPath": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "clipPathUnits": true,
			"externalResourcesRequired": true, "style": true, "transform": true,
		},
		Defaults:      map[string]string{"clipPathUnits": "userSpaceOnUse"},
		ContentGroups: StringSet{"animation": true, "descriptive": true, "shape": true},
		Content:       StringSet{"text": true, "use": true},
	},
	"color-profile": {
		AttrsGroups:   StringSet{"core": true, "xlink": true},
		Attrs:         StringSet{"local": true, "name": true, "rendering-intent": true},
		Defaults:      map[string]string{"name": "sRGB", "rendering-intent": "auto"},
		Deprecated:    &DeprecatedAttrs{Unsafe: StringSet{"name": true}},
		ContentGroups: StringSet{"descriptive": true},
	},
	"cursor": {
		AttrsGroups: StringSet{
			"core": true, "conditionalProcessing": true, "xlink": true,
		},
		Attrs:         StringSet{"externalResourcesRequired": true, "x": true, "y": true},
		Defaults:      map[string]string{"x": "0", "y": "0"},
		ContentGroups: StringSet{"descriptive": true},
	},
	"defs": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"style": true, "transform": true,
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"desc": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"class": true, "style": true},
	},
	"ellipse": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "cx": true, "cy": true,
			"externalResourcesRequired": true, "rx": true, "ry": true,
			"style": true, "transform": true,
		},
		Defaults:      map[string]string{"cx": "0", "cy": "0"},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"feBlend": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true, "in2": true, "mode": true},
		Defaults:    map[string]string{"mode": "normal"},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feColorMatrix": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true, "type": true, "values": true},
		Defaults:    map[string]string{"type": "matrix"},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feComponentTransfer": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true},
		Content:     StringSet{"feFuncA": true, "feFuncB": true, "feFuncG": true, "feFuncR": true},
	},
	"feComposite": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs: StringSet{
			"class": true, "in": true, "in2": true, "k1": true,
			"k2": true, "k3": true, "k4": true, "operator": true, "style": true,
		},
		Defaults: map[string]string{
			"operator": "over", "k1": "0", "k2": "0", "k3": "0", "k4": "0",
		},
		Content: StringSet{"animate": true, "set": true},
	},
	"feConvolveMatrix": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs: StringSet{
			"class": true, "in": true, "kernelMatrix": true, "order": true,
			"style": true, "bias": true, "divisor": true, "edgeMode": true,
			"targetX": true, "targetY": true, "kernelUnitLength": true, "preserveAlpha": true,
		},
		Defaults: map[string]string{
			"order": "3", "bias": "0", "edgeMode": "duplicate", "preserveAlpha": "false",
		},
		Content: StringSet{"animate": true, "set": true},
	},
	"feDiffuseLighting": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs: StringSet{
			"class": true, "diffuseConstant": true, "in": true,
			"kernelUnitLength": true, "style": true, "surfaceScale": true,
		},
		Defaults:      map[string]string{"surfaceScale": "1", "diffuseConstant": "1"},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"feDistantLight": true, "fePointLight": true, "feSpotLight": true},
	},
	"feDisplacementMap": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs: StringSet{
			"class": true, "in": true, "in2": true, "scale": true,
			"style": true, "xChannelSelector": true, "yChannelSelector": true,
		},
		Defaults: map[string]string{
			"scale": "0", "xChannelSelector": "A", "yChannelSelector": "A",
		},
		Content: StringSet{"animate": true, "set": true},
	},
	"feDistantLight": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"azimuth": true, "elevation": true},
		Defaults:    map[string]string{"azimuth": "0", "elevation": "0"},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feFlood": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true},
		Content:     StringSet{"animate": true, "animateColor": true, "set": true},
	},
	"feFuncA": {
		AttrsGroups: StringSet{"core": true, "transferFunction": true},
		Content:     StringSet{"set": true, "animate": true},
	},
	"feFuncB": {
		AttrsGroups: StringSet{"core": true, "transferFunction": true},
		Content:     StringSet{"set": true, "animate": true},
	},
	"feFuncG": {
		AttrsGroups: StringSet{"core": true, "transferFunction": true},
		Content:     StringSet{"set": true, "animate": true},
	},
	"feFuncR": {
		AttrsGroups: StringSet{"core": true, "transferFunction": true},
		Content:     StringSet{"set": true, "animate": true},
	},
	"feGaussianBlur": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true, "stdDeviation": true},
		Defaults:    map[string]string{"stdDeviation": "0"},
		Content:     StringSet{"set": true, "animate": true},
	},
	"feImage": {
		AttrsGroups: StringSet{
			"core": true, "presentation": true, "filterPrimitive": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true, "href": true,
			"preserveAspectRatio": true, "style": true, "xlink:href": true,
		},
		Defaults: map[string]string{"preserveAspectRatio": "xMidYMid meet"},
		Content:  StringSet{"animate": true, "animateTransform": true, "set": true},
	},
	"feMerge": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true},
		Content:     StringSet{"feMergeNode": true},
	},
	"feMergeNode": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"in": true},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feMorphology": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true, "operator": true, "radius": true},
		Defaults:    map[string]string{"operator": "erode", "radius": "0"},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feOffset": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true, "dx": true, "dy": true},
		Defaults:    map[string]string{"dx": "0", "dy": "0"},
		Content:     StringSet{"animate": true, "set": true},
	},
	"fePointLight": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"x": true, "y": true, "z": true},
		Defaults:    map[string]string{"x": "0", "y": "0", "z": "0"},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feSpecularLighting": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs: StringSet{
			"class": true, "in": true, "kernelUnitLength": true,
			"specularConstant": true, "specularExponent": true,
			"style": true, "surfaceScale": true,
		},
		Defaults:      map[string]string{"surfaceScale": "1", "specularConstant": "1", "specularExponent": "1"},
		ContentGroups: StringSet{"descriptive": true, "lightSource": true},
	},
	"feSpotLight": {
		AttrsGroups: StringSet{"core": true},
		Attrs: StringSet{
			"limitingConeAngle": true, "pointsAtX": true, "pointsAtY": true,
			"pointsAtZ": true, "specularExponent": true, "x": true, "y": true, "z": true,
		},
		Defaults: map[string]string{
			"x": "0", "y": "0", "z": "0",
			"pointsAtX": "0", "pointsAtY": "0", "pointsAtZ": "0",
			"specularExponent": "1",
		},
		Content: StringSet{"animate": true, "set": true},
	},
	"feTile": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs:       StringSet{"class": true, "style": true, "in": true},
		Content:     StringSet{"animate": true, "set": true},
	},
	"feTurbulence": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "filterPrimitive": true},
		Attrs: StringSet{
			"baseFrequency": true, "class": true, "numOctaves": true,
			"seed": true, "stitchTiles": true, "style": true, "type": true,
		},
		Defaults: map[string]string{
			"baseFrequency": "0", "numOctaves": "1", "seed": "0",
			"stitchTiles": "noStitch", "type": "turbulence",
		},
		Content: StringSet{"animate": true, "set": true},
	},
	"filter": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "xlink": true},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true, "filterRes": true,
			"filterUnits": true, "height": true, "href": true, "primitiveUnits": true,
			"style": true, "width": true, "x": true, "xlink:href": true, "y": true,
		},
		Defaults: map[string]string{
			"primitiveUnits": "userSpaceOnUse",
			"x":              "-10%", "y": "-10%", "width": "120%", "height": "120%",
		},
		Deprecated:    &DeprecatedAttrs{Unsafe: StringSet{"filterRes": true}},
		ContentGroups: StringSet{"descriptive": true, "filterPrimitive": true},
		Content:       StringSet{"animate": true, "set": true},
	},
	"font": {
		AttrsGroups: StringSet{"core": true, "presentation": true},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"horiz-adv-x": true, "horiz-origin-x": true, "horiz-origin-y": true,
			"style": true, "vert-adv-y": true, "vert-origin-x": true, "vert-origin-y": true,
		},
		Defaults: map[string]string{
			"horiz-origin-x": "0", "horiz-origin-y": "0",
		},
		Deprecated: &DeprecatedAttrs{
			Unsafe: StringSet{
				"horiz-origin-x": true, "horiz-origin-y": true,
				"vert-adv-y": true, "vert-origin-x": true, "vert-origin-y": true,
			},
		},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"font-face": true, "glyph": true, "hkern": true, "missing-glyph": true, "vkern": true},
	},
	"font-face": {
		AttrsGroups: StringSet{"core": true},
		Attrs: StringSet{
			"font-family": true, "font-style": true, "font-variant": true,
			"font-weight": true, "font-stretch": true, "font-size": true,
			"unicode-range": true, "units-per-em": true, "panose-1": true,
			"stemv": true, "stemh": true, "slope": true, "cap-height": true,
			"x-height": true, "accent-height": true, "ascent": true, "descent": true,
			"widths": true, "bbox": true, "ideographic": true, "alphabetic": true,
			"mathematical": true, "hanging": true, "v-ideographic": true,
			"v-alphabetic": true, "v-mathematical": true, "v-hanging": true,
			"underline-position": true, "underline-thickness": true,
			"strikethrough-position": true, "strikethrough-thickness": true,
			"overline-position": true, "overline-thickness": true,
		},
		Defaults: map[string]string{
			"font-style": "all", "font-variant": "normal",
			"font-weight": "all", "font-stretch": "normal",
			"unicode-range": "U+0-10FFFF", "units-per-em": "1000",
			"panose-1": "0 0 0 0 0 0 0 0 0 0", "slope": "0",
		},
		Deprecated: &DeprecatedAttrs{
			Unsafe: StringSet{
				"accent-height": true, "alphabetic": true, "ascent": true,
				"bbox": true, "cap-height": true, "descent": true,
				"hanging": true, "ideographic": true, "mathematical": true,
				"panose-1": true, "slope": true, "stemh": true, "stemv": true,
				"unicode-range": true, "units-per-em": true,
				"v-alphabetic": true, "v-hanging": true,
				"v-ideographic": true, "v-mathematical": true,
				"widths": true, "x-height": true,
			},
		},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"font-face-src": true},
	},
	"font-face-format": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"string": true},
		Deprecated:  &DeprecatedAttrs{Unsafe: StringSet{"string": true}},
	},
	"font-face-name": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"name": true},
		Deprecated:  &DeprecatedAttrs{Unsafe: StringSet{"name": true}},
	},
	"font-face-src": {
		AttrsGroups: StringSet{"core": true},
		Content:     StringSet{"font-face-name": true, "font-face-uri": true},
	},
	"font-face-uri": {
		AttrsGroups: StringSet{"core": true, "xlink": true},
		Attrs:       StringSet{"href": true, "xlink:href": true},
		Content:     StringSet{"font-face-format": true},
	},
	"foreignObject": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"height": true, "style": true, "transform": true,
			"width": true, "x": true, "y": true,
		},
		Defaults: map[string]string{"x": "0", "y": "0"},
	},
	"g": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"style": true, "transform": true,
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"glyph": {
		AttrsGroups: StringSet{"core": true, "presentation": true},
		Attrs: StringSet{
			"arabic-form": true, "class": true, "d": true, "glyph-name": true,
			"horiz-adv-x": true, "lang": true, "orientation": true,
			"style": true, "unicode": true, "vert-adv-y": true,
			"vert-origin-x": true, "vert-origin-y": true,
		},
		Defaults: map[string]string{"arabic-form": "initial"},
		Deprecated: &DeprecatedAttrs{
			Unsafe: StringSet{
				"arabic-form": true, "glyph-name": true, "horiz-adv-x": true,
				"orientation": true, "unicode": true, "vert-adv-y": true,
				"vert-origin-x": true, "vert-origin-y": true,
			},
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"glyphRef": {
		AttrsGroups: StringSet{"core": true, "presentation": true},
		Attrs: StringSet{
			"class": true, "d": true, "horiz-adv-x": true, "style": true,
			"vert-adv-y": true, "vert-origin-x": true, "vert-origin-y": true,
		},
		Deprecated: &DeprecatedAttrs{
			Unsafe: StringSet{
				"horiz-adv-x": true, "vert-adv-y": true,
				"vert-origin-x": true, "vert-origin-y": true,
			},
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"hatch": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "xlink": true},
		Attrs: StringSet{
			"class": true, "hatchContentUnits": true, "hatchUnits": true,
			"pitch": true, "rotate": true, "style": true, "transform": true,
			"x": true, "y": true,
		},
		Defaults: map[string]string{
			"hatchUnits": "objectBoundingBox", "hatchContentUnits": "userSpaceOnUse",
			"x": "0", "y": "0", "pitch": "0", "rotate": "0",
		},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
		Content:       StringSet{"hatchPath": true},
	},
	"hatchPath": {
		AttrsGroups:   StringSet{"core": true, "presentation": true, "xlink": true},
		Attrs:         StringSet{"class": true, "style": true, "d": true, "offset": true},
		Defaults:      map[string]string{"offset": "0"},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"hkern": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"u1": true, "g1": true, "u2": true, "g2": true, "k": true},
		Deprecated:  &DeprecatedAttrs{Unsafe: StringSet{"g1": true, "g2": true, "k": true, "u1": true, "u2": true}},
	},
	"image": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true, "height": true,
			"href": true, "preserveAspectRatio": true, "style": true,
			"transform": true, "width": true, "x": true, "xlink:href": true, "y": true,
		},
		Defaults: map[string]string{
			"x": "0", "y": "0", "preserveAspectRatio": "xMidYMid meet",
		},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"line": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"style": true, "transform": true,
			"x1": true, "x2": true, "y1": true, "y2": true,
		},
		Defaults:      map[string]string{"x1": "0", "y1": "0", "x2": "0", "y2": "0"},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"linearGradient": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "xlink": true},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"gradientTransform": true, "gradientUnits": true, "href": true,
			"spreadMethod": true, "style": true,
			"x1": true, "x2": true, "xlink:href": true, "y1": true, "y2": true,
		},
		Defaults: map[string]string{
			"x1": "0", "y1": "0", "x2": "100%", "y2": "0", "spreadMethod": "pad",
		},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"animate": true, "animateTransform": true, "set": true, "stop": true},
	},
	"marker": {
		AttrsGroups: StringSet{"core": true, "presentation": true},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"markerHeight": true, "markerUnits": true, "markerWidth": true,
			"orient": true, "preserveAspectRatio": true,
			"refX": true, "refY": true, "style": true, "viewBox": true,
		},
		Defaults: map[string]string{
			"markerUnits": "strokeWidth", "refX": "0", "refY": "0",
			"markerWidth": "3", "markerHeight": "3",
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"mask": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true, "height": true,
			"mask-type": true, "maskContentUnits": true, "maskUnits": true,
			"style": true, "width": true, "x": true, "y": true,
		},
		Defaults: map[string]string{
			"maskUnits": "objectBoundingBox", "maskContentUnits": "userSpaceOnUse",
			"x": "-10%", "y": "-10%", "width": "120%", "height": "120%",
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"metadata": {
		AttrsGroups: StringSet{"core": true},
	},
	"missing-glyph": {
		AttrsGroups: StringSet{"core": true, "presentation": true},
		Attrs: StringSet{
			"class": true, "d": true, "horiz-adv-x": true, "style": true,
			"vert-adv-y": true, "vert-origin-x": true, "vert-origin-y": true,
		},
		Deprecated: &DeprecatedAttrs{
			Unsafe: StringSet{
				"horiz-adv-x": true, "vert-adv-y": true,
				"vert-origin-x": true, "vert-origin-y": true,
			},
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"mpath": {
		AttrsGroups:   StringSet{"core": true, "xlink": true},
		Attrs:         StringSet{"externalResourcesRequired": true, "href": true, "xlink:href": true},
		ContentGroups: StringSet{"descriptive": true},
	},
	"path": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "d": true, "externalResourcesRequired": true,
			"pathLength": true, "style": true, "transform": true,
		},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"pattern": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"presentation": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true, "height": true,
			"href": true, "patternContentUnits": true, "patternTransform": true,
			"patternUnits": true, "preserveAspectRatio": true, "style": true,
			"viewBox": true, "width": true, "x": true, "xlink:href": true, "y": true,
		},
		Defaults: map[string]string{
			"patternUnits": "objectBoundingBox", "patternContentUnits": "userSpaceOnUse",
			"x": "0", "y": "0", "width": "0", "height": "0",
			"preserveAspectRatio": "xMidYMid meet",
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"polygon": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"points": true, "style": true, "transform": true,
		},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"polyline": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"points": true, "style": true, "transform": true,
		},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"radialGradient": {
		AttrsGroups: StringSet{"core": true, "presentation": true, "xlink": true},
		Attrs: StringSet{
			"class": true, "cx": true, "cy": true,
			"externalResourcesRequired": true, "fr": true, "fx": true, "fy": true,
			"gradientTransform": true, "gradientUnits": true, "href": true,
			"r": true, "spreadMethod": true, "style": true, "xlink:href": true,
		},
		Defaults: map[string]string{
			"gradientUnits": "objectBoundingBox",
			"cx":            "50%", "cy": "50%", "r": "50%",
		},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"animate": true, "animateTransform": true, "set": true, "stop": true},
	},
	"meshGradient": {
		AttrsGroups:   StringSet{"core": true, "presentation": true, "xlink": true},
		Attrs:         StringSet{"class": true, "style": true, "x": true, "y": true, "gradientUnits": true, "transform": true},
		ContentGroups: StringSet{"descriptive": true, "paintServer": true, "animation": true},
		Content:       StringSet{"meshRow": true},
	},
	"meshRow": {
		AttrsGroups:   StringSet{"core": true, "presentation": true},
		Attrs:         StringSet{"class": true, "style": true},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"meshPatch": true},
	},
	"meshPatch": {
		AttrsGroups:   StringSet{"core": true, "presentation": true},
		Attrs:         StringSet{"class": true, "style": true},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"stop": true},
	},
	"rect": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"height": true, "rx": true, "ry": true,
			"style": true, "transform": true, "width": true, "x": true, "y": true,
		},
		Defaults:      map[string]string{"x": "0", "y": "0"},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"script": {
		AttrsGroups: StringSet{"core": true, "xlink": true},
		Attrs:       StringSet{"externalResourcesRequired": true, "type": true, "href": true, "xlink:href": true},
	},
	"set": {
		AttrsGroups: StringSet{
			"animation": true, "animationAttributeTarget": true,
			"animationTiming": true, "conditionalProcessing": true,
			"core": true, "xlink": true,
		},
		Attrs:         StringSet{"externalResourcesRequired": true, "to": true},
		ContentGroups: StringSet{"descriptive": true},
	},
	"solidColor": {
		AttrsGroups:   StringSet{"core": true, "presentation": true},
		Attrs:         StringSet{"class": true, "style": true},
		ContentGroups: StringSet{"paintServer": true},
	},
	"stop": {
		AttrsGroups: StringSet{"core": true, "presentation": true},
		Attrs:       StringSet{"class": true, "style": true, "offset": true, "path": true},
		Content:     StringSet{"animate": true, "animateColor": true, "set": true},
	},
	"style": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"type": true, "media": true, "title": true},
		Defaults:    map[string]string{"type": "text/css"},
	},
	"svg": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"documentEvent": true, "graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"baseProfile": true, "class": true, "contentScriptType": true,
			"contentStyleType": true, "height": true, "preserveAspectRatio": true,
			"style": true, "version": true, "viewBox": true, "width": true,
			"x": true, "y": true, "zoomAndPan": true,
		},
		Defaults: map[string]string{
			"x": "0", "y": "0", "width": "100%", "height": "100%",
			"preserveAspectRatio": "xMidYMid meet",
			"zoomAndPan":          "magnify", "version": "1.1", "baseProfile": "none",
			"contentScriptType": "application/ecmascript",
			"contentStyleType":  "text/css",
		},
		Deprecated: &DeprecatedAttrs{
			Safe:   StringSet{"version": true},
			Unsafe: StringSet{"baseProfile": true, "contentScriptType": true, "contentStyleType": true, "zoomAndPan": true},
		},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"switch": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"style": true, "transform": true,
		},
		ContentGroups: StringSet{"animation": true, "descriptive": true, "shape": true},
		Content: StringSet{
			"a": true, "foreignObject": true, "g": true, "image": true,
			"svg": true, "switch": true, "text": true, "use": true,
		},
	},
	"symbol": {
		AttrsGroups: StringSet{"core": true, "graphicalEvent": true, "presentation": true},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"preserveAspectRatio": true, "refX": true, "refY": true,
			"style": true, "viewBox": true,
		},
		Defaults: map[string]string{"refX": "0", "refY": "0"},
		ContentGroups: StringSet{
			"animation": true, "descriptive": true, "paintServer": true,
			"shape": true, "structural": true,
		},
		Content: StringSet{
			"a": true, "altGlyphDef": true, "clipPath": true, "color-profile": true,
			"cursor": true, "filter": true, "font-face": true, "font": true,
			"foreignObject": true, "image": true, "marker": true, "mask": true,
			"pattern": true, "script": true, "style": true, "switch": true,
			"text": true, "view": true,
		},
	},
	"text": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "dx": true, "dy": true,
			"externalResourcesRequired": true, "lengthAdjust": true,
			"rotate": true, "style": true, "textLength": true,
			"transform": true, "x": true, "y": true,
		},
		Defaults:      map[string]string{"x": "0", "y": "0", "lengthAdjust": "spacing"},
		ContentGroups: StringSet{"animation": true, "descriptive": true, "textContentChild": true},
		Content:       StringSet{"a": true},
	},
	"textPath": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "d": true, "externalResourcesRequired": true,
			"href": true, "method": true, "spacing": true,
			"startOffset": true, "style": true, "xlink:href": true,
		},
		Defaults:      map[string]string{"startOffset": "0", "method": "align", "spacing": "exact"},
		ContentGroups: StringSet{"descriptive": true},
		Content: StringSet{
			"a": true, "altGlyph": true, "animate": true, "animateColor": true,
			"set": true, "tref": true, "tspan": true,
		},
	},
	"title": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"class": true, "style": true},
	},
	"tref": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"href": true, "style": true, "xlink:href": true,
		},
		ContentGroups: StringSet{"descriptive": true},
		Content:       StringSet{"animate": true, "animateColor": true, "set": true},
	},
	"tspan": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true,
		},
		Attrs: StringSet{
			"class": true, "dx": true, "dy": true,
			"externalResourcesRequired": true, "lengthAdjust": true,
			"rotate": true, "style": true, "textLength": true, "x": true, "y": true,
		},
		ContentGroups: StringSet{"descriptive": true},
		Content: StringSet{
			"a": true, "altGlyph": true, "animate": true, "animateColor": true,
			"set": true, "tref": true, "tspan": true,
		},
	},
	"use": {
		AttrsGroups: StringSet{
			"conditionalProcessing": true, "core": true,
			"graphicalEvent": true, "presentation": true, "xlink": true,
		},
		Attrs: StringSet{
			"class": true, "externalResourcesRequired": true,
			"height": true, "href": true, "style": true, "transform": true,
			"width": true, "x": true, "xlink:href": true, "y": true,
		},
		Defaults:      map[string]string{"x": "0", "y": "0"},
		ContentGroups: StringSet{"animation": true, "descriptive": true},
	},
	"view": {
		AttrsGroups: StringSet{"core": true},
		Attrs: StringSet{
			"externalResourcesRequired": true, "preserveAspectRatio": true,
			"viewBox": true, "viewTarget": true, "zoomAndPan": true,
		},
		Deprecated:    &DeprecatedAttrs{Unsafe: StringSet{"viewTarget": true, "zoomAndPan": true}},
		ContentGroups: StringSet{"descriptive": true},
	},
	"vkern": {
		AttrsGroups: StringSet{"core": true},
		Attrs:       StringSet{"u1": true, "g1": true, "u2": true, "g2": true, "k": true},
		Deprecated:  &DeprecatedAttrs{Unsafe: StringSet{"g1": true, "g2": true, "k": true, "u1": true, "u2": true}},
	},
}
