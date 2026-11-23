package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
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
	recordDir := flag.String("record", "", "dump a scripted demo as PNG frames into this dir, then exit")
	recordFrames := flag.Int("frames", 120, "frame count for -record (fixed 30 fps timestep)")
	up := flag.String("up", "y", "model's up axis: x|y|z (rotates Z-up scans upright)")
	smoothStart := flag.Bool("smooth", false, "start with smooth (vertex) normals, generating them if the model has none")
	shadeStart := flag.Int("shade", 0, "start shading mode: 0 faces, 1 normals, 2 uv checker")
	debug := flag.Bool("debug", false, "log frame timings to stdout")
	flag.Usage = usage
	flag.Parse()

	models := flag.Args() // Tab cycles through these; dropped files are appended
	if len(models) == 0 {
		models = []string{"models/42.obj"}
	}
	modelPath := models[0]

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
	upRot := mathgl.Identity()
	switch *up {
	case "y":
	case "z":
		upRot = mathgl.RotateX(-math.Pi / 2)
	case "x":
		upRot = mathgl.RotateZ(math.Pi / 2)
	default:
		log.Fatalf("scop: -up must be x, y or z, got %q", *up)
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
		toCenter   mathgl.Mat4
		fromCenter mathgl.Mat4
		lightDir   = mathgl.Vec3{0.4, 0.7, 0.6}.Normalize()
		lastTime   = glfw.GetTime()
		moveSpeed  float32 // object translation units per second
		maxExtent  float32
		glMesh     *renderer.GLMesh
		curMesh    *objparser.Mesh
		groupTex   []uint32
		cam        *camera.Camera
		smooth     bool // use vertex normals (true) or derivative flat normals
		modelIdx   int

		mixFactor float32 // eased blend position, 0..1
		mixTarget float32 // 0 = flat color, 1 = textured
		wireframe = *wireframeStart
		paused    bool
		shadeMode = int32(*shadeStart % 3)
		bgIdx     int
		frameNo   int

		showFPS   = true
		fpsText   = make([]byte, 0, 32) // reused: no per-frame allocations
		fpsFrames int
		fpsAccum  float64

		pv         mathgl.Mat4
		planes     [6]mathgl.Vec4
		culled     int
		memStats   runtime.MemStats
		lastAllocs uint64
	)
	backgrounds := [...][3]float32{{0.12, 0.12, 0.14}, {0.02, 0.02, 0.03}, {0.9, 0.9, 0.92}, {0.1, 0.2, 0.35}}
	if *recordDir != "" {
		if err := os.MkdirAll(*recordDir, 0o755); err != nil {
			log.Fatalf("scop: %v", err)
		}
	}
	const fadeDur = 0.4 // seconds for the color <-> texture crossfade

	var enableSmooth func()
	// load uploads a parsed mesh and resets everything model-specific.
	load := func(m *objparser.Mesh) {
		if glMesh != nil {
			glMesh.Delete()
		}
		curMesh = m
		glMesh = renderer.UploadMesh(m)
		center := glMesh.Center()
		maxExtent = glMesh.MaxExtent()
		toCenter = mathgl.Translate(center.Scale(-1))
		fromCenter = mathgl.Translate(center)
		moveSpeed = maxExtent * 0.5
		cam = camera.New(center, 1.4*maxExtent)
		offset, angle = mathgl.Vec3{}, 0
		smooth = m.HasNormals
		groupTex = make([]uint32, len(m.Groups))
		for i, g := range m.Groups {
			groupTex[i] = fallbackTex
			if *texturePath == "" && g.Material.MapKd != "" {
				if t := loadTex(g.Material.MapKd); t != 0 {
					groupTex[i] = t
				}
			}
		}
		win.Handle().SetTitle("scop - " + filepath.Base(modelPath))
	}
	// switchTo parses a model file and swaps it in; failures keep the old one.
	switchTo := func(path string) {
		m, err := objparser.ParseOBJ(path)
		if err != nil {
			log.Printf("scop: %v", err)
			return
		}
		objparser.GenerateUVs(m, uvMode)
		log.Printf("%s: %d vertices, %d triangles, %d group(s)",
			path, len(m.Vertices), len(m.Indices)/3, len(m.Groups))
		modelPath = path
		load(m)
		lastTime = glfw.GetTime() // do not count the load as a frame
	}
	// enableSmooth builds vertex normals for a vn-less mesh and re-uploads it.
	enableSmooth = func() {
		objparser.SmoothNormals(curMesh)
		glMesh.Delete()
		glMesh = renderer.UploadMesh(curMesh)
		smooth = true
	}
	load(mesh)
	if *smoothStart {
		enableSmooth()
	}
	var dropped []string
	win.Handle().SetDropCallback(func(_ *glfw.Window, names []string) {
		dropped = append(dropped, names...)
	})

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now
		if *recordDir != "" {
			dt = 1.0 / 30 // deterministic timestep
			// Scripted demo: fade to texture at 1/3, back to color at 2/3.
			switch frameNo {
			case *recordFrames / 3, 2 * *recordFrames / 3:
				mixTarget = 1 - mixTarget
			}
		}
		if !paused {
			angle += float32(*speed) * dt
		}

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
		if input.Pressed(glfw.KeyTab) && len(models) > 1 {
			modelIdx = (modelIdx + 1) % len(models)
			switchTo(models[modelIdx])
		}
		for _, d := range dropped {
			models = append(models, d)
			modelIdx = len(models) - 1
			switchTo(d)
		}
		dropped = dropped[:0]
		if input.Pressed(glfw.KeyM) {
			if !curMesh.HasNormals {
				enableSmooth()
			} else {
				smooth = !smooth
			}
		}
		if input.Pressed(glfw.KeyP) {
			paused = !paused
		}
		if input.Pressed(glfw.KeyN) {
			shadeMode = (shadeMode + 1) % 3
		}
		if input.Pressed(glfw.KeyB) {
			bgIdx = (bgIdx + 1) % len(backgrounds)
			bg := backgrounds[bgIdx]
			gl.ClearColor(bg[0], bg[1], bg[2], 1)
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
				runtime.ReadMemStats(&memStats)
				allocsPerFrame := float64(memStats.Mallocs-lastAllocs) / float64(fpsFrames)
				lastAllocs = memStats.Mallocs
				log.Printf("frames=%d avg=%.3fms fps=%d culled=%d allocs/frame=%.2f",
					fpsFrames, avgMs, fps, culled, allocsPerFrame)
			}
			fpsFrames = 0
			fpsAccum = 0
			culled = 0
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
			Mul(fromCenter).Mul(mathgl.RotateY(angle)).Mul(upRot).Mul(toCenter)

		// Frustum-cull the model's world-space bounding box.
		pv = proj.Mul(view)
		renderer.ExtractFrustum(&pv, &planes)
		wmin, wmax := worldAABB(&model, glMesh.BBoxMin, glMesh.BBoxMax)
		visible := renderer.AABBVisible(&planes, wmin, wmax)
		if !visible {
			culled++
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		shader.Use()
		shader.SetMat4("model", &model)
		shader.SetMat4("view", &view)
		shader.SetMat4("proj", &proj)
		shader.SetInt("hasNormals", b2i(smooth))
		shader.SetVec3("lightDir", lightDir)
		shader.SetVec3("viewPos", cam.Eye())
		shader.SetFloat("mixFactor", smoothMix)
		shader.SetInt("tex", 0)
		shader.SetInt("shadeMode", shadeMode)
		gl.ActiveTexture(gl.TEXTURE0)

		glMesh.Bind()
		if !visible {
			// Nothing to draw; the box is entirely outside the frustum.
		} else if wireframe {
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

		if showFPS && *recordDir == "" {
			overlay.Draw(fpsText, fbW, fbH)
		}

		if *recordDir != "" {
			if _, err := renderer.ScreenshotTo(fbW, fbH, filepath.Join(*recordDir, fmt.Sprintf("f%04d.png", frameNo))); err != nil {
				log.Fatalf("scop: %v", err)
			}
			if frameNo++; frameNo >= *recordFrames {
				return
			}
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

// worldAABB transforms a local-space bounding box through the model matrix
// and returns the world-space box enclosing all eight corners.
func worldAABB(model *mathgl.Mat4, bmin, bmax mathgl.Vec3) (mathgl.Vec3, mathgl.Vec3) {
	var wmin, wmax mathgl.Vec3
	for c := 0; c < 8; c++ {
		corner := mathgl.Vec4{bmin[0], bmin[1], bmin[2], 1}
		if c&1 != 0 {
			corner[0] = bmax[0]
		}
		if c&2 != 0 {
			corner[1] = bmax[1]
		}
		if c&4 != 0 {
			corner[2] = bmax[2]
		}
		w := model.MulVec4(corner)
		if c == 0 {
			wmin = mathgl.Vec3{w[0], w[1], w[2]}
			wmax = wmin
			continue
		}
		for a := 0; a < 3; a++ {
			if w[a] < wmin[a] {
				wmin[a] = w[a]
			}
			if w[a] > wmax[a] {
				wmax[a] = w[a]
			}
		}
	}
	return wmin, wmax
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
  N                    cycle shading: faces / normals / uv checker
  M                    toggle smooth / flat normals
  Tab                  next model (several can be given; drop files onto the window)
  P                    pause / resume rotation
  B                    cycle background color
  R                    reset view and object transforms
  F1                   toggle FPS overlay
  F2                   toggle vsync
  F11                  toggle fullscreen
  F12                  screenshot (PNG)
  Esc                  quit
`)
}

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
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
