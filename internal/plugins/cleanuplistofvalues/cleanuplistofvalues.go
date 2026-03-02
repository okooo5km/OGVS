// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package cleanuplistofvalues implements the cleanupListOfValues SVGO plugin.
// It rounds list of values to the fixed precision.
package cleanuplistofvalues

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "cleanupListOfValues",
		Description: "rounds list of values to the fixed precision",
		Fn:          fn,
	})
}

var regNumericValues = regexp.MustCompile(`^([-+]?\d*\.?\d+([eE][-+]?\d+)?)(px|pt|pc|mm|cm|m|in|ft|em|ex|%)?$`)
var regSeparator = regexp.MustCompile(`\s+,?\s*|,\s*`)

var absoluteLengths = map[string]float64{
	"cm": 96.0 / 2.54,
	"mm": 96.0 / 25.4,
	"in": 96.0,
	"pt": 4.0 / 3.0,
	"pc": 16.0,
	"px": 1.0,
}

// listAttrs are the attributes that contain lists of values.
var listAttrs = []string{
	"points", "enable-background", "viewBox", "stroke-dasharray",
	"dx", "dy", "x", "y",
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	floatPrecision := 3
	leadingZero := true
	defaultPx := true
	convertToPx := true

	if v, ok := params["floatPrecision"].(float64); ok {
		floatPrecision = int(v)
	}
	if v, ok := params["leadingZero"].(bool); ok {
		leadingZero = v
	}
	if v, ok := params["defaultPx"].(bool); ok {
		defaultPx = v
	}
	if v, ok := params["convertToPx"].(bool); ok {
		convertToPx = v
	}

	roundValues := func(lists string) string {
		parts := regSeparator.Split(lists, -1)
		var roundedList []string

		for _, elem := range parts {
			match := regNumericValues.FindStringSubmatch(elem)

			if match != nil {
				numVal, err := strconv.ParseFloat(match[1], 64)
				if err != nil {
					roundedList = append(roundedList, elem)
					continue
				}

				num := tools.ToFixed(numVal, floatPrecision)
				units := match[3]

				// Convert absolute units to px
				if convertToPx && units != "" {
					if factor, ok := absoluteLengths[units]; ok {
						pxNum := tools.ToFixed(factor*numVal, floatPrecision)
						pxStr := formatNum(pxNum)
						if len(pxStr) < len(match[0]) {
							num = pxNum
							units = "px"
						}
					}
				}

				// Remove leading zero
				var str string
				if leadingZero {
					str = tools.RemoveLeadingZero(num)
				} else {
					str = formatNum(num)
				}

				// Remove default "px" units
				if defaultPx && units == "px" {
					units = ""
				}

				roundedList = append(roundedList, str+units)
			} else if elem == "new" {
				roundedList = append(roundedList, "new")
			} else if elem != "" {
				roundedList = append(roundedList, elem)
			}
		}

		return strings.Join(roundedList, " ")
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				for _, attr := range listAttrs {
					if val, ok := elem.Attributes.Get(attr); ok {
						elem.Attributes.Set(attr, roundValues(val))
					}
				}

				return nil
			},
		},
	}
}

// formatNum formats a float64 to string.
func formatNum(f float64) string {
	if f == math.Trunc(f) && !math.IsInf(f, 0) {
		return fmt.Sprintf("%g", f)
	}
	return fmt.Sprintf("%g", f)
}
