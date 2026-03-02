// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package moveelemsattrstogroup implements the moveElemsAttrsToGroup SVGO plugin.
// It moves common attributes of group children to the group element.
package moveelemsattrstogroup

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "moveElemsAttrsToGroup",
		Description: "Move common attributes of group children to the group",
		Fn:          fn,
	})
}

// orderedEntry is a key-value pair for maintaining insertion order.
type orderedEntry struct {
	name  string
	value string
}

func fn(root *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Find if any style element is present
	deoptimizedWithStyles := false
	svgast.Visit(root, &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				if elem.Name == "style" {
					deoptimizedWithStyles = true
				}
				return nil
			},
		},
	}, nil)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Exit: func(node svgast.Node, _ svgast.Parent) {
				elem := node.(*svgast.Element)

				// Process only groups with more than 1 child
				if elem.Name != "g" || len(elem.Children) <= 1 {
					return
				}

				// Deoptimize the plugin when style elements are present
				// selectors may rely on id, classes or tag names
				if deoptimizedWithStyles {
					return
				}

				// Find common attributes in group children.
				// Use ordered slice to preserve insertion order (matches JS Map behavior).
				var commonAttrs []orderedEntry
				commonIndex := make(map[string]int) // name -> index in commonAttrs
				initial := true
				everyChildIsPath := true

				for _, child := range elem.Children {
					if childElem, ok := child.(*svgast.Element); ok {
						if !collections.PathElems[childElem.Name] {
							everyChildIsPath = false
						}

						if initial {
							initial = false
							// Collect all inheritable attributes from first child element
							for _, entry := range childElem.Attributes.Entries() {
								if collections.InheritableAttrs[entry.Name] {
									commonIndex[entry.Name] = len(commonAttrs)
									commonAttrs = append(commonAttrs, orderedEntry{
										name: entry.Name, value: entry.Value,
									})
								}
							}
						} else {
							// Exclude uncommon attributes from initial list
							var filtered []orderedEntry
							newIndex := make(map[string]int)
							for _, ca := range commonAttrs {
								childVal, ok := childElem.Attributes.Get(ca.name)
								if ok && childVal == ca.value {
									newIndex[ca.name] = len(filtered)
									filtered = append(filtered, ca)
								}
							}
							commonAttrs = filtered
							commonIndex = newIndex
						}
					}
				}

				// Preserve transform on children when group has filter or clip-path or mask
				if elem.Attributes.Has("filter") ||
					elem.Attributes.Has("clip-path") ||
					elem.Attributes.Has("mask") {
					commonAttrs = removeByName(commonAttrs, "transform")
				}

				// Preserve transform when all children are paths
				// so the transform could be applied to path data by other plugins
				if everyChildIsPath {
					commonAttrs = removeByName(commonAttrs, "transform")
				}

				// Build a set from remaining common attributes for quick lookup
				commonSet := make(map[string]bool, len(commonAttrs))
				for _, ca := range commonAttrs {
					commonSet[ca.name] = true
				}

				// Add common children attributes to group
				for _, ca := range commonAttrs {
					if ca.name == "transform" {
						if existingTransform, ok := elem.Attributes.Get("transform"); ok {
							elem.Attributes.Set("transform", existingTransform+" "+ca.value)
						} else {
							elem.Attributes.Set("transform", ca.value)
						}
					} else {
						elem.Attributes.Set(ca.name, ca.value)
					}
				}

				// Delete common attributes from children
				for _, child := range elem.Children {
					if childElem, ok := child.(*svgast.Element); ok {
						for _, ca := range commonAttrs {
							childElem.Attributes.Delete(ca.name)
						}
					}
				}
			},
		},
	}
}

// removeByName removes an entry by name from the ordered slice.
func removeByName(entries []orderedEntry, name string) []orderedEntry {
	result := make([]orderedEntry, 0, len(entries))
	for _, e := range entries {
		if e.name != name {
			result = append(result, e)
		}
	}
	return result
}
