// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeemptyattrs implements the removeEmptyAttrs SVGO plugin.
// It removes attributes with empty values, except conditional processing attributes.
package removeemptyattrs

import (
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

// conditionalProcessing attributes that should not be removed even if empty,
// because empty values prevent elements from rendering.
var conditionalProcessing = map[string]bool{
	"requiredExtensions": true,
	"requiredFeatures":   true,
	"systemLanguage":     true,
}

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeEmptyAttrs",
		Description: "removes empty attributes",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				for _, entry := range elem.Attributes.Entries() {
					if entry.Value == "" && !conditionalProcessing[entry.Name] {
						elem.Attributes.Delete(entry.Name)
					}
				}
				return nil
			},
		},
	}
}
