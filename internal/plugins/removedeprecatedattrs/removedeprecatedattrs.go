// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removedeprecatedattrs implements the removeDeprecatedAttrs SVGO plugin.
// It removes deprecated attributes from SVG elements.
package removedeprecatedattrs

import (
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeDeprecatedAttrs",
		Description: "removes deprecated attributes",
		Fn:          fn,
	})
}

// extractAttributesInStylesheet extracts all attribute names used
// in CSS attribute selectors (e.g. [version="1.1"]) from the stylesheet.
// This is used to preserve attributes that are referenced in CSS selectors.
func extractAttributesInStylesheet(stylesheet *css.Stylesheet) map[string]bool {
	attrs := make(map[string]bool)
	for _, rule := range stylesheet.Rules {
		// Parse the original selector to find [attr] or [attr=value] patterns
		extractAttrsFromSelector(rule.OriginalSelector, attrs)
	}
	return attrs
}

// extractAttrsFromSelector extracts attribute names from CSS attribute selectors
// within a selector string.
func extractAttrsFromSelector(selector string, attrs map[string]bool) {
	i := 0
	for i < len(selector) {
		if selector[i] == '[' {
			i++ // skip [
			// Read attribute name (until =, ], ~, |, ^, $, *)
			start := i
			for i < len(selector) && selector[i] != '=' && selector[i] != ']' &&
				selector[i] != '~' && selector[i] != '|' && selector[i] != '^' &&
				selector[i] != '$' && selector[i] != '*' {
				i++
			}
			name := strings.TrimSpace(selector[start:i])
			if name != "" {
				attrs[name] = true
			}
			// Skip to closing ]
			for i < len(selector) && selector[i] != ']' {
				i++
			}
			if i < len(selector) {
				i++ // skip ]
			}
		} else {
			i++
		}
	}
}

// processAttributes removes deprecated attrs from the node, respecting
// the safe/unsafe distinction and skipping attrs used in CSS selectors.
func processAttributes(
	node *svgast.Element,
	deprecated *collections.DeprecatedAttrs,
	removeUnsafe bool,
	attributesInStylesheet map[string]bool,
) {
	if deprecated == nil {
		return
	}

	if deprecated.Safe != nil {
		for name := range deprecated.Safe {
			if attributesInStylesheet[name] {
				continue
			}
			node.Attributes.Delete(name)
		}
	}

	if removeUnsafe && deprecated.Unsafe != nil {
		for name := range deprecated.Unsafe {
			if attributesInStylesheet[name] {
				continue
			}
			node.Attributes.Delete(name)
		}
	}
}

func fn(root *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	removeUnsafe := false
	if v, ok := params["removeUnsafe"]; ok {
		if b, ok := v.(bool); ok {
			removeUnsafe = b
		}
	}

	stylesheet := css.CollectStylesheet(root)
	attributesInStylesheet := extractAttributesInStylesheet(stylesheet)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				elemConfig := collections.Elems[elem.Name]
				if elemConfig == nil {
					return nil
				}

				// Special case: removing deprecated xml:lang is safe
				// when the lang attribute exists.
				if elemConfig.AttrsGroups["core"] &&
					elem.Attributes.Has("xml:lang") &&
					!attributesInStylesheet["xml:lang"] &&
					elem.Attributes.Has("lang") {
					elem.Attributes.Delete("xml:lang")
				}

				// General cases: process deprecated attrs from attribute groups
				for attrsGroup := range elemConfig.AttrsGroups {
					processAttributes(
						elem,
						collections.AttrsGroupsDeprecatedFull[attrsGroup],
						removeUnsafe,
						attributesInStylesheet,
					)
				}

				// Process element-specific deprecated attrs
				processAttributes(
					elem,
					elemConfig.Deprecated,
					removeUnsafe,
					attributesInStylesheet,
				)

				return nil
			},
		},
	}
}
