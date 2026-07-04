package main

import (
	"flag"
	"log"
	"math"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"scop/internal/mathgl"
	"scop/internal/objparser"
	"scop/internal/renderer"
	"scop/internal/window"
	"scop/shaders"
)

func init() {
	// OpenGL and GLFW require all calls on the main OS thread.
	runtime.LockOSThread()
}

func main() {
	speed := flag.Float64("speed", 0.8, "rotation speed in radians per second")
	width := flag.Int("width", 1280, "window width")
	height := flag.Int("height", 720, "window height")
	uvFlag := flag.String("uv", "auto", "uv generation: auto|planar|cylindrical|spherical")
	flag.Parse()

	modelPath := "models/42.obj"
	if flag.NArg() > 0 {
		modelPath = flag.Arg(0)
	}

	uvMode, err := objparser.ParseUVMode(*uvFlag)
	if err != nil {
		log.Fatalf("scop: %v", err)
	}

	mesh, err := objparser.ParseOBJ(modelPath)
	if err != nil {
		log.Fatalf("scop: %v", err)
	}
	objparser.GenerateUVs(mesh, uvMode)
	log.Printf("%s: %d vertices, %d triangles, %d group(s)",
		modelPath, len(mesh.Vertices), len(mesh.Indices)/3, len(mesh.Groups))

	win, err := window.New(*width, *height, "scop")
	if err != nil {
		log.Fatalf("scop: %v", err)
	}
	defer win.Terminate()
	log.Printf("OpenGL %s", win.GLVersion())

	win.Handle().SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	shader, err := renderer.NewShader(shaders.ScopVert, shaders.ScopFrag)
	if err != nil {
		log.Fatalf("scop: %v", err)
	}

	glMesh := renderer.UploadMesh(mesh)
	center := glMesh.Center()
	dist := 2.5 * glMesh.MaxExtent()

	gl.Enable(gl.DEPTH_TEST)
	gl.ClearColor(0.12, 0.12, 0.14, 1)

	// All per-frame state lives outside the loop: zero steady-state allocs.
	var (
		angle      float32
		model      mathgl.Mat4
		view       mathgl.Mat4
		proj       mathgl.Mat4
		toCenter   = mathgl.Translate(center.Scale(-1))
		fromCenter = mathgl.Translate(center)
		lightDir   = mathgl.Vec3{0.4, 0.7, 0.6}.Normalize()
		lastTime   = glfw.GetTime()
	)
	eye := center.Add(mathgl.Vec3{0, 0, dist})
	view = mathgl.LookAt(eye, center, mathgl.Vec3{0, 1, 0})

	hasNormals := int32(0)
	if glMesh.HasNormals {
		hasNormals = 1
	}

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now
		angle += float32(*speed) * dt

		fbW, fbH := win.FramebufferSize()
		gl.Viewport(0, 0, int32(fbW), int32(fbH))
		aspect := float32(fbW) / float32(max(fbH, 1))
		proj = mathgl.Perspective(float32(math.Pi/4), aspect, 0.1, 100*glMesh.MaxExtent())

		// Spin around the model's own central vertical axis.
		model = fromCenter.Mul(mathgl.RotateY(angle)).Mul(toCenter)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		shader.Use()
		shader.SetMat4("model", &model)
		shader.SetMat4("view", &view)
		shader.SetMat4("proj", &proj)
		shader.SetInt("hasNormals", hasNormals)
		shader.SetVec3("lightDir", lightDir)

		glMesh.Bind()
		glMesh.DrawAll()

		win.SwapAndPoll()
	}
}
