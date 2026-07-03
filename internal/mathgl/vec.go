// Package mathgl is a from-scratch linear algebra library for the renderer:
// float32 vectors and column-major 4x4 matrices matching OpenGL conventions.
// All types are plain arrays; every operation is allocation-free.
package mathgl

import "math"

type Vec2 [2]float32

type Vec3 [3]float32

type Vec4 [4]float32

func (a Vec2) X() float32 { return a[0] }
func (a Vec2) Y() float32 { return a[1] }

func (a Vec3) X() float32 { return a[0] }
func (a Vec3) Y() float32 { return a[1] }
func (a Vec3) Z() float32 { return a[2] }

func (a Vec4) X() float32 { return a[0] }
func (a Vec4) Y() float32 { return a[1] }
func (a Vec4) Z() float32 { return a[2] }
func (a Vec4) W() float32 { return a[3] }

func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func (a Vec3) Scale(s float32) Vec3 {
	return Vec3{a[0] * s, a[1] * s, a[2] * s}
}

func (a Vec3) Dot(b Vec3) float32 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func (a Vec3) Len() float32 {
	return float32(math.Sqrt(float64(a.Dot(a))))
}

// Normalize returns the unit vector, or the zero vector for zero input.
func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}
