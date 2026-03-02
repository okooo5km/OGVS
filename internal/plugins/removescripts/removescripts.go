// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removescripts implements the removeScripts SVGO plugin.
// It removes scripts and event handler attributes.
package removescripts

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeScripts",
		Description: "removes scripts",
		Fn:          fn,
	})
}

// eventAttrs is the union of all event attribute sets.
var eventAttrs map[string]bool

func init() {
	eventAttrs = make(map[string]bool)
	for k := range collections.AnimationEventAttrs {
		eventAttrs[k] = true
	}
	for k := range collections.DocumentEventAttrs {
		eventAttrs[k] = true
	}
	for k := range collections.DocumentElementEventAttrs {
		eventAttrs[k] = true
	}
	for k := range collections.GlobalEventAttrs {
		eventAttrs[k] = true
	}
	for k := range collections.GraphicalEventAttrs {
		eventAttrs[k] = true
	}
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Remove <script> elements
				if elem.Name == "script" {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}

				// Remove event handler attributes
				for attr := range eventAttrs {
					if elem.Attributes.Has(attr) {
						elem.Attributes.Delete(attr)
					}
				}

				return nil
			},
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)
				if elem.Name != "a" {
					return
				}

				for _, entry := range elem.Attributes.Entries() {
					if entry.Name != "href" && !strings.HasSuffix(entry.Name, ":href") {
						continue
					}
					if !strings.HasPrefix(strings.TrimLeft(entry.Value, " \t\n\r"), "javascript:") {
						continue
					}

					// Replace <a> with its non-text children in parent
					parentElem, ok := parent.(*svgast.Element)
					if !ok {
						continue
					}

					var usefulChildren []svgast.Node
					for _, child := range elem.Children {
						if child.Type() != svgast.NodeText {
							usefulChildren = append(usefulChildren, child)
						}
					}

					// Splice: replace node with usefulChildren in parent
					newChildren := make([]svgast.Node, 0, len(parentElem.Children)-1+len(usefulChildren))
					for _, child := range parentElem.Children {
						if child == node {
							newChildren = append(newChildren, usefulChildren...)
						} else {
							newChildren = append(newChildren, child)
						}
					}
					parentElem.Children = newChildren
					return
				}
			},
		},
	}
}
