// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package converttransform implements the convertTransform SVGO plugin.
// It collapses multiple transformations and optimizes them — converting
// matrices to short aliases, merging transforms, and removing useless ones.
package converttransform

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/geom/transform"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertTransform",
		Description: "collapses multiple transformations and optimizes it",
		Fn:          fn,
	})
}

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	convertToShorts := getBoolParam(params, "convertToShorts", true)
	degPrecision := getOptIntParam(params, "degPrecision")
	floatPrecision := getIntParam(params, "floatPrecision", 3)
	transformPrecision := getIntParam(params, "transformPrecision", 5)
	matrixToTransformParam := getBoolParam(params, "matrixToTransform", true)
	shortTranslate := getBoolParam(params, "shortTranslate", true)
	shortScale := getBoolParam(params, "shortScale", true)
	shortRotate := getBoolParam(params, "shortRotate", true)
	removeUselessParam := getBoolParam(params, "removeUseless", true)
	collapseIntoOne := getBoolParam(params, "collapseIntoOne", true)
	leadingZero := getBoolParam(params, "leadingZero", true)
	negativeExtraSpace := getBoolParam(params, "negativeExtraSpace", false)

	newParams := &transformParams{
		convertToShorts:    convertToShorts,
		degPrecision:       degPrecision,
		floatPrecision:     floatPrecision,
		transformPrecision: transformPrecision,
		matrixToTransform:  matrixToTransformParam,
		shortTranslate:     shortTranslate,
		shortScale:         shortScale,
		shortRotate:        shortRotate,
		removeUseless:      removeUselessParam,
		collapseIntoOne:    collapseIntoOne,
		leadingZero:        leadingZero,
		negativeExtraSpace: negativeExtraSpace,
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, _ svgast.Parent) error {
				elem := node.(*svgast.Element)

				if _, ok := elem.Attributes.Get("transform"); ok {
					convertTransformAttr(elem, "transform", newParams)
				}
				if _, ok := elem.Attributes.Get("gradientTransform"); ok {
					convertTransformAttr(elem, "gradientTransform", newParams)
				}
				if _, ok := elem.Attributes.Get("patternTransform"); ok {
					convertTransformAttr(elem, "patternTransform", newParams)
				}

				return nil
			},
		},
	}
}

// transformParams holds all the parameters for transform optimization.
// This mirrors SVGO's TransformParams typedef.
type transformParams struct {
	convertToShorts    bool
	degPrecision       *int // nil means not explicitly set
	floatPrecision     int
	transformPrecision int
	matrixToTransform  bool
	shortTranslate     bool
	shortScale         bool
	shortRotate        bool
	removeUseless      bool
	collapseIntoOne    bool
	leadingZero        bool
	negativeExtraSpace bool
}

// clone creates a shallow copy of the params (for definePrecision modification).
func (p *transformParams) clone() *transformParams {
	c := *p
	if p.degPrecision != nil {
		v := *p.degPrecision
		c.degPrecision = &v
	}
	return &c
}

// toTransformParams converts to the geom/transform package's TransformParams.
func (p *transformParams) toTransformParams() *transform.TransformParams {
	return &transform.TransformParams{
		ConvertToShorts:    p.convertToShorts,
		DegPrecision:       p.degPrecision,
		FloatPrecision:     p.floatPrecision,
		TransformPrecision: p.transformPrecision,
		MatrixToTransform:  p.matrixToTransform,
		ShortTranslate:     p.shortTranslate,
		ShortScale:         p.shortScale,
		ShortRotate:        p.shortRotate,
		RemoveUseless:      p.removeUseless,
		CollapseIntoOne:    p.collapseIntoOne,
		LeadingZero:        p.leadingZero,
		NegativeExtraSpace: p.negativeExtraSpace,
	}
}

// convertTransformAttr processes a single transform attribute on an element.
func convertTransformAttr(elem *svgast.Element, attrName string, params *transformParams) {
	attrVal, _ := elem.Attributes.Get(attrName)
	data := transform.Transform2JS(attrVal)
	// SVGO's transform2js returns [] (empty array) when no transforms are found,
	// and the algorithm proceeds through all steps, eventually deleting the attribute
	// when data.length is 0. We match this behavior by treating nil as empty slice.
	if data == nil {
		data = []transform.TransformItem{}
	}

	params = definePrecision(data, params)

	if params.collapseIntoOne && len(data) > 1 {
		result := transform.TransformsMultiply(data)
		data = []transform.TransformItem{result}
	}

	if params.convertToShorts {
		data = convertToShorts(data, params)
	} else {
		tp := params.toTransformParams()
		for i := range data {
			transform.RoundTransform(&data[i], tp)
		}
	}

	if params.removeUseless {
		data = removeUseless(data)
	}

	if len(data) > 0 {
		elem.Attributes.Set(attrName, js2transform(data, params))
	} else {
		elem.Attributes.Delete(attrName)
	}
}

// definePrecision adapts precision settings based on matrix data.
// This matches SVGO's definePrecision function exactly.
func definePrecision(data []transform.TransformItem, params *transformParams) *transformParams {
	newParams := params.clone()

	var matrixData []float64
	for _, item := range data {
		if item.Name == "matrix" && len(item.Data) >= 4 {
			matrixData = append(matrixData, item.Data[:4]...)
		}
	}

	numberOfDigits := newParams.transformPrecision

	// Limit transform precision with matrix one
	if len(matrixData) > 0 {
		maxFloatDigits := 0
		for _, n := range matrixData {
			fd := floatDigits(n)
			if fd > maxFloatDigits {
				maxFloatDigits = fd
			}
		}
		// Math.max.apply(Math, matrixData.map(floatDigits)) || newParams.transformPrecision
		if maxFloatDigits == 0 {
			maxFloatDigits = newParams.transformPrecision
		}
		if maxFloatDigits < newParams.transformPrecision {
			newParams.transformPrecision = maxFloatDigits
		}

		// Count total digits in numbers
		maxTotalDigits := 0
		for _, n := range matrixData {
			td := totalDigits(n)
			if td > maxTotalDigits {
				maxTotalDigits = td
			}
		}
		numberOfDigits = maxTotalDigits
	}

	// No sense in angle precision more than number of significant digits in matrix
	if newParams.degPrecision == nil {
		degPrec := numberOfDigits - 2
		if degPrec < 0 {
			degPrec = 0
		}
		if newParams.floatPrecision < degPrec {
			degPrec = newParams.floatPrecision
		}
		newParams.degPrecision = &degPrec
	}

	return newParams
}

// regDigits matches all non-digit characters for counting total digits.
var regDigits = regexp.MustCompile(`\D+`)

// floatDigits returns the number of digits after the decimal point.
// Example: 0.125 → 3
func floatDigits(n float64) int {
	str := strconv.FormatFloat(n, 'f', -1, 64)
	dotIdx := strings.Index(str, ".")
	if dotIdx < 0 {
		return 0
	}
	return len(str) - dotIdx - 1
}

// totalDigits returns the total number of digits in the number's string representation.
// Example: 123.45 → 5
func totalDigits(n float64) int {
	str := strconv.FormatFloat(n, 'f', -1, 64)
	digits := regDigits.ReplaceAllString(str, "")
	return len(digits)
}

// convertToShorts converts transforms to shorthand alternatives.
// This matches SVGO's convertToShorts function.
func convertToShorts(transforms []transform.TransformItem, params *transformParams) []transform.TransformItem {
	tp := params.toTransformParams()

	for i := 0; i < len(transforms); i++ {
		t := &transforms[i]

		// Convert matrix to short aliases
		if params.matrixToTransform && t.Name == "matrix" {
			decomposed := transform.MatrixToTransform(t, tp)
			decomposedStr := transform.JS2Transform(decomposed, tp)
			originalStr := transform.JS2Transform([]transform.TransformItem{*t}, tp)
			if len(decomposedStr) <= len(originalStr) {
				// Splice: replace transforms[i] with decomposed...
				newTransforms := make([]transform.TransformItem, 0, len(transforms)-1+len(decomposed))
				newTransforms = append(newTransforms, transforms[:i]...)
				newTransforms = append(newTransforms, decomposed...)
				newTransforms = append(newTransforms, transforms[i+1:]...)
				transforms = newTransforms
			}
			if i < len(transforms) {
				t = &transforms[i]
			} else {
				continue
			}
		}

		// Fixed-point numbers rounding
		transform.RoundTransform(t, tp)

		// Convert long translate to short: translate(10 0) → translate(10)
		if params.shortTranslate &&
			t.Name == "translate" &&
			len(t.Data) == 2 &&
			t.Data[1] == 0 {
			t.Data = t.Data[:1]
		}

		// Convert long scale to short: scale(2 2) → scale(2)
		if params.shortScale &&
			t.Name == "scale" &&
			len(t.Data) == 2 &&
			t.Data[0] == t.Data[1] {
			t.Data = t.Data[:1]
		}

		// Convert long rotate to short:
		// translate(cx cy) rotate(a) translate(-cx -cy) → rotate(a cx cy)
		if params.shortRotate && i >= 2 &&
			transforms[i-2].Name == "translate" &&
			transforms[i-1].Name == "rotate" &&
			transforms[i].Name == "translate" &&
			len(transforms[i-2].Data) >= 2 &&
			len(transforms[i].Data) >= 2 &&
			transforms[i-2].Data[0] == -transforms[i].Data[0] &&
			transforms[i-2].Data[1] == -transforms[i].Data[1] {

			merged := transform.TransformItem{
				Name: "rotate",
				Data: []float64{
					transforms[i-1].Data[0],
					transforms[i-2].Data[0],
					transforms[i-2].Data[1],
				},
			}

			// Splice: replace transforms[i-2..i] with merged
			newTransforms := make([]transform.TransformItem, 0, len(transforms)-2)
			newTransforms = append(newTransforms, transforms[:i-2]...)
			newTransforms = append(newTransforms, merged)
			newTransforms = append(newTransforms, transforms[i+1:]...)
			transforms = newTransforms

			// Splice compensation
			i -= 2
		}
	}

	return transforms
}

// removeUseless removes identity transforms.
// This matches SVGO's removeUseless function.
func removeUseless(transforms []transform.TransformItem) []transform.TransformItem {
	var result []transform.TransformItem
	for _, t := range transforms {
		if isUseless(t) {
			continue
		}
		result = append(result, t)
	}
	return result
}

// isUseless checks if a transform is an identity (no-op).
func isUseless(t transform.TransformItem) bool {
	name := t.Name
	data := t.Data

	// translate(0), rotate(0[, cx, cy]), skewX(0), skewY(0)
	if (name == "translate" || name == "rotate" || name == "skewX" || name == "skewY") &&
		(len(data) == 1 || name == "rotate") &&
		data[0] == 0 {
		return true
	}

	// translate(0, 0)
	if name == "translate" &&
		len(data) >= 2 &&
		data[0] == 0 &&
		data[1] == 0 {
		return true
	}

	// scale(1) or scale(1, 1)
	if name == "scale" &&
		data[0] == 1 &&
		(len(data) < 2 || data[1] == 1) {
		return true
	}

	// matrix(1 0 0 1 0 0) — identity matrix
	if name == "matrix" &&
		len(data) >= 6 &&
		data[0] == 1 &&
		data[3] == 1 &&
		data[1] == 0 &&
		data[2] == 0 &&
		data[4] == 0 &&
		data[5] == 0 {
		return true
	}

	return false
}

// js2transform converts transforms to an SVG transform attribute string.
// This is a local wrapper that uses the geom/transform package's JS2Transform.
func js2transform(transforms []transform.TransformItem, params *transformParams) string {
	return transform.JS2Transform(transforms, params.toTransformParams())
}

// --- Parameter extraction helpers ---

func getBoolParam(params map[string]any, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

func getIntParam(params map[string]any, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

func getOptIntParam(params map[string]any, key string) *int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			val := int(n)
			return &val
		case int:
			return &n
		}
	}
	return nil
}
