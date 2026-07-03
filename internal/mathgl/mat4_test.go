package mathgl

import (
	"math"
	"testing"
)

func vec4NearEq(a, b Vec4) bool {
	return nearEq(a[0], b[0]) && nearEq(a[1], b[1]) &&
		nearEq(a[2], b[2]) && nearEq(a[3], b[3])
}

func TestIdentity(t *testing.T) {
	v := Vec4{1, 2, 3, 4}
	if got := Identity().MulVec4(v); got != v {
		t.Fatalf("got %v want %v", got, v)
	}
}

func TestIdentityMul(t *testing.T) {
	m := Translate(Vec3{1, 2, 3})
	if got := Identity().Mul(m); got != m {
		t.Fatalf("I*M != M: %v", got)
	}
	if got := m.Mul(Identity()); got != m {
		t.Fatalf("M*I != M: %v", got)
	}
}

func TestTranslate(t *testing.T) {
	got := Translate(Vec3{1, 2, 3}).MulVec4(Vec4{0, 0, 0, 1})
	want := Vec4{1, 2, 3, 1}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	// Direction vectors (w=0) are unaffected by translation.
	got = Translate(Vec3{1, 2, 3}).MulVec4(Vec4{1, 0, 0, 0})
	want = Vec4{1, 0, 0, 0}
	if got != want {
		t.Fatalf("direction: got %v want %v", got, want)
	}
}

func TestScale(t *testing.T) {
	got := ScaleVec(Vec3{2, 3, 4}).MulVec4(Vec4{1, 1, 1, 1})
	want := Vec4{2, 3, 4, 1}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	got = ScaleUniform(2).MulVec4(Vec4{1, 2, 3, 1})
	want = Vec4{2, 4, 6, 1}
	if got != want {
		t.Fatalf("uniform: got %v want %v", got, want)
	}
}

func TestRotations(t *testing.T) {
	halfPi := float32(math.Pi / 2)
	cases := []struct {
		name string
		m    Mat4
		in   Vec4
		want Vec4
	}{
		{"Y(90) x->-z", RotateY(halfPi), Vec4{1, 0, 0, 0}, Vec4{0, 0, -1, 0}},
		{"Y(90) z->x", RotateY(halfPi), Vec4{0, 0, 1, 0}, Vec4{1, 0, 0, 0}},
		{"X(90) y->z", RotateX(halfPi), Vec4{0, 1, 0, 0}, Vec4{0, 0, 1, 0}},
		{"Z(90) x->y", RotateZ(halfPi), Vec4{1, 0, 0, 0}, Vec4{0, 1, 0, 0}},
	}
	for _, c := range cases {
		if got := c.m.MulVec4(c.in); !vec4NearEq(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestMulComposition(t *testing.T) {
	// Translate then rotate: M = R * T applies T first.
	m := RotateY(float32(math.Pi / 2)).Mul(Translate(Vec3{1, 0, 0}))
	got := m.MulVec4(Vec4{0, 0, 0, 1})
	want := Vec4{0, 0, -1, 1} // origin -> (1,0,0) -> rotated to (0,0,-1)
	if !vec4NearEq(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPerspective(t *testing.T) {
	m := Perspective(float32(math.Pi/2), 1, 1, 10)
	checks := map[int]float32{
		0:  1,
		5:  1,
		10: -11.0 / 9.0,
		11: -1,
		14: -20.0 / 9.0,
		15: 0,
	}
	for idx, want := range checks {
		if !nearEq(m[idx], want) {
			t.Errorf("m[%d] = %v want %v", idx, m[idx], want)
		}
	}
	// A point farther away projects to smaller screen coordinates.
	near := m.MulVec4(Vec4{1, 0, -2, 1})
	far := m.MulVec4(Vec4{1, 0, -8, 1})
	if near[0]/near[3] <= far[0]/far[3] {
		t.Fatalf("perspective does not shrink with distance: near %v far %v", near, far)
	}
}

func TestLookAt(t *testing.T) {
	view := LookAt(Vec3{0, 0, 5}, Vec3{0, 0, 0}, Vec3{0, 1, 0})
	// The eye maps to the view-space origin.
	if got := view.MulVec4(Vec4{0, 0, 5, 1}); !vec4NearEq(got, Vec4{0, 0, 0, 1}) {
		t.Fatalf("eye: got %v", got)
	}
	// The target sits on the negative z axis at distance 5.
	if got := view.MulVec4(Vec4{0, 0, 0, 1}); !vec4NearEq(got, Vec4{0, 0, -5, 1}) {
		t.Fatalf("target: got %v", got)
	}
	// World +x stays +x for this eye/up combination.
	if got := view.MulVec4(Vec4{1, 0, 5, 1}); !vec4NearEq(got, Vec4{1, 0, 0, 1}) {
		t.Fatalf("right: got %v", got)
	}
}

func TestPtr(t *testing.T) {
	m := Identity()
	if m.Ptr() != &m[0] {
		t.Fatal("Ptr must point at first element")
	}
}
