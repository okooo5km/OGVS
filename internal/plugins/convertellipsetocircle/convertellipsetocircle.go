// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package convertellipsetocircle implements the convertEllipseToCircle SVGO plugin.
// It converts non-eccentric <ellipse>s to <circle>s.
package convertellipsetocircle

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertEllipseToCircle",
		Description: "converts non-eccentric <ellipse>s to <circle>s",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name != "ellipse" {
					return nil
				}

				rx, _ := elem.Attributes.Get("rx")
				if rx == "" {
					rx = "0"
				}
				ry, _ := elem.Attributes.Get("ry")
				if ry == "" {
					ry = "0"
				}

				if rx == ry || rx == "auto" || ry == "auto" {
					elem.Name = "circle"
					radius := rx
					if rx == "auto" {
						radius = ry
					}
					elem.Attributes.Delete("rx")
					elem.Attributes.Delete("ry")
					elem.Attributes.Set("r", radius)
				}

				return nil
			},
		},
	}
}
