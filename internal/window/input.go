package window

import "github.com/go-gl/glfw/v3.3/glfw"

// Input gathers GLFW events into per-frame state: accumulated mouse/scroll
// deltas, held keys, and edge-triggered presses. The render loop reads it
// and calls EndFrame once per frame; no allocations occur after Attach.
type Input struct {
	dx, dy     float64
	scroll     float64
	lastX      float64
	lastY      float64
	firstMouse bool

	leftDown   bool
	middleDown bool
	shiftDown  bool

	held    map[glfw.Key]bool
	pressed map[glfw.Key]bool
}

// Attach registers callbacks on the window and returns the input collector.
func Attach(w *glfw.Window) *Input {
	in := &Input{
		firstMouse: true,
		held:       make(map[glfw.Key]bool, 32),
		pressed:    make(map[glfw.Key]bool, 32),
	}

	w.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		in.shiftDown = mods&glfw.ModShift != 0
		switch action {
		case glfw.Press:
			in.held[key] = true
			in.pressed[key] = true
		case glfw.Release:
			in.held[key] = false
		}
	})

	w.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		if in.firstMouse {
			in.lastX, in.lastY = x, y
			in.firstMouse = false
			return
		}
		in.dx += x - in.lastX
		in.dy += y - in.lastY
		in.lastX, in.lastY = x, y
	})

	w.SetMouseButtonCallback(func(_ *glfw.Window, b glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		down := action == glfw.Press
		in.shiftDown = mods&glfw.ModShift != 0
		switch b {
		case glfw.MouseButtonLeft:
			in.leftDown = down
		case glfw.MouseButtonMiddle:
			in.middleDown = down
		}
	})

	w.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) {
		in.scroll += yoff
	})

	return in
}

// MouseDelta returns the cursor movement accumulated since the last EndFrame.
func (in *Input) MouseDelta() (float32, float32) {
	return float32(in.dx), float32(in.dy)
}

// Scroll returns the wheel movement accumulated since the last EndFrame.
func (in *Input) Scroll() float32 { return float32(in.scroll) }

// Down reports whether a key is currently held.
func (in *Input) Down(k glfw.Key) bool { return in.held[k] }

// Pressed reports a key press edge since the last EndFrame.
func (in *Input) Pressed(k glfw.Key) bool { return in.pressed[k] }

// Dragging reports plain left-button drag (orbit rotate).
func (in *Input) Dragging() bool { return in.leftDown && !in.shiftDown }

// Panning reports middle-button or shift+left drag (orbit pan).
func (in *Input) Panning() bool { return in.middleDown || (in.leftDown && in.shiftDown) }

// ResetMouse discards the next cursor delta (after cursor-mode switches, the
// jump to the recentered cursor must not turn the camera).
func (in *Input) ResetMouse() { in.firstMouse = true }

// Axis converts a pair of held keys into -1/0/+1.
func (in *Input) Axis(pos, neg glfw.Key) float32 {
	var v float32
	if in.held[pos] {
		v++
	}
	if in.held[neg] {
		v--
	}
	return v
}

// EndFrame clears per-frame deltas and press edges.
func (in *Input) EndFrame() {
	in.dx, in.dy = 0, 0
	in.scroll = 0
	clear(in.pressed)
}
