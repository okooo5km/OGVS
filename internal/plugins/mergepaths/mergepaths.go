// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package mergepaths implements the mergePaths SVGO plugin.
// It merges multiple paths in one if possible.
// Ported from SVGO's plugins/mergePaths.js.
package mergepaths

import (
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/geom/path"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "mergePaths",
		Description: "merges multiple paths in one if possible",
		Fn:          fn,
	})
}

// elementHasUrl checks if a computed style property contains a url() reference.
func elementHasUrl(computedStyle css.ComputedStyles, attName string) bool {
	style := computedStyle[attName]
	if style != nil && style.Type == css.StyleStatic {
		return tools.IncludesURLReference(style.Value)
	}
	return false
}

func fn(root *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	force := false
	floatPrecision := 3
	noSpaceAfterFlags := false

	if v, ok := params["force"].(bool); ok {
		force = v
	}
	if v, ok := params["floatPrecision"].(float64); ok {
		floatPrecision = int(v)
	}
	if v, ok := params["noSpaceAfterFlags"].(bool); ok {
		noSpaceAfterFlags = v
	}

	stylesheet := css.CollectStylesheet(root)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if len(elem.Children) <= 1 {
					return nil
				}

				var elementsToRemove []svgast.Node
				var prevChild *svgast.Element
				var prevPathData []path.PathDataItem

				updatePreviousPath := func(child *svgast.Element, pathData []path.PathDataItem) {
					path.JS2Path(child, pathData, floatPrecision, noSpaceAfterFlags)
					prevPathData = nil
				}

				for i := 0; i < len(elem.Children); i++ {
					child := elem.Children[i]

					if i == 0 {
						if e, ok := child.(*svgast.Element); ok {
							prevChild = e
						} else {
							prevChild = nil
						}
						continue
					}

					// Check prevChild is a valid path element
					if prevChild == nil || prevChild.Name != "path" ||
						len(prevChild.Children) != 0 {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						if e, ok := child.(*svgast.Element); ok {
							prevChild = e
						} else {
							prevChild = nil
						}
						continue
					}
					_, hasPrevD := prevChild.Attributes.Get("d")
					if !hasPrevD {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						if e, ok := child.(*svgast.Element); ok {
							prevChild = e
						} else {
							prevChild = nil
						}
						continue
					}

					// Check current child is a valid path element
					childElem, isElem := child.(*svgast.Element)
					if !isElem || childElem.Name != "path" ||
						len(childElem.Children) != 0 {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						if isElem {
							prevChild = childElem
						} else {
							prevChild = nil
						}
						continue
					}
					_, hasChildD := childElem.Attributes.Get("d")
					if !hasChildD {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						prevChild = childElem
						continue
					}

					// Check computed style for markers, clip-path, mask, url references
					computedStyle := css.ComputeStyle(stylesheet, childElem)
					if computedStyle["marker-start"] != nil ||
						computedStyle["marker-mid"] != nil ||
						computedStyle["marker-end"] != nil ||
						computedStyle["clip-path"] != nil ||
						computedStyle["mask"] != nil ||
						computedStyle["mask-image"] != nil {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						prevChild = childElem
						continue
					}

					urlAttrs := []string{"fill", "filter", "stroke"}
					hasUrlRef := false
					for _, attName := range urlAttrs {
						if elementHasUrl(computedStyle, attName) {
							hasUrlRef = true
							break
						}
					}
					if hasUrlRef {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						prevChild = childElem
						continue
					}

					// Check attributes match
					childAttrs := childElem.Attributes.Entries()
					prevAttrs := prevChild.Attributes.Entries()
					if len(childAttrs) != len(prevAttrs) {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						prevChild = childElem
						continue
					}

					attrsNotEqual := false
					for _, entry := range childAttrs {
						if entry.Name != "d" {
							prevVal, hasPrev := prevChild.Attributes.Get(entry.Name)
							if !hasPrev || prevVal != entry.Value {
								attrsNotEqual = true
								break
							}
						}
					}

					if attrsNotEqual {
						if prevPathData != nil && prevChild != nil {
							updatePreviousPath(prevChild, prevPathData)
						}
						prevChild = childElem
						continue
					}

					hasPrevPath := prevPathData != nil
					currentPathData := path.Path2JS(childElem)
					if prevPathData == nil {
						prevPathData = path.Path2JS(prevChild)
					}

					if force || !path.Intersects(prevPathData, currentPathData) {
						prevPathData = append(prevPathData, currentPathData...)
						elementsToRemove = append(elementsToRemove, child)
						continue
					}

					if hasPrevPath {
						updatePreviousPath(prevChild, prevPathData)
					}

					prevChild = childElem
					prevPathData = nil
				}

				if prevPathData != nil && prevChild != nil {
					updatePreviousPath(prevChild, prevPathData)
				}

				// Remove merged elements
				if len(elementsToRemove) > 0 {
					removeSet := make(map[svgast.Node]bool)
					for _, r := range elementsToRemove {
						removeSet[r] = true
					}
					filtered := make([]svgast.Node, 0, len(elem.Children)-len(elementsToRemove))
					for _, c := range elem.Children {
						if !removeSet[c] {
							filtered = append(filtered, c)
						}
					}
					elem.Children = filtered
				}

				return nil
			},
		},
	}
}
