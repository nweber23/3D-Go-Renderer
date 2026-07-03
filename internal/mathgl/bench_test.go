package mathgl

import "testing"

var (
	sinkMat Mat4
	sinkVec Vec4
)

func BenchmarkMat4Mul(b *testing.B) {
	m := RotateY(0.5)
	n := Translate(Vec3{1, 2, 3})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkMat = m.Mul(n)
	}
}

func BenchmarkMat4MulVec4(b *testing.B) {
	m := RotateY(0.5).Mul(Translate(Vec3{1, 2, 3}))
	v := Vec4{1, 2, 3, 1}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkVec = m.MulVec4(v)
	}
}

func BenchmarkLookAt(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkMat = LookAt(Vec3{0, 1, 5}, Vec3{}, Vec3{0, 1, 0})
	}
}

func BenchmarkPerspective(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkMat = Perspective(1.0, 1.77, 0.1, 100)
	}
}
