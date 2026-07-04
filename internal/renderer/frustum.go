package renderer

import (
	"math"

	"scop/internal/mathgl"
)

// ExtractFrustum derives the six clip planes from a projection*view matrix
// (Gribb-Hartmann). Each plane is (a,b,c,d) with the normal pointing into
// the frustum; planes are normalized so distances are in world units.
// Writes into the caller's array: no allocations.
func ExtractFrustum(pv *mathgl.Mat4, planes *[6]mathgl.Vec4) {
	row := func(i int) mathgl.Vec4 {
		return mathgl.Vec4{pv[i], pv[4+i], pv[8+i], pv[12+i]}
	}
	r0, r1, r2, r3 := row(0), row(1), row(2), row(3)

	planes[0] = addV4(r3, r0) // left
	planes[1] = subV4(r3, r0) // right
	planes[2] = addV4(r3, r1) // bottom
	planes[3] = subV4(r3, r1) // top
	planes[4] = addV4(r3, r2) // near
	planes[5] = subV4(r3, r2) // far

	for i := range planes {
		p := &planes[i]
		l := float32(math.Sqrt(float64(p[0]*p[0] + p[1]*p[1] + p[2]*p[2])))
		if l > 0 {
			p[0] /= l
			p[1] /= l
			p[2] /= l
			p[3] /= l
		}
	}
}

// AABBVisible reports whether the axis-aligned box intersects the frustum,
// using the positive-vertex test: if the box corner farthest along a
// plane's normal is behind that plane, the whole box is outside.
func AABBVisible(planes *[6]mathgl.Vec4, min, max mathgl.Vec3) bool {
	for i := range planes {
		p := &planes[i]
		var v mathgl.Vec3
		for a := 0; a < 3; a++ {
			if p[a] >= 0 {
				v[a] = max[a]
			} else {
				v[a] = min[a]
			}
		}
		if p[0]*v[0]+p[1]*v[1]+p[2]*v[2]+p[3] < 0 {
			return false
		}
	}
	return true
}

func addV4(a, b mathgl.Vec4) mathgl.Vec4 {
	return mathgl.Vec4{a[0] + b[0], a[1] + b[1], a[2] + b[2], a[3] + b[3]}
}

func subV4(a, b mathgl.Vec4) mathgl.Vec4 {
	return mathgl.Vec4{a[0] - b[0], a[1] - b[1], a[2] - b[2], a[3] - b[3]}
}
