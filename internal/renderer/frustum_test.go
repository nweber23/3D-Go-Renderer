package renderer

import (
	"math"
	"testing"

	"scop/internal/mathgl"
)

// testFrustum: camera at (0,0,10) looking at the origin, 45-degree fov,
// square aspect, near 0.1, far 100.
func testFrustum() [6]mathgl.Vec4 {
	proj := mathgl.Perspective(float32(math.Pi/4), 1, 0.1, 100)
	view := mathgl.LookAt(mathgl.Vec3{0, 0, 10}, mathgl.Vec3{}, mathgl.Vec3{0, 1, 0})
	pv := proj.Mul(view)
	var planes [6]mathgl.Vec4
	ExtractFrustum(&pv, &planes)
	return planes
}

func TestAABBVisibleAtOrigin(t *testing.T) {
	planes := testFrustum()
	if !AABBVisible(&planes, mathgl.Vec3{-1, -1, -1}, mathgl.Vec3{1, 1, 1}) {
		t.Fatal("unit box at origin must be visible")
	}
}

func TestAABBCulledBehindCamera(t *testing.T) {
	planes := testFrustum()
	if AABBVisible(&planes, mathgl.Vec3{-1, -1, 150}, mathgl.Vec3{1, 1, 152}) {
		t.Fatal("box behind the camera must be culled")
	}
}

func TestAABBCulledFarLeft(t *testing.T) {
	planes := testFrustum()
	if AABBVisible(&planes, mathgl.Vec3{-500, -1, -1}, mathgl.Vec3{-498, 1, 1}) {
		t.Fatal("box far outside the fov must be culled")
	}
}

func TestAABBCulledBeyondFar(t *testing.T) {
	planes := testFrustum()
	if AABBVisible(&planes, mathgl.Vec3{-1, -1, -300}, mathgl.Vec3{1, 1, -290}) {
		t.Fatal("box beyond the far plane must be culled")
	}
}

func TestAABBStraddlingPlaneVisible(t *testing.T) {
	planes := testFrustum()
	// Straddles the left frustum plane: partially inside.
	if !AABBVisible(&planes, mathgl.Vec3{-50, -1, -1}, mathgl.Vec3{0, 1, 1}) {
		t.Fatal("box straddling a frustum plane must be visible")
	}
}

func BenchmarkFrustumCull(b *testing.B) {
	planes := testFrustum()
	min := mathgl.Vec3{-1, -1, -1}
	max := mathgl.Vec3{1, 1, 1}
	b.ReportAllocs()
	sink := false
	for i := 0; i < b.N; i++ {
		sink = AABBVisible(&planes, min, max)
	}
	_ = sink
}
