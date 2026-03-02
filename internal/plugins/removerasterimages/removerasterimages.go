// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removerasterimages implements the removeRasterImages SVGO plugin.
// It removes raster image references in <image>.
package removerasterimages

import (
	"regexp"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeRasterImages",
		Description: "removes raster images",
		Fn:          fn,
	})
}

var regRasterImage = regexp.MustCompile(`(\.|image\/)(jpe?g|png|gif)`)

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "image" {
					if href, ok := elem.Attributes.Get("xlink:href"); ok {
						if regRasterImage.MatchString(href) {
							svgast.DetachNodeFromParent(node, parent)
						}
					}
				}
				return nil
			},
		},
	}
}
