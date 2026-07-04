package renderer

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	overlayMaxChars = 64 // capacity of the preallocated text quad buffer
	overlayScale    = 2  // pixels per font pixel
)

const overlayVert = `#version 410 core
layout(location = 0) in vec2 aPos; // pixel coordinates, origin top-left
layout(location = 1) in vec2 aUV;
uniform vec2 fbSize;
out vec2 vUV;
void main() {
    vec2 ndc = vec2(2.0*aPos.x/fbSize.x - 1.0, 1.0 - 2.0*aPos.y/fbSize.y);
    vUV = aUV;
    gl_Position = vec4(ndc, 0.0, 1.0);
}`

const overlayFrag = `#version 410 core
in vec2 vUV;
uniform sampler2D font;
out vec4 fragColor;
void main() {
    float a = texture(font, vUV).r;
    fragColor = vec4(1.0, 1.0, 1.0, a);
}`

// Overlay draws short text lines (the FPS counter) as textured quads from
// a hand-built font atlas. The vertex buffer is allocated once; Draw
// performs no heap allocations.
type Overlay struct {
	shader *Shader
	atlas  uint32
	vao    uint32
	vbo    uint32
	verts  []float32 // reused scratch: 6 verts * 4 floats per char
}

// NewOverlay compiles the overlay shader and uploads the font atlas.
func NewOverlay() (*Overlay, error) {
	shader, err := NewShader(overlayVert, overlayFrag)
	if err != nil {
		return nil, fmt.Errorf("overlay: %w", err)
	}
	o := &Overlay{
		shader: shader,
		verts:  make([]float32, 0, overlayMaxChars*6*4),
	}

	pix := buildFontAtlas()
	atlasW := int32(8 * len(glyphOrder))
	gl.GenTextures(1, &o.atlas)
	gl.BindTexture(gl.TEXTURE_2D, o.atlas)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R8, atlasW, 8, 0,
		gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	gl.BindTexture(gl.TEXTURE_2D, 0)

	gl.GenVertexArrays(1, &o.vao)
	gl.GenBuffers(1, &o.vbo)
	gl.BindVertexArray(o.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, o.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, overlayMaxChars*6*4*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)
	gl.BindVertexArray(0)
	return o, nil
}

// Draw renders text anchored at the top-left corner. Call with depth test
// enabled; it disables and restores GL state itself.
func (o *Overlay) Draw(text []byte, fbW, fbH int) {
	if len(text) == 0 {
		return
	}
	if len(text) > overlayMaxChars {
		text = text[:overlayMaxChars]
	}

	o.verts = o.verts[:0]
	const cell = 8 * overlayScale
	x := float32(12)
	y := float32(12)
	du := 1.0 / float32(len(glyphOrder))
	for _, c := range text {
		u0 := float32(glyphColumn(c)) * du
		u1 := u0 + du
		x0, y0, x1, y1 := x, y, x+cell, y+cell
		o.verts = append(o.verts,
			x0, y0, u0, 0,
			x0, y1, u0, 1,
			x1, y1, u1, 1,
			x0, y0, u0, 0,
			x1, y1, u1, 1,
			x1, y0, u1, 0,
		)
		x += cell
	}

	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	o.shader.Use()
	gl.Uniform2f(o.shader.loc("fbSize"), float32(fbW), float32(fbH))
	gl.Uniform1i(o.shader.loc("font"), 0)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, o.atlas)

	gl.BindVertexArray(o.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, o.vbo)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(o.verts)*4, gl.Ptr(o.verts))
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(o.verts)/4))
	gl.BindVertexArray(0)

	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
}
