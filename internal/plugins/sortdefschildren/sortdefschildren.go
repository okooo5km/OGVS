// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package sortdefschildren implements the sortDefsChildren SVGO plugin.
// It sorts children of <defs> to improve compression.
package sortdefschildren

import (
	"sort"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "sortDefsChildren",
		Description: "Sorts children of <defs> to improve compression",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name != "defs" {
					return nil
				}

				// Count element name frequencies
				frequencies := make(map[string]int)
				for _, child := range elem.Children {
					if ce, ok := child.(*svgast.Element); ok {
						frequencies[ce.Name]++
					}
				}

				// Sort children: higher frequency first, longer name first, reverse alphabetical
				sort.SliceStable(elem.Children, func(i, j int) bool {
					a, aOk := elem.Children[i].(*svgast.Element)
					b, bOk := elem.Children[j].(*svgast.Element)
					if !aOk || !bOk {
						return false
					}

					aFreq := frequencies[a.Name]
					bFreq := frequencies[b.Name]
					if aFreq != bFreq {
						return aFreq > bFreq
					}

					if len(a.Name) != len(b.Name) {
						return len(a.Name) > len(b.Name)
					}

					if a.Name != b.Name {
						return a.Name > b.Name
					}

					return false
				})

				return nil
			},
		},
	}
}
