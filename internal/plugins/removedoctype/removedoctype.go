// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removedoctype implements the removeDoctype SVGO plugin.
// It removes DOCTYPE declarations from SVG files.
package removedoctype

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeDoctype",
		Description: "removes doctype declaration",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Doctype: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				svgast.DetachNodeFromParent(node, parent)
				return nil
			},
		},
	}
}
