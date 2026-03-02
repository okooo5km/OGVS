// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package collections

// ReferencesProps are properties that can contain URL references.
var ReferencesProps = StringSet{
	"clip-path": true, "color-profile": true, "fill": true,
	"filter": true, "marker-end": true, "marker-mid": true,
	"marker-start": true, "mask": true, "stroke": true, "style": true,
}

// InheritableAttrs are CSS-inheritable presentation attributes.
var InheritableAttrs = StringSet{
	"clip-rule": true, "color-interpolation-filters": true,
	"color-interpolation": true, "color-profile": true,
	"color-rendering": true, "color": true, "cursor": true,
	"direction": true, "dominant-baseline": true, "fill-opacity": true,
	"fill-rule": true, "fill": true, "font-family": true,
	"font-size-adjust": true, "font-size": true, "font-stretch": true,
	"font-style": true, "font-variant": true, "font-weight": true,
	"font": true, "glyph-orientation-horizontal": true,
	"glyph-orientation-vertical": true, "image-rendering": true,
	"letter-spacing": true, "marker-end": true, "marker-mid": true,
	"marker-start": true, "marker": true, "paint-order": true,
	"pointer-events": true, "shape-rendering": true,
	"stroke-dasharray": true, "stroke-dashoffset": true,
	"stroke-linecap": true, "stroke-linejoin": true,
	"stroke-miterlimit": true, "stroke-opacity": true,
	"stroke-width": true, "stroke": true, "text-anchor": true,
	"text-rendering": true, "transform": true, "visibility": true,
	"word-spacing": true, "writing-mode": true,
}

// PresentationNonInheritableGroupAttrs are non-inheritable presentation
// attributes that apply to groups.
var PresentationNonInheritableGroupAttrs = StringSet{
	"clip-path": true, "display": true, "filter": true,
	"mask": true, "opacity": true, "text-decoration": true,
	"transform": true, "unicode-bidi": true,
}

// EditorNamespaces are namespace URIs used by SVG editors.
var EditorNamespaces = StringSet{
	"http://creativecommons.org/ns#":                         true,
	"http://inkscape.sourceforge.net/DTD/sodipodi-0.dtd":     true,
	"http://krita.org/namespaces/svg/krita":                  true,
	"http://ns.adobe.com/AdobeIllustrator/10.0/":             true,
	"http://ns.adobe.com/AdobeSVGViewerExtensions/3.0/":      true,
	"http://ns.adobe.com/Extensibility/1.0/":                 true,
	"http://ns.adobe.com/Flows/1.0/":                         true,
	"http://ns.adobe.com/GenericCustomNamespace/1.0/":        true,
	"http://ns.adobe.com/Graphs/1.0/":                        true,
	"http://ns.adobe.com/ImageReplacement/1.0/":              true,
	"http://ns.adobe.com/SaveForWeb/1.0/":                    true,
	"http://ns.adobe.com/Variables/1.0/":                     true,
	"http://ns.adobe.com/XPath/1.0/":                         true,
	"http://purl.org/dc/elements/1.1/":                       true,
	"http://schemas.microsoft.com/visio/2003/SVGExtensions/": true,
	"http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd":     true,
	"http://taptrix.com/vectorillustrator/svg_extensions":    true,
	"http://www.bohemiancoding.com/sketch/ns":                true,
	"http://www.figma.com/figma/ns":                          true,
	"http://www.inkscape.org/namespaces/inkscape":            true,
	"http://www.serif.com/":                                  true,
	"http://www.vector.evaxdesign.sk":                        true,
	"http://www.w3.org/1999/02/22-rdf-syntax-ns#":            true,
	"https://boxy-svg.com":                                   true,
}
