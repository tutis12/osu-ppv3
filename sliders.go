package main

import (
	"math"
	"ppv3/dotosu"
)

// === constants chosen to mirror osu!lazer PathApproximator ===
const (
	bezTolSq   = 0.25 * 0.25 // BEZIER_TOLERANCE^2
	arcTol     = 0.10        // CIRCULAR_ARC_TOLERANCE (sagitta)
	catmullDet = 50          // CATMULL_DETAIL (samples per segment)
)

// ApproximateSliderPath returns a polyline approximation for a slider path.
// Points are absolute playfield coordinates; first element is the slider head.
// Duplicates and zero-length steps are removed.
func ApproximateSliderPath(slider dotosu.Slider) []Vec {
	path := slider.Path
	var poly []Vec

	add := func(v Vec) {
		n := len(poly)
		if n == 0 || (poly[n-1].X != v.X || poly[n-1].Y != v.Y) {
			poly = append(poly, v)
		}
	}
	appendMany := func(pts []Vec) {
		for i := range pts {
			add(pts[i])
		}
	}

	switch path.Type {
	case dotosu.PathLinear:
		for _, p := range path.Segments[0].Points {
			add(Vec{float64(p.X), float64(p.Y)})
		}

	case dotosu.PathCatmull:
		pts := path.Segments[0].Points
		appendMany(approximateCatmull(toVecs(pts)))

	case dotosu.PathPerfect:
		pts := path.Segments[0].Points
		v := toVecs(pts)
		// Perfect circle only for exactly 3 points; otherwise lazer falls back to Bezier.
		if len(v) == 3 {
			appendMany(approximateCircularArc(v[0], v[1], v[2]))
		} else {
			appendMany(approximateBezier(v))
		}

	default: // Bezier with red-anchor segmentation
		for si, seg := range path.Segments {
			v := toVecs(seg.Points)
			if len(v) < 2 {
				continue
			}
			pts := approximateBezier(v)
			// Avoid duplicating the shared point between consecutive Bezier segments.
			if si > 0 && len(pts) > 0 && len(poly) > 0 && almostEq(poly[len(poly)-1], pts[0]) {
				pts = pts[1:]
			}
			appendMany(pts)
		}
	}

	// compact collinear/zero-length steps
	poly = dedupeCollinear(poly)
	return poly
}

// --- Bezier (adaptive subdivision, identical strategy to lazer) ---

func approximateBezier(cp []Vec) []Vec {
	if len(cp) == 0 {
		return nil
	}
	var out []Vec
	stack := make([][]Vec, 0, 32)
	stack = append(stack, cp)

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if bezierFlatEnough(cur) {
			// For a flat segment, emit its start point.
			out = append(out, cur[0])
			continue
		}
		// Subdivide (de Casteljau) into left & right halves and process right first
		// so the resulting points come out in correct order.
		l, r := bezierSubdivide(cur)
		stack = append(stack, r)
		stack = append(stack, l)
	}
	// Finally, emit the original end point.
	out = append(out, cp[len(cp)-1])
	return out
}

func bezierFlatEnough(cp []Vec) bool {
	// Check second differences against tolerance.
	for i := 1; i < len(cp)-1; i++ {
		dx := cp[i-1].X - 2*cp[i].X + cp[i+1].X
		dy := cp[i-1].Y - 2*cp[i].Y + cp[i+1].Y
		if dx*dx+dy*dy > bezTolSq {
			return false
		}
	}
	return true
}

func bezierSubdivide(cp []Vec) (left, right []Vec) {
	n := len(cp)
	buf := make([]Vec, n*(n+1)/2)

	// first row = control points
	for i := 0; i < n; i++ {
		buf[i] = cp[i]
	}

	rowStart := 0
	nextRowStart := n
	for r := 1; r < n; r++ {
		for i := 0; i < n-r; i++ {
			a := buf[rowStart+i]
			b := buf[rowStart+i+1]
			buf[nextRowStart+i] = Vec{(a.X + b.X) * 0.5, (a.Y + b.Y) * 0.5}
		}
		rowStart = nextRowStart
		nextRowStart += n - r
	}

	// Left: first element of each row
	left = make([]Vec, n)
	rowStart = 0
	for r := 0; r < n; r++ {
		left[r] = buf[rowStart]
		rowStart += n - r
	}

	// Right: last element of each row, **reversed** (midpoint -> end)
	right = make([]Vec, n)
	rowStart = 0
	rowEnd := n - 1
	for r := 0; r < n; r++ {
		right[n-1-r] = buf[rowStart+rowEnd] // note the reverse index
		rowStart += n - r
		rowEnd--
	}
	return left, right
}

// --- Catmull-Rom (uniform, with lazer's detail count) ---

func approximateCatmull(pts []Vec) []Vec {
	n := len(pts)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []Vec{pts[0]}
	}
	out := make([]Vec, 0, (n-1)*catmullDet+1)
	for i := 0; i < n-1; i++ {
		p0 := pts[maxi(i-1, 0)]
		p1 := pts[i]
		p2 := pts[i+1]
		p3 := pts[mini(i+2, n-1)]
		// First point of the whole path
		if i == 0 {
			out = append(out, p1)
		}
		// Sample (0,1] for segments after the very first 0 to avoid duplicates
		for s := 1; s <= catmullDet; s++ {
			t := float64(s) / float64(catmullDet)
			out = append(out, catmullPoint(p0, p1, p2, p3, t))
		}
	}
	return out
}

func catmullPoint(p0, p1, p2, p3 Vec, t float64) Vec {
	t2 := t * t
	t3 := t2 * t
	return Vec{
		X: 0.5 * ((2 * p1.X) + (-p0.X+p2.X)*t + (2*p0.X-5*p1.X+4*p2.X-p3.X)*t2 + (-p0.X+3*p1.X-3*p2.X+p3.X)*t3),
		Y: 0.5 * ((2 * p1.Y) + (-p0.Y+p2.Y)*t + (2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2 + (-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3),
	}
}

// --- Perfect circle (3 points), with arc step from sagitta tolerance ---
func approximateCircularArc(p1, p2, p3 Vec) []Vec {
	// Collinear → straight line
	if collinear(p1, p2, p3) {
		return []Vec{p1, p3}
	}

	cx, cy, ok := circumcenter(p1, p2, p3)
	if !ok {
		return []Vec{p1, p3}
	}
	c := Vec{cx, cy}
	r := dist(c, p1)

	a1 := math.Atan2(p1.Y-cy, p1.X-cx)
	a3 := math.Atan2(p3.Y-cy, p3.X-cx)

	// Direction by cross((p2-p1),(p3-p2))
	dir := 1.0
	if cross(sub(p2, p1), sub(p3, p2)) < 0 {
		dir = -1.0
	}
	// Sweep from a1 -> a3 in chosen direction
	delta := angleDiff(a1, a3, dir)

	// Step angle from sagitta tolerance (matches lazer approach)
	step := 2 * math.Acos(clamp(1.0-arcTol/r, -1, 1))
	if step <= 0 || math.IsNaN(step) || step > math.Pi {
		step = math.Pi
	}
	steps := int(math.Ceil(math.Abs(delta) / step))
	if steps < 2 { // lazer keeps a minimum of two segments
		steps = 2
	}
	step = math.Copysign(step, dir)

	out := make([]Vec, 0, steps+1)
	out = append(out, p1)
	for i := 1; i < steps; i++ {
		a := a1 + float64(i)*step
		out = append(out, Vec{cx + math.Cos(a)*r, cy + math.Sin(a)*r})
	}
	out = append(out, p3)
	return out
}

// --- helpers ---

func toVecs(v2 []dotosu.Vec2) []Vec {
	out := make([]Vec, len(v2))
	for i, p := range v2 {
		out[i] = Vec{float64(p.X), float64(p.Y)}
	}
	return out
}

func dedupeCollinear(pts []Vec) []Vec {
	if len(pts) <= 2 {
		return pts
	}
	out := []Vec{pts[0]}
	for i := 1; i < len(pts)-1; i++ {
		a, b, c := out[len(out)-1], pts[i], pts[i+1]
		// remove exact duplicates or nearly-collinear middle points
		if almostEq(a, b) {
			continue
		}
		if math.Abs(cross(sub(b, a), sub(c, b))) < 1e-7 &&
			dot(norm(sub(b, a)), norm(sub(c, b))) > 0.999999 {
			continue
		}
		out = append(out, b)
	}
	if !almostEq(out[len(out)-1], pts[len(pts)-1]) {
		out = append(out, pts[len(pts)-1])
	}
	return out
}

func collinear(a, b, c Vec) bool {
	return math.Abs(cross(sub(b, a), sub(c, b))) < 1e-6
}

func circumcenter(a, b, c Vec) (x, y float64, ok bool) {
	d := 2 * (a.X*(b.Y-c.Y) + b.X*(c.Y-a.Y) + c.X*(a.Y-b.Y))
	if math.Abs(d) < 1e-8 {
		return 0, 0, false
	}
	a2 := a.X*a.X + a.Y*a.Y
	b2 := b.X*b.X + b.Y*b.Y
	c2 := c.X*c.X + c.Y*c.Y
	x = (a2*(b.Y-c.Y) + b2*(c.Y-a.Y) + c2*(a.Y-b.Y)) / d
	y = (a2*(c.X-b.X) + b2*(a.X-c.X) + c2*(b.X-a.X)) / d
	return x, y, true
}

func angleDiff(aStart, aEnd, dir float64) float64 {
	d := aEnd - aStart
	// Wrap to (-pi, pi]
	for d <= -math.Pi {
		d += 2 * math.Pi
	}
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	if dir < 0 && d > 0 {
		d -= 2 * math.Pi
	} else if dir > 0 && d < 0 {
		d += 2 * math.Pi
	}
	return d
}

func sub(a, b Vec) Vec       { return Vec{a.X - b.X, a.Y - b.Y} }
func dot(a, b Vec) float64   { return a.X*b.X + a.Y*b.Y }
func cross(a, b Vec) float64 { return a.X*b.Y - a.Y*b.X }
func dist(a, b Vec) float64  { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func norm(v Vec) Vec {
	l := math.Hypot(v.X, v.Y)
	if l == 0 {
		return Vec{0, 0}
	}
	return Vec{v.X / l, v.Y / l}
}
func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
func almostEq(a, b Vec) bool {
	return math.Abs(a.X-b.X) < 1e-9 && math.Abs(a.Y-b.Y) < 1e-9
}
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func GetSliderPosition(poly []Vec, progress float64) Vec {
	for i := 1; i < len(poly); i++ {
		dir := Vec{
			X: poly[i].X - poly[i-1].X,
			Y: poly[i].Y - poly[i-1].Y,
		}
		l := math.Hypot(dir.X, dir.Y)
		if progress <= l {
			return Vec{
				X: poly[i-1].X + dir.X*progress/l,
				Y: poly[i-1].Y + dir.Y*progress/l,
			}
		} else {
			progress -= l
		}
	}
	if len(poly) < 2 {
		panic("wtf")
		//return Vec{poly[0].X + progress, poly[0].Y}
	}
	from := poly[len(poly)-1]
	dir := Vec{
		X: poly[len(poly)-1].X - poly[len(poly)-2].X,
		Y: poly[len(poly)-1].Y - poly[len(poly)-2].Y,
	}

	l := math.Hypot(dir.X, dir.Y)

	return Vec{
		X: from.X + dir.X*progress/l,
		Y: from.Y + dir.Y*progress/l,
	}
}
