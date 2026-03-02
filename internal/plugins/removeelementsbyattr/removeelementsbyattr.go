// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeelementsbyattr implements the removeElementsByAttr SVGO plugin.
// It removes arbitrary elements by ID or className.
package removeelementsbyattr

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeElementsByAttr",
		Description: "removes arbitrary elements by ID or className",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	ids := normalizeStringSlice(params["id"])
	classes := normalizeStringSlice(params["class"])

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Remove by ID
				if id, ok := elem.Attributes.Get("id"); ok && len(ids) > 0 {
					for _, target := range ids {
						if id == target {
							svgast.DetachNodeFromParent(node, parent)
							return nil
						}
					}
				}

				// Remove by class
				if classAttr, ok := elem.Attributes.Get("class"); ok && len(classes) > 0 {
					classList := strings.Split(classAttr, " ")
					for _, target := range classes {
						for _, cls := range classList {
							if cls == target {
								svgast.DetachNodeFromParent(node, parent)
								return nil
							}
						}
					}
				}

				return nil
			},
		},
	}
}

// normalizeStringSlice converts a param value (string or []any) to []string.
func normalizeStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		return []string{val}
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
