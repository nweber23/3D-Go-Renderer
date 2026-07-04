package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"runtime"
	"strconv"

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
	wireframeStart := flag.Bool("wireframe-start", false, "start with the wireframe overlay enabled")
	vsync := flag.Bool("vsync", true, "synchronize buffer swaps with the display")
	debug := flag.Bool("debug", false, "log frame timings to stdout")
	flag.Usage = usage
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
	win.SetVSync(*vsync)

	shader, err := renderer.NewShader(shaders.ScopVert, shaders.ScopFrag)
	if err != nil {
		log.Fatalf("scop: %v", err)
	}
	overlay, err := renderer.NewOverlay()
	if err != nil {
		log.Fatalf("scop: %v", err)
	}

	glMesh := renderer.UploadMesh(mesh)
	center := glMesh.Center()
	maxExtent := glMesh.MaxExtent()
	cam := camera.New(center, 2.5*maxExtent)

	// Texture priority per group: -texture flag > mtl map_Kd > generated
	// default. Loaded once here; groups arrive pre-sorted by texture so the
	// draw loop changes bindings as rarely as possible.
	texCache := make(map[string]uint32)
	loadTex := func(path string) uint32 {
		if t, ok := texCache[path]; ok {
			return t
		}
		t, err := renderer.LoadTexture(path)
		if err != nil {
			log.Printf("scop: %v", err)
			t = 0
		}
		texCache[path] = t
		return t
	}
	fallbackTex := loadTex("models/default_texture.png")
	if *texturePath != "" {
		if t := loadTex(*texturePath); t != 0 {
			fallbackTex = t
		}
	}
	groupTex := make([]uint32, len(mesh.Groups))
	for i, g := range mesh.Groups {
		groupTex[i] = fallbackTex
		if *texturePath == "" && g.Material.MapKd != "" {
			if t := loadTex(g.Material.MapKd); t != 0 {
				groupTex[i] = t
			}
		}
	}
	texture := fallbackTex // any texture available enables the T toggle

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
		wireframe = *wireframeStart

		showFPS   = true
		fpsText   = make([]byte, 0, 32) // reused: no per-frame allocations
		fpsFrames int
		fpsAccum  float64
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
		if input.Pressed(glfw.KeyF) {
			wireframe = !wireframe
		}
		if input.Pressed(glfw.KeyF1) {
			showFPS = !showFPS
		}
		if input.Pressed(glfw.KeyF2) {
			win.SetVSync(!win.VSync())
		}
		if input.Pressed(glfw.KeyF11) {
			win.ToggleFullscreen()
		}
		if input.Pressed(glfw.KeyF12) {
			w, h := win.FramebufferSize()
			if name, err := renderer.Screenshot(w, h); err != nil {
				log.Printf("scop: %v", err)
			} else {
				log.Printf("saved %s", name)
			}
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

		// Aggregate frame timings; refresh the readout twice a second.
		fpsFrames++
		fpsAccum += float64(dt)
		if fpsAccum >= 0.5 {
			avgMs := fpsAccum / float64(fpsFrames) * 1000
			fps := int64(float64(fpsFrames)/fpsAccum + 0.5)
			fpsText = fpsText[:0]
			fpsText = append(fpsText, "FPS "...)
			fpsText = strconv.AppendInt(fpsText, fps, 10)
			fpsText = append(fpsText, " / "...)
			fpsText = strconv.AppendFloat(fpsText, avgMs, 'f', 2, 64)
			fpsText = append(fpsText, "ms"...)
			if *debug {
				log.Printf("frames=%d avg=%.3fms fps=%d", fpsFrames, avgMs, fps)
			}
			fpsFrames = 0
			fpsAccum = 0
		}

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
		shader.SetVec3("viewPos", cam.Eye())
		shader.SetFloat("mixFactor", smoothMix)
		shader.SetInt("tex", 0)
		gl.ActiveTexture(gl.TEXTURE0)

		glMesh.Bind()
		if wireframe {
			// Solid pass pushed back slightly so the line pass on top
			// cannot z-fight with it.
			gl.Enable(gl.POLYGON_OFFSET_FILL)
			gl.PolygonOffset(1, 1)
			drawGroups(shader, glMesh, groupTex)
			gl.Disable(gl.POLYGON_OFFSET_FILL)

			gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
			shader.SetInt("wireframePass", 1)
			glMesh.DrawAll()
			gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
		} else {
			drawGroups(shader, glMesh, groupTex)
		}

		if showFPS {
			overlay.Draw(fpsText, fbW, fbH)
		}

		win.SwapAndPoll()
	}
}

// drawGroups renders each material group with its uniforms and texture.
// Groups are sorted by texture at parse time, so rebinds are minimal.
func drawGroups(shader *renderer.Shader, m *renderer.GLMesh, groupTex []uint32) {
	shader.SetInt("wireframePass", 0)
	var bound uint32 = ^uint32(0)
	for i := range m.Groups {
		mat := &m.Groups[i].Material
		shader.SetVec3("matKd", mat.Kd)
		shader.SetVec3("matKs", mat.Ks)
		shader.SetFloat("matNs", mat.Ns)
		if groupTex[i] != bound {
			gl.BindTexture(gl.TEXTURE_2D, groupTex[i])
			bound = groupTex[i]
		}
		m.DrawGroup(i)
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(),
		`usage: scop [flags] [model.obj]

Renders a Wavefront .obj model (default models/42.obj).

Flags:
`)
	flag.PrintDefaults()
	fmt.Fprint(flag.CommandLine.Output(), `
Controls:
  drag / scroll        orbit camera / zoom (orbit mode)
  shift-drag / middle  pan (orbit mode)
  WASD + mouse         move / look (fly mode)  [Space/Ctrl: up/down]
  C                    toggle camera mode (orbit <-> fly)
  arrows, IJKL, U/O    translate object on X/Y/Z
  T                    toggle color <-> texture (crossfade)
  F                    toggle wireframe overlay
  R                    reset view and object transforms
  F1                   toggle FPS overlay
  F2                   toggle vsync
  F11                  toggle fullscreen
  F12                  screenshot (PNG)
  Esc                  quit
`)
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
