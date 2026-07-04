// Package camera implements the two view modes: an orbit camera circling a
// target point (drag rotate, pan, scroll zoom) and a free-fly camera
// (WASD + mouse-look). All motion is delta-time based; the package holds no
// GL state and is fully unit-testable.
package camera

import (
	"math"

	"scop/internal/mathgl"
)

// Mode selects how input drives the camera.
type Mode int

const (
	Orbit Mode = iota
	Fly
)

const (
	maxPitch    = 89 * math.Pi / 180
	minDistFrac = 0.001 // zoom clamp as a fraction of the start distance
)

// Camera holds both orbit and fly state so toggling between modes is
// seamless: the eye position and look direction carry over.
type Camera struct {
	defTarget mathgl.Vec3
	defDist   float32

	mode Mode

	// Orbit state: eye = target + dist * direction(yaw, pitch).
	target     mathgl.Vec3
	yaw, pitch float32
	dist       float32

	// Fly state.
	pos              mathgl.Vec3
	flyYaw, flyPitch float32
}

// New creates an orbit camera looking at target from distance dist along +z.
func New(target mathgl.Vec3, dist float32) *Camera {
	c := &Camera{defTarget: target, defDist: dist}
	c.Reset()
	return c
}

// Reset restores the construction pose and orbit mode.
func (c *Camera) Reset() {
	c.mode = Orbit
	c.target = c.defTarget
	c.dist = c.defDist
	c.yaw, c.pitch = 0, 0
	c.pos = mathgl.Vec3{}
	c.flyYaw, c.flyPitch = 0, 0
}

func (c *Camera) Mode() Mode { return c.mode }

// orbitDir is the unit vector from target toward the eye.
func orbitDir(yaw, pitch float32) mathgl.Vec3 {
	sy, cy := sincos(yaw)
	sp, cp := sincos(pitch)
	return mathgl.Vec3{cp * sy, sp, cp * cy}
}

// flyForward is the unit look direction for the fly pose.
func flyForward(yaw, pitch float32) mathgl.Vec3 {
	sy, cy := sincos(yaw)
	sp, cp := sincos(pitch)
	return mathgl.Vec3{cp * sy, sp, cp * cy}
}

func sincos(rad float32) (float32, float32) {
	s, cc := math.Sincos(float64(rad))
	return float32(s), float32(cc)
}

// Eye returns the camera position in world space.
func (c *Camera) Eye() mathgl.Vec3 {
	if c.mode == Fly {
		return c.pos
	}
	return c.target.Add(orbitDir(c.yaw, c.pitch).Scale(c.dist))
}

// View returns the view matrix for the current pose.
func (c *Camera) View() mathgl.Mat4 {
	up := mathgl.Vec3{0, 1, 0}
	if c.mode == Fly {
		return mathgl.LookAt(c.pos, c.pos.Add(flyForward(c.flyYaw, c.flyPitch)), up)
	}
	return mathgl.LookAt(c.Eye(), c.target, up)
}

// ToggleMode switches orbit <-> fly, carrying the eye position and look
// direction over so the view does not jump.
func (c *Camera) ToggleMode() {
	if c.mode == Orbit {
		// The fly pose looks from the orbit eye back at the target.
		c.pos = c.Eye()
		c.flyYaw = c.yaw + math.Pi
		c.flyPitch = -c.pitch
		c.mode = Fly
		return
	}
	// Orbit around the point the camera is looking at, at the current
	// distance from the old target (keeps framing familiar).
	d := c.pos.Sub(c.target)
	c.dist = d.Len()
	if c.dist < minDistFrac*c.defDist {
		c.dist = minDistFrac * c.defDist
	}
	dir := d.Normalize()
	c.pitch = float32(math.Asin(float64(clamp(dir[1], -1, 1))))
	c.yaw = float32(math.Atan2(float64(dir[0]), float64(dir[2])))
	c.mode = Orbit
}

// Orbit rotates the eye around the target (radians).
func (c *Camera) Orbit(dx, dy float32) {
	c.yaw += dx
	c.pitch = clamp(c.pitch+dy, -maxPitch, maxPitch)
}

// Pan shifts the target (and with it the eye) in the view plane. Deltas are
// in pixels; movement scales with distance so panning feels constant.
func (c *Camera) Pan(dx, dy float32) {
	f := c.target.Sub(c.Eye()).Normalize()
	right := f.Cross(mathgl.Vec3{0, 1, 0}).Normalize()
	up := right.Cross(f)
	s := c.dist * 0.002
	c.target = c.target.Add(right.Scale(-dx * s)).Add(up.Scale(dy * s))
}

// Zoom moves the eye toward/away from the target multiplicatively; scroll
// is in wheel ticks. The distance never collapses to zero.
func (c *Camera) Zoom(scroll float32) {
	c.dist *= float32(math.Pow(0.9, float64(scroll)))
	min := minDistFrac * c.defDist
	if c.dist < min {
		c.dist = min
	}
}

// Look turns the fly camera (mouse-look, radians); pitch clamps at ±89°.
func (c *Camera) Look(dx, dy float32) {
	c.flyYaw += dx
	c.flyPitch = clamp(c.flyPitch+dy, -maxPitch, maxPitch)
}

// MoveSpeed is the fly translation speed in world units per second, scaled
// to the model size via the construction distance.
func (c *Camera) MoveSpeed() float32 {
	return c.defDist * 0.5
}

// Move translates the fly camera: fwd along the look direction, right
// perpendicular to it, up along world y. Magnitudes are key states (-1..1)
// integrated over dt seconds.
func (c *Camera) Move(fwd, right, up, dt float32) {
	f := flyForward(c.flyYaw, c.flyPitch)
	r := f.Cross(mathgl.Vec3{0, 1, 0}).Normalize()
	step := c.MoveSpeed() * dt
	c.pos = c.pos.
		Add(f.Scale(fwd * step)).
		Add(r.Scale(right * step)).
		Add(mathgl.Vec3{0, up * step, 0})
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
