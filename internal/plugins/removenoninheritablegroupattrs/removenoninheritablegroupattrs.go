// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removenoninheritablegroupattrs implements the removeNonInheritableGroupAttrs SVGO plugin.
// It removes non-inheritable group's presentational attributes.
package removenoninheritablegroupattrs

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeNonInheritableGroupAttrs",
		Description: "removes non-inheritable group's presentational attributes",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name != "g" {
					return nil
				}
				for _, entry := range elem.Attributes.Entries() {
					if collections.PresentationAttrs[entry.Name] &&
						!collections.InheritableAttrs[entry.Name] &&
						!collections.PresentationNonInheritableGroupAttrs[entry.Name] {
						elem.Attributes.Delete(entry.Name)
					}
				}
				return nil
			},
		},
	}
}
