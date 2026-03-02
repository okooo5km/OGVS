// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removexmlns implements the removeXMLNS SVGO plugin.
// It removes the xmlns attribute from the root <svg> element.
// Useful for inline SVG where xmlns is inherited from parent HTML.
package removexmlns

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeXMLNS",
		Description: "removes xmlns attribute (for inline svg)",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "svg" {
					elem.Attributes.Delete("xmlns")
				}
				return nil
			},
		},
	}
}
