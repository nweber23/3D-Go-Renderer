package mathgl

import "math"

// Mat4 is a 4x4 matrix in column-major order (OpenGL convention):
// element (row r, col c) lives at m[c*4+r].
type Mat4 [16]float32

func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Mul returns m * n (n is applied first when transforming vectors).
func (m Mat4) Mul(n Mat4) Mat4 {
	var r Mat4
	for c := 0; c < 4; c++ {
		for row := 0; row < 4; row++ {
			r[c*4+row] = m[row]*n[c*4] +
				m[4+row]*n[c*4+1] +
				m[8+row]*n[c*4+2] +
				m[12+row]*n[c*4+3]
		}
	}
	return r
}

func (m Mat4) MulVec4(v Vec4) Vec4 {
	return Vec4{
		m[0]*v[0] + m[4]*v[1] + m[8]*v[2] + m[12]*v[3],
		m[1]*v[0] + m[5]*v[1] + m[9]*v[2] + m[13]*v[3],
		m[2]*v[0] + m[6]*v[1] + m[10]*v[2] + m[14]*v[3],
		m[3]*v[0] + m[7]*v[1] + m[11]*v[2] + m[15]*v[3],
	}
}

func Translate(v Vec3) Mat4 {
	m := Identity()
	m[12] = v[0]
	m[13] = v[1]
	m[14] = v[2]
	return m
}

func ScaleUniform(s float32) Mat4 {
	return ScaleVec(Vec3{s, s, s})
}

func ScaleVec(v Vec3) Mat4 {
	var m Mat4
	m[0] = v[0]
	m[5] = v[1]
	m[10] = v[2]
	m[15] = 1
	return m
}

func RotateX(rad float32) Mat4 {
	s, c := sincos(rad)
	m := Identity()
	m[5] = c
	m[6] = s
	m[9] = -s
	m[10] = c
	return m
}

func RotateY(rad float32) Mat4 {
	s, c := sincos(rad)
	m := Identity()
	m[0] = c
	m[2] = -s
	m[8] = s
	m[10] = c
	return m
}

func RotateZ(rad float32) Mat4 {
	s, c := sincos(rad)
	m := Identity()
	m[0] = c
	m[1] = s
	m[4] = -s
	m[5] = c
	return m
}

func sincos(rad float32) (float32, float32) {
	s, c := math.Sincos(float64(rad))
	return float32(s), float32(c)
}

// Perspective builds a right-handed perspective projection mapping the
// frustum to clip space, looking down -z. fovy is in radians.
func Perspective(fovyRad, aspect, near, far float32) Mat4 {
	f := float32(1 / math.Tan(float64(fovyRad)/2))
	var m Mat4
	m[0] = f / aspect
	m[5] = f
	m[10] = (far + near) / (near - far)
	m[11] = -1
	m[14] = (2 * far * near) / (near - far)
	return m
}

// LookAt builds a view matrix for an eye looking at center with the given
// up direction (gluLookAt semantics).
func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)

	var m Mat4
	m[0] = s[0]
	m[4] = s[1]
	m[8] = s[2]
	m[1] = u[0]
	m[5] = u[1]
	m[9] = u[2]
	m[2] = -f[0]
	m[6] = -f[1]
	m[10] = -f[2]
	m[12] = -s.Dot(eye)
	m[13] = -u.Dot(eye)
	m[14] = f.Dot(eye)
	m[15] = 1
	return m
}

// Ptr returns a pointer to the first element for glUniformMatrix4fv.
func (m *Mat4) Ptr() *float32 { return &m[0] }
