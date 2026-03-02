// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removedimensions implements the removeDimensions SVGO plugin.
// It removes width/height attributes and adds viewBox if missing.
package removedimensions

import (
	"fmt"
	"strconv"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeDimensions",
		Description: "removes width and height in presence of viewBox (opposite to removeViewBox)",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name != "svg" {
					return nil
				}

				if elem.Attributes.Has("viewBox") {
					// viewBox exists: just remove width and height
					elem.Attributes.Delete("width")
					elem.Attributes.Delete("height")
				} else {
					// No viewBox: try to create one from width and height
					widthStr, hasW := elem.Attributes.Get("width")
					heightStr, hasH := elem.Attributes.Get("height")
					if hasW && hasH {
						w, errW := strconv.ParseFloat(widthStr, 64)
						h, errH := strconv.ParseFloat(heightStr, 64)
						if errW == nil && errH == nil {
							elem.Attributes.Set("viewBox", fmt.Sprintf("0 0 %s %s", formatNumber(w), formatNumber(h)))
							elem.Attributes.Delete("width")
							elem.Attributes.Delete("height")
						}
					}
				}
				return nil
			},
		},
	}
}

// formatNumber formats a float64 as an integer if it has no fractional part,
// otherwise as a decimal.
func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
