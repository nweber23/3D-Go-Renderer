package objparser

import (
	"math"
	"testing"

	"scop/internal/mathgl"
)

func TestParseUVMode(t *testing.T) {
	cases := map[string]UVMode{
		"auto":        UVAuto,
		"planar":      UVPlanar,
		"cylindrical": UVCylindrical,
		"spherical":   UVSpherical,
	}
	for s, want := range cases {
		got, err := ParseUVMode(s)
		if err != nil || got != want {
			t.Errorf("ParseUVMode(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	if _, err := ParseUVMode("banana"); err == nil {
		t.Error("ParseUVMode(banana) must error")
	}
}

func TestChooseUVMode(t *testing.T) {
	cases := []struct {
		name     string
		min, max mathgl.Vec3
		want     UVMode
	}{
		{"tall pole -> cylindrical", mathgl.Vec3{0, 0, 0}, mathgl.Vec3{1, 10, 1}, UVCylindrical},
		{"flat plate -> planar", mathgl.Vec3{0, 0, 0}, mathgl.Vec3{4, 0.2, 4}, UVPlanar},
		{"cube -> spherical", mathgl.Vec3{0, 0, 0}, mathgl.Vec3{2, 2, 2}, UVSpherical},
		// 42 logo bbox: a flat extruded plaque.
		{"42 logo -> planar", mathgl.Vec3{-0.227, -1.217, 1.134}, mathgl.Vec3{0.232, 1.12, 2.843}, UVPlanar},
		// teapot bbox: roundish.
		{"teapot -> spherical", mathgl.Vec3{-3, 0, -2}, mathgl.Vec3{3.434, 3.15, 2}, UVSpherical},
	}
	for _, c := range cases {
		if got := chooseUVMode(c.min, c.max); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestGenerateUVsPlanarCoversUnitSquare(t *testing.T) {
	m := &Mesh{
		Vertices: []Vertex{
			{Pos: mathgl.Vec3{0, 0, 0}},
			{Pos: mathgl.Vec3{4, 0, 0}},
			{Pos: mathgl.Vec3{4, 0.2, 4}},
			{Pos: mathgl.Vec3{0, 0, 4}},
		},
		Indices: []uint32{0, 1, 2, 0, 2, 3},
		BBoxMin: mathgl.Vec3{0, 0, 0},
		BBoxMax: mathgl.Vec3{4, 0.2, 4},
	}
	GenerateUVs(m, UVPlanar)
	if !m.HasUVs {
		t.Fatal("HasUVs not set")
	}
	minU, maxU := float32(9e9), float32(-9e9)
	for _, v := range m.Vertices {
		if v.UV[0] < minU {
			minU = v.UV[0]
		}
		if v.UV[0] > maxU {
			maxU = v.UV[0]
		}
		if v.UV[0] < 0 || v.UV[0] > 1 || v.UV[1] < 0 || v.UV[1] > 1 {
			t.Fatalf("planar uv out of [0,1]: %v", v.UV)
		}
	}
	if minU > 1e-4 || maxU < 1-1e-4 {
		t.Fatalf("planar u range [%v,%v] does not cover [0,1]", minU, maxU)
	}
}

func TestGenerateUVsTeapotInRange(t *testing.T) {
	m, err := ParseOBJ("../../models/teapot.obj")
	if err != nil {
		t.Fatal(err)
	}
	GenerateUVs(m, UVAuto)
	if !m.HasUVs {
		t.Fatal("HasUVs not set")
	}
	for i, v := range m.Vertices {
		// Seam-duplicated vertices may carry u up to 1.5.
		if v.UV[0] < -1e-4 || v.UV[0] > 1.5+1e-4 || v.UV[1] < -1e-4 || v.UV[1] > 1+1e-4 {
			t.Fatalf("vertex %d uv out of range: %v", i, v.UV)
		}
	}
}

func TestGenerateUVsAutoNoopWhenPresent(t *testing.T) {
	m, err := ParseOBJ("testdata/full.obj")
	if err != nil {
		t.Fatal(err)
	}
	orig := m.Vertices[1].UV
	GenerateUVs(m, UVAuto)
	if m.Vertices[1].UV != orig {
		t.Fatalf("UVAuto overwrote existing uvs: %v -> %v", orig, m.Vertices[1].UV)
	}
}

func TestGenerateUVsCylindricalSeam(t *testing.T) {
	// A full ring of quads around the y axis: without seam handling the
	// wrap-around triangles span u from ~0.9 back to 0.
	const n = 8
	m := &Mesh{}
	for ring := 0; ring < 2; ring++ {
		for i := 0; i < n; i++ {
			a := 2 * math.Pi * float64(i) / n
			m.Vertices = append(m.Vertices, Vertex{Pos: mathgl.Vec3{
				float32(math.Cos(a)), float32(ring), float32(math.Sin(a)),
			}})
		}
	}
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		a, b := uint32(i), uint32(j)
		c, d := uint32(n+j), uint32(n+i)
		m.Indices = append(m.Indices, a, b, c, a, c, d)
	}
	m.BBoxMin = mathgl.Vec3{-1, 0, -1}
	m.BBoxMax = mathgl.Vec3{1, 1, 1}
	GenerateUVs(m, UVCylindrical)

	for t3 := 0; t3+2 < len(m.Indices); t3 += 3 {
		u0 := m.Vertices[m.Indices[t3]].UV[0]
		u1 := m.Vertices[m.Indices[t3+1]].UV[0]
		u2 := m.Vertices[m.Indices[t3+2]].UV[0]
		span := math.Max(float64(u0), math.Max(float64(u1), float64(u2))) -
			math.Min(float64(u0), math.Min(float64(u1), float64(u2)))
		if span > 0.5 {
			t.Fatalf("triangle %d spans u=%v: seam not fixed (u %v %v %v)",
				t3/3, span, u0, u1, u2)
		}
	}
}
