// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package removeoffcanvaspaths implements the removeOffCanvasPaths SVGO plugin.
// It removes elements that are drawn outside of the viewBox.
package removeoffcanvaspaths

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/geom/path"
	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
)

func init() {
	plugin.Register(&plugin.Plugin{
		Name:        "removeOffCanvasPaths",
		Description: "removes elements that are drawn outside of the viewBox",
		Fn:          fn,
	})
}

// viewBox holds the parsed viewBox boundaries.
type viewBox struct {
	left   float64
	top    float64
	right  float64
	bottom float64
	width  float64
	height float64
}

// Regex patterns for viewBox parsing.
var (
	reCommasPlusPixels = regexp.MustCompile(`[,+]|px`)
	reWhitespace       = regexp.MustCompile(`\s+`)
	reViewBox          = regexp.MustCompile(`^(-?\d*\.?\d+) (-?\d*\.?\d+) (\d*\.?\d+) (\d*\.?\d+)$`)
)

func fn(_ *svgast.Root, _ map[string]any, _ *plugin.PluginInfo) *svgast.Visitor {
	var vbData *viewBox

	return &svgast.Visitor{
		Element: &svgast.VisitorCallbacks{
			Enter: func(node svgast.Node, parent svgast.Parent) error {
				elem := node.(*svgast.Element)

				// Parse viewBox from root <svg> element
				if elem.Name == "svg" && isRootParent(parent) {
					vbStr := ""
					if v, ok := elem.Attributes.Get("viewBox"); ok {
						vbStr = v
					} else if w, wOk := elem.Attributes.Get("width"); wOk {
						if h, hOk := elem.Attributes.Get("height"); hOk {
							vbStr = "0 0 " + w + " " + h
						}
					}

					// Remove commas and plus signs, normalize and trim whitespace
					vbStr = reCommasPlusPixels.ReplaceAllString(vbStr, " ")
					vbStr = reWhitespace.ReplaceAllString(vbStr, " ")
					vbStr = strings.TrimSpace(vbStr)

					// Ensure that the dimensions are 4 values separated by space
					m := reViewBox.FindStringSubmatch(vbStr)
					if m == nil {
						return nil
					}

					left, _ := strconv.ParseFloat(m[1], 64)
					top, _ := strconv.ParseFloat(m[2], 64)
					width, _ := strconv.ParseFloat(m[3], 64)
					height, _ := strconv.ParseFloat(m[4], 64)

					vbData = &viewBox{
						left:   left,
						top:    top,
						right:  left + width,
						bottom: top + height,
						width:  width,
						height: height,
					}
				}

				// Consider that any item with a transform attribute is visible
				if elem.Attributes.Has("transform") {
					return svgast.ErrVisitSkip
				}

				if elem.Name == "path" && elem.Attributes.Has("d") && vbData != nil {
					d, _ := elem.Attributes.Get("d")
					pathData := path.ParsePathData(d)

					// Consider that an M command within the viewBox is visible
					visible := false
					for _, item := range pathData {
						if item.Command == 'M' && len(item.Args) >= 2 {
							x, y := item.Args[0], item.Args[1]
							if x >= vbData.left && x <= vbData.right &&
								y >= vbData.top && y <= vbData.bottom {
								visible = true
							}
						}
					}
					if visible {
						return nil
					}

					// Close the path if too short for intersects()
					if len(pathData) == 2 {
						pathData = append(pathData, path.PathDataItem{
							Command: 'z',
							Args:    nil,
						})
					}

					// Build viewBox path data for intersection test
					viewBoxPathData := []path.PathDataItem{
						{Command: 'M', Args: []float64{vbData.left, vbData.top}},
						{Command: 'h', Args: []float64{vbData.width}},
						{Command: 'v', Args: []float64{vbData.height}},
						{Command: 'H', Args: []float64{vbData.left}},
						{Command: 'z', Args: nil},
					}

					if !path.Intersects(viewBoxPathData, pathData) {
						svgast.DetachNodeFromParent(node, parent)
					}
				}

				return nil
			},
		},
	}
}

// isRootParent returns true if parent is the document root.
func isRootParent(parent svgast.Parent) bool {
	_, ok := parent.(*svgast.Root)
	return ok
}
