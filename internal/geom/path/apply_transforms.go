// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Apply SVG transforms to path data.
// Ported from SVGO's plugins/applyTransforms.js.

package path

import (
	"math"
	"regexp"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/geom/transform"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/tools"
)

var regNumericValues = regexp.MustCompile(`[-+]?(?:\d*\.\d+|\d+\.?)(?:[eE][-+]?\d+)?`)

// ApplyTransformsParams controls the applyTransforms behavior.
type ApplyTransformsParams struct {
	TransformPrecision     int
	ApplyTransformsStroked bool
}

// ApplyTransformsVisitor returns a visitor that applies transforms to path data.
// Ported from SVGO's applyTransforms plugin in plugins/applyTransforms.js.
func ApplyTransformsVisitor(root *svgast.Root, params *ApplyTransformsParams) *svgast.Visitor {
	stylesheet := css.CollectStylesheet(root)
	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)
				applyTransformsToElement(elem, stylesheet, params)
				return nil
			},
		},
	}
}

// applyTransformsToElement applies the transform attribute to a path element's d attribute.
func applyTransformsToElement(elem *svgast.Element, stylesheet *css.Stylesheet, params *ApplyTransformsParams) {
	dVal, hasD := elem.Attributes.Get("d")
	if !hasD || dVal == "" {
		return
	}

	// stroke and stroke-width can be redefined with <use>
	if _, hasID := elem.Attributes.Get("id"); hasID {
		return
	}

	transformVal, hasTransform := elem.Attributes.Get("transform")
	if !hasTransform || transformVal == "" {
		return
	}

	// styles are not considered when applying transform
	if _, hasStyle := elem.Attributes.Get("style"); hasStyle {
		return
	}

	// Check for references to other objects (gradients, clip-paths) which are also subjects to transform
	for _, entry := range elem.Attributes.Entries() {
		if collections.ReferencesProps[entry.Name] && tools.IncludesURLReference(entry.Value) {
			return
		}
	}

	computedStyle := css.ComputeStyle(stylesheet, elem)
	transformStyle := computedStyle["transform"]

	// Transform overridden in <style> tag which is not considered
	if transformStyle != nil && transformStyle.Type == css.StyleStatic &&
		transformStyle.Value != transformVal {
		return
	}

	matrix := transform.TransformsMultiply(
		transform.Transform2JS(transformVal),
	)

	var stroke string
	if cs := computedStyle["stroke"]; cs != nil && cs.Type == css.StyleStatic {
		stroke = cs.Value
	}

	var strokeWidth string
	if cs := computedStyle["stroke-width"]; cs != nil && cs.Type == css.StyleStatic {
		strokeWidth = cs.Value
	}

	transformPrecision := params.TransformPrecision

	// If stroke or stroke-width is dynamic, don't apply transform
	if (computedStyle["stroke"] != nil && computedStyle["stroke"].Type == css.StyleDynamic) ||
		(computedStyle["stroke-width"] != nil && computedStyle["stroke-width"].Type == css.StyleDynamic) {
		return
	}

	scale := math.Round(math.Hypot(matrix.Data[0], matrix.Data[1])*
		math.Pow(10, float64(transformPrecision))) / math.Pow(10, float64(transformPrecision))

	if stroke != "" && stroke != "none" {
		if !params.ApplyTransformsStroked {
			return
		}

		// stroke cannot be transformed with different vertical and horizontal scale or skew
		if (matrix.Data[0] != matrix.Data[3] || matrix.Data[1] != -matrix.Data[2]) &&
			(matrix.Data[0] != -matrix.Data[3] || matrix.Data[1] != matrix.Data[2]) {
			return
		}

		// apply transform to stroke-width, stroke-dashoffset and stroke-dasharray
		if scale != 1 {
			vectorEffect, _ := elem.Attributes.Get("vector-effect")
			if vectorEffect != "non-scaling-stroke" {
				sw := strokeWidth
				if sw == "" {
					sw = collections.AttrsGroupsDefaults["presentation"]["stroke-width"]
				}
				newSW := strings.TrimSpace(sw)
				newSW = regNumericValues.ReplaceAllStringFunc(newSW, func(num string) string {
					v := parseFloatStr(num)
					return tools.RemoveLeadingZero(v * scale)
				})
				elem.Attributes.Set("stroke-width", newSW)

				if sdo, hasSdo := elem.Attributes.Get("stroke-dashoffset"); hasSdo {
					newSDO := strings.TrimSpace(sdo)
					newSDO = regNumericValues.ReplaceAllStringFunc(newSDO, func(num string) string {
						v := parseFloatStr(num)
						return tools.RemoveLeadingZero(v * scale)
					})
					elem.Attributes.Set("stroke-dashoffset", newSDO)
				}

				if sda, hasSda := elem.Attributes.Get("stroke-dasharray"); hasSda {
					newSDA := strings.TrimSpace(sda)
					newSDA = regNumericValues.ReplaceAllStringFunc(newSDA, func(num string) string {
						v := parseFloatStr(num)
						return tools.RemoveLeadingZero(v * scale)
					})
					elem.Attributes.Set("stroke-dasharray", newSDA)
				}
			}
		}
	}

	pathData := Path2JS(elem)
	ApplyMatrixToPathData(pathData, matrix.Data)

	// Write the transformed path data back to the d attribute.
	// In JS, path2js() caches pathData on the node, so subsequent calls
	// (e.g., from convertPathData) see the transformed data. Since Go
	// re-parses d each time, we must serialize the transformed data back.
	// Use precision=-1 (no rounding) to avoid precision loss.
	JS2Path(elem, pathData, -1, false)

	// remove transform attr
	elem.Attributes.Delete("transform")
}

// parseFloatStr parses a numeric string to float64.
func parseFloatStr(s string) float64 {
	return parseFloat(s)
}

// Path2JS converts a path element's d attribute to a slice of PathDataItems.
// Ported from SVGO's path2js() in plugins/_path.js.
// First moveto is always made absolute.
func Path2JS(elem *svgast.Element) []PathDataItem {
	dVal, ok := elem.Attributes.Get("d")
	if !ok {
		return nil
	}
	pathData := ParsePathData(dVal)
	// First moveto is actually absolute
	if len(pathData) > 0 && pathData[0].Command == 'm' {
		pathData[0].Command = 'M'
	}
	return pathData
}

// JS2Path converts path data items back to a d attribute string and sets it on the element.
// Also removes consecutive moveto commands (last one wins).
// Ported from SVGO's js2path() in plugins/_path.js.
func JS2Path(elem *svgast.Element, data []PathDataItem, floatPrecision int, noSpaceAfterFlags bool) {
	// Remove moveto commands which are followed by moveto commands
	var pathData []PathDataItem
	for _, item := range data {
		if len(pathData) != 0 &&
			(item.Command == 'M' || item.Command == 'm') {
			last := pathData[len(pathData)-1]
			if last.Command == 'M' || last.Command == 'm' {
				pathData = pathData[:len(pathData)-1]
			}
		}
		pathData = append(pathData, PathDataItem{
			Command: item.Command,
			Args:    item.Args,
		})
	}

	elem.Attributes.Set("d", StringifyPathData(&StringifyPathDataOptions{
		PathData:               pathData,
		Precision:              floatPrecision,
		DisableSpaceAfterFlags: noSpaceAfterFlags,
	}))
}

// transformAbsolutePoint applies a matrix to an absolute point.
func transformAbsolutePoint(matrix []float64, x, y float64) (float64, float64) {
	newX := matrix[0]*x + matrix[2]*y + matrix[4]
	newY := matrix[1]*x + matrix[3]*y + matrix[5]
	return newX, newY
}

// transformRelativePoint applies a matrix to a relative point (no translation).
func transformRelativePoint(matrix []float64, x, y float64) (float64, float64) {
	newX := matrix[0]*x + matrix[2]*y
	newY := matrix[1]*x + matrix[3]*y
	return newX, newY
}

// ApplyMatrixToPathData applies a transformation matrix to path data in-place.
// Ported from SVGO's applyMatrixToPathData() in plugins/applyTransforms.js.
func ApplyMatrixToPathData(pathData []PathDataItem, matrix []float64) {
	start := [2]float64{0, 0}
	cursor := [2]float64{0, 0}

	for pi := range pathData {
		pathItem := &pathData[pi]
		command := pathItem.Command
		args := pathItem.Args

		// moveto (x y)
		if command == 'M' {
			cursor[0] = args[0]
			cursor[1] = args[1]
			start[0] = cursor[0]
			start[1] = cursor[1]
			x, y := transformAbsolutePoint(matrix, args[0], args[1])
			args[0] = x
			args[1] = y
		}
		if command == 'm' {
			cursor[0] += args[0]
			cursor[1] += args[1]
			start[0] = cursor[0]
			start[1] = cursor[1]
			x, y := transformRelativePoint(matrix, args[0], args[1])
			args[0] = x
			args[1] = y
		}

		// horizontal lineto (x) → convert to lineto
		if command == 'H' {
			command = 'L'
			args = []float64{args[0], cursor[1]}
		}
		if command == 'h' {
			command = 'l'
			args = []float64{args[0], 0}
		}

		// vertical lineto (y) → convert to lineto
		if command == 'V' {
			command = 'L'
			args = []float64{cursor[0], args[0]}
		}
		if command == 'v' {
			command = 'l'
			args = []float64{0, args[0]}
		}

		// lineto (x y)
		if command == 'L' {
			cursor[0] = args[0]
			cursor[1] = args[1]
			x, y := transformAbsolutePoint(matrix, args[0], args[1])
			args[0] = x
			args[1] = y
		}
		if command == 'l' {
			cursor[0] += args[0]
			cursor[1] += args[1]
			x, y := transformRelativePoint(matrix, args[0], args[1])
			args[0] = x
			args[1] = y
		}

		// curveto (x1 y1 x2 y2 x y)
		if command == 'C' {
			cursor[0] = args[4]
			cursor[1] = args[5]
			x1, y1 := transformAbsolutePoint(matrix, args[0], args[1])
			x2, y2 := transformAbsolutePoint(matrix, args[2], args[3])
			x, y := transformAbsolutePoint(matrix, args[4], args[5])
			args[0] = x1
			args[1] = y1
			args[2] = x2
			args[3] = y2
			args[4] = x
			args[5] = y
		}
		if command == 'c' {
			cursor[0] += args[4]
			cursor[1] += args[5]
			x1, y1 := transformRelativePoint(matrix, args[0], args[1])
			x2, y2 := transformRelativePoint(matrix, args[2], args[3])
			x, y := transformRelativePoint(matrix, args[4], args[5])
			args[0] = x1
			args[1] = y1
			args[2] = x2
			args[3] = y2
			args[4] = x
			args[5] = y
		}

		// smooth curveto (x2 y2 x y)
		if command == 'S' {
			cursor[0] = args[2]
			cursor[1] = args[3]
			x2, y2 := transformAbsolutePoint(matrix, args[0], args[1])
			x, y := transformAbsolutePoint(matrix, args[2], args[3])
			args[0] = x2
			args[1] = y2
			args[2] = x
			args[3] = y
		}
		if command == 's' {
			cursor[0] += args[2]
			cursor[1] += args[3]
			x2, y2 := transformRelativePoint(matrix, args[0], args[1])
			x, y := transformRelativePoint(matrix, args[2], args[3])
			args[0] = x2
			args[1] = y2
			args[2] = x
			args[3] = y
		}

		// quadratic Bezier curveto (x1 y1 x y)
		if command == 'Q' {
			cursor[0] = args[2]
			cursor[1] = args[3]
			x1, y1 := transformAbsolutePoint(matrix, args[0], args[1])
			x, y := transformAbsolutePoint(matrix, args[2], args[3])
			args[0] = x1
			args[1] = y1
			args[2] = x
			args[3] = y
		}
		if command == 'q' {
			cursor[0] += args[2]
			cursor[1] += args[3]
			x1, y1 := transformRelativePoint(matrix, args[0], args[1])
			x, y := transformRelativePoint(matrix, args[2], args[3])
			args[0] = x1
			args[1] = y1
			args[2] = x
			args[3] = y
		}

		// smooth quadratic Bezier curveto (x y)
		if command == 'T' {
			cursor[0] = args[0]
			cursor[1] = args[1]
			x, y := transformAbsolutePoint(matrix, args[0], args[1])
			args[0] = x
			args[1] = y
		}
		if command == 't' {
			cursor[0] += args[0]
			cursor[1] += args[1]
			x, y := transformRelativePoint(matrix, args[0], args[1])
			args[0] = x
			args[1] = y
		}

		// elliptical arc (rx ry x-axis-rotation large-arc-flag sweep-flag x y)
		if command == 'A' {
			transform.TransformArc(cursor, args, matrix)
			cursor[0] = args[5]
			cursor[1] = args[6]
			// reduce number of digits in rotation angle
			if math.Abs(args[2]) > 80 {
				a := args[0]
				rotation := args[2]
				args[0] = args[1]
				args[1] = a
				if rotation > 0 {
					args[2] = rotation - 90
				} else {
					args[2] = rotation + 90
				}
			}
			x, y := transformAbsolutePoint(matrix, args[5], args[6])
			args[5] = x
			args[6] = y
		}
		if command == 'a' {
			transform.TransformArc([2]float64{0, 0}, args, matrix)
			cursor[0] += args[5]
			cursor[1] += args[6]
			// reduce number of digits in rotation angle
			if math.Abs(args[2]) > 80 {
				a := args[0]
				rotation := args[2]
				args[0] = args[1]
				args[1] = a
				if rotation > 0 {
					args[2] = rotation - 90
				} else {
					args[2] = rotation + 90
				}
			}
			x, y := transformRelativePoint(matrix, args[5], args[6])
			args[5] = x
			args[6] = y
		}

		// closepath
		if command == 'z' || command == 'Z' {
			cursor[0] = start[0]
			cursor[1] = start[1]
		}

		pathItem.Command = command
		pathItem.Args = args
	}
}
