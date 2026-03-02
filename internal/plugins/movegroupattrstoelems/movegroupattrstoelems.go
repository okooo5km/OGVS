// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package movegroupattrstoelems implements the moveGroupAttrsToElems SVGO plugin.
// It moves some group attributes to the content elements.
package movegroupattrstoelems

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "moveGroupAttrsToElems",
		Description: "moves some group attributes to the content elements",
		Fn:          fn,
	})
}

// pathElemsWithGroupsAndText includes path elements plus g and text.
var pathElemsWithGroupsAndText map[string]bool

func init() {
	pathElemsWithGroupsAndText = make(map[string]bool)
	for k := range collections.PathElems {
		pathElemsWithGroupsAndText[k] = true
	}
	pathElemsWithGroupsAndText["g"] = true
	pathElemsWithGroupsAndText["text"] = true
}

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				if elem.Name != "g" || len(elem.Children) == 0 {
					return nil
				}
				if !elem.Attributes.Has("transform") {
					return nil
				}

				// Check no attribute has URL reference
				for _, entry := range elem.Attributes.Entries() {
					if collections.ReferencesProps[entry.Name] && tools.IncludesURLReference(entry.Value) {
						return nil
					}
				}

				// Check all children are eligible elements
				for _, child := range elem.Children {
					childElem, ok := child.(*svgast.Element)
					if !ok {
						return nil
					}
					if !pathElemsWithGroupsAndText[childElem.Name] {
						return nil
					}
					if childElem.Attributes.Has("id") {
						return nil
					}
				}

				// Move transform to children
				transform, _ := elem.Attributes.Get("transform")
				for _, child := range elem.Children {
					childElem := child.(*svgast.Element)
					if existingTransform, ok := childElem.Attributes.Get("transform"); ok {
						childElem.Attributes.Set("transform", transform+" "+existingTransform)
					} else {
						childElem.Attributes.Set("transform", transform)
					}
				}

				elem.Attributes.Delete("transform")
				return nil
			},
		},
	}
}
