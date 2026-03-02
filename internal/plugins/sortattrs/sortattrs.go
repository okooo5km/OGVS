// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package sortattrs implements the sortAttrs SVGO plugin.
// It sorts element attributes for better compression.
package sortattrs

import (
	"sort"
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

var defaultOrder = []string{
	"id",
	"width", "height",
	"x", "x1", "x2",
	"y", "y1", "y2",
	"cx", "cy", "r",
	"fill", "stroke", "marker",
	"d", "points",
}

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "sortAttrs",
		Description: "Sort element attributes for better compression",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	order := defaultOrder
	if o, ok := params["order"]; ok {
		if arr, ok := o.([]any); ok {
			order = make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					order = append(order, s)
				}
			}
		}
	}

	xmlnsOrder := "front"
	if v, ok := params["xmlnsOrder"]; ok {
		if s, ok := v.(string); ok {
			xmlnsOrder = s
		}
	}

	// Build order index for O(1) lookup
	orderIndex := make(map[string]int, len(order))
	for i, name := range order {
		orderIndex[name] = i
	}

	getNsPriority := func(name string) int {
		if xmlnsOrder == "front" {
			if name == "xmlns" {
				return 3
			}
			if strings.HasPrefix(name, "xmlns:") {
				return 2
			}
		}
		if strings.Contains(name, ":") {
			return 1
		}
		return 0
	}

	compareAttrs := func(a, b svgast.AttrEntry) bool {
		aName := a.Name
		bName := b.Name

		// Sort by namespace priority
		aPriority := getNsPriority(aName)
		bPriority := getNsPriority(bName)
		if aPriority != bPriority {
			return aPriority > bPriority
		}

		// Extract first part (before '-')
		aPart := aName
		if idx := strings.Index(aName, "-"); idx >= 0 {
			aPart = aName[:idx]
		}
		bPart := bName
		if idx := strings.Index(bName, "-"); idx >= 0 {
			bPart = bName[:idx]
		}

		if aPart != bPart {
			_, aInOrder := orderIndex[aPart]
			_, bInOrder := orderIndex[bPart]

			if aInOrder && bInOrder {
				return orderIndex[aPart] < orderIndex[bPart]
			}
			if aInOrder != bInOrder {
				return aInOrder
			}
		}

		// Alphabetical sort
		return aName < bName
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				entries := elem.Attributes.Entries()
				sort.SliceStable(entries, func(i, j int) bool {
					return compareAttrs(entries[i], entries[j])
				})
				elem.Attributes.SetEntries(entries)
				return nil
			},
		},
	}
}
