// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeattrsbyselector removes attributes from elements matching CSS selectors.
package removeattrsbyselector

import (
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeAttributesBySelector",
		Description: "removes attributes of elements that match a css selector",
		Fn:          fn,
	})
}

func fn(root *svgast.Root, params map[string]any, info *plugin.PluginInfo) *svgast.Visitor {
	// Build parent map for CSS selector matching
	parents := buildParentMap(root)

	// Normalize params into a list of {selector, attributes} pairs
	type selectorConfig struct {
		Selector   string
		Attributes []string
	}
	var configs []selectorConfig

	if selectorsRaw, ok := params["selectors"]; ok {
		// Multiple selectors mode
		if arr, ok := selectorsRaw.([]any); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					sel, _ := m["selector"].(string)
					attrs := extractStringOrSlice(m["attributes"])
					if sel != "" && len(attrs) > 0 {
						configs = append(configs, selectorConfig{Selector: sel, Attributes: attrs})
					}
				}
			}
		}
	} else {
		// Single selector mode
		sel, _ := params["selector"].(string)
		attrs := extractStringOrSlice(params["attributes"])
		if sel != "" && len(attrs) > 0 {
			configs = append(configs, selectorConfig{Selector: sel, Attributes: attrs})
		}
	}

	// Apply all selector configs
	for _, cfg := range configs {
		matched := css.QuerySelectorAll(root, cfg.Selector, parents)
		for _, elem := range matched {
			for _, attr := range cfg.Attributes {
				elem.Attributes.Delete(attr)
			}
		}
	}

	// Return empty visitor — all work done during initialization
	return &svgast.Visitor{}
}

// extractStringOrSlice extracts a string or []string from an any value.
func extractStringOrSlice(v any) []string {
	switch val := v.(type) {
	case string:
		return []string{val}
	case []any:
		var result []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	}
	return nil
}

// buildParentMap builds a parent map for the entire tree.
func buildParentMap(root *svgast.Root) map[svgast.Node]svgast.Parent {
	parents := make(map[svgast.Node]svgast.Parent)
	var walk func(parent svgast.Parent, children []svgast.Node)
	walk = func(parent svgast.Parent, children []svgast.Node) {
		for _, child := range children {
			parents[child] = parent
			if elem, ok := child.(*svgast.Element); ok {
				walk(elem, elem.Children)
			}
		}
	}
	walk(root, root.Children)
	return parents
}
