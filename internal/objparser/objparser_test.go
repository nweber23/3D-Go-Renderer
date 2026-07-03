package objparser

import (
	"path/filepath"
	"testing"

	"scop/internal/mathgl"
)

func parseFile(t *testing.T, name string) *Mesh {
	t.Helper()
	m, err := ParseOBJ(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ParseOBJ(%s): %v", name, err)
	}
	return m
}

func TestParseTriangle(t *testing.T) {
	m := parseFile(t, "tri.obj")
	if len(m.Vertices) != 3 {
		t.Fatalf("vertices = %d, want 3", len(m.Vertices))
	}
	if len(m.Indices) != 3 {
		t.Fatalf("indices = %d, want 3", len(m.Indices))
	}
	want := []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}
	for i, w := range want {
		if m.Vertices[i].Pos != w {
			t.Errorf("vertex %d pos = %v, want %v", i, m.Vertices[i].Pos, w)
		}
	}
	if m.HasUVs || m.HasNormals {
		t.Fatalf("HasUVs=%v HasNormals=%v, want false/false", m.HasUVs, m.HasNormals)
	}
	if m.BBoxMin != (mathgl.Vec3{0, 0, 0}) || m.BBoxMax != (mathgl.Vec3{1, 1, 0}) {
		t.Fatalf("bbox = %v..%v", m.BBoxMin, m.BBoxMax)
	}
}

func TestParseQuadTriangulates(t *testing.T) {
	m := parseFile(t, "quad.obj")
	if len(m.Vertices) != 4 {
		t.Fatalf("vertices = %d, want 4 (deduplicated)", len(m.Vertices))
	}
	if len(m.Indices) != 6 {
		t.Fatalf("indices = %d, want 6 (two triangles)", len(m.Indices))
	}
}

func TestParseNegativeIndices(t *testing.T) {
	m := parseFile(t, "neg.obj")
	if len(m.Vertices) != 3 || len(m.Indices) != 3 {
		t.Fatalf("verts=%d indices=%d, want 3/3", len(m.Vertices), len(m.Indices))
	}
	if m.Vertices[m.Indices[2]].Pos != (mathgl.Vec3{0, 1, 0}) {
		t.Fatalf("last corner = %v", m.Vertices[m.Indices[2]].Pos)
	}
}

func TestParseSharedVerticesDeduped(t *testing.T) {
	m := parseFile(t, "shared.obj")
	if len(m.Vertices) != 4 {
		t.Fatalf("vertices = %d, want 4 (shared edge deduplicated)", len(m.Vertices))
	}
	if len(m.Indices) != 6 {
		t.Fatalf("indices = %d, want 6", len(m.Indices))
	}
}

func TestParseMalformedSkipsBadLines(t *testing.T) {
	m := parseFile(t, "malformed.obj")
	// Only "f 1 3 4" is a valid face; "f 1 2" has too few corners.
	if len(m.Indices) != 3 {
		t.Fatalf("indices = %d, want 3", len(m.Indices))
	}
}

func TestParseFullFaceSyntax(t *testing.T) {
	m := parseFile(t, "full.obj")
	if !m.HasUVs || !m.HasNormals {
		t.Fatalf("HasUVs=%v HasNormals=%v, want true/true", m.HasUVs, m.HasNormals)
	}
	if m.Vertices[1].UV != (mathgl.Vec2{1, 0}) {
		t.Fatalf("vertex 1 uv = %v, want {1,0}", m.Vertices[1].UV)
	}
	if m.Vertices[0].Normal != (mathgl.Vec3{0, 0, 1}) {
		t.Fatalf("vertex 0 normal = %v", m.Vertices[0].Normal)
	}
}

func TestParseNoFacesErrors(t *testing.T) {
	if _, err := parseOBJBytes([]byte("v 0 0 0\nv 1 0 0\n"), "."); err == nil {
		t.Fatal("expected error for obj without faces")
	}
}

func TestParseOutOfRangeIndexErrors(t *testing.T) {
	if _, err := parseOBJBytes([]byte("v 0 0 0\nf 1 2 9\n"), "."); err == nil {
		t.Fatal("expected error for out-of-range index")
	}
	if _, err := parseOBJBytes([]byte("v 0 0 0\nf 0 1 1\n"), "."); err == nil {
		t.Fatal("expected error for zero index")
	}
}

func TestParseSuppliedModels(t *testing.T) {
	cases := []struct {
		path        string
		wantIndices int
	}{
		// 42.obj: 18 tris + 29 quads = 18 + 58 = 76 triangles.
		{"../../models/42.obj", 76 * 3},
		// both teapots ship pre-triangulated with 6320 faces.
		{"../../models/teapot.obj", 6320 * 3},
		{"../../models/teapot2.obj", 6320 * 3},
	}
	for _, c := range cases {
		m, err := ParseOBJ(c.path)
		if err != nil {
			t.Fatalf("ParseOBJ(%s): %v", c.path, err)
		}
		if len(m.Indices) != c.wantIndices {
			t.Errorf("%s: indices = %d, want %d", c.path, len(m.Indices), c.wantIndices)
		}
		if len(m.Groups) < 1 {
			t.Errorf("%s: no material groups", c.path)
		}
		total := 0
		for _, g := range m.Groups {
			total += g.IndexCount
		}
		if total != len(m.Indices) {
			t.Errorf("%s: group counts sum %d != %d indices", c.path, total, len(m.Indices))
		}
	}
}
