package mathgl

import (
	"math"
	"testing"
)

const eps = 1e-6

func nearEq(a, b float32) bool {
	return math.Abs(float64(a-b)) < eps
}

func vecNearEq(a, b Vec3) bool {
	return nearEq(a[0], b[0]) && nearEq(a[1], b[1]) && nearEq(a[2], b[2])
}

func TestVec3Add(t *testing.T) {
	got := Vec3{1, 2, 3}.Add(Vec3{4, 5, 6})
	want := Vec3{5, 7, 9}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestVec3Sub(t *testing.T) {
	got := Vec3{4, 5, 6}.Sub(Vec3{1, 2, 3})
	want := Vec3{3, 3, 3}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestVec3Scale(t *testing.T) {
	got := Vec3{1, -2, 3}.Scale(2)
	want := Vec3{2, -4, 6}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestVec3Dot(t *testing.T) {
	if got := (Vec3{1, 2, 3}).Dot(Vec3{4, 5, 6}); got != 32 {
		t.Fatalf("got %v want 32", got)
	}
	if got := (Vec3{1, 0, 0}).Dot(Vec3{0, 1, 0}); got != 0 {
		t.Fatalf("orthogonal dot: got %v want 0", got)
	}
}

func TestVec3Cross(t *testing.T) {
	got := Vec3{1, 0, 0}.Cross(Vec3{0, 1, 0})
	want := Vec3{0, 0, 1}
	if got != want {
		t.Fatalf("x cross y: got %v want %v", got, want)
	}
	got = Vec3{2, 3, 4}.Cross(Vec3{5, 6, 7})
	want = Vec3{-3, 6, -3} // (3*7-4*6, 4*5-2*7, 2*6-3*5)
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestVec3Len(t *testing.T) {
	if got := (Vec3{3, 4, 0}).Len(); !nearEq(got, 5) {
		t.Fatalf("got %v want 5", got)
	}
	if got := (Vec3{}).Len(); got != 0 {
		t.Fatalf("zero len: got %v want 0", got)
	}
}

func TestVec3Normalize(t *testing.T) {
	got := Vec3{3, 4, 0}.Normalize()
	want := Vec3{0.6, 0.8, 0}
	if !vecNearEq(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if !nearEq(got.Len(), 1) {
		t.Fatalf("normalized len %v want 1", got.Len())
	}
}

func TestVec3NormalizeZero(t *testing.T) {
	if got := (Vec3{}).Normalize(); got != (Vec3{}) {
		t.Fatalf("got %v want zero vec", got)
	}
}

func TestAccessors(t *testing.T) {
	v3 := Vec3{1, 2, 3}
	if v3.X() != 1 || v3.Y() != 2 || v3.Z() != 3 {
		t.Fatalf("Vec3 accessors wrong: %v", v3)
	}
	v2 := Vec2{4, 5}
	if v2.X() != 4 || v2.Y() != 5 {
		t.Fatalf("Vec2 accessors wrong: %v", v2)
	}
	v4 := Vec4{6, 7, 8, 9}
	if v4.X() != 6 || v4.Y() != 7 || v4.Z() != 8 || v4.W() != 9 {
		t.Fatalf("Vec4 accessors wrong: %v", v4)
	}
}
