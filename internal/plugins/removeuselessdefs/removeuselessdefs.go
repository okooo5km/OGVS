// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeuselessdefs implements the removeUselessDefs SVGO plugin.
// It removes elements in <defs> without id.
package removeuselessdefs

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeUselessDefs",
		Description: "removes elements in <defs> without id",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if elem.Name == "defs" ||
					(collections.NonRenderingElems[elem.Name] && !elem.Attributes.Has("id")) {
					var usefulNodes []svgast.Node
					collectUsefulNodes(elem, &usefulNodes)
					if len(usefulNodes) == 0 {
						svgast.DetachNodeFromParent(node, parent)
					}
					elem.Children = usefulNodes
				}

				return nil
			},
		},
	}
}

// collectUsefulNodes recursively collects child elements that have an id or are style elements.
func collectUsefulNodes(node *svgast.Element, usefulNodes *[]svgast.Node) {
	for _, child := range node.Children {
		if childElem, ok := child.(*svgast.Element); ok {
			if childElem.Attributes.Has("id") || childElem.Name == "style" {
				*usefulNodes = append(*usefulNodes, child)
			} else {
				collectUsefulNodes(childElem, usefulNodes)
			}
		}
	}
}
