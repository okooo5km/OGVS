// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeattrs implements the removeAttrs SVGO plugin.
// It removes specified attributes based on pattern matching.
package removeattrs

import (
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeAttrs",
		Description: "removes specified attributes",
		Fn:          fn,
	})
}

// matcher wraps either a standard regexp.Regexp or a regexp2.Regexp.
// Go's RE2 engine does not support lookaheads; regexp2 provides PCRE-like
// features as a fallback.
type matcher struct {
	re  *regexp.Regexp
	re2 *regexp2.Regexp
}

func compileMatcher(pattern string) (*matcher, error) {
	re, err := regexp.Compile(pattern)
	if err == nil {
		return &matcher{re: re}, nil
	}
	// Fallback to regexp2 for advanced features (lookaheads, etc.)
	re2, err2 := regexp2.Compile(pattern, regexp2.IgnoreCase)
	if err2 != nil {
		return nil, err2
	}
	return &matcher{re2: re2}, nil
}

func (m *matcher) matchString(s string) bool {
	if m.re != nil {
		return m.re.MatchString(s)
	}
	ok, _ := m.re2.MatchString(s)
	return ok
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	if params["attrs"] == nil {
		return nil
	}

	elemSeparator := ":"
	if v, ok := params["elemSeparator"].(string); ok {
		elemSeparator = v
	}

	preserveCurrentColor := false
	if v, ok := params["preserveCurrentColor"].(bool); ok {
		preserveCurrentColor = v
	}

	// Normalize attrs to []string
	var attrs []string
	switch v := params["attrs"].(type) {
	case string:
		attrs = []string{v}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				attrs = append(attrs, s)
			}
		}
	}

	// Pre-compile regex patterns
	type attrPattern struct {
		elemRe  *matcher
		nameRe  *matcher
		valueRe *matcher
	}

	var patterns []attrPattern
	for _, pattern := range attrs {
		// Normalize pattern to 3 parts
		if !strings.Contains(pattern, elemSeparator) {
			pattern = ".*" + elemSeparator + pattern + elemSeparator + ".*"
		} else if len(strings.Split(pattern, elemSeparator)) < 3 {
			pattern = pattern + elemSeparator + ".*"
		}

		parts := strings.SplitN(pattern, elemSeparator, 3)
		for i, p := range parts {
			if p == "*" {
				parts[i] = ".*"
			}
		}

		elemRe, err1 := compileMatcher("(?i)^" + parts[0] + "$")
		nameRe, err2 := compileMatcher("(?i)^" + parts[1] + "$")
		valueRe, err3 := compileMatcher("(?i)^" + parts[2] + "$")
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		patterns = append(patterns, attrPattern{elemRe, nameRe, valueRe})
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				for _, pat := range patterns {
					if !pat.elemRe.matchString(elem.Name) {
						continue
					}

					for _, entry := range elem.Attributes.Entries() {
						isCurrentColor := strings.EqualFold(entry.Value, "currentcolor")
						if preserveCurrentColor && entry.Name == "fill" && isCurrentColor {
							continue
						}
						if preserveCurrentColor && entry.Name == "stroke" && isCurrentColor {
							continue
						}

						if pat.nameRe.matchString(entry.Name) && pat.valueRe.matchString(entry.Value) {
							elem.Attributes.Delete(entry.Name)
						}
					}
				}

				return nil
			},
		},
	}
}
