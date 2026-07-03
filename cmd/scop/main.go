package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"

	"scop/internal/renderer"
	"scop/internal/window"
	"scop/shaders"
)

func init() {
	// OpenGL and GLFW require all calls on the main OS thread.
	runtime.LockOSThread()
}

func main() {
	win, err := window.New(1280, 720, "scop")
	if err != nil {
		log.Fatalf("scop: %v", err)
	}
	defer win.Terminate()
	log.Printf("OpenGL %s", win.GLVersion())

	shader, err := renderer.NewShader(shaders.BasicVert, shaders.BasicFrag)
	if err != nil {
		log.Fatalf("scop: %v", err)
	}

	// Hello triangle: position (3) + color (3), interleaved.
	verts := []float32{
		-0.6, -0.5, 0, 1, 0.3, 0.2,
		0.6, -0.5, 0, 0.2, 1, 0.3,
		0.0, 0.7, 0, 0.3, 0.2, 1,
	}
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 6*4, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, 6*4, 3*4)

	gl.ClearColor(0.12, 0.12, 0.14, 1)
	for !win.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		shader.Use()
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		win.SwapAndPoll()
	}
}
