// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Path intersection detection using convex hull + GJK algorithm.
// Ported from SVGO's plugins/_path.js (lines ~241-530).

package path

import (
	"math"
)

// vec2 is a 2D vector / point.
type vec2 [2]float64

// hullPoint represents a set of 2D points with indices of extreme points.
type hullPoint struct {
	list []vec2
	minX int
	minY int
	maxX int
	maxY int
}

// hullPoints represents the collection of all subpath point sets.
type hullPoints struct {
	list []hullPoint
	minX float64
	minY float64
	maxX float64
	maxY float64
}

// Intersects checks if two paths have an intersection by checking convex hulls
// collision using Gilbert-Johnson-Keerthi distance algorithm.
// Ported from SVGO's intersects() in plugins/_path.js.
func Intersects(path1, path2 []PathDataItem) bool {
	// Collect points of every subpath.
	points1 := gatherPoints(convertRelativeToAbsolute(path1))
	points2 := gatherPoints(convertRelativeToAbsolute(path2))

	// Axis-aligned bounding box check.
	if points1.maxX <= points2.minX ||
		points2.maxX <= points1.minX ||
		points1.maxY <= points2.minY ||
		points2.maxY <= points1.minY {
		return false
	}

	// Per-subpath AABB check — JS: points1.list.every(set1 => points2.list.every(set2 => ...))
	allSeparated := true
	for _, set1 := range points1.list {
		for _, set2 := range points2.list {
			if !(set1.list[set1.maxX][0] <= set2.list[set2.minX][0] ||
				set2.list[set2.maxX][0] <= set1.list[set1.minX][0] ||
				set1.list[set1.maxY][1] <= set2.list[set2.minY][1] ||
				set2.list[set2.maxY][1] <= set1.list[set1.minY][1]) {
				allSeparated = false
				break
			}
		}
		if !allSeparated {
			break
		}
	}
	if allSeparated {
		return false
	}

	// Get a convex hull from points of each subpath.
	hullNest1 := make([]hullPoint, len(points1.list))
	for i, p := range points1.list {
		hullNest1[i] = convexHull(p)
	}
	hullNest2 := make([]hullPoint, len(points2.list))
	for i, p := range points2.list {
		hullNest2[i] = convexHull(p)
	}

	// Check intersection of every subpath of the first path with every subpath of the second.
	for _, hull1 := range hullNest1 {
		if len(hull1.list) < 3 {
			continue
		}
		for _, hull2 := range hullNest2 {
			if len(hull2.list) < 3 {
				continue
			}
			if gjkIntersects(&hull1, &hull2) {
				return true
			}
		}
	}

	return false
}

// gjkIntersects performs GJK collision detection between two convex hulls.
func gjkIntersects(hull1, hull2 *hullPoint) bool {
	simplex := []vec2{getSupport(hull1, hull2, vec2{1, 0})}
	direction := vecMinus(simplex[0])

	iterations := 10000 // infinite loop protection

	for {
		if iterations <= 0 {
			// safety: assume intersection to avoid merging
			return true
		}
		iterations--

		// add a new point
		newPoint := getSupport(hull1, hull2, direction)
		simplex = append(simplex, newPoint)

		// see if the new point was on the correct side of the origin
		if vecDot(direction, simplex[len(simplex)-1]) <= 0 {
			return false
		}

		// process the simplex
		if processSimplexGJK(&simplex, &direction) {
			return true
		}
	}
}

// getSupport returns the Minkowski difference support point.
func getSupport(a, b *hullPoint, direction vec2) vec2 {
	return vecSub(supportPoint(a, direction), supportPoint(b, vecMinus(direction)))
}

// supportPoint computes the farthest polygon point in a particular direction.
// Uses knowledge of min/max x and y coordinates to choose a quadrant to search in.
func supportPoint(polygon *hullPoint, direction vec2) vec2 {
	var index int
	if direction[1] >= 0 {
		if direction[0] < 0 {
			index = polygon.maxY
		} else {
			index = polygon.maxX
		}
	} else {
		if direction[0] < 0 {
			index = polygon.minX
		} else {
			index = polygon.minY
		}
	}

	maxVal := math.Inf(-1)
	for {
		value := vecDot(polygon.list[index], direction)
		if value <= maxVal {
			break
		}
		maxVal = value
		index++
		index = index % len(polygon.list)
	}

	// return the previous point (the one that had the max dot product)
	retIdx := index
	if retIdx == 0 {
		retIdx = len(polygon.list)
	}
	retIdx--
	return polygon.list[retIdx]
}

// processSimplexGJK processes the simplex and updates the direction.
// Returns true if the origin is contained in the simplex.
// The simplex and direction are modified in place (matching JS mutation semantics).
func processSimplexGJK(simplex *[]vec2, direction *vec2) bool {
	s := *simplex

	if len(s) == 2 {
		// 1-simplex (line segment)
		a := s[1]
		b := s[0]
		ao := vecMinus(s[1])
		ab := vecSub(b, a)

		if vecDot(ao, ab) > 0 {
			// get the vector perpendicular to AB facing O
			result := vecOrth(ab, a)
			*direction = result
		} else {
			*direction = ao
			// only A remains in the simplex: simplex.shift()
			*simplex = []vec2{a}
		}
		return false
	}

	// 2-simplex (triangle)
	a := s[2]
	b := s[1]
	c := s[0]
	ab := vecSub(b, a)
	ac := vecSub(c, a)
	ao := vecMinus(a)
	acb := vecOrth(ab, ac) // perpendicular to AB facing away from C
	abc := vecOrth(ac, ab) // perpendicular to AC facing away from B

	if vecDot(acb, ao) > 0 {
		if vecDot(ab, ao) > 0 {
			// region 4
			*direction = acb
			// simplex.shift() → remove first element → simplex = [b, a]
			*simplex = []vec2{b, a}
		} else {
			// region 5
			*direction = ao
			// simplex.splice(0, 2) → simplex = [a]
			*simplex = []vec2{a}
		}
	} else if vecDot(abc, ao) > 0 {
		if vecDot(ac, ao) > 0 {
			// region 6
			*direction = abc
			// simplex.splice(1, 1) → simplex = [c, a]
			*simplex = []vec2{c, a}
		} else {
			// region 5 (again)
			*direction = ao
			// simplex.splice(0, 2) → simplex = [a]
			*simplex = []vec2{a}
		}
	} else {
		// region 7 — origin is inside the triangle
		return true
	}
	return false
}

// vecMinus negates a vector.
func vecMinus(v vec2) vec2 {
	return vec2{-v[0], -v[1]}
}

// vecSub subtracts v2 from v1.
func vecSub(v1, v2 vec2) vec2 {
	return vec2{v1[0] - v2[0], v1[1] - v2[1]}
}

// vecDot returns the dot product of two vectors.
func vecDot(v1, v2 vec2) float64 {
	return v1[0]*v2[0] + v1[1]*v2[1]
}

// vecOrth returns a vector perpendicular to v, facing away from "from".
func vecOrth(v, from vec2) vec2 {
	o := vec2{-v[1], v[0]}
	if vecDot(o, vecMinus(from)) < 0 {
		return vecMinus(o)
	}
	return o
}

// cross2D returns the cross product of vectors OA and OB.
func cross2D(o, a, b vec2) float64 {
	return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0])
}

// convertRelativeToAbsolute converts relative path commands to absolute.
// Ported from SVGO's convertRelativeToAbsolute in plugins/_path.js.
func convertRelativeToAbsolute(data []PathDataItem) []PathDataItem {
	newData := make([]PathDataItem, 0, len(data))
	start := vec2{0, 0}
	cursor := vec2{0, 0}

	for _, item := range data {
		command := item.Command
		args := make([]float64, len(item.Args))
		copy(args, item.Args)

		// moveto (x y)
		if command == 'm' {
			args[0] += cursor[0]
			args[1] += cursor[1]
			command = 'M'
		}
		if command == 'M' {
			cursor[0] = args[0]
			cursor[1] = args[1]
			start[0] = cursor[0]
			start[1] = cursor[1]
		}

		// horizontal lineto (x)
		if command == 'h' {
			args[0] += cursor[0]
			command = 'H'
		}
		if command == 'H' {
			cursor[0] = args[0]
		}

		// vertical lineto (y)
		if command == 'v' {
			args[0] += cursor[1]
			command = 'V'
		}
		if command == 'V' {
			cursor[1] = args[0]
		}

		// lineto (x y)
		if command == 'l' {
			args[0] += cursor[0]
			args[1] += cursor[1]
			command = 'L'
		}
		if command == 'L' {
			cursor[0] = args[0]
			cursor[1] = args[1]
		}

		// curveto (x1 y1 x2 y2 x y)
		if command == 'c' {
			args[0] += cursor[0]
			args[1] += cursor[1]
			args[2] += cursor[0]
			args[3] += cursor[1]
			args[4] += cursor[0]
			args[5] += cursor[1]
			command = 'C'
		}
		if command == 'C' {
			cursor[0] = args[4]
			cursor[1] = args[5]
		}

		// smooth curveto (x2 y2 x y)
		if command == 's' {
			args[0] += cursor[0]
			args[1] += cursor[1]
			args[2] += cursor[0]
			args[3] += cursor[1]
			command = 'S'
		}
		if command == 'S' {
			cursor[0] = args[2]
			cursor[1] = args[3]
		}

		// quadratic Bezier curveto (x1 y1 x y)
		if command == 'q' {
			args[0] += cursor[0]
			args[1] += cursor[1]
			args[2] += cursor[0]
			args[3] += cursor[1]
			command = 'Q'
		}
		if command == 'Q' {
			cursor[0] = args[2]
			cursor[1] = args[3]
		}

		// smooth quadratic Bezier curveto (x y)
		if command == 't' {
			args[0] += cursor[0]
			args[1] += cursor[1]
			command = 'T'
		}
		if command == 'T' {
			cursor[0] = args[0]
			cursor[1] = args[1]
		}

		// elliptical arc (rx ry x-axis-rotation large-arc-flag sweep-flag x y)
		if command == 'a' {
			args[5] += cursor[0]
			args[6] += cursor[1]
			command = 'A'
		}
		if command == 'A' {
			cursor[0] = args[5]
			cursor[1] = args[6]
		}

		// closepath
		if command == 'z' || command == 'Z' {
			cursor[0] = start[0]
			cursor[1] = start[1]
			command = 'z'
		}

		newData = append(newData, PathDataItem{Command: command, Args: args})
	}
	return newData
}

// a2c converts an arc to cubic Bezier curves.
// Based on code from Snap.svg (Apache 2 license).
// Ported from SVGO's a2c() in plugins/_path.js.
func a2c(x1, y1, rx, ry, angle, largeArcFlag, sweepFlag, x2, y2 float64, recursive []float64) []float64 {
	_120 := math.Pi * 120 / 180
	rad := math.Pi / 180 * angle

	var res []float64
	rotateX := func(x, y, rad float64) float64 {
		return x*math.Cos(rad) - y*math.Sin(rad)
	}
	rotateY := func(x, y, rad float64) float64 {
		return x*math.Sin(rad) + y*math.Cos(rad)
	}

	var f1, f2, cx, cy float64

	if recursive == nil {
		x1 = rotateX(x1, y1, -rad)
		y1 = rotateY(x1, y1, -rad)
		x2 = rotateX(x2, y2, -rad)
		y2 = rotateY(x2, y2, -rad)
		x := (x1 - x2) / 2
		y := (y1 - y2) / 2
		h := (x*x)/(rx*rx) + (y*y)/(ry*ry)
		if h > 1 {
			h = math.Sqrt(h)
			rx = h * rx
			ry = h * ry
		}
		rx2 := rx * rx
		ry2 := ry * ry
		sign := -1.0
		if largeArcFlag != sweepFlag {
			sign = 1.0
		}
		k := sign * math.Sqrt(math.Abs(
			(rx2*ry2-rx2*y*y-ry2*x*x)/(rx2*y*y+ry2*x*x),
		))
		cx = (k*rx*y)/ry + (x1+x2)/2
		cy = (k*-ry*x)/rx + (y1+y2)/2

		// toFixed(9) equivalent
		f1val := (y1 - cy) / ry
		f1val = math.Round(f1val*1e9) / 1e9
		f1 = math.Asin(f1val)

		f2val := (y2 - cy) / ry
		f2val = math.Round(f2val*1e9) / 1e9
		f2 = math.Asin(f2val)

		if x1 < cx {
			f1 = math.Pi - f1
		}
		if x2 < cx {
			f2 = math.Pi - f2
		}
		if f1 < 0 {
			f1 = math.Pi*2 + f1
		}
		if f2 < 0 {
			f2 = math.Pi*2 + f2
		}
		if sweepFlag != 0 && f1 > f2 {
			f1 = f1 - math.Pi*2
		}
		if sweepFlag == 0 && f2 > f1 {
			f2 = f2 - math.Pi*2
		}
	} else {
		f1 = recursive[0]
		f2 = recursive[1]
		cx = recursive[2]
		cy = recursive[3]
	}

	df := f2 - f1
	if math.Abs(df) > _120 {
		f2old := f2
		x2old := x2
		y2old := y2
		sign := -1.0
		if sweepFlag != 0 && f2 > f1 {
			sign = 1.0
		}
		f2 = f1 + _120*sign
		x2 = cx + rx*math.Cos(f2)
		y2 = cy + ry*math.Sin(f2)
		res = a2c(x2, y2, rx, ry, angle, 0, sweepFlag, x2old, y2old, []float64{f2, f2old, cx, cy})
	}

	df = f2 - f1
	c1 := math.Cos(f1)
	s1 := math.Sin(f1)
	c2 := math.Cos(f2)
	s2 := math.Sin(f2)
	t := math.Tan(df / 4)
	hx := (4.0 / 3) * rx * t
	hy := (4.0 / 3) * ry * t
	m := []float64{
		-hx * s1,
		hy * c1,
		x2 + hx*s2 - x1,
		y2 - hy*c2 - y1,
		x2 - x1,
		y2 - y1,
	}

	if recursive != nil {
		return append(m, res...)
	}

	res = append(m, res...)
	newres := make([]float64, len(res))
	for i := 0; i < len(res); i++ {
		if i%2 != 0 {
			newres[i] = rotateY(res[i-1], res[i], rad)
		} else {
			newres[i] = rotateX(res[i], res[i+1], rad)
		}
	}
	return newres
}

// gatherPoints collects sample points from absolute path commands.
// Ported from SVGO's gatherPoints() in plugins/_path.js.
func gatherPoints(pathData []PathDataItem) hullPoints {
	points := hullPoints{}

	var prevCtrlPoint vec2

	addPoint := func(path *hullPoint, point vec2) {
		if len(path.list) == 0 || point[1] > path.list[path.maxY][1] {
			path.maxY = len(path.list)
			if len(points.list) > 0 {
				points.maxY = math.Max(point[1], points.maxY)
			} else {
				points.maxY = point[1]
			}
		}
		if len(path.list) == 0 || point[0] > path.list[path.maxX][0] {
			path.maxX = len(path.list)
			if len(points.list) > 0 {
				points.maxX = math.Max(point[0], points.maxX)
			} else {
				points.maxX = point[0]
			}
		}
		if len(path.list) == 0 || point[1] < path.list[path.minY][1] {
			path.minY = len(path.list)
			if len(points.list) > 0 {
				points.minY = math.Min(point[1], points.minY)
			} else {
				points.minY = point[1]
			}
		}
		if len(path.list) == 0 || point[0] < path.list[path.minX][0] {
			path.minX = len(path.list)
			if len(points.list) > 0 {
				points.minX = math.Min(point[0], points.minX)
			} else {
				points.minX = point[0]
			}
		}
		path.list = append(path.list, point)
	}

	for i := 0; i < len(pathData); i++ {
		pathDataItem := pathData[i]
		var subPath *hullPoint
		if len(points.list) == 0 {
			subPath = &hullPoint{}
		} else {
			subPath = &points.list[len(points.list)-1]
		}

		var prev *PathDataItem
		if i > 0 {
			prev = &pathData[i-1]
		}

		var basePoint *vec2
		if len(subPath.list) > 0 {
			bp := subPath.list[len(subPath.list)-1]
			basePoint = &bp
		}

		data := pathDataItem.Args
		var ctrlPoint *vec2
		if basePoint != nil {
			cp := *basePoint
			ctrlPoint = &cp
		}

		// toAbsolute converts relative coord n at index idx to absolute
		toAbsolute := func(n float64, idx int) float64 {
			if basePoint == nil {
				return n
			}
			return n + basePoint[idx%2]
		}

		switch pathDataItem.Command {
		case 'M':
			newSubPath := hullPoint{}
			points.list = append(points.list, newSubPath)
			subPath = &points.list[len(points.list)-1]

		case 'H':
			if basePoint != nil {
				addPoint(subPath, vec2{data[0], basePoint[1]})
			}

		case 'V':
			if basePoint != nil {
				addPoint(subPath, vec2{basePoint[0], data[0]})
			}

		case 'Q':
			addPoint(subPath, vec2{data[0], data[1]})
			prevCtrlPoint = vec2{data[2] - data[0], data[3] - data[1]}

		case 'T':
			if basePoint != nil && prev != nil &&
				(prev.Command == 'Q' || prev.Command == 'T') {
				cp := vec2{
					basePoint[0] + prevCtrlPoint[0],
					basePoint[1] + prevCtrlPoint[1],
				}
				ctrlPoint = &cp
				addPoint(subPath, *ctrlPoint)
				prevCtrlPoint = vec2{data[0] - ctrlPoint[0], data[1] - ctrlPoint[1]}
			}

		case 'C':
			if basePoint != nil {
				addPoint(subPath, vec2{
					0.5 * (basePoint[0] + data[0]),
					0.5 * (basePoint[1] + data[1]),
				})
			}
			addPoint(subPath, vec2{
				0.5 * (data[0] + data[2]),
				0.5 * (data[1] + data[3]),
			})
			addPoint(subPath, vec2{
				0.5 * (data[2] + data[4]),
				0.5 * (data[3] + data[5]),
			})
			prevCtrlPoint = vec2{data[4] - data[2], data[5] - data[3]}

		case 'S':
			if basePoint != nil && prev != nil &&
				(prev.Command == 'C' || prev.Command == 'S') {
				addPoint(subPath, vec2{
					basePoint[0] + 0.5*prevCtrlPoint[0],
					basePoint[1] + 0.5*prevCtrlPoint[1],
				})
				cp := vec2{
					basePoint[0] + prevCtrlPoint[0],
					basePoint[1] + prevCtrlPoint[1],
				}
				ctrlPoint = &cp
			}
			if ctrlPoint != nil {
				addPoint(subPath, vec2{
					0.5 * (ctrlPoint[0] + data[0]),
					0.5 * (ctrlPoint[1] + data[1]),
				})
			}
			addPoint(subPath, vec2{
				0.5 * (data[0] + data[2]),
				0.5 * (data[1] + data[3]),
			})
			prevCtrlPoint = vec2{data[2] - data[0], data[3] - data[1]}

		case 'A':
			if basePoint != nil {
				// Convert the arc to Bezier curves and use the same approximation
				curves := a2c(basePoint[0], basePoint[1],
					data[0], data[1], data[2], data[3], data[4], data[5], data[6], nil)

				bp := *basePoint
				for len(curves) >= 6 {
					cData := make([]float64, 6)
					copy(cData, curves[:6])
					curves = curves[6:]
					// map toAbsolute
					for ci := range cData {
						cData[ci] = toAbsolute(cData[ci], ci)
					}

					addPoint(subPath, vec2{
						0.5 * (bp[0] + cData[0]),
						0.5 * (bp[1] + cData[1]),
					})
					addPoint(subPath, vec2{
						0.5 * (cData[0] + cData[2]),
						0.5 * (cData[1] + cData[3]),
					})
					addPoint(subPath, vec2{
						0.5 * (cData[2] + cData[4]),
						0.5 * (cData[3] + cData[5]),
					})

					if len(curves) > 0 {
						bp = vec2{cData[4], cData[5]}
						basePointCopy := bp
						basePoint = &basePointCopy
						addPoint(subPath, bp)
					}
				}
			}
		}

		// Save final command coordinates
		if len(data) >= 2 {
			addPoint(subPath, vec2{data[len(data)-2], data[len(data)-1]})
		}

		// Write back updated subPath to points.list
		if len(points.list) > 0 {
			points.list[len(points.list)-1] = *subPath
		}
	}

	return points
}

// convexHull computes the convex hull of a set of points using monotone chain algorithm.
// Ported from SVGO's convexHull() in plugins/_path.js.
func convexHull(points hullPoint) hullPoint {
	// Sort points by x, then by y
	sortPoints(points.list)

	lower := make([]vec2, 0)
	minY := 0
	bottom := 0
	for i := 0; i < len(points.list); i++ {
		for len(lower) >= 2 &&
			cross2D(lower[len(lower)-2], lower[len(lower)-1], points.list[i]) <= 0 {
			lower = lower[:len(lower)-1]
		}
		if points.list[i][1] < points.list[minY][1] {
			minY = i
			bottom = len(lower)
		}
		lower = append(lower, points.list[i])
	}

	upper := make([]vec2, 0)
	maxY := len(points.list) - 1
	top := 0
	for i := len(points.list) - 1; i >= 0; i-- {
		for len(upper) >= 2 &&
			cross2D(upper[len(upper)-2], upper[len(upper)-1], points.list[i]) <= 0 {
			upper = upper[:len(upper)-1]
		}
		if points.list[i][1] > points.list[maxY][1] {
			maxY = i
			top = len(upper)
		}
		upper = append(upper, points.list[i])
	}

	// Last points are equal to starting points of the other part
	if len(upper) > 0 {
		upper = upper[:len(upper)-1]
	}
	if len(lower) > 0 {
		lower = lower[:len(lower)-1]
	}

	hullList := append(lower, upper...)

	hullLen := len(hullList)
	if hullLen == 0 {
		return hullPoint{list: hullList}
	}

	hull := hullPoint{
		list: hullList,
		minX: 0, // by sorting
		maxX: len(lower),
		minY: bottom,
		maxY: (len(lower) + top) % hullLen,
	}

	return hull
}

// sortPoints sorts a slice of vec2 by x coordinate, then by y.
func sortPoints(pts []vec2) {
	// Insertion sort matching JS Array.sort comparator behavior:
	// a[0] == b[0] ? a[1] - b[1] : a[0] - b[0]
	n := len(pts)
	for i := 1; i < n; i++ {
		key := pts[i]
		j := i - 1
		for j >= 0 && (pts[j][0] > key[0] || (pts[j][0] == key[0] && pts[j][1] > key[1])) {
			pts[j+1] = pts[j]
			j--
		}
		pts[j+1] = key
	}
}
