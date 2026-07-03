package objparser

import (
	"fmt"
	"math"

	"scop/internal/mathgl"
)

// UVMode selects the projection used when generating texture coordinates.
type UVMode int

const (
	UVAuto UVMode = iota
	UVPlanar
	UVCylindrical
	UVSpherical
)

// ParseUVMode maps the -uv flag values to a UVMode.
func ParseUVMode(s string) (UVMode, error) {
	switch s {
	case "auto":
		return UVAuto, nil
	case "planar":
		return UVPlanar, nil
	case "cylindrical":
		return UVCylindrical, nil
	case "spherical":
		return UVSpherical, nil
	}
	return UVAuto, fmt.Errorf("unknown uv mode %q (want auto|planar|cylindrical|spherical)", s)
}

// GenerateUVs assigns texture coordinates by projecting the mesh's shape.
// With UVAuto it keeps existing coordinates and otherwise picks the
// projection that best fits the bounding-box proportions. Wrapping
// projections duplicate vertices along the u seam so no triangle spans the
// wrap discontinuity.
func GenerateUVs(m *Mesh, mode UVMode) {
	if mode == UVAuto {
		if m.HasUVs {
			return
		}
		mode = chooseUVMode(m.BBoxMin, m.BBoxMax)
	}
	switch mode {
	case UVPlanar:
		planarUVs(m)
	case UVCylindrical:
		cylindricalUVs(m)
		fixSeam(m)
	case UVSpherical:
		sphericalUVs(m)
		fixSeam(m)
	}
	m.HasUVs = true
}

// chooseUVMode picks a projection from the bounding-box proportions:
// clearly elongated shapes wrap cylindrically around their long axis, flat
// shapes project onto their large face, everything else maps spherically.
func chooseUVMode(min, max mathgl.Vec3) UVMode {
	e := extents(min, max)
	a := axesByExtent(e)
	e0, e1, e2 := e[a[0]], e[a[1]], e[a[2]]
	switch {
	case e0 > 1.8*e1:
		return UVCylindrical
	case e2 < 0.25*e0:
		return UVPlanar
	default:
		return UVSpherical
	}
}

func extents(min, max mathgl.Vec3) mathgl.Vec3 {
	return max.Sub(min)
}

// axesByExtent returns the axis indices sorted by descending extent.
func axesByExtent(e mathgl.Vec3) [3]int {
	a := [3]int{0, 1, 2}
	for i := 0; i < 2; i++ {
		for j := i + 1; j < 3; j++ {
			if e[a[j]] > e[a[i]] {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
	return a
}

// planarUVs projects onto the plane spanned by the two largest axes,
// normalized to [0,1] over the bounding box.
func planarUVs(m *Mesh) {
	e := extents(m.BBoxMin, m.BBoxMax)
	a := axesByExtent(e)
	u, v := a[0], a[1]
	du, dv := safeExtent(e[u]), safeExtent(e[v])
	for i := range m.Vertices {
		p := m.Vertices[i].Pos
		m.Vertices[i].UV = mathgl.Vec2{
			(p[u] - m.BBoxMin[u]) / du,
			(p[v] - m.BBoxMin[v]) / dv,
		}
	}
}

// cylindricalUVs wraps u around the longest axis and maps v to the height
// along it.
func cylindricalUVs(m *Mesh) {
	e := extents(m.BBoxMin, m.BBoxMax)
	a := axesByExtent(e)
	h := a[0]          // height axis
	x, y := a[1], a[2] // circular plane axes
	dh := safeExtent(e[h])
	c := m.Center()
	for i := range m.Vertices {
		p := m.Vertices[i].Pos
		angle := math.Atan2(float64(p[y]-c[y]), float64(p[x]-c[x]))
		m.Vertices[i].UV = mathgl.Vec2{
			float32(angle/(2*math.Pi)) + 0.5,
			(p[h] - m.BBoxMin[h]) / dh,
		}
	}
}

// sphericalUVs maps direction from the bbox center to longitude/latitude.
func sphericalUVs(m *Mesh) {
	c := m.Center()
	for i := range m.Vertices {
		d := m.Vertices[i].Pos.Sub(c)
		r := float64(d.Len())
		if r == 0 {
			m.Vertices[i].UV = mathgl.Vec2{0.5, 0.5}
			continue
		}
		u := math.Atan2(float64(d[2]), float64(d[0]))/(2*math.Pi) + 0.5
		v := math.Asin(float64(d[1])/r)/math.Pi + 0.5
		m.Vertices[i].UV = mathgl.Vec2{float32(u), float32(v)}
	}
}

func safeExtent(e float32) float32 {
	if e <= 0 {
		return 1
	}
	return e
}

// fixSeam repairs triangles crossing the u wrap line (u jumps from ~1 back
// to ~0): corners on the low side are re-pointed at duplicated vertices
// with u+1 so interpolation runs forward across the seam (GL_REPEAT shows
// the correct texels for u > 1).
func fixSeam(m *Mesh) {
	dup := make(map[uint32]uint32)
	for t := 0; t+2 < len(m.Indices); t += 3 {
		i0, i1, i2 := m.Indices[t], m.Indices[t+1], m.Indices[t+2]
		u0 := m.Vertices[i0].UV[0]
		u1 := m.Vertices[i1].UV[0]
		u2 := m.Vertices[i2].UV[0]
		maxU := maxf(u0, maxf(u1, u2))
		minU := minf(u0, minf(u1, u2))
		if maxU-minU <= 0.5 {
			continue
		}
		for k := 0; k < 3; k++ {
			idx := m.Indices[t+k]
			if m.Vertices[idx].UV[0] < 0.5 {
				m.Indices[t+k] = dupVertex(m, dup, idx)
			}
		}
	}
}

func dupVertex(m *Mesh, dup map[uint32]uint32, idx uint32) uint32 {
	if d, ok := dup[idx]; ok {
		return d
	}
	v := m.Vertices[idx]
	v.UV[0] += 1
	d := uint32(len(m.Vertices))
	m.Vertices = append(m.Vertices, v)
	dup[idx] = d
	return d
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
