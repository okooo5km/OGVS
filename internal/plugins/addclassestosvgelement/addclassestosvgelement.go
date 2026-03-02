// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package addclassestosvgelement implements the addClassesToSVGElement SVGO plugin.
// It adds classnames to an outer <svg> element.
package addclassestosvgelement

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "addClassesToSVGElement",
		Description: "adds classnames to an outer <svg> element",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Normalize params: support both "classNames" (array) and "className" (single)
	var classNames []string
	if names, ok := params["classNames"].([]any); ok {
		for _, n := range names {
			if s, ok := n.(string); ok {
				classNames = append(classNames, s)
			}
		}
	} else if name, ok := params["className"].(string); ok {
		classNames = []string{name}
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

				// Build ordered unique set of classes
				seen := make(map[string]bool)
				var result []string

				if existing, ok := elem.Attributes.Get("class"); ok {
					for _, cls := range strings.Split(existing, " ") {
						if cls != "" && !seen[cls] {
							seen[cls] = true
							result = append(result, cls)
						}
					}
				}

				for _, cls := range classNames {
					if !seen[cls] {
						seen[cls] = true
						result = append(result, cls)
					}
				}

				elem.Attributes.Set("class", strings.Join(result, " "))

				return nil
			},
		},
	}
}
