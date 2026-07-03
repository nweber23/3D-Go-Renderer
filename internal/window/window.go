// Package window wraps GLFW window creation and the OpenGL context.
package window

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// Window owns the GLFW window and its OpenGL 4.1 core context.
type Window struct {
	handle *glfw.Window
}

// New initializes GLFW, creates a window with an OpenGL 4.1 core profile
// context and loads the OpenGL function pointers. Must be called from the
// main OS thread (runtime.LockOSThread in the caller's init).
func New(width, height int, title string) (*Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("glfw init: %w", err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True) // required on macOS
	glfw.WindowHint(glfw.Samples, 4)

	handle, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("create window: %w", err)
	}
	handle.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("gl init: %w", err)
	}
	return &Window{handle: handle}, nil
}

// Handle exposes the underlying GLFW window for callback registration.
func (w *Window) Handle() *glfw.Window { return w.handle }

// GLVersion reports the version string of the created context.
func (w *Window) GLVersion() string {
	return gl.GoStr(gl.GetString(gl.VERSION))
}

// FramebufferSize returns the current framebuffer size in pixels.
func (w *Window) FramebufferSize() (int, int) {
	return w.handle.GetFramebufferSize()
}

func (w *Window) ShouldClose() bool { return w.handle.ShouldClose() }

// SwapAndPoll presents the back buffer and processes pending events.
func (w *Window) SwapAndPoll() {
	w.handle.SwapBuffers()
	glfw.PollEvents()
}

// Terminate destroys the window and shuts down GLFW.
func (w *Window) Terminate() {
	w.handle.Destroy()
	glfw.Terminate()
}
