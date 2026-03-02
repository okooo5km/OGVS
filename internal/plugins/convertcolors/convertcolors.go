// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package convertcolors implements the convertColors SVGO plugin.
// It converts colors: rgb() to #rrggbb and #rrggbb to #rgb.
package convertcolors

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertColors",
		Description: "converts colors: rgb() to #rrggbb and #rrggbb to #rgb",
		Fn:          fn,
	})
}

var (
	rNumber = `([+-]?(?:\d*\.\d+|\d+\.?)%?)`
	rComma  = `(?:\s*,\s*|\s+)`
	regRGB  = regexp.MustCompile(`^rgb\(\s*` + rNumber + rComma + rNumber + rComma + rNumber + `\s*\)$`)
)

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	currentColor := false
	currentColorStr := ""
	currentColorIsStr := false
	names2hex := true
	rgb2hex := true
	convertCase := "lower"
	shorthex := true
	shortname := true

	if v, ok := params["currentColor"]; ok {
		switch cv := v.(type) {
		case bool:
			currentColor = cv
		case string:
			currentColor = true
			currentColorIsStr = true
			currentColorStr = cv
		}
	}
	if v, ok := params["names2hex"].(bool); ok {
		names2hex = v
	}
	if v, ok := params["rgb2hex"].(bool); ok {
		rgb2hex = v
	}
	if v, ok := params["convertCase"]; ok {
		switch cv := v.(type) {
		case string:
			convertCase = cv
		case bool:
			if !cv {
				convertCase = ""
			}
		}
	}
	if v, ok := params["shorthex"].(bool); ok {
		shorthex = v
	}
	if v, ok := params["shortname"].(bool); ok {
		shortname = v
	}

	maskCounter := 0

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				if elem.Name == "mask" {
					maskCounter++
				}

				for _, entry := range elem.Attributes.Entries() {
					if !collections.ColorsProps[entry.Name] {
						continue
					}

					val := entry.Value

					// Convert to currentColor
					if currentColor && maskCounter == 0 {
						matched := false
						if currentColorIsStr {
							matched = val == currentColorStr
						} else {
							matched = val != "none"
						}
						if matched {
							val = "currentColor"
						}
					}

					// Convert color name to hex
					if names2hex {
						colorName := strings.ToLower(val)
						if hex, ok := collections.ColorsNames[colorName]; ok {
							val = hex
						}
					}

					// Convert rgb() to hex
					if rgb2hex {
						if match := regRGB.FindStringSubmatch(val); match != nil {
							r := parseRGBComponent(match[1])
							g := parseRGBComponent(match[2])
							b := parseRGBComponent(match[3])
							val = convertRGBToHex(r, g, b)
						}
					}

					// Convert case (skip url references and currentColor)
					if convertCase != "" && !tools.IncludesURLReference(val) && val != "currentColor" {
						if convertCase == "lower" {
							val = strings.ToLower(val)
						} else if convertCase == "upper" {
							val = strings.ToUpper(val)
						}
					}

					// Convert long hex to short hex
					if shorthex {
						if isShortableHex(val) {
							val = "#" + string(val[1]) + string(val[3]) + string(val[5])
						}
					}

					// Convert hex to shorter color name
					if shortname {
						colorName := strings.ToLower(val)
						if name, ok := collections.ColorsShortNames[colorName]; ok {
							val = name
						}
					}

					elem.Attributes.Set(entry.Name, val)
				}

				return nil
			},
			Exit: func(node svgast.Node, _ svgast.Parent) {
				elem := node.(*svgast.Element)
				if elem.Name == "mask" {
					maskCounter--
				}
			},
		},
	}
}

// parseRGBComponent parses an RGB component value (integer or percentage).
func parseRGBComponent(s string) int {
	if strings.HasSuffix(s, "%") {
		pct, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0
		}
		return clamp(int(math.Round(pct*2.55)), 0, 255)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		// Try parsing as float
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		n = int(f)
	}
	return clamp(n, 0, 255)
}

// convertRGBToHex converts r,g,b values to a hex color string.
func convertRGBToHex(r, g, b int) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// isShortableHex checks if a 7-char hex color (#aabbcc) can be shortened to #abc.
// Go regexp doesn't support backreferences, so we check manually.
func isShortableHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for i := 0; i < 6; i++ {
		c := s[i+1]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	// Check pairs: s[1]==s[2], s[3]==s[4], s[5]==s[6]
	return s[1] == s[2] && s[3] == s[4] && s[5] == s[6]
}

// clamp restricts a value to [min, max].
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
