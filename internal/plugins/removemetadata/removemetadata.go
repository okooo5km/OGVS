// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removemetadata implements the removeMetadata SVGO plugin.
// It removes <metadata> elements from SVG files.
package removemetadata

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeMetadata",
		Description: "removes <metadata>",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "metadata" {
					svgast.DetachNodeFromParent(node, parent)
				}
				return nil
			},
		},
	}
}
