// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package convertonestopgradients implements the convertOneStopGradients
// SVGO plugin. It converts one-stop (single color) gradients to a plain color.
package convertonestopgradients

import (
	"fmt"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertOneStopGradients",
		Description: "converts one-stop (single color) gradients to a plain color",
		Fn:          fn,
	})
}

// querySelector finds the first element matching a CSS ID selector like "#id".
func querySelector(root *svgast.Root, selector string) *svgast.Element {
	if !strings.HasPrefix(selector, "#") {
		return nil
	}
	id := selector[1:]
	var found *svgast.Element
	var walk func(children []svgast.Node)
	walk = func(children []svgast.Node) {
		for _, child := range children {
			if found != nil {
				return
			}
			if elem, ok := child.(*svgast.Element); ok {
				if v, ok := elem.Attributes.Get("id"); ok && v == id {
					found = elem
					return
				}
				walk(elem.Children)
			}
		}
	}
	walk(root.Children)
	return found
}

// getStopChildren returns all direct child <stop> elements.
func getStopChildren(node *svgast.Element) []*svgast.Element {
	var stops []*svgast.Element
	for _, child := range node.Children {
		if elem, ok := child.(*svgast.Element); ok && elem.Name == "stop" {
			stops = append(stops, elem)
		}
	}
	return stops
}

// findElementsWithStyleContaining walks the tree and returns all elements
// whose "style" attribute contains the given substring.
func findElementsWithStyleContaining(root *svgast.Root, substr string) []*svgast.Element {
	var results []*svgast.Element
	var walk func(children []svgast.Node)
	walk = func(children []svgast.Node) {
		for _, child := range children {
			if elem, ok := child.(*svgast.Element); ok {
				if style, ok := elem.Attributes.Get("style"); ok && strings.Contains(style, substr) {
					results = append(results, elem)
				}
				walk(elem.Children)
			}
		}
	}
	walk(root.Children)
	return results
}

func fn(root *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	stylesheet := css.CollectStylesheet(root)

	type defsEntry struct {
		node   *svgast.Element
		parent svgast.Parent
	}

	type gradientEntry struct {
		node   *svgast.Element
		parent svgast.Parent
	}

	effectedDefs := make(map[*svgast.Element]bool)
	var allDefs []defsEntry
	allDefsSet := make(map[*svgast.Element]bool)
	var gradientsToDetach []gradientEntry
	gradientsToDetachSet := make(map[*svgast.Element]bool)
	xlinkHrefCount := 0

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if elem.Attributes.Has("xlink:href") {
					xlinkHrefCount++
				}

				if elem.Name == "defs" {
					if !allDefsSet[elem] {
						allDefs = append(allDefs, defsEntry{node: elem, parent: parent})
						allDefsSet[elem] = true
					}
					return nil
				}

				if elem.Name != "linearGradient" && elem.Name != "radialGradient" {
					return nil
				}

				stops := getStopChildren(elem)

				href, hrefOk := elem.Attributes.Get("xlink:href")
				if !hrefOk {
					href, hrefOk = elem.Attributes.Get("href")
				}

				var effectiveNode *svgast.Element
				if len(stops) == 0 && hrefOk && strings.HasPrefix(href, "#") {
					effectiveNode = querySelector(root, href)
				} else {
					effectiveNode = elem
				}

				if effectiveNode == nil {
					// Schedule gradient for removal
					if !gradientsToDetachSet[elem] {
						gradientsToDetach = append(gradientsToDetach, gradientEntry{node: elem, parent: parent})
						gradientsToDetachSet[elem] = true
					}
					return nil
				}

				effectiveStops := getStopChildren(effectiveNode)
				if len(effectiveStops) != 1 {
					return nil
				}

				// Mark the parent defs as effected
				if parentElem, ok := parent.(*svgast.Element); ok && parentElem.Name == "defs" {
					effectedDefs[parentElem] = true
				}

				// Schedule gradient for removal
				if !gradientsToDetachSet[elem] {
					gradientsToDetach = append(gradientsToDetach, gradientEntry{node: elem, parent: parent})
					gradientsToDetachSet[elem] = true
				}

				// Get color from the single stop
				var color string
				computedStyles := css.ComputeStyle(stylesheet, effectiveStops[0])
				if style, ok := computedStyles["stop-color"]; ok && style != nil && style.Type == css.StyleStatic {
					color = style.Value
				}

				// Build selector: [fill="url(#id)"], [stroke="url(#id)"], etc.
				id, _ := elem.Attributes.Get("id")
				selectorVal := fmt.Sprintf("url(#%s)", id)

				var selectorParts []string
				for prop := range collections.ColorsProps {
					selectorParts = append(selectorParts, fmt.Sprintf(`[%s="%s"]`, prop, selectorVal))
				}
				selector := strings.Join(selectorParts, ",")

				elements := css.QuerySelectorAll(root, selector, stylesheet.Parents)
				for _, el := range elements {
					for prop := range collections.ColorsProps {
						if v, ok := el.Attributes.Get(prop); ok && v == selectorVal {
							if color != "" {
								el.Attributes.Set(prop, color)
							} else {
								el.Attributes.Delete(prop)
							}
						}
					}
				}

				// Handle elements with url() in style attribute
				styledElements := findElementsWithStyleContaining(root, selectorVal)
				for _, el := range styledElements {
					if style, ok := el.Attributes.Get("style"); ok {
						replacement := color
						if replacement == "" {
							replacement = collections.AttrsGroupsDefaults["presentation"]["stop-color"]
						}
						el.Attributes.Set("style", strings.ReplaceAll(style, selectorVal, replacement))
					}
				}

				return nil
			},
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)
				if elem.Name != "svg" {
					return
				}

				// Detach gradients
				for _, g := range gradientsToDetach {
					if g.node.Attributes.Has("xlink:href") {
						xlinkHrefCount--
					}
					svgast.DetachNodeFromParent(g.node, g.parent)
				}

				// Remove xmlns:xlink if no more xlink:href references
				if xlinkHrefCount == 0 {
					elem.Attributes.Delete("xmlns:xlink")
				}

				// Remove empty effected defs
				for _, d := range allDefs {
					if effectedDefs[d.node] && len(d.node.Children) == 0 {
						svgast.DetachNodeFromParent(d.node, d.parent)
					}
				}
			},
		},
	}
}
