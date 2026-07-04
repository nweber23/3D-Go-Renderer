package camera

import (
	"math"
	"testing"

	"scop/internal/mathgl"
)

const eps = 1e-4

func nearEq(a, b float32) bool {
	return math.Abs(float64(a-b)) < eps
}

func vecNearEq(a, b mathgl.Vec3) bool {
	return nearEq(a[0], b[0]) && nearEq(a[1], b[1]) && nearEq(a[2], b[2])
}

func TestNewLooksAtTargetFromZ(t *testing.T) {
	target := mathgl.Vec3{1, 2, 3}
	c := New(target, 10)
	wantEye := mathgl.Vec3{1, 2, 13}
	if !vecNearEq(c.Eye(), wantEye) {
		t.Fatalf("eye = %v, want %v", c.Eye(), wantEye)
	}
	want := mathgl.LookAt(wantEye, target, mathgl.Vec3{0, 1, 0})
	got := c.View()
	for i := range got {
		if !nearEq(got[i], want[i]) {
			t.Fatalf("view[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestOrbitMovesEyeAroundTarget(t *testing.T) {
	target := mathgl.Vec3{0, 0, 0}
	c := New(target, 5)
	c.Orbit(float32(math.Pi/2), 0)
	wantEye := mathgl.Vec3{5, 0, 0}
	if !vecNearEq(c.Eye(), wantEye) {
		t.Fatalf("eye = %v, want %v", c.Eye(), wantEye)
	}
	if d := c.Eye().Sub(target).Len(); !nearEq(d, 5) {
		t.Fatalf("distance = %v, want 5", d)
	}
}

func TestOrbitPitchClamped(t *testing.T) {
	c := New(mathgl.Vec3{}, 5)
	c.Orbit(0, 10) // way past vertical
	eye := c.Eye()
	// Even fully pitched, the eye never flips over the pole.
	if eye[1] < 0 || nearEq(eye[2], 0) && nearEq(eye[0], 0) && eye[1] >= 5 {
		t.Fatalf("pitch not clamped: eye = %v", eye)
	}
	maxY := float32(5 * math.Sin(89*math.Pi/180))
	if eye[1] > maxY+eps {
		t.Fatalf("eye y = %v beyond 89-degree clamp %v", eye[1], maxY)
	}
}

func TestZoom(t *testing.T) {
	c := New(mathgl.Vec3{}, 10)
	c.Zoom(1)
	closer := c.Eye().Len()
	if closer >= 10 {
		t.Fatalf("zoom in did not reduce distance: %v", closer)
	}
	c.Zoom(-1)
	if d := c.Eye().Len(); !nearEq(d, 10) {
		t.Fatalf("zoom out did not restore distance: %v", d)
	}
	// Clamp: zooming in forever never reaches the target.
	for i := 0; i < 1000; i++ {
		c.Zoom(1)
	}
	if d := c.Eye().Len(); d < 1e-4 {
		t.Fatalf("zoom clamp failed: distance %v", d)
	}
}

func TestPanMovesTarget(t *testing.T) {
	c := New(mathgl.Vec3{}, 10)
	before := c.Eye()
	c.Pan(100, 0)
	after := c.Eye()
	if vecNearEq(before, after) {
		t.Fatal("pan did not move the camera")
	}
	// Pan preserves viewing direction (eye and target shift together).
	if d := after.Sub(before); !nearEq(d[2], 0) {
		t.Fatalf("pan moved along the view axis: %v", d)
	}
}

func TestToggleModePreservesEye(t *testing.T) {
	c := New(mathgl.Vec3{1, 0, -2}, 7)
	c.Orbit(0.7, 0.3)
	eye := c.Eye()
	c.ToggleMode()
	if c.Mode() != Fly {
		t.Fatal("mode did not switch to Fly")
	}
	if !vecNearEq(c.Eye(), eye) {
		t.Fatalf("toggle to fly moved eye %v -> %v", eye, c.Eye())
	}
	c.ToggleMode()
	if c.Mode() != Orbit {
		t.Fatal("mode did not switch back to Orbit")
	}
	if !vecNearEq(c.Eye(), eye) {
		t.Fatalf("toggle to orbit moved eye %v -> %v", eye, c.Eye())
	}
}

func TestFlyMoveAdvancesAlongView(t *testing.T) {
	c := New(mathgl.Vec3{}, 10)
	c.ToggleMode()
	before := c.Eye()
	c.Move(1, 0, 0, 0.1)
	delta := c.Eye().Sub(before)
	// From the default pose the camera looks toward -z.
	wantLen := c.MoveSpeed() * 0.1
	if !nearEq(delta.Len(), wantLen) {
		t.Fatalf("moved %v, want %v", delta.Len(), wantLen)
	}
	dir := delta.Normalize()
	if !vecNearEq(dir, mathgl.Vec3{0, 0, -1}) {
		t.Fatalf("move direction = %v, want -z", dir)
	}
}

func TestFlyLookPitchClamped(t *testing.T) {
	c := New(mathgl.Vec3{}, 10)
	c.ToggleMode()
	c.Look(0, 100) // crank pitch far beyond vertical
	before := c.Eye()
	c.Move(1, 0, 0, 1)
	dir := c.Eye().Sub(before).Normalize()
	// Clamped at 89 degrees: still a small horizontal component, no flip.
	if math.Abs(float64(dir[1])) > math.Sin(89.001*math.Pi/180) {
		t.Fatalf("pitch not clamped: dir %v", dir)
	}
}

func TestReset(t *testing.T) {
	target := mathgl.Vec3{1, 2, 3}
	c := New(target, 10)
	orig := c.View()
	c.Orbit(1, 0.5)
	c.Zoom(3)
	c.Pan(50, 20)
	c.ToggleMode()
	c.Move(1, 1, 1, 2)
	c.Reset()
	if c.Mode() != Orbit {
		t.Fatal("reset did not restore orbit mode")
	}
	got := c.View()
	for i := range got {
		if !nearEq(got[i], orig[i]) {
			t.Fatalf("view[%d] = %v, want %v after reset", i, got[i], orig[i])
		}
	}
}
