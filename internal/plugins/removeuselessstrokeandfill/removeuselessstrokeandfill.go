// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeuselessstrokeandfill implements the removeUselessStrokeAndFill SVGO plugin.
// It removes useless stroke and fill attributes.
package removeuselessstrokeandfill

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeUselessStrokeAndFill",
		Description: "removes useless stroke and fill attributes",
		Fn:          fn,
	})
}

func fn(root *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	removeStroke := true
	removeFill := true
	removeNone := false

	if v, ok := params["stroke"]; ok {
		if b, ok := v.(bool); ok {
			removeStroke = b
		}
	}
	if v, ok := params["fill"]; ok {
		if b, ok := v.(bool); ok {
			removeFill = b
		}
	}
	if v, ok := params["removeNone"]; ok {
		if b, ok := v.(bool); ok {
			removeNone = b
		}
	}

	// style and script elements deoptimize this plugin
	hasStyleOrScript := false
	svgast.Visit(root, &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "style" || tools.HasScripts(elem) {
					hasStyleOrScript = true
				}
				return nil
			},
		},
	}, nil)

	if hasStyleOrScript {
		return nil
	}

	stylesheet := css.CollectStylesheet(root)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parentNode svgast.Parent) error {
				elem := node.(*svgast.Element)

				// id attribute deoptimizes the whole subtree
				if elem.Attributes.Has("id") {
					return svgast.ErrVisitSkip
				}

				if !collections.ShapeElems[elem.Name] {
					return nil
				}

				computedStyle := css.ComputeStyle(stylesheet, elem)
				stroke := computedStyle["stroke"]
				strokeOpacity := computedStyle["stroke-opacity"]
				strokeWidth := computedStyle["stroke-width"]
				markerEnd := computedStyle["marker-end"]
				fill := computedStyle["fill"]
				fillOpacity := computedStyle["fill-opacity"]

				var parentStroke *css.ComputedStyle
				if parentElem, ok := parentNode.(*svgast.Element); ok {
					computedParentStyle := css.ComputeStyle(stylesheet, parentElem)
					parentStroke = computedParentStyle["stroke"]
				}

				// remove stroke*
				if removeStroke {
					if stroke == nil ||
						(stroke.Type == css.StyleStatic && stroke.Value == "none") ||
						(strokeOpacity != nil && strokeOpacity.Type == css.StyleStatic && strokeOpacity.Value == "0") ||
						(strokeWidth != nil && strokeWidth.Type == css.StyleStatic && strokeWidth.Value == "0") {

						// stroke-width may affect the size of marker-end
						// marker is not visible when stroke-width is 0
						if (strokeWidth != nil && strokeWidth.Type == css.StyleStatic && strokeWidth.Value == "0") ||
							markerEnd == nil {
							// Remove all stroke* attributes
							for _, entry := range elem.Attributes.Entries() {
								if strings.HasPrefix(entry.Name, "stroke") {
									elem.Attributes.Delete(entry.Name)
								}
							}
							// Set explicit none to not inherit from parent
							if parentStroke != nil &&
								parentStroke.Type == css.StyleStatic &&
								parentStroke.Value != "none" {
								elem.Attributes.Set("stroke", "none")
							}
						}
					}
				}

				// remove fill*
				if removeFill {
					if (fill != nil && fill.Type == css.StyleStatic && fill.Value == "none") ||
						(fillOpacity != nil && fillOpacity.Type == css.StyleStatic && fillOpacity.Value == "0") {
						// Remove all fill-* attributes (not fill itself)
						for _, entry := range elem.Attributes.Entries() {
							if strings.HasPrefix(entry.Name, "fill-") {
								elem.Attributes.Delete(entry.Name)
							}
						}
						if fill == nil || (fill.Type == css.StyleStatic && fill.Value != "none") {
							elem.Attributes.Set("fill", "none")
						}
					}
				}

				if removeNone {
					strokeAttrVal, _ := elem.Attributes.Get("stroke")
					fillAttrVal, _ := elem.Attributes.Get("fill")

					if (stroke == nil || strokeAttrVal == "none") &&
						((fill != nil && fill.Type == css.StyleStatic && fill.Value == "none") ||
							fillAttrVal == "none") {
						svgast.DetachNodeFromParent(node, parentNode)
					}
				}

				return nil
			},
		},
	}
}
