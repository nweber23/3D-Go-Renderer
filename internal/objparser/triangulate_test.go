package objparser

import (
	"math"
	"testing"

	"scop/internal/mathgl"
)

// triArea returns the area of triangle t (three corner indices into pts).
func triArea(pts []mathgl.Vec3, a, b, c uint32) float64 {
	ab := pts[b].Sub(pts[a])
	ac := pts[c].Sub(pts[a])
	return 0.5 * float64(ab.Cross(ac).Len())
}

func totalArea(pts []mathgl.Vec3, tris []uint32) float64 {
	sum := 0.0
	for i := 0; i+2 < len(tris); i += 3 {
		sum += triArea(pts, tris[i], tris[i+1], tris[i+2])
	}
	return sum
}

func seqIndices(n int) []uint32 {
	out := make([]uint32, n)
	for i := range out {
		out[i] = uint32(i)
	}
	return out
}

// pointInPolygon2D: ray casting on the xy plane.
func pointInPolygon2D(px, py float64, poly [][2]float64) bool {
	in := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if (yi > py) != (yj > py) &&
			px < (xj-xi)*(py-yi)/(yj-yi)+xi {
			in = !in
		}
		j = i
	}
	return in
}

func TestTriangulateSquare(t *testing.T) {
	pts := []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}}
	tris := triangulate(pts, seqIndices(4), nil)
	if len(tris) != 6 {
		t.Fatalf("got %d indices, want 6", len(tris))
	}
	if a := totalArea(pts, tris); math.Abs(a-1) > 1e-5 {
		t.Fatalf("area = %v, want 1", a)
	}
}

func TestTriangulateConvexPentagon(t *testing.T) {
	pts := []mathgl.Vec3{
		{0, 0, 0}, {2, 0, 0}, {2.6, 1.8, 0}, {1, 3, 0}, {-0.6, 1.8, 0},
	}
	tris := triangulate(pts, seqIndices(5), nil)
	if len(tris) != 9 {
		t.Fatalf("got %d indices, want 9 (3 triangles)", len(tris))
	}
	// Shoelace area of the pentagon.
	want := 0.0
	for i := 0; i < 5; i++ {
		a, b := pts[i], pts[(i+1)%5]
		want += float64(a[0]*b[1] - b[0]*a[1])
	}
	want = math.Abs(want) / 2
	if a := totalArea(pts, tris); math.Abs(a-want) > 1e-4 {
		t.Fatalf("area = %v, want %v", a, want)
	}
}

func TestTriangulateConcaveL(t *testing.T) {
	// L-shape: 2x2 square missing its top-right 1x1 quadrant. Area 3.
	poly2d := [][2]float64{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2}}
	pts := make([]mathgl.Vec3, len(poly2d))
	for i, p := range poly2d {
		pts[i] = mathgl.Vec3{float32(p[0]), float32(p[1]), 0}
	}
	tris := triangulate(pts, seqIndices(6), nil)
	if len(tris) != 12 {
		t.Fatalf("got %d indices, want 12 (4 triangles)", len(tris))
	}
	if a := totalArea(pts, tris); math.Abs(a-3) > 1e-4 {
		t.Fatalf("area = %v, want 3 (triangles escaped the concavity)", a)
	}
	for i := 0; i+2 < len(tris); i += 3 {
		a, b, c := pts[tris[i]], pts[tris[i+1]], pts[tris[i+2]]
		cx := float64(a[0]+b[0]+c[0]) / 3
		cy := float64(a[1]+b[1]+c[1]) / 3
		if !pointInPolygon2D(cx, cy, poly2d) {
			t.Errorf("triangle %d centroid (%v,%v) outside the L", i/3, cx, cy)
		}
		// Winding must match the polygon (CCW -> +z normal).
		n := b.Sub(a).Cross(c.Sub(a))
		if n[2] <= 0 {
			t.Errorf("triangle %d has inverted winding (normal %v)", i/3, n)
		}
	}
}

func TestTriangulateCollinearPoint(t *testing.T) {
	// Unit square with a redundant collinear point on the top edge.
	pts := []mathgl.Vec3{
		{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0.5, 1, 0}, {0, 1, 0},
	}
	tris := triangulate(pts, seqIndices(5), nil)
	if a := totalArea(pts, tris); math.Abs(a-1) > 1e-5 {
		t.Fatalf("area = %v, want 1", a)
	}
	for i := 0; i+2 < len(tris); i += 3 {
		if triArea(pts, tris[i], tris[i+1], tris[i+2]) < 1e-9 {
			t.Fatalf("zero-area triangle emitted at %d", i/3)
		}
	}
}

func TestTriangulateDegenerate(t *testing.T) {
	// Fewer than 3 corners.
	pts := []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}}
	if tris := triangulate(pts, seqIndices(2), nil); len(tris) != 0 {
		t.Fatalf("2-gon: got %d indices, want 0", len(tris))
	}
	// All points collinear: no area, nothing to emit.
	pts = []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}, {2, 0, 0}, {3, 0, 0}}
	if tris := triangulate(pts, seqIndices(4), nil); len(tris) != 0 {
		t.Fatalf("collinear: got %d indices, want 0", len(tris))
	}
	// All points coincident.
	pts = []mathgl.Vec3{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	if tris := triangulate(pts, seqIndices(4), nil); len(tris) != 0 {
		t.Fatalf("coincident: got %d indices, want 0", len(tris))
	}
}

func TestTriangulateNonCoplanarQuad(t *testing.T) {
	pts := []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}, {1, 1, 0.5}, {0, 1, 0}}
	tris := triangulate(pts, seqIndices(4), nil)
	if len(tris) != 6 {
		t.Fatalf("got %d indices, want 6", len(tris))
	}
	for i := 0; i+2 < len(tris); i += 3 {
		if triArea(pts, tris[i], tris[i+1], tris[i+2]) < 1e-6 {
			t.Fatalf("degenerate triangle in non-coplanar quad")
		}
	}
}

func TestTriangulateRemapsPolyIndices(t *testing.T) {
	// poly values are arbitrary mesh indices, not 0..n-1.
	pts := []mathgl.Vec3{{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}}
	tris := triangulate(pts, []uint32{7, 9, 11, 13}, nil)
	if len(tris) != 6 {
		t.Fatalf("got %d indices, want 6", len(tris))
	}
	for _, idx := range tris {
		if idx != 7 && idx != 9 && idx != 11 && idx != 13 {
			t.Fatalf("output index %d not from input poly", idx)
		}
	}
}
