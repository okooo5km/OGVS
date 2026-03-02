// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeeditorsnsdata implements the removeEditorsNSData SVGO plugin.
// It removes editor-specific namespace declarations, elements, and attributes.
package removeeditorsnsdata

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

// editorNamespaces is the set of known SVG editor namespace URIs.
var editorNamespaces = map[string]bool{
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

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeEditorsNSData",
		Description: "removes editors namespaces, elements and attributes",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Build namespace set including additional namespaces from params
	namespaces := make(map[string]bool, len(editorNamespaces))
	for k, v := range editorNamespaces {
		namespaces[k] = v
	}
	if additional, ok := params["additionalNamespaces"]; ok {
		if arr, ok := additional.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					namespaces[s] = true
				}
			}
		}
	}

	var prefixes []string

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Collect namespace prefixes from svg element
				if elem.Name == "svg" {
					for _, entry := range elem.Attributes.Entries() {
						if strings.HasPrefix(entry.Name, "xmlns:") && namespaces[entry.Value] {
							prefixes = append(prefixes, entry.Name[len("xmlns:"):])
							elem.Attributes.Delete(entry.Name)
						}
					}
				}

				// Remove editor attributes (e.g. sodipodi:nodetypes)
				for _, entry := range elem.Attributes.Entries() {
					if strings.Contains(entry.Name, ":") {
						prefix := entry.Name[:strings.Index(entry.Name, ":")]
						if containsStr(prefixes, prefix) {
							elem.Attributes.Delete(entry.Name)
						}
					}
				}

				// Remove editor elements (e.g. <sodipodi:namedview>)
				if strings.Contains(elem.Name, ":") {
					prefix := elem.Name[:strings.Index(elem.Name, ":")]
					if containsStr(prefixes, prefix) {
						svgast.DetachNodeFromParent(node, parent)
					}
				}

				return nil
			},
		},
	}
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
