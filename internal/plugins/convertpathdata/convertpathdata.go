// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package convertpathdata implements the convertPathData SVGO plugin.
// It optimizes path data: writes in shorter form, applies transformations.
// Ported from SVGO's plugins/convertPathData.js.
package convertpathdata

import (
	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/css"
	"github.com/okooo5km/ogvs/internal/geom/path"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertPathData",
		Description: "optimizes path data: writes in shorter form, applies transformations",
		Fn:          fn,
	})
}

func fn(root *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	// Parse params with defaults matching SVGO
	applyTransformsParam := true
	applyTransformsStroked := true
	var makeArcs *path.MakeArcsParams = &path.MakeArcsParams{
		Threshold: 2.5,
		Tolerance: 0.5,
	}
	straightCurves := true
	convertToQ := true
	lineShorthands := true
	convertToZ := true
	curveSmoothShorthands := true
	floatPrecision := 3
	floatPrecisionEnabled := true
	transformPrecision := 5
	smartArcRounding := true
	removeUseless := true
	collapseRepeated := true
	utilizeAbsolute := true
	leadingZero := true
	negativeExtraSpace := true
	noSpaceAfterFlags := false
	forceAbsolutePath := false

	if v, ok := params["applyTransforms"].(bool); ok {
		applyTransformsParam = v
	}
	if v, ok := params["applyTransformsStroked"].(bool); ok {
		applyTransformsStroked = v
	}
	if v, ok := params["makeArcs"]; ok {
		if v == false {
			makeArcs = nil
		} else if m, ok := v.(map[string]any); ok {
			if t, ok := m["threshold"].(float64); ok {
				makeArcs.Threshold = t
			}
			if t, ok := m["tolerance"].(float64); ok {
				makeArcs.Tolerance = t
			}
		}
	}
	if v, ok := params["straightCurves"].(bool); ok {
		straightCurves = v
	}
	if v, ok := params["convertToQ"].(bool); ok {
		convertToQ = v
	}
	if v, ok := params["lineShorthands"].(bool); ok {
		lineShorthands = v
	}
	if v, ok := params["convertToZ"].(bool); ok {
		convertToZ = v
	}
	if v, ok := params["curveSmoothShorthands"].(bool); ok {
		curveSmoothShorthands = v
	}
	if v, ok := params["floatPrecision"]; ok {
		if vb, ok := v.(bool); ok && !vb {
			floatPrecisionEnabled = false
		} else if vf, ok := v.(float64); ok {
			floatPrecision = int(vf)
		}
	}
	if v, ok := params["transformPrecision"].(float64); ok {
		transformPrecision = int(v)
	}
	if v, ok := params["smartArcRounding"].(bool); ok {
		smartArcRounding = v
	}
	if v, ok := params["removeUseless"].(bool); ok {
		removeUseless = v
	}
	if v, ok := params["collapseRepeated"].(bool); ok {
		collapseRepeated = v
	}
	if v, ok := params["utilizeAbsolute"].(bool); ok {
		utilizeAbsolute = v
	}
	if v, ok := params["leadingZero"].(bool); ok {
		leadingZero = v
	}
	if v, ok := params["negativeExtraSpace"].(bool); ok {
		negativeExtraSpace = v
	}
	if v, ok := params["noSpaceAfterFlags"].(bool); ok {
		noSpaceAfterFlags = v
	}
	if v, ok := params["forceAbsolutePath"].(bool); ok {
		forceAbsolutePath = v
	}

	convertParams := &path.ConvertPathDataParams{
		ApplyTransforms:        applyTransformsParam,
		ApplyTransformsStroked: applyTransformsStroked,
		MakeArcs:               makeArcs,
		StraightCurves:         straightCurves,
		ConvertToQ:             convertToQ,
		LineShorthands:         lineShorthands,
		ConvertToZ:             convertToZ,
		CurveSmoothShorthands:  curveSmoothShorthands,
		FloatPrecision:         floatPrecision,
		FloatPrecisionEnabled:  floatPrecisionEnabled,
		TransformPrecision:     transformPrecision,
		SmartArcRounding:       smartArcRounding,
		RemoveUseless:          removeUseless,
		CollapseRepeated:       collapseRepeated,
		UtilizeAbsolute:        utilizeAbsolute,
		LeadingZero:            leadingZero,
		NegativeExtraSpace:     negativeExtraSpace,
		NoSpaceAfterFlags:      noSpaceAfterFlags,
		ForceAbsolutePath:      forceAbsolutePath,
	}

	// Invoke applyTransforms plugin
	if applyTransformsParam {
		visitor := path.ApplyTransformsVisitor(root, &path.ApplyTransformsParams{
			TransformPrecision:     transformPrecision,
			ApplyTransformsStroked: applyTransformsStroked,
		})
		svgast.Visit(root, visitor, nil)
	}

	stylesheet := css.CollectStylesheet(root)

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				if !collections.PathElems[elem.Name] {
					return nil
				}
				dVal, hasD := elem.Attributes.Get("d")
				if !hasD || dVal == "" {
					return nil
				}

				computedStyle := css.ComputeStyle(stylesheet, elem)

				hasMarkerMid := computedStyle["marker-mid"] != nil

				maybeHasStroke := computedStyle["stroke"] != nil &&
					(computedStyle["stroke"].Type == css.StyleDynamic ||
						computedStyle["stroke"].Value != "none")
				maybeHasLinecap := computedStyle["stroke-linecap"] != nil &&
					(computedStyle["stroke-linecap"].Type == css.StyleDynamic ||
						computedStyle["stroke-linecap"].Value != "butt")
				maybeHasStrokeAndLinecap := maybeHasStroke && maybeHasLinecap

				isSafeToUseZ := true
				if maybeHasStroke {
					isSafeToUseZ = computedStyle["stroke-linecap"] != nil &&
						computedStyle["stroke-linecap"].Type == css.StyleStatic &&
						computedStyle["stroke-linecap"].Value == "round" &&
						computedStyle["stroke-linejoin"] != nil &&
						computedStyle["stroke-linejoin"].Type == css.StyleStatic &&
						computedStyle["stroke-linejoin"].Value == "round"
				}

				rawData := path.Path2JS(elem)

				if len(rawData) == 0 {
					return nil
				}

				includesVertices := false
				for _, item := range rawData {
					if item.Command != 'm' && item.Command != 'M' {
						includesVertices = true
						break
					}
				}

				// Convert to ExtendedPathItem
				data := make([]path.ExtendedPathItem, len(rawData))
				for i, item := range rawData {
					data[i] = path.ExtendedPathItem{
						Command: item.Command,
						Args:    append([]float64(nil), item.Args...),
					}
				}

				path.ConvertToRelative(data)

				data = path.Filters(data, convertParams,
					isSafeToUseZ, maybeHasStrokeAndLinecap, hasMarkerMid)

				if utilizeAbsolute {
					data = path.ConvertToMixed(data, convertParams)
				}

				// Check if markers-only path
				hasMarker := false
				if _, ok := elem.Attributes.Get("marker-start"); ok {
					hasMarker = true
				}
				if _, ok := elem.Attributes.Get("marker-end"); ok {
					hasMarker = true
				}
				isMarkersOnlyPath := false
				if hasMarker && includesVertices {
					allMoveTo := true
					for _, item := range data {
						if item.Command != 'm' && item.Command != 'M' {
							allMoveTo = false
							break
						}
					}
					isMarkersOnlyPath = allMoveTo
				}
				if isMarkersOnlyPath {
					data = append(data, path.ExtendedPathItem{
						Command: 'z',
						Args:    nil,
					})
				}

				// Convert back to PathDataItem for JS2Path
				pathDataItems := make([]path.PathDataItem, len(data))
				for i, item := range data {
					pathDataItems[i] = path.PathDataItem{
						Command: item.Command,
						Args:    item.Args,
					}
				}

				// When floatPrecision is false (disabled), pass -1 for no rounding
				outPrecision := floatPrecision
				if !floatPrecisionEnabled {
					outPrecision = -1
				}
				path.JS2Path(elem, pathDataItems, outPrecision, noSpaceAfterFlags)

				return nil
			},
		},
	}
}
