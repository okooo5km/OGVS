// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removestyleelement implements the removeStyleElement SVGO plugin.
// It removes <style> elements.
package removestyleelement

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeStyleElement",
		Description: "removes <style> element",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "style" {
					svgast.DetachNodeFromParent(node, parent)
				}
				return nil
			},
		},
	}
}
