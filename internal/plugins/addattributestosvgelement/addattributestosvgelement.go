// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package addattributestosvgelement implements the addAttributesToSVGElement SVGO plugin.
// It adds attributes to an outer <svg> element.
package addattributestosvgelement

import (
	"fmt"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "addAttributesToSVGElement",
		Description: "adds attributes to an outer <svg> element",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Normalize params: support both "attributes" (array) and "attribute" (single)
	var attributes []any
	if attrs, ok := params["attributes"].([]any); ok {
		attributes = attrs
	} else if attr := params["attribute"]; attr != nil {
		attributes = []any{attr}
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Only process root <svg>
				if elem.Name != "svg" {
					return nil
				}
				if _, isRoot := parent.(*svgast.Root); !isRoot {
					return nil
				}

				for _, attr := range attributes {
					switch v := attr.(type) {
					case string:
						if !elem.Attributes.Has(v) {
							elem.Attributes.Set(v, svgast.UndefinedAttrValue)
						}
					case map[string]any:
						for key, val := range v {
							if !elem.Attributes.Has(key) {
								elem.Attributes.Set(key, fmt.Sprintf("%v", val))
							}
						}
					}
				}

				return nil
			},
		},
	}
}
