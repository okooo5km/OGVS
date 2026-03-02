// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package convertshapetopath implements the convertShapeToPath SVGO plugin.
// It converts basic shapes to more compact path form.
package convertshapetopath

import (
	"math"
	"regexp"
	"strconv"

	"github.com/okooo5km/ogvs/internal/geom/path"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "convertShapeToPath",
		Description: "converts basic shapes to more compact path form",
		Fn:          fn,
	})
}

var regNumber = regexp.MustCompile(`[-+]?(?:\d*\.\d+|\d+\.?)(?:[eE][-+]?\d+)?`)

func fn(_ *svgast.Root, params map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	convertArcs := false
	precision := -1 // -1 means no rounding

	if v, ok := params["convertArcs"].(bool); ok {
		convertArcs = v
	}
	if v, ok := params["floatPrecision"].(float64); ok {
		precision = int(v)
	}

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				switch elem.Name {
				case "rect":
					convertRect(elem, precision)
				case "line":
					convertLine(elem, precision)
				case "polyline":
					convertPolyline(elem, parent, precision)
				case "polygon":
					convertPolygon(elem, parent, precision)
				case "circle":
					if convertArcs {
						convertCircle(elem, precision)
					}
				case "ellipse":
					if convertArcs {
						convertEllipse(elem, precision)
					}
				}

				return nil
			},
		},
	}
}

func convertRect(elem *svgast.Element, precision int) {
	if !elem.Attributes.Has("width") || !elem.Attributes.Has("height") {
		return
	}
	if elem.Attributes.Has("rx") || elem.Attributes.Has("ry") {
		return
	}

	x := getNumAttr(elem, "x", 0)
	y := getNumAttr(elem, "y", 0)
	width := getNumAttr(elem, "width", 0)
	height := getNumAttr(elem, "height", 0)

	if math.IsNaN(x-y+width-height) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(width, 0) || math.IsInf(height, 0) {
		return
	}

	pathData := []path.PathDataItem{
		{Command: 'M', Args: []float64{x, y}},
		{Command: 'H', Args: []float64{x + width}},
		{Command: 'V', Args: []float64{y + height}},
		{Command: 'H', Args: []float64{x}},
		{Command: 'z', Args: []float64{}},
	}

	elem.Name = "path"
	elem.Attributes.Set("d", path.StringifyPathData(&path.StringifyPathDataOptions{
		PathData:  pathData,
		Precision: precision,
	}))
	elem.Attributes.Delete("x")
	elem.Attributes.Delete("y")
	elem.Attributes.Delete("width")
	elem.Attributes.Delete("height")
}

func convertLine(elem *svgast.Element, precision int) {
	x1 := getNumAttr(elem, "x1", 0)
	y1 := getNumAttr(elem, "y1", 0)
	x2 := getNumAttr(elem, "x2", 0)
	y2 := getNumAttr(elem, "y2", 0)

	if math.IsNaN(x1 - y1 + x2 - y2) {
		return
	}

	pathData := []path.PathDataItem{
		{Command: 'M', Args: []float64{x1, y1}},
		{Command: 'L', Args: []float64{x2, y2}},
	}

	elem.Name = "path"
	elem.Attributes.Set("d", path.StringifyPathData(&path.StringifyPathDataOptions{
		PathData:  pathData,
		Precision: precision,
	}))
	elem.Attributes.Delete("x1")
	elem.Attributes.Delete("y1")
	elem.Attributes.Delete("x2")
	elem.Attributes.Delete("y2")
}

func convertPolyline(elem *svgast.Element, parent svgast.Parent, precision int) {
	pointsStr, ok := elem.Attributes.Get("points")
	if !ok {
		return
	}

	coords := parsePoints(pointsStr)
	if len(coords) < 4 {
		svgast.DetachNodeFromParent(elem, parent)
		return
	}

	pathData := buildPolyPathData(coords, false)

	elem.Name = "path"
	elem.Attributes.Set("d", path.StringifyPathData(&path.StringifyPathDataOptions{
		PathData:  pathData,
		Precision: precision,
	}))
	elem.Attributes.Delete("points")
}

func convertPolygon(elem *svgast.Element, parent svgast.Parent, precision int) {
	pointsStr, ok := elem.Attributes.Get("points")
	if !ok {
		return
	}

	coords := parsePoints(pointsStr)
	if len(coords) < 4 {
		svgast.DetachNodeFromParent(elem, parent)
		return
	}

	pathData := buildPolyPathData(coords, true)

	elem.Name = "path"
	elem.Attributes.Set("d", path.StringifyPathData(&path.StringifyPathDataOptions{
		PathData:  pathData,
		Precision: precision,
	}))
	elem.Attributes.Delete("points")
}

func convertCircle(elem *svgast.Element, precision int) {
	cx := getNumAttr(elem, "cx", 0)
	cy := getNumAttr(elem, "cy", 0)
	r := getNumAttr(elem, "r", 0)

	if math.IsNaN(cx - cy + r) {
		return
	}

	pathData := []path.PathDataItem{
		{Command: 'M', Args: []float64{cx, cy - r}},
		{Command: 'A', Args: []float64{r, r, 0, 1, 0, cx, cy + r}},
		{Command: 'A', Args: []float64{r, r, 0, 1, 0, cx, cy - r}},
		{Command: 'z', Args: []float64{}},
	}

	elem.Name = "path"
	elem.Attributes.Set("d", path.StringifyPathData(&path.StringifyPathDataOptions{
		PathData:  pathData,
		Precision: precision,
	}))
	elem.Attributes.Delete("cx")
	elem.Attributes.Delete("cy")
	elem.Attributes.Delete("r")
}

func convertEllipse(elem *svgast.Element, precision int) {
	cx := getNumAttr(elem, "cx", 0)
	cy := getNumAttr(elem, "cy", 0)
	rx := getNumAttr(elem, "rx", 0)
	ry := getNumAttr(elem, "ry", 0)

	if math.IsNaN(cx - cy + rx - ry) {
		return
	}

	pathData := []path.PathDataItem{
		{Command: 'M', Args: []float64{cx, cy - ry}},
		{Command: 'A', Args: []float64{rx, ry, 0, 1, 0, cx, cy + ry}},
		{Command: 'A', Args: []float64{rx, ry, 0, 1, 0, cx, cy - ry}},
		{Command: 'z', Args: []float64{}},
	}

	elem.Name = "path"
	elem.Attributes.Set("d", path.StringifyPathData(&path.StringifyPathDataOptions{
		PathData:  pathData,
		Precision: precision,
	}))
	elem.Attributes.Delete("cx")
	elem.Attributes.Delete("cy")
	elem.Attributes.Delete("rx")
	elem.Attributes.Delete("ry")
}

// getNumAttr gets a numeric attribute value, returning defaultVal if absent.
func getNumAttr(elem *svgast.Element, name string, defaultVal float64) float64 {
	val, ok := elem.Attributes.Get(name)
	if !ok || val == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

// parsePoints extracts numeric coordinates from a points attribute string.
func parsePoints(s string) []float64 {
	matches := regNumber.FindAllString(s, -1)
	coords := make([]float64, 0, len(matches))
	for _, m := range matches {
		f, err := strconv.ParseFloat(m, 64)
		if err != nil {
			continue
		}
		coords = append(coords, f)
	}
	return coords
}

// buildPolyPathData converts a list of coordinates to path data items.
func buildPolyPathData(coords []float64, close bool) []path.PathDataItem {
	var pathData []path.PathDataItem
	for i := 0; i < len(coords)-1; i += 2 {
		cmd := byte('L')
		if i == 0 {
			cmd = 'M'
		}
		pathData = append(pathData, path.PathDataItem{
			Command: cmd,
			Args:    coords[i : i+2],
		})
	}
	if close {
		pathData = append(pathData, path.PathDataItem{
			Command: 'z',
			Args:    []float64{},
		})
	}
	return pathData
}
