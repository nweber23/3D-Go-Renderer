package objparser

import (
	"testing"

	"scop/internal/mathgl"
)

// Two coplanar triangles sharing an edge, but with split (seam) vertices:
// welding by position must still give every vertex the same +Z normal.
func TestSmoothNormalsWeldsSeams(t *testing.T) {
	p := []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}}
	m := &Mesh{Indices: []uint32{0, 1, 2, 3, 4, 5}}
	for _, v := range p {
		m.Vertices = append(m.Vertices, Vertex{Pos: v})
	}
	SmoothNormals(m)
	if !m.HasNormals {
		t.Fatal("HasNormals not set")
	}
	for i, v := range m.Vertices {
		if v.Normal != (mathgl.Vec3{0, 0, 1}) {
			t.Errorf("vertex %d normal = %v, want +Z", i, v.Normal)
		}
	}
}
