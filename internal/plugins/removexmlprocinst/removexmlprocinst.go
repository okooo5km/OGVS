// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removexmlprocinst implements the removeXMLProcInst SVGO plugin.
// It removes XML processing instructions (<?xml ...?>).
package removexmlprocinst

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeXMLProcInst",
		Description: "removes XML processing instructions",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Instruction: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				inst := node.(*svgast.Instruction)
				if inst.Name == "xml" {
					svgast.DetachNodeFromParent(node, parent)
				}
				return nil
			},
		},
	}
}
