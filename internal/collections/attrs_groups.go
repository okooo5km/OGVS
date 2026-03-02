// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package collections

// Attribute groups based on SVG spec.

var AnimationAdditionAttrs = StringSet{
	"additive": true, "accumulate": true,
}

var AnimationAttributeTargetAttrs = StringSet{
	"attributeType": true, "attributeName": true,
}

var AnimationEventAttrs = StringSet{
	"onbegin": true, "onend": true, "onrepeat": true, "onload": true,
}

var AnimationTimingAttrs = StringSet{
	"begin": true, "dur": true, "end": true, "fill": true,
	"max": true, "min": true, "repeatCount": true, "repeatDur": true,
	"restart": true,
}

var AnimationValueAttrs = StringSet{
	"by": true, "calcMode": true, "from": true, "keySplines": true,
	"keyTimes": true, "to": true, "values": true,
}

var ConditionalProcessingAttrs = StringSet{
	"requiredExtensions": true, "requiredFeatures": true, "systemLanguage": true,
}

var CoreAttrs = StringSet{
	"id": true, "tabindex": true, "xml:base": true,
	"xml:lang": true, "xml:space": true,
}

var GraphicalEventAttrs = StringSet{
	"onactivate": true, "onclick": true, "onfocusin": true, "onfocusout": true,
	"onload": true, "onmousedown": true, "onmousemove": true, "onmouseout": true,
	"onmouseover": true, "onmouseup": true,
}

var PresentationAttrs = StringSet{
	"alignment-baseline": true, "baseline-shift": true, "clip-path": true,
	"clip-rule": true, "clip": true, "color-interpolation-filters": true,
	"color-interpolation": true, "color-profile": true, "color-rendering": true,
	"color": true, "cursor": true, "direction": true, "display": true,
	"dominant-baseline": true, "enable-background": true, "fill-opacity": true,
	"fill-rule": true, "fill": true, "filter": true, "flood-color": true,
	"flood-opacity": true, "font-family": true, "font-size-adjust": true,
	"font-size": true, "font-stretch": true, "font-style": true,
	"font-variant": true, "font-weight": true,
	"glyph-orientation-horizontal": true, "glyph-orientation-vertical": true,
	"image-rendering": true, "letter-spacing": true, "lighting-color": true,
	"marker-end": true, "marker-mid": true, "marker-start": true,
	"mask": true, "opacity": true, "overflow": true, "paint-order": true,
	"pointer-events": true, "shape-rendering": true, "stop-color": true,
	"stop-opacity": true, "stroke-dasharray": true, "stroke-dashoffset": true,
	"stroke-linecap": true, "stroke-linejoin": true, "stroke-miterlimit": true,
	"stroke-opacity": true, "stroke-width": true, "stroke": true,
	"text-anchor": true, "text-decoration": true, "text-overflow": true,
	"text-rendering": true, "transform-origin": true, "transform": true,
	"unicode-bidi": true, "vector-effect": true, "visibility": true,
	"word-spacing": true, "writing-mode": true,
}

var XlinkAttrs = StringSet{
	"xlink:actuate": true, "xlink:arcrole": true, "xlink:href": true,
	"xlink:role": true, "xlink:show": true, "xlink:title": true,
	"xlink:type": true,
}

var DocumentEventAttrs = StringSet{
	"onabort": true, "onerror": true, "onresize": true,
	"onscroll": true, "onunload": true, "onzoom": true,
}

var DocumentElementEventAttrs = StringSet{
	"oncopy": true, "oncut": true, "onpaste": true,
}

var GlobalEventAttrs = StringSet{
	"oncancel": true, "oncanplay": true, "oncanplaythrough": true,
	"onchange": true, "onclick": true, "onclose": true,
	"oncuechange": true, "ondblclick": true, "ondrag": true,
	"ondragend": true, "ondragenter": true, "ondragleave": true,
	"ondragover": true, "ondragstart": true, "ondrop": true,
	"ondurationchange": true, "onemptied": true, "onended": true,
	"onerror": true, "onfocus": true, "oninput": true,
	"oninvalid": true, "onkeydown": true, "onkeypress": true,
	"onkeyup": true, "onload": true, "onloadeddata": true,
	"onloadedmetadata": true, "onloadstart": true, "onmousedown": true,
	"onmouseenter": true, "onmouseleave": true, "onmousemove": true,
	"onmouseout": true, "onmouseover": true, "onmouseup": true,
	"onmousewheel": true, "onpause": true, "onplay": true,
	"onplaying": true, "onprogress": true, "onratechange": true,
	"onreset": true, "onresize": true, "onscroll": true,
	"onseeked": true, "onseeking": true, "onselect": true,
	"onshow": true, "onstalled": true, "onsubmit": true,
	"onsuspend": true, "ontimeupdate": true, "ontoggle": true,
	"onvolumechange": true, "onwaiting": true,
}

var FilterPrimitiveAttrs = StringSet{
	"x": true, "y": true, "width": true, "height": true, "result": true,
}

var TransferFunctionAttrs = StringSet{
	"amplitude": true, "exponent": true, "intercept": true,
	"offset": true, "slope": true, "tableValues": true, "type": true,
}

// AttrsGroups maps group names to their attribute sets.
// Matches SVGO's attrsGroups export.
var AttrsGroups = map[string]StringSet{
	"animationAddition":        AnimationAdditionAttrs,
	"animationAttributeTarget": AnimationAttributeTargetAttrs,
	"animationEvent":           AnimationEventAttrs,
	"animationTiming":          AnimationTimingAttrs,
	"animationValue":           AnimationValueAttrs,
	"conditionalProcessing":    ConditionalProcessingAttrs,
	"core":                     CoreAttrs,
	"graphicalEvent":           GraphicalEventAttrs,
	"presentation":             PresentationAttrs,
	"xlink":                    XlinkAttrs,
	"documentEvent":            DocumentEventAttrs,
	"documentElementEvent":     DocumentElementEventAttrs,
	"globalEvent":              GlobalEventAttrs,
	"filterPrimitive":          FilterPrimitiveAttrs,
	"transferFunction":         TransferFunctionAttrs,
}

// AttrsGroupsDefaults contains default values for attribute groups.
var AttrsGroupsDefaults = map[string]map[string]string{
	"core": {
		"xml:space": "default",
	},
	"presentation": {
		"clip": "auto", "clip-path": "none", "clip-rule": "nonzero",
		"mask": "none", "opacity": "1", "stop-color": "#000",
		"stop-opacity": "1", "fill-opacity": "1", "fill-rule": "nonzero",
		"fill": "#000", "stroke": "none", "stroke-width": "1",
		"stroke-linecap": "butt", "stroke-linejoin": "miter",
		"stroke-miterlimit": "4", "stroke-dasharray": "none",
		"stroke-dashoffset": "0", "stroke-opacity": "1",
		"paint-order": "normal", "vector-effect": "none",
		"display": "inline", "visibility": "visible",
		"marker-start": "none", "marker-mid": "none", "marker-end": "none",
		"color-interpolation": "sRGB", "color-interpolation-filters": "linearRGB",
		"color-rendering": "auto", "shape-rendering": "auto",
		"text-rendering": "auto", "image-rendering": "auto",
		"font-style": "normal", "font-variant": "normal",
		"font-weight": "normal", "font-stretch": "normal",
		"font-size": "medium", "font-size-adjust": "none",
		"kerning": "auto", "letter-spacing": "normal",
		"word-spacing": "normal", "text-decoration": "none",
		"text-anchor": "start", "text-overflow": "clip",
		"writing-mode": "lr-tb", "glyph-orientation-vertical": "auto",
		"glyph-orientation-horizontal": "0deg", "direction": "ltr",
		"unicode-bidi": "normal", "dominant-baseline": "auto",
		"alignment-baseline": "baseline", "baseline-shift": "baseline",
	},
	"transferFunction": {
		"slope": "1", "intercept": "0", "amplitude": "1",
		"exponent": "1", "offset": "0",
	},
}

// AttrsGroupsDeprecated contains deprecated attributes per group.
var AttrsGroupsDeprecated = map[string]StringSet{
	"animationAttributeTarget": {"attributeType": true},
	"conditionalProcessing":    {"requiredFeatures": true},
	"core":                     {"xml:base": true, "xml:lang": true, "xml:space": true},
	"presentation": {
		"clip": true, "color-profile": true, "enable-background": true,
		"glyph-orientation-horizontal": true, "glyph-orientation-vertical": true,
		"kerning": true,
	},
}
