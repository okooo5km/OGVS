// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package cleanupnumericvalues implements the cleanupNumericValues SVGO plugin.
// It rounds numeric values to fixed precision, removes default "px" units.
package cleanupnumericvalues

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
		Name:        "cleanupNumericValues",
		Description: "rounds numeric values to the fixed precision, removes default \"px\" units",
		Fn:          fn,
	})
}

var regNumericValues = regexp.MustCompile(`^([-+]?\d*\.?\d+([eE][-+]?\d+)?)(px|pt|pc|mm|cm|m|in|ft|em|ex|%)?$`)

// regViewBoxSep splits viewBox values: "0 0 100 100", "0, 0, 100, 100", "0,0,100,100"
var regViewBoxSep = regexp.MustCompile(`(?:\s,?|,)\s*`)

var absoluteLengths = map[string]float64{
	"cm": 96.0 / 2.54,
	"mm": 96.0 / 25.4,
	"in": 96.0,
	"pt": 4.0 / 3.0,
	"pc": 16.0,
	"px": 1.0,
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

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Special handling for viewBox
				if vb, ok := elem.Attributes.Get("viewBox"); ok {
					trimmed := strings.TrimSpace(vb)
					numbers := regViewBoxSep.Split(trimmed, -1)
					for i, v := range numbers {
						num, err := strconv.ParseFloat(v, 64)
						if err != nil {
							continue
						}
						rounded := tools.ToFixed(num, floatPrecision)
						numbers[i] = formatNum(rounded)
					}
					elem.Attributes.Set("viewBox", strings.Join(numbers, " "))
				}

				for _, entry := range elem.Attributes.Entries() {
					// version is a text string
					if entry.Name == "version" {
						continue
					}

					match := regNumericValues.FindStringSubmatch(entry.Value)
					if match == nil {
						continue
					}

					numVal, err := strconv.ParseFloat(match[1], 64)
					if err != nil {
						continue
					}

					// Round to precision
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

					elem.Attributes.Set(entry.Name, str+units)
				}

				return nil
			},
		},
	}
}

// formatNum formats a float64 to string, removing unnecessary trailing zeros.
func formatNum(f float64) string {
	if f == math.Trunc(f) && !math.IsInf(f, 0) {
		return fmt.Sprintf("%g", f)
	}
	return fmt.Sprintf("%g", f)
}
