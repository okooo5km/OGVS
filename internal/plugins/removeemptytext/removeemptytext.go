// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeemptytext implements the removeEmptyText SVGO plugin.
// It removes empty <text> elements.
package removeemptytext

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeEmptyText",
		Description: "removes empty <text> elements",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	text := true
	tspan := true
	tref := true

	if v, ok := params["text"].(bool); ok {
		text = v
	}
	if v, ok := params["tspan"].(bool); ok {
		tspan = v
	}
	if v, ok := params["tref"].(bool); ok {
		tref = v
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if text && elem.Name == "text" && len(elem.Children) == 0 {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}
				if tspan && elem.Name == "tspan" && len(elem.Children) == 0 {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}
				if tref && elem.Name == "tref" && !elem.Attributes.Has("xlink:href") {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}

				return nil
			},
		},
	}
}
