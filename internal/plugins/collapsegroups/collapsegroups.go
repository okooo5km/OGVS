// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package collapsegroups implements the collapseGroups SVGO plugin.
// It collapses useless groups by either moving attributes from a group
// to its single child or unwrapping children when the group has no attributes.
package collapsegroups

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "collapseGroups",
		Description: "collapses useless groups",
		Fn:          fn,
	})
}

// hasAnimatedAttr checks if any descendant animation element targets the
// given attribute name. This recursively walks the subtree.
func hasAnimatedAttr(node svgast.Node, name string) bool {
	elem, ok := node.(*svgast.Element)
	if !ok {
		return false
	}
	if collections.AnimationElems[elem.Name] {
		if attrName, has := elem.Attributes.Get("attributeName"); has && attrName == name {
			return true
		}
	}
	for _, child := range elem.Children {
		if hasAnimatedAttr(child, name) {
			return true
		}
	}
	return false
}

func fn(root *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	stylesheet := css.CollectStylesheet(root)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Exit: func(node svgast.Node, parent svgast.Parent) {
				elem := node.(*svgast.Element)

				// Skip if parent is root or switch
				if _, isRoot := parent.(*svgast.Root); isRoot {
					return
				}
				if parentElem, ok := parent.(*svgast.Element); ok && parentElem.Name == "switch" {
					return
				}

				// non-empty groups only
				if elem.Name != "g" || len(elem.Children) == 0 {
					return
				}

				// Move group attributes to the single child element
				if elem.Attributes.Len() != 0 && len(elem.Children) == 1 {
					firstChild, isElem := elem.Children[0].(*svgast.Element)
					if !isElem {
						goto collapseEmpty
					}

					// Check filter on group (attribute or computed style)
					nodeHasFilter := false
					if elem.Attributes.Has("filter") {
						nodeHasFilter = true
					} else {
						computedStyle := css.ComputeStyle(stylesheet, elem)
						if computedStyle["filter"] != nil {
							nodeHasFilter = true
						}
					}

					if firstChild.Attributes.Has("id") {
						goto collapseEmpty
					}
					if nodeHasFilter {
						goto collapseEmpty
					}

					// class conflict: both group and child have class
					groupHasClass := elem.Attributes.Has("class")
					childHasClass := firstChild.Attributes.Has("class")
					if groupHasClass && childHasClass {
						goto collapseEmpty
					}

					// clip-path/mask constraint:
					// if group has clip-path or mask, child must be <g>
					// and neither can have transform
					groupHasClipPath := elem.Attributes.Has("clip-path")
					groupHasMask := elem.Attributes.Has("mask")
					if groupHasClipPath || groupHasMask {
						if firstChild.Name != "g" {
							goto collapseEmpty
						}
						if elem.Attributes.Has("transform") || firstChild.Attributes.Has("transform") {
							goto collapseEmpty
						}
					}

					// Build new child attributes: copy child attrs first, then merge group attrs
					newChildAttrs := firstChild.Attributes.Clone()

					for _, entry := range elem.Attributes.Entries() {
						attrName := entry.Name
						attrValue := entry.Value

						// avoid copying to not conflict with animated attribute
						if hasAnimatedAttr(firstChild, attrName) {
							return
						}

						childVal, childHas := newChildAttrs.Get(attrName)
						if !childHas {
							// Child doesn't have this attr, copy from group
							newChildAttrs.Set(attrName, attrValue)
						} else if attrName == "transform" {
							// Concatenate transforms: group transform + child transform
							newChildAttrs.Set(attrName, attrValue+" "+childVal)
						} else if childVal == "inherit" {
							// Replace inherit with group value
							newChildAttrs.Set(attrName, attrValue)
						} else if !collections.InheritableAttrs[attrName] && childVal != attrValue {
							// Non-inheritable attr with different value: can't merge
							return
						}
						// For inheritable attrs with child having its own value,
						// child's value takes precedence (no action needed)
					}

					// Clear group attributes and apply merged attrs to child
					elem.Attributes.SetEntries(nil)
					firstChild.Attributes = newChildAttrs
				}

			collapseEmpty:
				// Collapse groups without attributes
				if elem.Attributes.Len() == 0 {
					// animation elements "add" attributes to group
					// group should be preserved
					for _, child := range elem.Children {
						if childElem, ok := child.(*svgast.Element); ok {
							if collections.AnimationElems[childElem.Name] {
								return
							}
						}
					}
					// Replace current node with all its children
					parentChildren := parent.GetChildren()
					var newChildren []svgast.Node
					for _, child := range parentChildren {
						if child == node {
							newChildren = append(newChildren, elem.Children...)
						} else {
							newChildren = append(newChildren, child)
						}
					}
					parent.SetChildren(newChildren)
				}
			},
		},
	}
}
