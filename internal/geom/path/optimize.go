// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Path optimization functions for convertPathData plugin.
// Ported from SVGO's plugins/convertPathData.js and plugins/_path.js.

package path

import (
	"math"

	"github.com/okooo5km/ogvs/internal/tools"
)

// ExtendedPathItem extends PathDataItem with absolute coordinate tracking
// used during optimization. Matches the JS pattern of attaching .base and .coords
// to path items.
type ExtendedPathItem struct {
	Command byte
	Args    []float64
	Base    []float64 // absolute coordinates of the previous point
	Coords  []float64 // absolute coordinates after this command
	Sdata   []float64 // saved curve data for arc conversion
}

// ConvertPathDataParams holds all the parameters for convertPathData.
type ConvertPathDataParams struct {
	ApplyTransforms        bool
	ApplyTransformsStroked bool
	MakeArcs               *MakeArcsParams
	StraightCurves         bool
	ConvertToQ             bool
	LineShorthands         bool
	ConvertToZ             bool
	CurveSmoothShorthands  bool
	FloatPrecision         int  // use -1 for false
	FloatPrecisionEnabled  bool // true when floatPrecision is not false
	TransformPrecision     int
	SmartArcRounding       bool
	RemoveUseless          bool
	CollapseRepeated       bool
	UtilizeAbsolute        bool
	LeadingZero            bool
	NegativeExtraSpace     bool
	NoSpaceAfterFlags      bool
	ForceAbsolutePath      bool
}

// MakeArcsParams controls arc conversion thresholds.
type MakeArcsParams struct {
	Threshold float64
	Tolerance float64
}

// circleResult represents a found circle from three curve points.
type circleResult struct {
	Center [2]float64
	Radius float64
}

// Module-level variables matching JS module globals.
// These are set per-invocation in the filter pipeline.
var (
	optimizePrecision    int
	optimizeError        float64
	optimizeArcThreshold float64
	optimizeArcTolerance float64
	optimizeRoundData    func(data []float64) []float64
)

// ConvertToRelative converts absolute path data coordinates to relative.
// Modifies items in place, attaching Base and Coords for later use.
// Ported from SVGO's convertToRelative() in plugins/convertPathData.js.
func ConvertToRelative(pathData []ExtendedPathItem) []ExtendedPathItem {
	start := [2]float64{0, 0}
	cursor := [2]float64{0, 0}
	prevCoords := []float64{0, 0}

	for i := range pathData {
		item := &pathData[i]
		command := item.Command
		args := item.Args

		// moveto (x y)
		if command == 'm' {
			cursor[0] += args[0]
			cursor[1] += args[1]
			start[0] = cursor[0]
			start[1] = cursor[1]
		} else if command == 'M' {
			// M → m (skip first moveto)
			if i != 0 {
				command = 'm'
			}
			args[0] -= cursor[0]
			args[1] -= cursor[1]
			cursor[0] += args[0]
			cursor[1] += args[1]
			start[0] = cursor[0]
			start[1] = cursor[1]
		} else if command == 'l' {
			cursor[0] += args[0]
			cursor[1] += args[1]
		} else if command == 'L' {
			command = 'l'
			args[0] -= cursor[0]
			args[1] -= cursor[1]
			cursor[0] += args[0]
			cursor[1] += args[1]
		} else if command == 'h' {
			cursor[0] += args[0]
		} else if command == 'H' {
			command = 'h'
			args[0] -= cursor[0]
			cursor[0] += args[0]
		} else if command == 'v' {
			cursor[1] += args[0]
		} else if command == 'V' {
			command = 'v'
			args[0] -= cursor[1]
			cursor[1] += args[0]
		} else if command == 'c' {
			cursor[0] += args[4]
			cursor[1] += args[5]
		} else if command == 'C' {
			command = 'c'
			args[0] -= cursor[0]
			args[1] -= cursor[1]
			args[2] -= cursor[0]
			args[3] -= cursor[1]
			args[4] -= cursor[0]
			args[5] -= cursor[1]
			cursor[0] += args[4]
			cursor[1] += args[5]
		} else if command == 's' {
			cursor[0] += args[2]
			cursor[1] += args[3]
		} else if command == 'S' {
			command = 's'
			args[0] -= cursor[0]
			args[1] -= cursor[1]
			args[2] -= cursor[0]
			args[3] -= cursor[1]
			cursor[0] += args[2]
			cursor[1] += args[3]
		} else if command == 'q' {
			cursor[0] += args[2]
			cursor[1] += args[3]
		} else if command == 'Q' {
			command = 'q'
			args[0] -= cursor[0]
			args[1] -= cursor[1]
			args[2] -= cursor[0]
			args[3] -= cursor[1]
			cursor[0] += args[2]
			cursor[1] += args[3]
		} else if command == 't' {
			cursor[0] += args[0]
			cursor[1] += args[1]
		} else if command == 'T' {
			command = 't'
			args[0] -= cursor[0]
			args[1] -= cursor[1]
			cursor[0] += args[0]
			cursor[1] += args[1]
		} else if command == 'a' {
			cursor[0] += args[5]
			cursor[1] += args[6]
		} else if command == 'A' {
			command = 'a'
			args[5] -= cursor[0]
			args[6] -= cursor[1]
			cursor[0] += args[5]
			cursor[1] += args[6]
		} else if command == 'Z' || command == 'z' {
			cursor[0] = start[0]
			cursor[1] = start[1]
		}

		item.Command = command
		item.Args = args
		item.Base = prevCoords
		item.Coords = []float64{cursor[0], cursor[1]}
		prevCoords = item.Coords
	}

	return pathData
}

// Filters is the main filter loop for path optimization.
// Ported from SVGO's filters() in plugins/convertPathData.js.
func Filters(
	pathItems []ExtendedPathItem,
	params *ConvertPathDataParams,
	isSafeToUseZ bool,
	maybeHasStrokeAndLinecap bool,
	hasMarkerMid bool,
) []ExtendedPathItem {
	precision := params.FloatPrecision
	precisionEnabled := params.FloatPrecisionEnabled

	if precisionEnabled {
		optimizePrecision = precision
		optimizeError = tools.ToFixed(math.Pow(0.1, float64(precision)), precision)
	} else {
		optimizePrecision = 0
		optimizeError = 1e-2
	}

	if precisionEnabled && precision > 0 && precision < 20 {
		optimizeRoundData = strongRound
	} else {
		optimizeRoundData = simpleRound
	}

	if params.MakeArcs != nil {
		optimizeArcThreshold = params.MakeArcs.Threshold
		optimizeArcTolerance = params.MakeArcs.Tolerance
	}

	stringify := func(items []ExtendedPathItem) string {
		return data2Path(params, items)
	}

	relSubpoint := [2]float64{0, 0}
	pathBase := [2]float64{0, 0}
	var prev *ExtendedPathItem
	var prevQControlPoint *[2]float64

	result := make([]ExtendedPathItem, 0, len(pathItems))

	for index := 0; index < len(pathItems); index++ {
		item := &pathItems[index]
		qControlPoint := prevQControlPoint

		command := item.Command
		data := item.Args
		var next *ExtendedPathItem
		if index+1 < len(pathItems) {
			next = &pathItems[index+1]
		}

		if command != 'Z' && command != 'z' {
			sdata := data
			var circle *circleResult

			if command == 's' {
				sdata = make([]float64, 0, 2+len(data))
				sdata = append(sdata, 0, 0)
				sdata = append(sdata, data...)

				if prev != nil {
					pdata := prev.Args
					n := len(pdata)
					// In JS, accessing array with negative index returns undefined,
					// and undefined in arithmetic yields NaN, which causes
					// isConvex/isCurveStraightLine to return false.
					// In Go, guard against n < 4 to avoid index out of range panic.
					if n >= 4 {
						sdata[0] = pdata[n-2] - pdata[n-4]
						sdata[1] = pdata[n-1] - pdata[n-3]
					}
					// When n < 4, sdata[0] and sdata[1] remain 0
				}
			}

			// convert curves to arcs if possible
			if params.MakeArcs != nil &&
				(command == 'c' || command == 's') &&
				isConvex(sdata) {
				circle = findCircle(sdata)
			}

			if circle != nil {
				r := optimizeRoundData([]float64{circle.Radius})[0]
				angle := findArcAngle(sdata, circle)
				sweep := 0.0
				if sdata[5]*sdata[0]-sdata[4]*sdata[1] > 0 {
					sweep = 1
				}
				arc := ExtendedPathItem{
					Command: 'a',
					Args:    []float64{r, r, 0, 0, sweep, sdata[4], sdata[5]},
					Coords:  append([]float64(nil), item.Coords...),
					Base:    item.Base,
				}
				output := []ExtendedPathItem{arc}
				relCenter := [2]float64{
					circle.Center[0] - sdata[4],
					circle.Center[1] - sdata[5],
				}
				relCircle := &circleResult{Center: relCenter, Radius: circle.Radius}
				arcCurves := []ExtendedPathItem{*item}
				hasPrev := 0
				suffix := ""
				var nextLonghand *ExtendedPathItem

				if prev != nil &&
					((prev.Command == 'c' && isConvex(prev.Args) && isArcPrev(prev.Args, circle)) ||
						(prev.Command == 'a' && prev.Sdata != nil && isArcPrev(prev.Sdata, circle))) {
					arcCurves = append([]ExtendedPathItem{*prev}, arcCurves...)
					arc.Base = prev.Base
					arc.Args[5] = arc.Coords[0] - arc.Base[0]
					arc.Args[6] = arc.Coords[1] - arc.Base[1]
					prevData := prev.Args
					if prev.Command == 'a' {
						prevData = prev.Sdata
					}
					prevAngle := findArcAngle(prevData, &circleResult{
						Center: [2]float64{
							prevData[4] + circle.Center[0],
							prevData[5] + circle.Center[1],
						},
						Radius: circle.Radius,
					})
					angle += prevAngle
					if angle > math.Pi {
						arc.Args[3] = 1
					}
					hasPrev = 1
					output[0] = arc
				}

				// check if next curves are fitting the arc
				j := index
				for {
					j++
					if j >= len(pathItems) {
						break
					}
					nextItem := &pathItems[j]
					if nextItem.Command != 'c' && nextItem.Command != 's' {
						break
					}

					nextData := nextItem.Args
					if nextItem.Command == 's' {
						nextLonghand = makeLonghandExt(
							&ExtendedPathItem{Command: 's', Args: append([]float64(nil), nextItem.Args...)},
							pathItems[j-1].Args,
						)
						nextData = nextLonghand.Args
						nextLonghand.Args = nextData[:2]
						suffix = stringify([]ExtendedPathItem{*nextLonghand})
					}
					if isConvex(nextData) && isArc(nextData, relCircle) {
						angle += findArcAngle(nextData, relCircle)
						if angle-2*math.Pi > 1e-3 {
							break
						}
						if angle > math.Pi {
							arc.Args[3] = 1
						}
						arcCurves = append(arcCurves, *nextItem)
						if 2*math.Pi-angle > 1e-3 {
							// less than 360
							arc.Coords = nextItem.Coords
							arc.Args[5] = arc.Coords[0] - arc.Base[0]
							arc.Args[6] = arc.Coords[1] - arc.Base[1]
						} else {
							// full circle
							arc.Args[5] = 2 * (relCircle.Center[0] - nextData[4])
							arc.Args[6] = 2 * (relCircle.Center[1] - nextData[5])
							arc.Coords = []float64{
								arc.Base[0] + arc.Args[5],
								arc.Base[1] + arc.Args[6],
							}
							newArc := ExtendedPathItem{
								Command: 'a',
								Args: []float64{
									r, r, 0, 0, sweep,
									nextItem.Coords[0] - arc.Coords[0],
									nextItem.Coords[1] - arc.Coords[1],
								},
								Coords: nextItem.Coords,
								Base:   arc.Coords,
							}
							output = append(output, newArc)
							j++
							break
						}
						relCenter[0] -= nextData[4]
						relCenter[1] -= nextData[5]
						relCircle = &circleResult{Center: relCenter, Radius: circle.Radius}
					} else {
						break
					}
				}

				// In JS, arc is an object reference shared with output[0].
				// Modifications to arc (Args, Coords) during the forward loop
				// are visible through output[0]. In Go, arc is a value copy,
				// so we must sync it back after the loop.
				output[0] = arc

				outputStr := stringify(output) + suffix
				arcStr := stringify(arcCurves)
				if len(outputStr) < len(arcStr) {
					if j < len(pathItems) && pathItems[j].Command == 's' {
						makeLonghandExt(&pathItems[j], pathItems[j-1].Args)
					}
					if hasPrev != 0 {
						prevArc := output[0]
						output = output[1:]
						optimizeRoundData(prevArc.Args)
						relSubpoint[0] += prevArc.Args[5] - prev.Args[len(prev.Args)-2]
						relSubpoint[1] += prevArc.Args[6] - prev.Args[len(prev.Args)-1]
						prev.Command = 'a'
						prev.Args = prevArc.Args
						item.Base = prevArc.Coords
						prev.Coords = prevArc.Coords
						// update prev in result
						if len(result) > 0 {
							result[len(result)-1] = *prev
						}
					}
					// arc = output.shift()
					var arcResult *ExtendedPathItem
					if len(output) > 0 {
						a := output[0]
						arcResult = &a
						output = output[1:]
					}
					if len(arcCurves) == 1 {
						item.Sdata = append([]float64(nil), sdata...)
					} else if len(arcCurves)-1-hasPrev > 0 {
						// splice: replace consumed items with remaining output
						count := len(arcCurves) - 1 - hasPrev
						newItems := make([]ExtendedPathItem, 0, len(pathItems)-count+len(output))
						newItems = append(newItems, pathItems[:index+1]...)
						newItems = append(newItems, output...)
						newItems = append(newItems, pathItems[index+1+count:]...)
						pathItems = newItems
					}
					if arcResult == nil {
						// return false — filter out this item
						continue
					}
					command = 'a'
					data = arcResult.Args
					item.Coords = arcResult.Coords
				}
			}

			// Rounding relative coordinates, taking into account accumulating error
			if precisionEnabled {
				if command == 'm' || command == 'l' || command == 't' ||
					command == 'q' || command == 's' || command == 'c' {
					for i := len(data) - 1; i >= 0; i-- {
						data[i] += item.Base[i%2] - relSubpoint[i%2]
					}
				} else if command == 'h' {
					data[0] += item.Base[0] - relSubpoint[0]
				} else if command == 'v' {
					data[0] += item.Base[1] - relSubpoint[1]
				} else if command == 'a' {
					data[5] += item.Base[0] - relSubpoint[0]
					data[6] += item.Base[1] - relSubpoint[1]
				}
				optimizeRoundData(data)

				if command == 'h' {
					relSubpoint[0] += data[0]
				} else if command == 'v' {
					relSubpoint[1] += data[0]
				} else {
					relSubpoint[0] += data[len(data)-2]
					relSubpoint[1] += data[len(data)-1]
				}
				optimizeRoundData(relSubpoint[:])

				if command == 'M' || command == 'm' {
					pathBase[0] = relSubpoint[0]
					pathBase[1] = relSubpoint[1]
				}
			}

			// round arc radius more accurately
			var sagitta *float64
			if command == 'a' {
				s := calculateSagitta(data)
				sagitta = s
			}
			if params.SmartArcRounding && sagitta != nil && precisionEnabled && precision > 0 {
				for precisionNew := precision; precisionNew >= 0; precisionNew-- {
					radius := tools.ToFixed(data[0], precisionNew)
					newArcData := make([]float64, len(data))
					copy(newArcData, data)
					newArcData[0] = radius
					newArcData[1] = radius
					sagittaNew := calculateSagitta(newArcData)
					if sagittaNew != nil && math.Abs(*sagitta-*sagittaNew) < optimizeError {
						data[0] = radius
						data[1] = radius
					} else {
						break
					}
				}
			}

			// convert straight curves into line segments
			if params.StraightCurves {
				if (command == 'c' && isCurveStraightLine(data)) ||
					(command == 's' && isCurveStraightLine(sdata)) {
					if next != nil && next.Command == 's' {
						makeLonghandExt(next, data)
					}
					command = 'l'
					data = data[len(data)-2:]
				} else if command == 'q' && isCurveStraightLine(data) {
					if next != nil && next.Command == 't' {
						makeLonghandExt(next, data)
					}
					command = 'l'
					data = data[len(data)-2:]
				} else if command == 't' &&
					prev != nil && prev.Command != 'q' && prev.Command != 't' {
					command = 'l'
					data = data[len(data)-2:]
				} else if command == 'a' &&
					(data[0] == 0 || data[1] == 0 ||
						(sagitta != nil && *sagitta < optimizeError)) {
					command = 'l'
					data = data[len(data)-2:]
				}
			}

			// degree-lower c to q when possible
			if params.ConvertToQ && command == 'c' {
				x1 := 0.75*(item.Base[0]+data[0]) - 0.25*item.Base[0]
				x2 := 0.75*(item.Base[0]+data[2]) - 0.25*(item.Base[0]+data[4])
				if math.Abs(x1-x2) < optimizeError*2 {
					y1 := 0.75*(item.Base[1]+data[1]) - 0.25*item.Base[1]
					y2 := 0.75*(item.Base[1]+data[3]) - 0.25*(item.Base[1]+data[5])
					if math.Abs(y1-y2) < optimizeError*2 {
						newData := append([]float64(nil), data...)
						// splice: replace first 4 elements with 2 new values
						newData = append([]float64{
							x1 + x2 - item.Base[0],
							y1 + y2 - item.Base[1],
						}, newData[4:]...)
						optimizeRoundData(newData)
						originalLength := len(cleanupOutDataStr(data, params))
						newLength := len(cleanupOutDataStr(newData, params))
						if newLength < originalLength {
							command = 'q'
							data = newData
							if next != nil && next.Command == 's' {
								makeLonghandExt(next, data)
							}
						}
					}
				}
			}

			// horizontal and vertical line shorthands
			if params.LineShorthands && command == 'l' {
				if data[1] == 0 {
					command = 'h'
					data = data[:1]
				} else if data[0] == 0 {
					command = 'v'
					data = data[1:]
				}
			}

			// collapse repeated commands
			if params.CollapseRepeated && !hasMarkerMid &&
				(command == 'm' || command == 'h' || command == 'v') &&
				prev != nil && prev.Command != 0 &&
				command == toLower(prev.Command) {
				if (command != 'h' && command != 'v') ||
					(prev.Args[0] >= 0) == (data[0] >= 0) {
					prev.Args[0] += data[0]
					if command != 'h' && command != 'v' {
						prev.Args[1] += data[1]
					}
					prev.Coords = item.Coords
					// update prev in result
					if len(result) > 0 {
						result[len(result)-1] = *prev
					}
					continue
				}
			}

			// convert curves into smooth shorthands
			if params.CurveSmoothShorthands && prev != nil && prev.Command != 0 {
				if command == 'c' {
					if prev.Command == 'c' &&
						math.Abs(data[0]-(-(prev.Args[2]-prev.Args[4]))) < optimizeError &&
						math.Abs(data[1]-(-(prev.Args[3]-prev.Args[5]))) < optimizeError {
						command = 's'
						data = data[2:]
					} else if prev.Command == 's' &&
						math.Abs(data[0]-(-(prev.Args[0]-prev.Args[2]))) < optimizeError &&
						math.Abs(data[1]-(-(prev.Args[1]-prev.Args[3]))) < optimizeError {
						command = 's'
						data = data[2:]
					} else if prev.Command != 'c' && prev.Command != 's' &&
						math.Abs(data[0]) < optimizeError &&
						math.Abs(data[1]) < optimizeError {
						command = 's'
						data = data[2:]
					}
				} else if command == 'q' {
					if prev.Command == 'q' &&
						math.Abs(data[0]-(prev.Args[2]-prev.Args[0])) < optimizeError &&
						math.Abs(data[1]-(prev.Args[3]-prev.Args[1])) < optimizeError {
						command = 't'
						data = data[2:]
					} else if prev.Command == 't' {
						predictedCP := reflectPoint(qControlPoint, item.Base)
						realCP := [2]float64{
							data[0] + item.Base[0],
							data[1] + item.Base[1],
						}
						if math.Abs(predictedCP[0]-realCP[0]) < optimizeError &&
							math.Abs(predictedCP[1]-realCP[1]) < optimizeError {
							command = 't'
							data = data[2:]
						}
					}
				}
			}

			// remove useless non-first path segments
			if params.RemoveUseless && !maybeHasStrokeAndLinecap {
				if (command == 'l' || command == 'h' || command == 'v' ||
					command == 'q' || command == 't' || command == 'c' || command == 's') &&
					allZero(data) {
					// skip this item, keep prev
					continue
				}
				if command == 'a' && data[5] == 0 && data[6] == 0 {
					continue
				}
			}

			// convert going home to z
			if params.ConvertToZ &&
				(isSafeToUseZ || (next != nil && (next.Command == 'Z' || next.Command == 'z'))) &&
				(command == 'l' || command == 'h' || command == 'v') {
				if math.Abs(pathBase[0]-item.Coords[0]) < optimizeError &&
					math.Abs(pathBase[1]-item.Coords[1]) < optimizeError {
					command = 'z'
					data = nil
				}
			}

			item.Command = command
			item.Args = data
		} else {
			// z resets coordinates
			relSubpoint[0] = pathBase[0]
			relSubpoint[1] = pathBase[1]
			if prev != nil && (prev.Command == 'Z' || prev.Command == 'z') {
				continue
			}
		}

		// Remove useless z when already at start
		if (command == 'Z' || command == 'z') &&
			params.RemoveUseless && isSafeToUseZ &&
			math.Abs(item.Base[0]-item.Coords[0]) < optimizeError/10 &&
			math.Abs(item.Base[1]-item.Coords[1]) < optimizeError/10 {
			continue
		}

		if command == 'q' {
			p := [2]float64{data[0] + item.Base[0], data[1] + item.Base[1]}
			prevQControlPoint = &p
		} else if command == 't' {
			if qControlPoint != nil {
				p := reflectPoint(qControlPoint, item.Base)
				prevQControlPoint = &p
			} else {
				p := [2]float64{item.Coords[0], item.Coords[1]}
				prevQControlPoint = &p
			}
		} else {
			prevQControlPoint = nil
		}

		result = append(result, *item)
		prev = &result[len(result)-1]
	}

	return result
}

// ConvertToMixed writes data in shortest form using absolute or relative coordinates.
// Ported from SVGO's convertToMixed() in plugins/convertPathData.js.
func ConvertToMixed(path []ExtendedPathItem, params *ConvertPathDataParams) []ExtendedPathItem {
	if len(path) == 0 {
		return path
	}
	prev := &path[0]

	result := []ExtendedPathItem{path[0]}

	for index := 1; index < len(path); index++ {
		item := &path[index]

		if item.Command == 'Z' || item.Command == 'z' {
			prev = item
			result = append(result, *item)
			continue
		}

		command := item.Command
		data := item.Args
		adata := append([]float64(nil), data...)
		rdata := append([]float64(nil), data...)

		if command == 'm' || command == 'l' || command == 't' ||
			command == 'q' || command == 's' || command == 'c' {
			for i := len(adata) - 1; i >= 0; i-- {
				adata[i] += item.Base[i%2]
			}
		} else if command == 'h' {
			adata[0] += item.Base[0]
		} else if command == 'v' {
			adata[0] += item.Base[1]
		} else if command == 'a' {
			adata[5] += item.Base[0]
			adata[6] += item.Base[1]
		}

		optimizeRoundData(adata)
		optimizeRoundData(rdata)

		absoluteDataStr := cleanupOutDataStr(adata, params)
		relativeDataStr := cleanupOutDataStr(rdata, params)

		if params.ForceAbsolutePath ||
			(len(absoluteDataStr) < len(relativeDataStr) &&
				!(params.NegativeExtraSpace &&
					command == prev.Command &&
					prev.Command > 96 && // lowercase
					len(absoluteDataStr) == len(relativeDataStr)-1 &&
					(data[0] < 0 ||
						(math.Floor(data[0]) == 0 &&
							!isIntegerFloat(data[0]) &&
							math.Mod(prev.Args[len(prev.Args)-1], 1) != 0)))) {
			item.Command = toUpper(command)
			item.Args = adata
		}

		prev = item
		result = append(result, *item)
	}

	return result
}

// Helper functions ported from SVGO's convertPathData.js

// strongRound decreases accuracy of floating-point numbers keeping a specified
// number of decimals. Smart rounds values like 2.3491 to 2.35 instead of 2.349.
func strongRound(data []float64) []float64 {
	precisionNum := optimizePrecision
	for i := len(data) - 1; i >= 0; i-- {
		fixed := tools.ToFixed(data[i], precisionNum)
		if fixed != data[i] {
			rounded := tools.ToFixed(data[i], precisionNum-1)
			diff := tools.ToFixed(math.Abs(rounded-data[i]), precisionNum+1)
			if diff >= optimizeError {
				data[i] = fixed
			} else {
				data[i] = rounded
			}
		}
	}
	return data
}

// simpleRound rounds all values to nearest integer (used when precision is 0 or false).
func simpleRound(data []float64) []float64 {
	for i := len(data) - 1; i >= 0; i-- {
		data[i] = math.Round(data[i])
	}
	return data
}

// isCurveStraightLine checks if a curve is actually a straight line.
func isCurveStraightLine(data []float64) bool {
	i := len(data) - 2
	a := -data[i+1] // y1 - y2 (y1 = 0)
	b := data[i]    // x2 - x1 (x1 = 0)
	d := 1.0 / (a*a + b*b)

	// curve that ends at start point isn't the case; also degenerate lines
	if i <= 1 || math.IsInf(d, 0) || math.IsNaN(d) {
		return false
	}

	// Distance from point (x0, y0) to the line is sqrt((c - a*x0 - b*y0)^2 / (a^2 + b^2))
	for i -= 2; i >= 0; i -= 2 {
		if math.Sqrt(math.Pow(a*data[i]+b*data[i+1], 2)*d) > optimizeError {
			return false
		}
	}
	return true
}

// calculateSagitta calculates the sagitta of an arc if possible.
func calculateSagitta(data []float64) *float64 {
	if data[3] == 1 {
		return nil
	}
	rx, ry := data[0], data[1]
	if math.Abs(rx-ry) > optimizeError {
		return nil
	}
	chord := math.Hypot(data[5], data[6])
	if chord > rx*2 {
		return nil
	}
	result := rx - math.Sqrt(rx*rx-0.25*chord*chord)
	return &result
}

// isConvex checks if curve is convex.
func isConvex(data []float64) bool {
	if len(data) < 6 {
		return false
	}
	center := getIntersection([]float64{
		0, 0, data[2], data[3],
		data[0], data[1], data[4], data[5],
	})
	if center == nil {
		return false
	}
	return (data[2] < center[0]) == (center[0] < 0) &&
		(data[3] < center[1]) == (center[1] < 0) &&
		(data[4] < center[0]) == (center[0] < data[0]) &&
		(data[5] < center[1]) == (center[1] < data[1])
}

// getIntersection computes line intersection from 8 coordinates.
func getIntersection(coords []float64) []float64 {
	a1 := coords[1] - coords[3]
	b1 := coords[2] - coords[0]
	c1 := coords[0]*coords[3] - coords[2]*coords[1]
	a2 := coords[5] - coords[7]
	b2 := coords[6] - coords[4]
	c2 := coords[4]*coords[7] - coords[5]*coords[6]
	denom := a1*b2 - a2*b1

	if denom == 0 {
		return nil
	}

	crossX := (b1*c2 - b2*c1) / denom
	crossY := (a1*c2 - a2*c1) / -denom

	if !math.IsNaN(crossX) && !math.IsNaN(crossY) &&
		!math.IsInf(crossX, 0) && !math.IsInf(crossY, 0) {
		return []float64{crossX, crossY}
	}
	return nil
}

// getCubicBezierPoint returns the point on a cubic Bezier at parameter t.
func getCubicBezierPoint(curve []float64, t float64) [2]float64 {
	sqrT := t * t
	cubT := sqrT * t
	mt := 1 - t
	sqrMt := mt * mt
	return [2]float64{
		3*sqrMt*t*curve[0] + 3*mt*sqrT*curve[2] + cubT*curve[4],
		3*sqrMt*t*curve[1] + 3*mt*sqrT*curve[3] + cubT*curve[5],
	}
}

// findCircle finds a circle from 3 points of a curve.
func findCircle(curve []float64) *circleResult {
	midPoint := getCubicBezierPoint(curve, 0.5)
	m1 := [2]float64{midPoint[0] / 2, midPoint[1] / 2}
	m2 := [2]float64{(midPoint[0] + curve[4]) / 2, (midPoint[1] + curve[5]) / 2}

	center := getIntersection([]float64{
		m1[0], m1[1],
		m1[0] + m1[1], m1[1] - m1[0],
		m2[0], m2[1],
		m2[0] + (m2[1] - midPoint[1]),
		m2[1] - (m2[0] - midPoint[0]),
	})

	if center == nil {
		return nil
	}

	radius := getDistance([2]float64{0, 0}, [2]float64{center[0], center[1]})
	tolerance := math.Min(
		optimizeArcThreshold*optimizeError,
		(optimizeArcTolerance*radius)/100,
	)

	if radius >= 1e15 {
		return nil
	}

	testPoints := []float64{0.25, 0.75}
	for _, pt := range testPoints {
		bp := getCubicBezierPoint(curve, pt)
		if math.Abs(getDistance(bp, [2]float64{center[0], center[1]})-radius) > tolerance {
			return nil
		}
	}

	return &circleResult{
		Center: [2]float64{center[0], center[1]},
		Radius: radius,
	}
}

// isArc checks if a curve fits the given circle.
func isArc(curve []float64, circle *circleResult) bool {
	tolerance := math.Min(
		optimizeArcThreshold*optimizeError,
		(optimizeArcTolerance*circle.Radius)/100,
	)
	testPoints := []float64{0, 0.25, 0.5, 0.75, 1}
	for _, pt := range testPoints {
		bp := getCubicBezierPoint(curve, pt)
		if math.Abs(getDistance(bp, circle.Center)-circle.Radius) > tolerance {
			return false
		}
	}
	return true
}

// isArcPrev checks if a previous curve fits the given circle.
func isArcPrev(curve []float64, circle *circleResult) bool {
	return isArc(curve, &circleResult{
		Center: [2]float64{circle.Center[0] + curve[4], circle.Center[1] + curve[5]},
		Radius: circle.Radius,
	})
}

// findArcAngle finds the angle of a curve fitting the given arc.
func findArcAngle(curve []float64, relCircle *circleResult) float64 {
	x1 := -relCircle.Center[0]
	y1 := -relCircle.Center[1]
	x2 := curve[4] - relCircle.Center[0]
	y2 := curve[5] - relCircle.Center[1]

	return math.Acos(
		(x1*x2 + y1*y2) / math.Sqrt((x1*x1+y1*y1)*(x2*x2+y2*y2)),
	)
}

// getDistance returns the distance between two points.
func getDistance(p1, p2 [2]float64) float64 {
	return math.Hypot(p1[0]-p2[0], p1[1]-p2[1])
}

// reflectPoint reflects a point across another point.
func reflectPoint(controlPoint *[2]float64, base []float64) [2]float64 {
	return [2]float64{2*base[0] - controlPoint[0], 2*base[1] - controlPoint[1]}
}

// makeLonghandExt converts a shorthand command to longhand form using previous data.
func makeLonghandExt(item *ExtendedPathItem, prevData []float64) *ExtendedPathItem {
	switch item.Command {
	case 's':
		item.Command = 'c'
	case 't':
		item.Command = 'q'
	}
	n := len(prevData)
	newArgs := make([]float64, 0, 2+len(item.Args))
	// In JS, accessing array with negative index returns undefined (NaN in arithmetic).
	// Guard against n < 4 to avoid index out of range; use 0 as fallback.
	var cp0, cp1 float64
	if n >= 4 {
		cp0 = prevData[n-2] - prevData[n-4]
		cp1 = prevData[n-1] - prevData[n-3]
	}
	newArgs = append(newArgs, cp0, cp1)
	newArgs = append(newArgs, item.Args...)
	item.Args = newArgs
	return item
}

// data2Path converts path data items to a string representation.
func data2Path(params *ConvertPathDataParams, pathData []ExtendedPathItem) string {
	var result []byte
	for _, item := range pathData {
		strData := ""
		if len(item.Args) > 0 {
			dataCopy := append([]float64(nil), item.Args...)
			optimizeRoundData(dataCopy)
			strData = cleanupOutDataStr(dataCopy, params)
		}
		result = append(result, item.Command)
		result = append(result, strData...)
	}
	return string(result)
}

// cleanupOutDataStr wraps the tools.CleanupOutData call for path items.
func cleanupOutDataStr(data []float64, params *ConvertPathDataParams) string {
	return tools.CleanupOutData(data, &tools.CleanupOutDataParams{
		LeadingZero:        params.LeadingZero,
		NegativeExtraSpace: params.NegativeExtraSpace,
		NoSpaceAfterFlags:  params.NoSpaceAfterFlags,
	}, 0)
}

// allZero checks if all elements are zero.
func allZero(data []float64) bool {
	for _, v := range data {
		if v != 0 {
			return false
		}
	}
	return true
}

// toLower converts a command byte to lowercase.
func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// toUpper converts a command byte to uppercase.
func toUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 32
	}
	return c
}

// isIntegerFloat checks if a float64 is an integer value (matching JS Number.isInteger).
func isIntegerFloat(n float64) bool {
	return n == math.Trunc(n) && !math.IsInf(n, 0) && !math.IsNaN(n)
}
