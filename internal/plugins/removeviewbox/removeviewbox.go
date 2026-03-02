// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeviewbox implements the removeViewBox SVGO plugin.
// It removes viewBox attribute when it coincides with width/height.
package removeviewbox

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeViewBox",
		Description: "removes viewBox attribute when possible",
		Fn:          fn,
	})
}

var viewBoxElems = map[string]bool{
	"svg":     true,
	"pattern": true,
	"symbol":  true,
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if !viewBoxElems[elem.Name] {
					return nil
				}

				viewBox, hasViewBox := elem.Attributes.Get("viewBox")
				width, hasWidth := elem.Attributes.Get("width")
				height, hasHeight := elem.Attributes.Get("height")

				if !hasViewBox || !hasWidth || !hasHeight {
					return nil
				}

				// Skip nested <svg> (parent is not Root)
				if elem.Name == "svg" {
					if _, isRoot := parent.(*svgast.Root); !isRoot {
						return nil
					}
				}

				// Split viewBox by spaces and/or commas
				numbers := splitViewBox(viewBox)
				if len(numbers) != 4 {
					return nil
				}

				if numbers[0] == "0" &&
					numbers[1] == "0" &&
					strings.TrimSuffix(width, "px") == numbers[2] &&
					strings.TrimSuffix(height, "px") == numbers[3] {
					elem.Attributes.Delete("viewBox")
				}

				return nil
			},
		},
	}
}

// splitViewBox splits a viewBox string by whitespace and/or commas.
func splitViewBox(s string) []string {
	var result []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r'
	}) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
