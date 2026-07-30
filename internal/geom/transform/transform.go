// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package transform provides SVG transform parsing, matrix operations,
// and decomposition, ported from SVGO's plugins/_transforms.js.
package transform

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/tools"
)

// Products in the ported arithmetic below are wrapped in explicit float64
// conversions. Go is free to contract an expression such as a*b + c*d into a
// single fused multiply-add, which carries more intermediate precision than the
// JavaScript being mirrored; the conversions force a rounding after each
// multiplication so results agree with SVGO to the last bit. Dropping them
// shifts decomposed rotation angles by an ULP, which is enough to flip a
// Math.round at a halfway value.

// TransformItem represents a single transform function with its parameters.
type TransformItem struct {
	Name string
	Data []float64
}

// TransformParams controls transform optimization and output formatting.
type TransformParams struct {
	ConvertToShorts    bool
	DegPrecision       *int // nil means not set
	FloatPrecision     int
	TransformPrecision int
	MatrixToTransform  bool
	ShortTranslate     bool
	ShortScale         bool
	ShortRotate        bool
	RemoveUseless      bool
	CollapseIntoOne    bool
	LeadingZero        bool
	NegativeExtraSpace bool
	NoSpaceAfterFlags  bool
}

var transformTypes = map[string]bool{
	"matrix":    true,
	"rotate":    true,
	"scale":     true,
	"skewX":     true,
	"skewY":     true,
	"translate": true,
}

var regTransformSplit = regexp.MustCompile(
	`\s*(matrix|translate|scale|rotate|skewX|skewY)\s*\(\s*(.+?)\s*\)[\s,]*`)
var regNumericValues = regexp.MustCompile(
	`[-+]?(?:\d*\.\d+|\d+\.?)(?:[eE][-+]?\d+)?`)

// Transform2JS converts a transform attribute string to a slice of TransformItems.
// Returns nil if the string is malformed, matching SVGO's empty-array result.
//
// The traversal mirrors SVGO's `transformString.split(regTransformSplit)`: the
// split regexp carries capture groups, so the resulting chunks alternate between
// the text between matches, the transform name and the parenthesised arguments.
// Numbers found in any chunk — including text trailing the last closing paren —
// are appended to the transform currently being built, and a bare transform name
// outside parentheses starts a new, empty one. That empty trailing transform is
// what makes the whole attribute malformed.
func Transform2JS(transformString string) []TransformItem {
	var transforms []TransformItem
	current := -1

	for _, chunk := range splitWithCaptures(regTransformSplit, transformString) {
		if chunk == "" {
			continue
		}
		if transformTypes[chunk] {
			transforms = append(transforms, TransformItem{Name: chunk})
			current = len(transforms) - 1
			continue
		}
		for _, numStr := range regNumericValues.FindAllString(chunk, -1) {
			if current >= 0 {
				transforms[current].Data = append(transforms[current].Data, jsNumber(numStr))
			}
		}
	}

	if current < 0 || len(transforms[current].Data) == 0 {
		return nil
	}

	return transforms
}

// splitWithCaptures reproduces JavaScript's String.prototype.split with a
// regexp containing capture groups: the text between matches interleaved with
// each match's captures, with non-participating groups yielding "".
func splitWithCaptures(re *regexp.Regexp, s string) []string {
	var out []string
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		out = append(out, s[last:m[0]])
		for g := 1; 2*g+1 < len(m); g++ {
			if m[2*g] < 0 {
				out = append(out, "")
			} else {
				out = append(out, s[m[2*g]:m[2*g+1]])
			}
		}
		last = m[1]
	}
	return append(out, s[last:])
}

// jsNumber converts a numeric literal to a float64 the way JavaScript's
// Number() does. Out-of-range magnitudes saturate to ±Inf (or 0) rather than
// being discarded, so a six-argument matrix always yields six values.
func jsNumber(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return math.NaN()
	}
	return v
}

// TransformsMultiply multiplies multiple transforms into a single matrix.
//
// The result always carries at least six values. SVGO leaves short matrices
// short and lets the missing arguments read as undefined, which every consumer
// then propagates as NaN; padding with NaN here is the same arithmetic without
// an out-of-range index on every read.
func TransformsMultiply(transforms []TransformItem) TransformItem {
	var result []float64
	for i := range transforms {
		m := transformToMatrix(&transforms[i])
		if i == 0 {
			result = m
			continue
		}
		result = MultiplyTransformMatrices(result, m)
	}

	return TransformItem{Name: "matrix", Data: padMatrix(result)}
}

// padMatrix returns a copy of data holding at least six values, filling any
// missing ones with NaN. Extra values are preserved.
func padMatrix(data []float64) []float64 {
	size := len(data)
	if size < 6 {
		size = 6
	}
	out := make([]float64, size)
	copy(out, data)
	for i := len(data); i < size; i++ {
		out[i] = math.NaN()
	}
	return out
}

// Math utilities for degree-based trig.
func rad(deg float64) float64 {
	return deg * math.Pi / 180
}

func deg(r float64) float64 {
	return r * 180 / math.Pi
}

func cosDeg(d float64) float64 {
	return math.Cos(rad(d))
}

func sinDeg(d float64) float64 {
	return math.Sin(rad(d))
}

func tanDeg(d float64) float64 {
	return math.Tan(rad(d))
}

// AcosDeg returns acos in degrees, rounded to floatPrecision.
func AcosDeg(val float64, floatPrecision int) float64 {
	return tools.ToFixed(deg(math.Acos(val)), floatPrecision)
}

// AsinDeg returns asin in degrees, rounded to floatPrecision.
func AsinDeg(val float64, floatPrecision int) float64 {
	return tools.ToFixed(deg(math.Asin(val)), floatPrecision)
}

// AtanDeg returns atan in degrees, rounded to floatPrecision.
func AtanDeg(val float64, floatPrecision int) float64 {
	return tools.ToFixed(deg(math.Atan(val)), floatPrecision)
}

// transformToMatrix converts a transform to a 6-element matrix [a, b, c, d, e, f].
// Arguments the attribute did not supply read as NaN, except where SVGO applies
// a `|| 0` or `?? data[0]` default.
func transformToMatrix(t *TransformItem) []float64 {
	if t.Name == "matrix" {
		return t.Data
	}

	data := t.Data
	switch t.Name {
	case "translate":
		return []float64{1, 0, 0, 1, tools.JSIndex(data, 0), tools.JSOr(tools.JSIndex(data, 1), 0)}

	case "scale":
		sy := tools.JSIndex(data, 0)
		if len(data) > 1 {
			sy = data[1]
		}
		return []float64{tools.JSIndex(data, 0), 0, 0, sy, 0, 0}

	case "rotate":
		angle := tools.JSIndex(data, 0)
		cos := cosDeg(angle)
		sin := sinDeg(angle)
		cx := tools.JSOr(tools.JSIndex(data, 1), 0)
		cy := tools.JSOr(tools.JSIndex(data, 2), 0)
		return []float64{
			cos, sin, -sin, cos,
			float64((1-cos)*cx) + float64(sin*cy),
			float64((1-cos)*cy) - float64(sin*cx),
		}

	case "skewX":
		return []float64{1, 0, tanDeg(tools.JSIndex(data, 0)), 1, 0, 0}

	case "skewY":
		return []float64{1, tanDeg(tools.JSIndex(data, 0)), 0, 1, 0, 0}
	}

	return []float64{1, 0, 0, 1, 0, 0} // identity
}

// MultiplyTransformMatrices multiplies two 2D affine transformation matrices.
// Each matrix is [a, b, c, d, e, f]; missing components contribute NaN.
func MultiplyTransformMatrices(a, b []float64) []float64 {
	a0, a1, a2, a3, a4, a5 := matrixComponents(a)
	b0, b1, b2, b3, b4, b5 := matrixComponents(b)
	return []float64{
		float64(a0*b0) + float64(a2*b1),
		float64(a1*b0) + float64(a3*b1),
		float64(a0*b2) + float64(a2*b3),
		float64(a1*b2) + float64(a3*b3),
		float64(a0*b4) + float64(a2*b5) + a4,
		float64(a1*b4) + float64(a3*b5) + a5,
	}
}

// matrixComponents destructures the six components of a matrix, yielding NaN for
// any the slice does not hold.
func matrixComponents(data []float64) (a, b, c, d, e, f float64) {
	return tools.JSIndex(data, 0), tools.JSIndex(data, 1), tools.JSIndex(data, 2),
		tools.JSIndex(data, 3), tools.JSIndex(data, 4), tools.JSIndex(data, 5)
}

// decomposeQRAB decomposes a matrix using QR decomposition (method A/B).
// Returns translate → rotate → scale → skewX, or nil if singular.
func decomposeQRAB(matrix *TransformItem) []TransformItem {
	a, b, c, d, e, f := matrixComponents(matrix.Data)

	delta := float64(a*d) - float64(b*c)
	if delta == 0 {
		return nil
	}

	r := math.Hypot(a, b)
	if r == 0 {
		return nil
	}

	var decomposition []TransformItem

	if !tools.JSFalsy(e) || !tools.JSFalsy(f) {
		decomposition = append(decomposition, TransformItem{
			Name: "translate", Data: []float64{e, f},
		})
	}

	cosOfRotationAngle := a / r
	if cosOfRotationAngle != 1 {
		rotationAngleRads := math.Acos(cosOfRotationAngle)
		if b < 0 {
			rotationAngleRads = -rotationAngleRads
		}
		decomposition = append(decomposition, TransformItem{
			Name: "rotate", Data: []float64{deg(rotationAngleRads), 0, 0},
		})
	}

	sx := r
	sy := delta / sx
	if sx != 1 || sy != 1 {
		decomposition = append(decomposition, TransformItem{
			Name: "scale", Data: []float64{sx, sy},
		})
	}

	acPlusBD := float64(a*c) + float64(b*d)
	if !tools.JSFalsy(acPlusBD) {
		decomposition = append(decomposition, TransformItem{
			Name: "skewX", Data: []float64{deg(math.Atan(acPlusBD / (float64(a*a) + float64(b*b))))},
		})
	}

	return decomposition
}

// decomposeQRCD decomposes a matrix using QR decomposition (method C/D).
// Returns translate → rotate → scale → skewY, or nil if singular.
func decomposeQRCD(matrix *TransformItem) []TransformItem {
	a, b, c, d, e, f := matrixComponents(matrix.Data)

	delta := float64(a*d) - float64(b*c)
	if delta == 0 {
		return nil
	}

	s := math.Hypot(c, d)
	if s == 0 {
		return nil
	}

	var decomposition []TransformItem

	if !tools.JSFalsy(e) || !tools.JSFalsy(f) {
		decomposition = append(decomposition, TransformItem{
			Name: "translate", Data: []float64{e, f},
		})
	}

	sign := 1.0
	if d < 0 {
		sign = -1.0
	}
	rotationAngleRads := math.Pi/2 - sign*math.Acos(-c/s)
	decomposition = append(decomposition, TransformItem{
		Name: "rotate", Data: []float64{deg(rotationAngleRads), 0, 0},
	})

	sx := delta / s
	sy := s
	if sx != 1 || sy != 1 {
		decomposition = append(decomposition, TransformItem{
			Name: "scale", Data: []float64{sx, sy},
		})
	}

	acPlusBD := float64(a*c) + float64(b*d)
	if !tools.JSFalsy(acPlusBD) {
		decomposition = append(decomposition, TransformItem{
			Name: "skewY", Data: []float64{deg(math.Atan(acPlusBD / (float64(c*c) + float64(d*d))))},
		})
	}

	return decomposition
}

// mergeTranslateAndRotate converts translate(tx,ty)rotate(a) to rotate(a,cx,cy).
func mergeTranslateAndRotate(tx, ty, a float64) TransformItem {
	rotationAngleRads := rad(a)
	d := 1 - math.Cos(rotationAngleRads)
	e := math.Sin(rotationAngleRads)
	cy := (float64(d*ty) + float64(e*tx)) / (float64(d*d) + float64(e*e))
	cx := (tx - float64(e*cy)) / d
	return TransformItem{Name: "rotate", Data: []float64{a, cx, cy}}
}

// isIdentityTransform checks if a transform is the identity.
func isIdentityTransform(t *TransformItem) bool {
	switch t.Name {
	case "rotate", "skewX", "skewY":
		return tools.JSIndex(t.Data, 0) == 0
	case "scale":
		return tools.JSIndex(t.Data, 0) == 1 && tools.JSIndex(t.Data, 1) == 1
	case "translate":
		return tools.JSIndex(t.Data, 0) == 0 && tools.JSIndex(t.Data, 1) == 0
	}
	return false
}

// createScaleTransform creates a scale transform, using short form if sx == sy.
func createScaleTransform(data []float64) TransformItem {
	end := 2
	if tools.JSIndex(data, 0) == tools.JSIndex(data, 1) {
		end = 1
	}
	return TransformItem{Name: "scale", Data: sliceData(data, 0, end)}
}

// sliceData copies data[start:end] with JavaScript Array.prototype.slice
// semantics, clamping end to the length of data.
func sliceData(data []float64, start, end int) []float64 {
	if end > len(data) {
		end = len(data)
	}
	if start > end {
		start = end
	}
	return append([]float64(nil), data[start:end]...)
}

// optimize optimizes a sequence of rounded transforms, removing identities
// and merging translate+rotate where possible.
func optimize(roundedTransforms, rawTransforms []TransformItem) []TransformItem {
	var optimized []TransformItem

	for index := 0; index < len(roundedTransforms); index++ {
		rt := &roundedTransforms[index]

		if isIdentityTransform(rt) {
			continue
		}

		data := rt.Data
		switch rt.Name {
		case "rotate":
			if angle := tools.JSIndex(data, 0); angle == 180 || angle == -180 {
				if index+1 < len(roundedTransforms) && roundedTransforms[index+1].Name == "scale" {
					next := &roundedTransforms[index+1]
					negated := make([]float64, len(next.Data))
					for i, v := range next.Data {
						negated[i] = -v
					}
					optimized = append(optimized, createScaleTransform(negated))
					index++
				} else {
					optimized = append(optimized, TransformItem{
						Name: "scale", Data: []float64{-1},
					})
				}
				continue
			}
			end := 1
			if !tools.JSFalsy(tools.JSIndex(data, 1)) || !tools.JSFalsy(tools.JSIndex(data, 2)) {
				end = 3
			}
			optimized = append(optimized, TransformItem{
				Name: "rotate", Data: sliceData(data, 0, end),
			})

		case "scale":
			optimized = append(optimized, createScaleTransform(data))

		case "skewX", "skewY":
			optimized = append(optimized, TransformItem{
				Name: rt.Name, Data: []float64{tools.JSIndex(data, 0)},
			})

		case "translate":
			if index+1 < len(roundedTransforms) {
				next := &roundedTransforms[index+1]
				nextAngle := tools.JSIndex(next.Data, 0)
				if next.Name == "rotate" &&
					nextAngle != 180 && nextAngle != -180 && nextAngle != 0 &&
					tools.JSIndex(next.Data, 1) == 0 && tools.JSIndex(next.Data, 2) == 0 {
					rawData := rawTransforms[index].Data
					optimized = append(optimized, mergeTranslateAndRotate(
						tools.JSIndex(rawData, 0), tools.JSIndex(rawData, 1),
						tools.JSIndex(rawTransforms[index+1].Data, 0)))
					index++
					continue
				}
			}
			end := 1
			if !tools.JSFalsy(tools.JSIndex(data, 1)) {
				end = 2
			}
			optimized = append(optimized, TransformItem{
				Name: "translate", Data: sliceData(data, 0, end),
			})
		}
	}

	if len(optimized) == 0 {
		return []TransformItem{{Name: "scale", Data: []float64{1}}}
	}
	return optimized
}

// MatrixToTransform decomposes a matrix into simple transforms and optimizes.
func MatrixToTransform(origMatrix *TransformItem, params *TransformParams) []TransformItem {
	decompositions := [][]TransformItem{}

	if qrab := decomposeQRAB(origMatrix); qrab != nil {
		decompositions = append(decompositions, qrab)
	}
	if qrcd := decomposeQRCD(origMatrix); qrcd != nil {
		decompositions = append(decompositions, qrcd)
	}

	var shortest []TransformItem
	shortestLen := math.MaxInt64

	for _, decomposition := range decompositions {
		rounded := make([]TransformItem, len(decomposition))
		for i, item := range decomposition {
			dataCopy := make([]float64, len(item.Data))
			copy(dataCopy, item.Data)
			rounded[i] = TransformItem{Name: item.Name, Data: dataCopy}
			RoundTransform(&rounded[i], params)
		}

		optimized := optimize(rounded, decomposition)
		str := JS2Transform(optimized, params)
		if len(str) < shortestLen {
			shortest = optimized
			shortestLen = len(str)
		}
	}

	if shortest == nil {
		return []TransformItem{*origMatrix}
	}
	return shortest
}

// TransformArc applies a transformation matrix to an elliptical arc.
// cursor is [x, y], arc is [rx, ry, angle, large-arc, sweep, x, y].
// transform is [a, b, c, d, e, f].
func TransformArc(cursor [2]float64, arc []float64, transform []float64) []float64 {
	x := arc[5] - cursor[0]
	y := arc[6] - cursor[1]
	a := arc[0]
	b := arc[1]
	rot := arc[2] * math.Pi / 180
	cos := math.Cos(rot)
	sin := math.Sin(rot)

	// skip if radius is 0
	if a > 0 && b > 0 {
		h := math.Pow(float64(x*cos)+float64(y*sin), 2)/(4*a*a) +
			math.Pow(float64(y*cos)-float64(x*sin), 2)/(4*b*b)
		if h > 1 {
			h = math.Sqrt(h)
			a *= h
			b *= h
		}
	}

	ellipse := []float64{a * cos, a * sin, -b * sin, b * cos, 0, 0}
	m := MultiplyTransformMatrices(transform, ellipse)

	lastCol := float64(m[2]*m[2]) + float64(m[3]*m[3])
	squareSum := float64(m[0]*m[0]) + float64(m[1]*m[1]) + lastCol
	root := math.Hypot(m[0]-m[3], m[1]+m[2]) * math.Hypot(m[0]+m[3], m[1]-m[2])

	if root == 0 {
		// circle
		arc[0] = math.Sqrt(squareSum / 2)
		arc[1] = arc[0]
		arc[2] = 0
	} else {
		majorAxisSqr := (squareSum + root) / 2
		minorAxisSqr := (squareSum - root) / 2
		major := math.Abs(majorAxisSqr-lastCol) > 1e-6
		sub := majorAxisSqr - lastCol
		if !major {
			sub = minorAxisSqr - lastCol
		}
		rowsSum := float64(m[0]*m[2]) + float64(m[1]*m[3])
		term1 := float64(m[0]*sub) + float64(m[2]*rowsSum)
		term2 := float64(m[1]*sub) + float64(m[3]*rowsSum)

		arc[0] = math.Sqrt(majorAxisSqr)
		arc[1] = math.Sqrt(minorAxisSqr)

		sign := -1.0
		if major {
			if term2 >= 0 {
				sign = 1.0
			}
		} else {
			if term1 <= 0 {
				sign = 1.0
			}
		}
		cosVal := term1
		if !major {
			cosVal = term2
		}
		arc[2] = sign * math.Acos(cosVal/math.Hypot(term1, term2)) * 180 / math.Pi
	}

	// Flip the sweep flag if coordinates are being flipped horizontally XOR vertically
	if (transform[0] < 0) != (transform[3] < 0) {
		arc[4] = 1 - arc[4]
	}

	return arc
}

// RoundTransform rounds transform data based on the params.
func RoundTransform(t *TransformItem, params *TransformParams) {
	switch t.Name {
	case "translate":
		t.Data = floatRound(t.Data, params)
	case "rotate":
		degPart := degRound(sliceData(t.Data, 0, 1), params)
		floatPart := floatRound(sliceData(t.Data, 1, len(t.Data)), params)
		t.Data = append(degPart, floatPart...)
	case "skewX", "skewY":
		t.Data = degRound(t.Data, params)
	case "scale":
		t.Data = transformRound(t.Data, params)
	case "matrix":
		transformPart := transformRound(sliceData(t.Data, 0, 4), params)
		floatPart := floatRound(sliceData(t.Data, 4, len(t.Data)), params)
		t.Data = append(transformPart, floatPart...)
	}
}

func degRound(data []float64, params *TransformParams) []float64 {
	if params.DegPrecision != nil && *params.DegPrecision >= 1 && params.FloatPrecision < 20 {
		return smartRound(*params.DegPrecision, data)
	}
	return roundSlice(data)
}

func floatRound(data []float64, params *TransformParams) []float64 {
	if params.FloatPrecision >= 1 && params.FloatPrecision < 20 {
		return smartRound(params.FloatPrecision, data)
	}
	return roundSlice(data)
}

func transformRound(data []float64, params *TransformParams) []float64 {
	if params.TransformPrecision >= 1 && params.FloatPrecision < 20 {
		return smartRound(params.TransformPrecision, data)
	}
	return roundSlice(data)
}

func roundSlice(data []float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = tools.JSRound(v)
	}
	return result
}

// smartRound decreases accuracy of floating-point numbers keeping a specified
// number of decimals. Smart rounds values like 2.349 to 2.35.
//
// SVGO mixes both rounding helpers here: the guard uses tools.toFixed
// (Math.round semantics) while the tolerance and every produced value use the
// native Number.prototype.toFixed, which breaks ties away from zero. The split
// is what decides which way a negative halfway component such as -0.0625 goes.
func smartRound(precision int, data []float64) []float64 {
	result := make([]float64, len(data))
	copy(result, data)

	tolerance := tools.NativeToFixed(math.Pow(0.1, float64(precision)), precision)

	for i := len(result) - 1; i >= 0; i-- {
		fixed := tools.ToFixed(result[i], precision)
		if fixed != result[i] {
			rounded := tools.NativeToFixed(result[i], precision-1)
			diff := tools.NativeToFixed(math.Abs(rounded-result[i]), precision+1)
			if diff >= tolerance {
				result[i] = tools.NativeToFixed(result[i], precision)
			} else {
				result[i] = rounded
			}
		}
	}

	return result
}

// JS2Transform converts transforms to an SVG transform attribute string.
func JS2Transform(transforms []TransformItem, params *TransformParams) string {
	var sb strings.Builder

	for _, t := range transforms {
		tCopy := TransformItem{Name: t.Name, Data: make([]float64, len(t.Data))}
		copy(tCopy.Data, t.Data)
		RoundTransform(&tCopy, params)
		sb.WriteString(tCopy.Name)
		sb.WriteByte('(')
		sb.WriteString(tools.CleanupOutData(tCopy.Data, &tools.CleanupOutDataParams{
			LeadingZero:        params.LeadingZero,
			NegativeExtraSpace: params.NegativeExtraSpace,
			NoSpaceAfterFlags:  params.NoSpaceAfterFlags,
		}, 0))
		sb.WriteByte(')')
	}

	return sb.String()
}
