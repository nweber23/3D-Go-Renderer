// Package renderer owns all OpenGL state: shader programs, buffers and draws.
package renderer

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Shader is a linked GLSL program with cached uniform locations.
type Shader struct {
	ID   uint32
	locs map[string]int32
}

// NewShader compiles the given vertex and fragment sources and links them
// into a program. Errors carry the full GL info log.
func NewShader(vertSrc, fragSrc string) (*Shader, error) {
	vert, err := compile(vertSrc, gl.VERTEX_SHADER, "vertex")
	if err != nil {
		return nil, err
	}
	defer gl.DeleteShader(vert)

	frag, err := compile(fragSrc, gl.FRAGMENT_SHADER, "fragment")
	if err != nil {
		return nil, err
	}
	defer gl.DeleteShader(frag)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen)+1)
		gl.GetProgramInfoLog(prog, logLen, nil, gl.Str(log))
		gl.DeleteProgram(prog)
		return nil, fmt.Errorf("link program: %s", log)
	}
	s := &Shader{ID: prog}
	s.initUniforms()
	return s, nil
}

func compile(src string, kind uint32, name string) (uint32, error) {
	shader := gl.CreateShader(kind)
	csrc, free := gl.Strs(src + "\x00")
	gl.ShaderSource(shader, 1, csrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen)+1)
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(log))
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("compile %s shader: %s", name, log)
	}
	return shader, nil
}

// Use makes the program current.
func (s *Shader) Use() { gl.UseProgram(s.ID) }
