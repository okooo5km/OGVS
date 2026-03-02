// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removedesc implements the removeDesc SVGO plugin.
// It removes <desc> elements, with smart defaults that preserve
// accessibility descriptions but remove editor-generated ones.
package removedesc

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeDesc",
		Description: "removes <desc>",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	removeAny := false
	if ra, ok := params["removeAny"]; ok {
		if b, ok := ra.(bool); ok {
			removeAny = b
		}
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name != "desc" {
					return nil
				}

				if removeAny {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}

				// Remove empty desc
				if len(elem.Children) == 0 {
					svgast.DetachNodeFromParent(node, parent)
					return nil
				}

				// Remove desc with standard editor-generated text
				if len(elem.Children) > 0 {
					if text, ok := elem.Children[0].(*svgast.Text); ok {
						if strings.HasPrefix(text.Value, "Created with") ||
							strings.HasPrefix(text.Value, "Created using") {
							svgast.DetachNodeFromParent(node, parent)
						}
					}
				}

				return nil
			},
		},
	}
}
