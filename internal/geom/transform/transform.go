// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package transform provides SVG transform parsing, matrix operations,
// and decomposition, ported from SVGO's plugins/_transforms.js.
package transform

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/tools"
)

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
// Returns nil if the string is malformed.
func Transform2JS(transformString string) []TransformItem {
	var transforms []TransformItem
	var current *TransformItem

	parts := regTransformSplit.Split(transformString, -1)
	matches := regTransformSplit.FindAllStringSubmatch(transformString, -1)

	// The split/match approach: matches give us transform names and their param strings
	_ = parts
	for _, m := range matches {
		name := m[1]
		paramStr := m[2]
		if !transformTypes[name] {
			continue
		}
		item := TransformItem{Name: name}
		nums := regNumericValues.FindAllString(paramStr, -1)
		for _, numStr := range nums {
			if v, err := strconv.ParseFloat(numStr, 64); err == nil {
				item.Data = append(item.Data, v)
			}
		}
		transforms = append(transforms, item)
		current = &transforms[len(transforms)-1]
	}

	if current == nil || len(current.Data) == 0 {
		return nil
	}

	return transforms
}

// TransformsMultiply multiplies multiple transforms into a single matrix.
func TransformsMultiply(transforms []TransformItem) TransformItem {
	if len(transforms) == 0 {
		return TransformItem{Name: "matrix", Data: nil}
	}

	result := transformToMatrix(&transforms[0])
	for i := 1; i < len(transforms); i++ {
		m := transformToMatrix(&transforms[i])
		result = MultiplyTransformMatrices(result, m)
	}

	return TransformItem{Name: "matrix", Data: result}
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
func transformToMatrix(t *TransformItem) []float64 {
	if t.Name == "matrix" {
		return t.Data
	}

	switch t.Name {
	case "translate":
		ty := 0.0
		if len(t.Data) > 1 {
			ty = t.Data[1]
		}
		return []float64{1, 0, 0, 1, t.Data[0], ty}

	case "scale":
		sy := t.Data[0]
		if len(t.Data) > 1 {
			sy = t.Data[1]
		}
		return []float64{t.Data[0], 0, 0, sy, 0, 0}

	case "rotate":
		cos := cosDeg(t.Data[0])
		sin := sinDeg(t.Data[0])
		cx := 0.0
		cy := 0.0
		if len(t.Data) > 1 {
			cx = t.Data[1]
		}
		if len(t.Data) > 2 {
			cy = t.Data[2]
		}
		return []float64{
			cos, sin, -sin, cos,
			(1-cos)*cx + sin*cy,
			(1-cos)*cy - sin*cx,
		}

	case "skewX":
		return []float64{1, 0, tanDeg(t.Data[0]), 1, 0, 0}

	case "skewY":
		return []float64{1, tanDeg(t.Data[0]), 0, 1, 0, 0}
	}

	return []float64{1, 0, 0, 1, 0, 0} // identity
}

// MultiplyTransformMatrices multiplies two 2D affine transformation matrices.
// Each matrix is [a, b, c, d, e, f].
func MultiplyTransformMatrices(a, b []float64) []float64 {
	return []float64{
		a[0]*b[0] + a[2]*b[1],
		a[1]*b[0] + a[3]*b[1],
		a[0]*b[2] + a[2]*b[3],
		a[1]*b[2] + a[3]*b[3],
		a[0]*b[4] + a[2]*b[5] + a[4],
		a[1]*b[4] + a[3]*b[5] + a[5],
	}
}

// decomposeQRAB decomposes a matrix using QR decomposition (method A/B).
// Returns translate → rotate → scale → skewX, or nil if singular.
func decomposeQRAB(matrix *TransformItem) []TransformItem {
	data := matrix.Data
	a, b, c, d, e, f := data[0], data[1], data[2], data[3], data[4], data[5]

	delta := a*d - b*c
	if delta == 0 {
		return nil
	}

	r := math.Hypot(a, b)
	if r == 0 {
		return nil
	}

	var decomposition []TransformItem

	if e != 0 || f != 0 {
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

	acPlusBD := a*c + b*d
	if acPlusBD != 0 {
		decomposition = append(decomposition, TransformItem{
			Name: "skewX", Data: []float64{deg(math.Atan(acPlusBD / (a*a + b*b)))},
		})
	}

	return decomposition
}

// decomposeQRCD decomposes a matrix using QR decomposition (method C/D).
// Returns translate → rotate → scale → skewY, or nil if singular.
func decomposeQRCD(matrix *TransformItem) []TransformItem {
	data := matrix.Data
	a, b, c, d, e, f := data[0], data[1], data[2], data[3], data[4], data[5]

	delta := a*d - b*c
	if delta == 0 {
		return nil
	}

	s := math.Hypot(c, d)
	if s == 0 {
		return nil
	}

	var decomposition []TransformItem

	if e != 0 || f != 0 {
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

	acPlusBD := a*c + b*d
	if acPlusBD != 0 {
		decomposition = append(decomposition, TransformItem{
			Name: "skewY", Data: []float64{deg(math.Atan(acPlusBD / (c*c + d*d)))},
		})
	}

	return decomposition
}

// mergeTranslateAndRotate converts translate(tx,ty)rotate(a) to rotate(a,cx,cy).
func mergeTranslateAndRotate(tx, ty, a float64) TransformItem {
	rotationAngleRads := rad(a)
	d := 1 - math.Cos(rotationAngleRads)
	e := math.Sin(rotationAngleRads)
	cy := (d*ty + e*tx) / (d*d + e*e)
	cx := (tx - e*cy) / d
	return TransformItem{Name: "rotate", Data: []float64{a, cx, cy}}
}

// isIdentityTransform checks if a transform is the identity.
func isIdentityTransform(t *TransformItem) bool {
	switch t.Name {
	case "rotate", "skewX", "skewY":
		return t.Data[0] == 0
	case "scale":
		return t.Data[0] == 1 && (len(t.Data) < 2 || t.Data[1] == 1)
	case "translate":
		return t.Data[0] == 0 && (len(t.Data) < 2 || t.Data[1] == 0)
	}
	return false
}

// createScaleTransform creates a scale transform, using short form if sx == sy.
func createScaleTransform(data []float64) TransformItem {
	if len(data) >= 2 && data[0] == data[1] {
		return TransformItem{Name: "scale", Data: []float64{data[0]}}
	}
	result := make([]float64, len(data))
	copy(result, data)
	if len(result) > 2 {
		result = result[:2]
	}
	return TransformItem{Name: "scale", Data: result}
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
			if data[0] == 180 || data[0] == -180 {
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
			if len(data) >= 3 && (data[1] != 0 || data[2] != 0) {
				end = 3
			}
			optimized = append(optimized, TransformItem{
				Name: "rotate", Data: append([]float64(nil), data[:end]...),
			})

		case "scale":
			optimized = append(optimized, createScaleTransform(data))

		case "skewX", "skewY":
			optimized = append(optimized, TransformItem{
				Name: rt.Name, Data: []float64{data[0]},
			})

		case "translate":
			if index+1 < len(roundedTransforms) {
				next := &roundedTransforms[index+1]
				if next.Name == "rotate" &&
					next.Data[0] != 180 && next.Data[0] != -180 && next.Data[0] != 0 &&
					len(next.Data) >= 3 && next.Data[1] == 0 && next.Data[2] == 0 {
					rawData := rawTransforms[index].Data
					optimized = append(optimized,
						mergeTranslateAndRotate(rawData[0], rawData[1], rawTransforms[index+1].Data[0]))
					index++
					continue
				}
			}
			end := 1
			if len(data) >= 2 && data[1] != 0 {
				end = 2
			}
			optimized = append(optimized, TransformItem{
				Name: "translate", Data: append([]float64(nil), data[:end]...),
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
		h := math.Pow(x*cos+y*sin, 2)/(4*a*a) +
			math.Pow(y*cos-x*sin, 2)/(4*b*b)
		if h > 1 {
			h = math.Sqrt(h)
			a *= h
			b *= h
		}
	}

	ellipse := []float64{a * cos, a * sin, -b * sin, b * cos, 0, 0}
	m := MultiplyTransformMatrices(transform, ellipse)

	lastCol := m[2]*m[2] + m[3]*m[3]
	squareSum := m[0]*m[0] + m[1]*m[1] + lastCol
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
		rowsSum := m[0]*m[2] + m[1]*m[3]
		term1 := m[0]*sub + m[2]*rowsSum
		term2 := m[1]*sub + m[3]*rowsSum

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
		if len(t.Data) >= 1 {
			degPart := degRound(t.Data[:1], params)
			floatPart := floatRound(t.Data[1:], params)
			t.Data = append(degPart, floatPart...)
		}
	case "skewX", "skewY":
		t.Data = degRound(t.Data, params)
	case "scale":
		t.Data = transformRound(t.Data, params)
	case "matrix":
		if len(t.Data) >= 6 {
			transformPart := transformRound(t.Data[:4], params)
			floatPart := floatRound(t.Data[4:], params)
			t.Data = append(transformPart, floatPart...)
		}
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
		result[i] = math.Round(v)
	}
	return result
}

// smartRound decreases accuracy of floating-point numbers keeping a specified
// number of decimals. Smart rounds values like 2.349 to 2.35.
func smartRound(precision int, data []float64) []float64 {
	result := make([]float64, len(data))
	copy(result, data)

	tolerance := math.Pow(0.1, float64(precision))
	tolerance = tools.ToFixed(tolerance, precision)

	for i := len(result) - 1; i >= 0; i-- {
		fixed := tools.ToFixed(result[i], precision)
		if fixed != result[i] {
			rounded := tools.ToFixed(result[i], precision-1)
			diff := tools.ToFixed(math.Abs(rounded-result[i]), precision+1)
			if diff >= tolerance {
				result[i] = tools.ToFixed(result[i], precision)
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
