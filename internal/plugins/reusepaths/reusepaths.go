// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package reusepaths implements the reusePaths SVGO plugin.
// It finds <path> elements with the same d, fill, and stroke, and converts
// them to <use> elements referencing a single <path> def.
package reusepaths

import (
	"fmt"

	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "reusePaths",
		Description: "finds <path> elements with the same d, fill, and stroke, and converts them to <use> elements referencing a single <path> def",
		Fn:          fn,
	})
}

// findElementsWithHref searches the tree for elements with matching href or xlink:href.
func findElementsWithHref(root svgast.Node, id string) []*svgast.Element {
	target := "#" + id
	var results []*svgast.Element
	var walk func(children []svgast.Node)
	walk = func(children []svgast.Node) {
		for _, child := range children {
			if elem, ok := child.(*svgast.Element); ok {
				for _, name := range []string{"href", "xlink:href"} {
					if v, ok := elem.Attributes.Get(name); ok && v == target {
						results = append(results, elem)
					}
				}
				walk(elem.Children)
			}
		}
	}
	switch n := root.(type) {
	case *svgast.Root:
		walk(n.Children)
	case *svgast.Element:
		walk(n.Children)
	}
	return results
}

func fn(root *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	stylesheet := css.CollectStylesheet(root)

	// Use ordered map pattern: a slice for insertion-order keys and a map for values.
	type pathGroup struct {
		key   string
		nodes []*svgast.Element
	}
	var pathGroups []pathGroup
	pathGroupIdx := make(map[string]int)

	var svgDefs *svgast.Element
	hrefs := make(map[string]bool)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if elem.Name == "path" {
					d, dOk := elem.Attributes.Get("d")
					if dOk {
						fill, _ := elem.Attributes.Get("fill")
						stroke, _ := elem.Attributes.Get("stroke")
						key := d + ";s:" + stroke + ";f:" + fill

						if idx, ok := pathGroupIdx[key]; ok {
							pathGroups[idx].nodes = append(pathGroups[idx].nodes, elem)
						} else {
							pathGroupIdx[key] = len(pathGroups)
							pathGroups = append(pathGroups, pathGroup{
								key:   key,
								nodes: []*svgast.Element{elem},
							})
						}
					}
				}

				if svgDefs == nil && elem.Name == "defs" {
					if parentElem, ok := parent.(*svgast.Element); ok && parentElem.Name == "svg" {
						svgDefs = elem
					}
				}

				if elem.Name == "use" {
					for _, name := range []string{"href", "xlink:href"} {
						if href, ok := elem.Attributes.Get(name); ok && len(href) > 1 && href[0] == '#' {
							hrefs[href[1:]] = true
						}
					}
				}

				return nil
			},
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)
				if elem.Name != "svg" {
					return
				}
				if parent == nil || parent.Type() != svgast.NodeRoot {
					return
				}

				defsTag := svgDefs
				if defsTag == nil {
					defsTag = &svgast.Element{
						Name:       "defs",
						Attributes: svgast.NewOrderedAttrs(),
					}
				}

				index := 0
				for _, group := range pathGroups {
					list := group.nodes
					if len(list) <= 1 {
						continue
					}

					// Create reusable path in defs
					reusablePath := &svgast.Element{
						Name:       "path",
						Attributes: svgast.NewOrderedAttrs(),
					}

					for _, attr := range []string{"fill", "stroke", "d"} {
						if v, ok := list[0].Attributes.Get(attr); ok {
							reusablePath.Attributes.Set(attr, v)
						}
					}

					// Determine the ID for the reusable path
					originalId, hasOriginalId := list[0].Attributes.Get("id")
					if !hasOriginalId || hrefs[originalId] || stylesheetHasSelector(stylesheet, "#"+originalId) {
						reusablePath.Attributes.Set("id", fmt.Sprintf("reuse-%d", index))
						index++
					} else {
						reusablePath.Attributes.Set("id", originalId)
						list[0].Attributes.Delete("id")
					}

					defsTag.Children = append(defsTag.Children, reusablePath)

					reusableId, _ := reusablePath.Attributes.Get("id")

					for _, pathNode := range list {
						pathNode.Attributes.Delete("d")
						pathNode.Attributes.Delete("stroke")
						pathNode.Attributes.Delete("fill")

						// Check if pathNode is in defs and has no children
						if isChildOf(pathNode, defsTag) && len(pathNode.Children) == 0 {
							if pathNode.Attributes.Len() == 0 {
								svgast.DetachNodeFromParent(pathNode, defsTag)
								continue
							}
							if pathNode.Attributes.Len() == 1 && pathNode.Attributes.Has("id") {
								pathNodeId, _ := pathNode.Attributes.Get("id")
								svgast.DetachNodeFromParent(pathNode, defsTag)
								// Update any references to the old id
								elements := findElementsWithHref(elem, pathNodeId)
								for _, child := range elements {
									for _, name := range []string{"href", "xlink:href"} {
										if _, ok := child.Attributes.Get(name); ok {
											child.Attributes.Set(name, "#"+reusableId)
										}
									}
								}
								continue
							}
						}

						// Convert path to use
						pathNode.Name = "use"
						pathNode.Attributes.Set("xlink:href", "#"+reusableId)
					}
				}

				if len(defsTag.Children) != 0 {
					if !elem.Attributes.Has("xmlns:xlink") {
						elem.Attributes.Set("xmlns:xlink", "http://www.w3.org/1999/xlink")
					}
					if svgDefs == nil {
						// Prepend defsTag as first child
						elem.Children = append([]svgast.Node{defsTag}, elem.Children...)
					}
				}
			},
		},
	}
}

// isChildOf checks if a node is a direct child of a parent element.
func isChildOf(node *svgast.Element, parent *svgast.Element) bool {
	for _, child := range parent.Children {
		if child == node {
			return true
		}
	}
	return false
}

// stylesheetHasSelector checks if any stylesheet rule matches the given selector.
func stylesheetHasSelector(stylesheet *css.Stylesheet, selector string) bool {
	for _, rule := range stylesheet.Rules {
		if rule.Selector == selector {
			return true
		}
	}
	return false
}
