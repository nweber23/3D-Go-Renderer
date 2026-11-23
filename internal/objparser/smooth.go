package objparser

import "scop/internal/mathgl"

// SmoothNormals gives a mesh without vn data per-vertex normals: the
// area-weighted average of every adjacent face normal, welded by position
// so UV/material seams that split vertices still shade continuously.
// No-op when the mesh already has normals.
func SmoothNormals(m *Mesh) {
	if m.HasNormals {
		return
	}
	sum := make(map[mathgl.Vec3]mathgl.Vec3, len(m.Vertices))
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Vertices[m.Indices[i]].Pos, m.Vertices[m.Indices[i+1]].Pos, m.Vertices[m.Indices[i+2]].Pos
		fn := b.Sub(a).Cross(c.Sub(a)) // length = 2*area, so area-weighted
		for _, p := range [3]mathgl.Vec3{a, b, c} {
			sum[p] = sum[p].Add(fn)
		}
	}
	for i := range m.Vertices {
		if n := sum[m.Vertices[i].Pos]; n.Len() > 0 {
			m.Vertices[i].Normal = n.Normalize()
		}
	}
	m.HasNormals = true
}
