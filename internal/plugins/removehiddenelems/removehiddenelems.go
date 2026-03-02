// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removehiddenelems implements the removeHiddenElems SVGO plugin.
// It removes hidden elements with disabled rendering: display:none, opacity:0,
// zero-sized shapes, empty path data, etc.
package removehiddenelems

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	pathpkg "github.com/okooo5km/ogvs/internal/geom/path"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeHiddenElems",
		Description: "removes hidden elements (zero sized, with absent attributes)",
		Fn:          fn,
	})
}

type nodeParentPair struct {
	node   *svgast.Element
	parent svgast.Parent
}

type useRef struct {
	node       *svgast.Element
	parentNode svgast.Parent
}

// hasVisibleDescendant checks if any descendant element has
// visibility="visible" as an attribute. This mirrors SVGO's
// querySelector(node, '[visibility=visible]').
func hasVisibleDescendant(node *svgast.Element) bool {
	for _, child := range node.Children {
		childElem, ok := child.(*svgast.Element)
		if !ok {
			continue
		}
		if vis, has := childElem.Attributes.Get("visibility"); has && vis == "visible" {
			return true
		}
		if hasVisibleDescendant(childElem) {
			return true
		}
	}
	return false
}

func fn(root *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Parse parameters with defaults
	isHidden := getBoolParam(params, "isHidden", true)
	displayNone := getBoolParam(params, "displayNone", true)
	opacity0 := getBoolParam(params, "opacity0", true)
	circleR0 := getBoolParam(params, "circleR0", true)
	ellipseRX0 := getBoolParam(params, "ellipseRX0", true)
	ellipseRY0 := getBoolParam(params, "ellipseRY0", true)
	rectWidth0 := getBoolParam(params, "rectWidth0", true)
	rectHeight0 := getBoolParam(params, "rectHeight0", true)
	patternWidth0 := getBoolParam(params, "patternWidth0", true)
	patternHeight0 := getBoolParam(params, "patternHeight0", true)
	imageWidth0 := getBoolParam(params, "imageWidth0", true)
	imageHeight0 := getBoolParam(params, "imageHeight0", true)
	pathEmptyD := getBoolParam(params, "pathEmptyD", true)
	polylineEmptyPoints := getBoolParam(params, "polylineEmptyPoints", true)
	polygonEmptyPoints := getBoolParam(params, "polygonEmptyPoints", true)

	stylesheet := css.CollectStylesheet(root)

	// nonRenderedNodes: skip non-rendered nodes initially, and only detach if
	// they have no ID, or their ID is not referenced by another node.
	// Using ordered slice to preserve iteration order (matches JS Map insertion order).
	var nonRenderedNodes []nodeParentPair

	// removedDefIds: IDs for removed hidden definitions
	removedDefIds := make(map[string]bool)

	// allDefs: all <defs> elements found
	var allDefs []nodeParentPair

	// allReferences: set of all referenced IDs
	allReferences := make(map[string]bool)

	// referencesById: <use> references grouped by target ID
	referencesById := make(map[string][]useRef)

	deoptimized := false

	// canRemoveNonRenderingNode checks if a non-rendering node can be removed.
	// Nodes can't be removed if they or any of their children have an ID
	// attribute that is referenced.
	var canRemoveNonRenderingNode func(node *svgast.Element) bool
	canRemoveNonRenderingNode = func(node *svgast.Element) bool {
		if id, has := node.Attributes.Get("id"); has && allReferences[id] {
			return false
		}
		for _, child := range node.Children {
			if childElem, ok := child.(*svgast.Element); ok {
				if !canRemoveNonRenderingNode(childElem) {
					return false
				}
			}
		}
		return true
	}

	// removeElement detaches a node from parent and tracks removed def IDs.
	removeElement := func(node svgast.Node, parentNode svgast.Parent) {
		if elem, ok := node.(*svgast.Element); ok {
			if id, has := elem.Attributes.Get("id"); has {
				if parentElem, ok := parentNode.(*svgast.Element); ok && parentElem.Name == "defs" {
					removedDefIds[id] = true
				}
			}
		}
		svgast.DetachNodeFromParent(node, parentNode)
	}

	// --- First pass: pre-visit to handle nonRendering and opacity:0 ---
	// This mirrors the SVGO visit() call at the top level of fn().
	// Note: SVGO's visit() with visitSkip does NOT call exit and does NOT
	// visit children. So for nonRendering elements, we skip entirely.
	var firstPassVisit func(node svgast.Node, parentNode svgast.Parent)
	firstPassVisit = func(node svgast.Node, parentNode svgast.Parent) {
		switch n := node.(type) {
		case *svgast.Root:
			// Copy children to handle mutations during iteration
			children := make([]svgast.Node, len(n.Children))
			copy(children, n.Children)
			for _, child := range children {
				firstPassVisit(child, n)
			}
		case *svgast.Element:
			// transparent non-rendering elements still apply where referenced
			if collections.NonRenderingElems[n.Name] {
				nonRenderedNodes = append(nonRenderedNodes, nodeParentPair{n, parentNode})
				return // visitSkip: no children, no exit
			}

			computedStyle := css.ComputeStyle(stylesheet, n)
			// opacity="0"
			if opacity0 &&
				computedStyle["opacity"] != nil &&
				computedStyle["opacity"].Type == css.StyleStatic &&
				computedStyle["opacity"].Value == "0" {
				if n.Name == "path" {
					nonRenderedNodes = append(nonRenderedNodes, nodeParentPair{n, parentNode})
					return // visitSkip
				}
				removeElement(n, parentNode)
				// Note: no return here in SVGO — removed node's children are not visited
				// because the node is detached (parentNode.children.includes check fails)
			}

			// Visit children if still attached
			if parentNode != nil && nodeInChildren(node, parentNode.GetChildren()) {
				children := make([]svgast.Node, len(n.Children))
				copy(children, n.Children)
				for _, child := range children {
					firstPassVisit(child, n)
				}
			}
		}
	}
	firstPassVisit(root, nil)

	// --- Second pass: returned visitor ---
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Deoptimize if style or scripts are present
				if (elem.Name == "style" && len(elem.Children) != 0) ||
					tools.HasScripts(elem) {
					deoptimized = true
					return nil
				}

				// Track <defs> elements
				if elem.Name == "defs" {
					allDefs = append(allDefs, nodeParentPair{elem, parent})
				}

				// Track <use> references
				if elem.Name == "use" {
					for _, entry := range elem.Attributes.Entries() {
						if entry.Name != "href" && !strings.HasSuffix(entry.Name, ":href") {
							continue
						}
						value := entry.Value
						id := value[1:] // strip leading '#'

						referencesById[id] = append(referencesById[id], useRef{elem, parent})
					}
				}

				// Circle with zero radius
				if circleR0 &&
					elem.Name == "circle" &&
					len(elem.Children) == 0 {
					if r, has := elem.Attributes.Get("r"); has && r == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Ellipse with zero x-axis radius
				if ellipseRX0 &&
					elem.Name == "ellipse" &&
					len(elem.Children) == 0 {
					if rx, has := elem.Attributes.Get("rx"); has && rx == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Ellipse with zero y-axis radius
				if ellipseRY0 &&
					elem.Name == "ellipse" &&
					len(elem.Children) == 0 {
					if ry, has := elem.Attributes.Get("ry"); has && ry == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Rectangle with zero width
				if rectWidth0 &&
					elem.Name == "rect" &&
					len(elem.Children) == 0 {
					if w, has := elem.Attributes.Get("width"); has && w == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Rectangle with zero height
				// Note: SVGO checks both rectHeight0 AND rectWidth0
				if rectHeight0 && rectWidth0 &&
					elem.Name == "rect" &&
					len(elem.Children) == 0 {
					if h, has := elem.Attributes.Get("height"); has && h == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Pattern with zero width
				if patternWidth0 &&
					elem.Name == "pattern" {
					if w, has := elem.Attributes.Get("width"); has && w == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Pattern with zero height
				if patternHeight0 &&
					elem.Name == "pattern" {
					if h, has := elem.Attributes.Get("height"); has && h == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Image with zero width
				if imageWidth0 &&
					elem.Name == "image" {
					if w, has := elem.Attributes.Get("width"); has && w == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Image with zero height
				if imageHeight0 &&
					elem.Name == "image" {
					if h, has := elem.Attributes.Get("height"); has && h == "0" {
						removeElement(node, parent)
						return nil
					}
				}

				// Polyline with empty points (points attribute is null/undefined)
				if polylineEmptyPoints &&
					elem.Name == "polyline" &&
					!elem.Attributes.Has("points") {
					removeElement(node, parent)
					return nil
				}

				// Polygon with empty points (points attribute is null/undefined)
				if polygonEmptyPoints &&
					elem.Name == "polygon" &&
					!elem.Attributes.Has("points") {
					removeElement(node, parent)
					return nil
				}

				// Removes hidden elements (visibility:hidden)
				computedStyle := css.ComputeStyle(stylesheet, elem)
				if isHidden &&
					computedStyle["visibility"] != nil &&
					computedStyle["visibility"].Type == css.StyleStatic &&
					computedStyle["visibility"].Value == "hidden" &&
					!hasVisibleDescendant(elem) {
					removeElement(node, parent)
					return nil
				}

				// display="none"
				if displayNone &&
					computedStyle["display"] != nil &&
					computedStyle["display"].Type == css.StyleStatic &&
					computedStyle["display"].Value == "none" &&
					elem.Name != "marker" {
					removeElement(node, parent)
					return nil
				}

				// Path with empty data
				if pathEmptyD && elem.Name == "path" {
					d, hasD := elem.Attributes.Get("d")
					if !hasD {
						removeElement(node, parent)
						return nil
					}
					pathData := pathpkg.ParsePathData(d)
					if len(pathData) == 0 {
						removeElement(node, parent)
						return nil
					}
					// keep single point paths for markers
					if len(pathData) == 1 &&
						computedStyle["marker-start"] == nil &&
						computedStyle["marker-end"] == nil {
						removeElement(node, parent)
						return nil
					}
				}

				// Collect all references from this element's attributes
				for _, entry := range elem.Attributes.Entries() {
					ids := tools.FindReferences(entry.Name, entry.Value)
					for _, id := range ids {
						allReferences[id] = true
					}
				}

				return nil
			},
		},
		Root: &svgast.VisitorCallbacks{
			Exit: func(_ svgast.Node, _ svgast.Parent) {
				// Remove <use> elements that referenced removed defs
				for id := range removedDefIds {
					if refs, ok := referencesById[id]; ok {
						for _, ref := range refs {
							svgast.DetachNodeFromParent(ref.node, ref.parentNode)
						}
					}
				}

				// Remove unreferenced non-rendering nodes (unless deoptimized by styles)
				if !deoptimized {
					for _, pair := range nonRenderedNodes {
						if canRemoveNonRenderingNode(pair.node) {
							svgast.DetachNodeFromParent(pair.node, pair.parent)
						}
					}
				}

				// Remove empty <defs> elements
				for _, pair := range allDefs {
					if len(pair.node.Children) == 0 {
						svgast.DetachNodeFromParent(pair.node, pair.parent)
					}
				}
			},
		},
	}
}

// getBoolParam retrieves a boolean parameter with a default value.
func getBoolParam(params map[string]any, name string, defaultVal bool) bool {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[name].(bool); ok {
		return v
	}
	return defaultVal
}

// nodeInChildren checks if a node is still in the parent's children list.
func nodeInChildren(node svgast.Node, children []svgast.Node) bool {
	for _, child := range children {
		if child == node {
			return true
		}
	}
	return false
}
