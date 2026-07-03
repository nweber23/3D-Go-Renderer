package objparser

import (
	"math"

	"scop/internal/mathgl"
)

// triangulate converts a polygon into triangles by ear clipping, appending
// index triples to out. corners[i] is the position of polygon corner i and
// poly[i] is the (mesh) index emitted for it. Handles concave polygons,
// collinear/duplicate points and non-coplanar polygons (via projection onto
// the dominant plane of the Newell normal). Degenerate polygons (under 3
// corners, or zero area) produce no output. Output triangles preserve the
// polygon's winding.
func triangulate(corners []mathgl.Vec3, poly []uint32, out []uint32) []uint32 {
	n := len(poly)
	if n < 3 || len(corners) != n {
		return out
	}

	normal := newellNormal(corners)
	nx, ny, nz := abs(normal[0]), abs(normal[1]), abs(normal[2])
	if nx+ny+nz == 0 {
		return out // zero area: collinear or coincident points
	}

	// Project onto the plane dropping the dominant normal axis, keeping the
	// remaining axes in cyclic order so the 2D signed area keeps the sign of
	// the dominant normal component.
	var u, v int
	var sign float64
	switch {
	case nx >= ny && nx >= nz:
		u, v = 1, 2
		sign = signOf(normal[0])
	case ny >= nz:
		u, v = 2, 0
		sign = signOf(normal[1])
	default:
		u, v = 0, 1
		sign = signOf(normal[2])
	}
	pts := make([][2]float64, n)
	scale := 0.0
	for i, c := range corners {
		pts[i] = [2]float64{float64(c[u]), float64(c[v])}
		scale = math.Max(scale, math.Max(math.Abs(pts[i][0]), math.Abs(pts[i][1])))
	}
	epsArea := 1e-12 * scale * scale

	// Working list of remaining corner slots (indices into pts/poly).
	rem := make([]int, n)
	for i := range rem {
		rem[i] = i
	}

	for len(rem) > 3 {
		clipped := false
		for i := 0; i < len(rem); i++ {
			prev := rem[(i+len(rem)-1)%len(rem)]
			cur := rem[i]
			next := rem[(i+1)%len(rem)]

			cross := cross2(pts[prev], pts[cur], pts[next]) * sign
			if cross < -epsArea {
				continue // reflex corner, not an ear
			}
			if cross <= epsArea {
				// Collinear or spike corner: clipping it loses no area.
				rem = append(rem[:i], rem[i+1:]...)
				clipped = true
				break
			}
			if earContainsPoint(pts, rem, prev, cur, next, sign, epsArea) {
				continue
			}
			out = append(out, poly[prev], poly[cur], poly[next])
			rem = append(rem[:i], rem[i+1:]...)
			clipped = true
			break
		}
		if !clipped {
			// Numerically stuck (e.g. self-intersecting input): fan out the
			// remainder rather than dropping it.
			for i := 1; i < len(rem)-1; i++ {
				out = append(out, poly[rem[0]], poly[rem[i]], poly[rem[i+1]])
			}
			return out
		}
	}
	if len(rem) == 3 {
		if cross2(pts[rem[0]], pts[rem[1]], pts[rem[2]])*sign > epsArea {
			out = append(out, poly[rem[0]], poly[rem[1]], poly[rem[2]])
		}
	}
	return out
}

// newellNormal computes the polygon normal by Newell's method: robust for
// concave and non-coplanar polygons.
func newellNormal(corners []mathgl.Vec3) mathgl.Vec3 {
	var n mathgl.Vec3
	for i, a := range corners {
		b := corners[(i+1)%len(corners)]
		n[0] += (a[1] - b[1]) * (a[2] + b[2])
		n[1] += (a[2] - b[2]) * (a[0] + b[0])
		n[2] += (a[0] - b[0]) * (a[1] + b[1])
	}
	return n
}

// cross2 is the z component of (b-a) x (c-b): twice the signed area.
func cross2(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-b[1]) - (b[1]-a[1])*(c[0]-b[0])
}

// earContainsPoint reports whether any remaining corner other than the ear's
// own corners lies inside triangle (prev, cur, next). Corners coincident
// with the ear's corners (duplicate points) do not block the ear.
func earContainsPoint(pts [][2]float64, rem []int, prev, cur, next int, sign, epsArea float64) bool {
	a, b, c := pts[prev], pts[cur], pts[next]
	for _, j := range rem {
		if j == prev || j == cur || j == next {
			continue
		}
		p := pts[j]
		if coincident(p, a) || coincident(p, b) || coincident(p, c) {
			continue
		}
		if cross2(a, b, p)*sign >= -epsArea &&
			cross2(b, c, p)*sign >= -epsArea &&
			cross2(c, a, p)*sign >= -epsArea {
			return true
		}
	}
	return false
}

func coincident(a, b [2]float64) bool {
	const eps = 1e-12
	dx, dy := a[0]-b[0], a[1]-b[1]
	return dx*dx+dy*dy < eps
}

func abs(f float32) float64 {
	return math.Abs(float64(f))
}

func signOf(f float32) float64 {
	if f < 0 {
		return -1
	}
	return 1
}
