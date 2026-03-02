// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeemptycontainers implements the removeEmptyContainers SVGO plugin.
// It removes empty container elements.
package removeemptycontainers

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeEmptyContainers",
		Description: "removes empty container elements",
		Fn:          fn,
	})
}

type useRef struct {
	node   svgast.Node
	parent svgast.Parent
}

func fn(root *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	stylesheet := css.CollectStylesheet(root)
	removedIds := make(map[string]bool)
	usesById := make(map[string][]useRef)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if elem.Name == "use" {
					for _, entry := range elem.Attributes.Entries() {
						ids := tools.FindReferences(entry.Name, entry.Value)
						for _, id := range ids {
							usesById[id] = append(usesById[id], useRef{node, parent})
						}
					}
				}

				return nil
			},
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)

				// Skip svg
				if elem.Name == "svg" {
					return
				}
				// Skip non-container elements
				if !collections.ContainerElems[elem.Name] {
					return
				}
				// Skip non-empty containers
				if len(elem.Children) != 0 {
					return
				}
				// Empty patterns may contain reusable configuration
				if elem.Name == "pattern" && len(elem.Attributes.Entries()) != 0 {
					return
				}
				// Empty <mask> hides masked element
				if elem.Name == "mask" && elem.Attributes.Has("id") {
					return
				}
				// Skip inside <switch>
				if parentElem, ok := parent.(*svgast.Element); ok && parentElem.Name == "switch" {
					return
				}
				// <g> with filter may create rendered content
				if elem.Name == "g" {
					if elem.Attributes.Has("filter") {
						return
					}
					computedStyle := css.ComputeStyle(stylesheet, elem)
					if computedStyle["filter"] != nil {
						return
					}
				}

				svgast.DetachNodeFromParent(node, parent)
				if id, ok := elem.Attributes.Get("id"); ok {
					removedIds[id] = true
				}
			},
		},
		Root: &svgast.VisitorCallbacks{
			Exit: func(_ svgast.Node, _ svgast.Parent) {
				// Remove <use> elements that referenced removed containers
				for id := range removedIds {
					if refs, ok := usesById[id]; ok {
						for _, ref := range refs {
							svgast.DetachNodeFromParent(ref.node, ref.parent)
						}
					}
				}
			},
		},
	}
}
