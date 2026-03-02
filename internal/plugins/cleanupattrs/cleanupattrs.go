// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package cleanupattrs implements the cleanupAttrs SVGO plugin.
// It cleans up attribute values from newlines, trailing and repeating spaces.
package cleanupattrs

import (
	"regexp"
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

var (
	regNewlinesNeedSpace = regexp.MustCompile(`(\S)\r?\n(\S)`)
	regNewlines          = regexp.MustCompile(`\r?\n`)
	regSpaces            = regexp.MustCompile(`\s{2,}`)
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "cleanupAttrs",
		Description: "cleanups attributes from newlines, trailing and repeating spaces",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	newlines := getBoolParam(params, "newlines", true)
	trim := getBoolParam(params, "trim", true)
	spaces := getBoolParam(params, "spaces", true)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)
				for _, entry := range elem.Attributes.Entries() {
					val := entry.Value
					if newlines {
						// new line which requires a space instead
						val = regNewlinesNeedSpace.ReplaceAllString(val, "${1} ${2}")
						// simple new line
						val = regNewlines.ReplaceAllString(val, "")
					}
					if trim {
						val = strings.TrimSpace(val)
					}
					if spaces {
						val = regSpaces.ReplaceAllString(val, " ")
					}
					if val != entry.Value {
						elem.Attributes.Set(entry.Name, val)
					}
				}
				return nil
			},
		},
	}
}

func getBoolParam(params map[string]any, key string, def bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
