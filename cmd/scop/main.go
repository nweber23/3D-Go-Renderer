package main

import (
	"flag"
	"log"
	"math"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"scop/internal/camera"
	"scop/internal/mathgl"
	"scop/internal/objparser"
	"scop/internal/renderer"
	"scop/internal/window"
	"scop/shaders"
)

const (
	orbitSensitivity = 0.005 // radians per pixel dragged
	lookSensitivity  = 0.002 // radians per pixel in fly mode
)

func init() {
	// OpenGL and GLFW require all calls on the main OS thread.
	runtime.LockOSThread()
}

func main() {
	speed := flag.Float64("speed", 0.8, "rotation speed in radians per second")
	texturePath := flag.String("texture", "", "texture image (png/jpeg); overrides mtl map_Kd")
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

	input := window.Attach(win.Handle())

	shader, err := renderer.NewShader(shaders.ScopVert, shaders.ScopFrag)
	if err != nil {
		log.Fatalf("scop: %v", err)
	}

	glMesh := renderer.UploadMesh(mesh)
	center := glMesh.Center()
	maxExtent := glMesh.MaxExtent()
	cam := camera.New(center, 2.5*maxExtent)

	// Texture priority: -texture flag > first mtl map_Kd > generated default.
	texFile := *texturePath
	if texFile == "" {
		for _, g := range mesh.Groups {
			if g.Material.MapKd != "" {
				texFile = g.Material.MapKd
				break
			}
		}
	}
	if texFile == "" {
		texFile = "models/default_texture.png"
	}
	texture, err := renderer.LoadTexture(texFile)
	if err != nil {
		log.Printf("scop: %v (texture mode disabled)", err)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.ClearColor(0.12, 0.12, 0.14, 1)

	// All per-frame state lives outside the loop: zero steady-state allocs.
	var (
		angle      float32
		offset     mathgl.Vec3 // object translation from IJKL/arrows/UO
		model      mathgl.Mat4
		view       mathgl.Mat4
		proj       mathgl.Mat4
		toCenter   = mathgl.Translate(center.Scale(-1))
		fromCenter = mathgl.Translate(center)
		lightDir   = mathgl.Vec3{0.4, 0.7, 0.6}.Normalize()
		lastTime   = glfw.GetTime()
		moveSpeed  = maxExtent * 0.5 // object translation units per second

		mixFactor float32 // eased blend position, 0..1
		mixTarget float32 // 0 = flat color, 1 = textured
	)
	const fadeDur = 0.4 // seconds for the color <-> texture crossfade

	hasNormals := int32(0)
	if glMesh.HasNormals {
		hasNormals = 1
	}

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now
		angle += float32(*speed) * dt

		// --- input -----------------------------------------------------
		if input.Pressed(glfw.KeyEscape) {
			win.Handle().SetShouldClose(true)
		}
		if input.Pressed(glfw.KeyC) {
			cam.ToggleMode()
			if cam.Mode() == camera.Fly {
				win.Handle().SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
			} else {
				win.Handle().SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			}
			input.ResetMouse()
		}
		if input.Pressed(glfw.KeyT) && texture != 0 {
			mixTarget = 1 - mixTarget
		}
		if input.Pressed(glfw.KeyR) {
			cam.Reset()
			offset = mathgl.Vec3{}
			angle = 0
			win.Handle().SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			input.ResetMouse()
		}

		dx, dy := input.MouseDelta()
		if cam.Mode() == camera.Fly {
			cam.Look(-dx*lookSensitivity, -dy*lookSensitivity)
			cam.Move(
				input.Axis(glfw.KeyW, glfw.KeyS),
				input.Axis(glfw.KeyD, glfw.KeyA),
				input.Axis(glfw.KeySpace, glfw.KeyLeftControl),
				dt,
			)
		} else {
			switch {
			case input.Panning():
				cam.Pan(dx, dy)
			case input.Dragging():
				cam.Orbit(-dx*orbitSensitivity, -dy*orbitSensitivity)
			}
			if s := input.Scroll(); s != 0 {
				cam.Zoom(s)
			}
		}

		// Object translation on all three axes, both directions.
		step := moveSpeed * dt
		offset[0] += (input.Axis(glfw.KeyRight, glfw.KeyLeft) + input.Axis(glfw.KeyL, glfw.KeyJ)) * step
		offset[1] += (input.Axis(glfw.KeyUp, glfw.KeyDown) + input.Axis(glfw.KeyI, glfw.KeyK)) * step
		offset[2] += input.Axis(glfw.KeyU, glfw.KeyO) * step

		input.EndFrame()

		// Ease the crossfade toward its target; smoothstep the shader value.
		fadeStep := dt / fadeDur
		if mixFactor < mixTarget {
			mixFactor = minf(mixFactor+fadeStep, mixTarget)
		} else if mixFactor > mixTarget {
			mixFactor = maxf(mixFactor-fadeStep, mixTarget)
		}
		smoothMix := mixFactor * mixFactor * (3 - 2*mixFactor)

		// --- render ----------------------------------------------------
		fbW, fbH := win.FramebufferSize()
		gl.Viewport(0, 0, int32(fbW), int32(fbH))
		aspect := float32(fbW) / float32(max(fbH, 1))
		proj = mathgl.Perspective(float32(math.Pi/4), aspect, 0.01*maxExtent, 100*maxExtent)
		view = cam.View()

		// Spin around the model's own central vertical axis, then apply
		// the user translation.
		model = mathgl.Translate(offset).
			Mul(fromCenter).Mul(mathgl.RotateY(angle)).Mul(toCenter)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		shader.Use()
		shader.SetMat4("model", &model)
		shader.SetMat4("view", &view)
		shader.SetMat4("proj", &proj)
		shader.SetInt("hasNormals", hasNormals)
		shader.SetVec3("lightDir", lightDir)
		shader.SetFloat("mixFactor", smoothMix)
		shader.SetInt("tex", 0)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture)

		glMesh.Bind()
		glMesh.DrawAll()

		win.SwapAndPoll()
	}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
