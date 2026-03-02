// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removecomments implements the removeComments SVGO plugin.
// It removes XML comments, optionally preserving those matching patterns.
package removecomments

import (
	"regexp"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeComments",
		Description: "removes comments",
		Fn:          fn,
	})
}

// defaultPreservePatterns preserves comments starting with "!" (e.g. copyright notices).
var defaultPreservePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^!`),
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Resolve preservePatterns param
	patterns := defaultPreservePatterns

	if pp, ok := params["preservePatterns"]; ok {
		switch v := pp.(type) {
		case bool:
			if !v {
				// preservePatterns: false → remove all comments
				patterns = nil
			}
		case []any:
			patterns = make([]*regexp.Regexp, 0, len(v))
			for _, p := range v {
				if s, ok := p.(string); ok {
					if re, err := regexp.Compile(s); err == nil {
						patterns = append(patterns, re)
					}
				}
			}
		}
	}

	return &svgast.Visitor{
		Comment: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				comment := node.(*svgast.Comment)

				// Check if comment matches any preserve pattern
				if len(patterns) > 0 {
					for _, re := range patterns {
						if re.MatchString(comment.Value) {
							return nil // preserve this comment
						}
					}
				}

				svgast.DetachNodeFromParent(node, parent)
				return nil
			},
		},
	}
}
