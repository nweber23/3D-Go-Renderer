package renderer

import (
	"github.com/go-gl/gl/v4.1-core/gl"

	"scop/internal/mathgl"
)

// initUniforms enumerates the program's active uniforms once at link time
// so per-frame updates are map lookups instead of GL name queries (which
// would allocate C strings every frame).
func (s *Shader) initUniforms() {
	s.locs = make(map[string]int32)
	var count int32
	gl.GetProgramiv(s.ID, gl.ACTIVE_UNIFORMS, &count)
	buf := make([]uint8, 256)
	for i := int32(0); i < count; i++ {
		var length, size int32
		var xtype uint32
		gl.GetActiveUniform(s.ID, uint32(i), int32(len(buf)-1), &length, &size, &xtype, &buf[0])
		name := string(buf[:length])
		s.locs[name] = gl.GetUniformLocation(s.ID, &buf[0])
	}
}

func (s *Shader) loc(name string) int32 {
	if l, ok := s.locs[name]; ok {
		return l
	}
	return -1 // glUniform* silently ignores -1
}

func (s *Shader) SetMat4(name string, m *mathgl.Mat4) {
	gl.UniformMatrix4fv(s.loc(name), 1, false, m.Ptr())
}

func (s *Shader) SetInt(name string, v int32) {
	gl.Uniform1i(s.loc(name), v)
}

func (s *Shader) SetFloat(name string, v float32) {
	gl.Uniform1f(s.loc(name), v)
}

func (s *Shader) SetVec3(name string, v mathgl.Vec3) {
	gl.Uniform3f(s.loc(name), v[0], v[1], v[2])
}
