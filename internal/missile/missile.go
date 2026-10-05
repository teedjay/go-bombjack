// Package missile implements Jack's homing missiles. It is headless (no
// ebiten) so the flight can be unit-tested.
//
// Flight model: each missile follows a cubic Bézier curve from its launch
// point to its target. The first control point lies along the launch
// heading, the second is offset sideways from the target, so the missile
// swings round in a wide arc instead of flying straight. The end points
// track the moving target every tick, the curve parameter advances by
// arc length so the missile accelerates smoothly, and a decaying sideways
// wobble is added on top. The drawn angle eases toward the actual motion.
package missile

import (
	"math"

	"bombjack/internal/world"
)

// Tuning (pixels, ticks).
const (
	LaunchSpeed = 1.0  // px/tick at launch
	Accel       = 0.07 // px/tick² until MaxSpeed
	MaxSpeed    = 6.5
	WobbleAmp   = 4.0  // px sideways at launch, fading to 0 at the target
	WobbleFreq  = 0.32 // radians per tick
	TurnEase    = 0.3  // fraction of the angle error corrected per tick
	MaxAge      = 600  // ticks before a missile self-destructs
	hitRadius   = 4    // half-size of the missile's hitbox
)

// Missile is one projectile in flight. Pos is its centre.
type Missile struct {
	Pos    world.Vec
	Angle  float64 // radians, for drawing (0 = facing right)
	Speed  float64
	Age    int
	Done   bool
	target world.Enemy
	p0, p1 world.Vec // curve start and launch-side control point
	side   float64   // which side the curve swings round (+1/-1)
	t      float64   // curve parameter 0..1
	phase  float64
}

// Manager owns all missiles in flight. It is a world.System.
type Manager struct {
	Missiles []*Missile
	launches int
}

// Fire launches a missile from pos towards the closest enemy. It returns
// false (and launches nothing) when there is no enemy to aim at.
func (m *Manager) Fire(w *world.World, pos world.Vec, facingLeft bool) bool {
	target := m.pickTarget(w, pos, nil)
	if target == nil {
		return false
	}
	dir := 1.0
	if facingLeft {
		dir = -1
	}
	m.launches++
	ms := &Missile{
		Pos:   pos,
		Angle: math.Atan2(-1, 0.6*dir), // launch up and forward
		Speed: LaunchSpeed,
		phase: float64(m.launches) * 1.7,
	}
	// alternate the swing side so salvoes fan out
	ms.side = 1
	if m.launches%2 == 0 {
		ms.side = -1
	}
	ms.retarget(target)
	m.Missiles = append(m.Missiles, ms)
	w.Emit(world.Event{Kind: world.EvMissileFired, Pos: pos})
	return true
}

// pickTarget returns the closest live enemy, preferring ones no other
// missile is chasing. skip is excluded (the enemy just lost).
func (m *Manager) pickTarget(w *world.World, from world.Vec, skip world.Enemy) world.Enemy {
	chased := map[world.Enemy]bool{}
	for _, o := range m.Missiles {
		if !o.Done && o.target != nil {
			chased[o.target] = true
		}
	}
	var best, bestFree world.Enemy
	bd, bfd := math.Inf(1), math.Inf(1)
	for _, e := range w.Enemies {
		if e.Removed() || e == skip {
			continue
		}
		d := dist(from, center(e))
		if d < bd {
			best, bd = e, d
		}
		if !chased[e] && d < bfd {
			bestFree, bfd = e, d
		}
	}
	if bestFree != nil {
		return bestFree
	}
	return best
}

// retarget starts a new curve from the current position and heading.
func (ms *Missile) retarget(e world.Enemy) {
	ms.target = e
	ms.t = 0
	ms.p0 = ms.Pos
	l := math.Max(dist(ms.Pos, center(e)), 60)
	ms.p1 = ms.Pos.Add(world.Vec{X: math.Cos(ms.Angle), Y: math.Sin(ms.Angle)}.Scale(l * 0.45))
}

// controls returns the current Bézier control points (target may move).
func (ms *Missile) controls() (p0, p1, p2, p3 world.Vec) {
	p3 = center(ms.target)
	d := p3.Add(ms.p0.Scale(-1))
	l := math.Hypot(d.X, d.Y)
	if l < 1 {
		l = 1
	}
	perp := world.Vec{X: -d.Y / l, Y: d.X / l}
	p2 = p3.Add(perp.Scale(ms.side * math.Max(l, 60) * 0.55))
	return ms.p0, ms.p1, p2, p3
}

func bezier(p0, p1, p2, p3 world.Vec, t float64) world.Vec {
	u := 1 - t
	a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return world.Vec{
		X: a*p0.X + b*p1.X + c*p2.X + d*p3.X,
		Y: a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y,
	}
}

// curveLen approximates the remaining arc length from t to 1.
func curveLen(p0, p1, p2, p3 world.Vec, t float64) float64 {
	const n = 12
	l, prev := 0.0, bezier(p0, p1, p2, p3, t)
	for i := 1; i <= n; i++ {
		q := bezier(p0, p1, p2, p3, t+(1-t)*float64(i)/n)
		l += dist(prev, q)
		prev = q
	}
	return l
}

// Update moves every missile and resolves hits. It implements world.System.
func (m *Manager) Update(w *world.World) {
	for _, ms := range m.Missiles {
		if !ms.Done {
			m.step(w, ms)
		}
	}
	n := 0
	for _, ms := range m.Missiles {
		if !ms.Done {
			m.Missiles[n] = ms
			n++
		}
	}
	m.Missiles = m.Missiles[:n]
}

func (m *Manager) step(w *world.World, ms *Missile) {
	ms.Age++
	ms.Speed = math.Min(ms.Speed+Accel, MaxSpeed)
	if ms.target == nil || ms.target.Removed() {
		if e := m.pickTarget(w, ms.Pos, ms.target); e != nil {
			ms.retarget(e)
		} else {
			ms.target = nil
		}
	}
	prev := ms.Pos
	if ms.target == nil {
		// nothing left to chase: keep flying straight until off-field
		ms.Pos = ms.Pos.Add(world.Vec{X: math.Cos(ms.Angle), Y: math.Sin(ms.Angle)}.Scale(ms.Speed))
	} else {
		p0, p1, p2, p3 := ms.controls()
		remaining := curveLen(p0, p1, p2, p3, ms.t)
		if remaining > 0.01 {
			ms.t = math.Min(1, ms.t+(1-ms.t)*ms.Speed/remaining)
		} else {
			ms.t = 1
		}
		base := bezier(p0, p1, p2, p3, ms.t)
		// sideways wobble, perpendicular to the curve, fading out near the end
		ahead := bezier(p0, p1, p2, p3, math.Min(1, ms.t+0.02))
		tx, ty := ahead.X-base.X, ahead.Y-base.Y
		if tl := math.Hypot(tx, ty); tl > 1e-6 {
			wob := math.Sin(float64(ms.Age)*WobbleFreq+ms.phase) * WobbleAmp * (1 - ms.t)
			base = base.Add(world.Vec{X: -ty / tl * wob, Y: tx / tl * wob})
		}
		ms.Pos = base
	}
	// ease the drawn angle toward the direction actually travelled
	if dx, dy := ms.Pos.X-prev.X, ms.Pos.Y-prev.Y; dx*dx+dy*dy > 1e-6 {
		ms.Angle += wrap(math.Atan2(dy, dx)-ms.Angle) * TurnEase
	}

	// hits: any enemy the missile touches, or reaching the end of the curve
	box := world.Rect{X: ms.Pos.X - hitRadius, Y: ms.Pos.Y - hitRadius, W: 2 * hitRadius, H: 2 * hitRadius}
	for _, e := range w.Enemies {
		if !e.Removed() && box.Overlaps(e.Hitbox()) {
			m.explode(w, ms, e)
			return
		}
	}
	if ms.target != nil && ms.t >= 1 {
		m.explode(w, ms, ms.target)
		return
	}
	out := ms.Pos.X < -8 || ms.Pos.X > world.FieldW+8 || ms.Pos.Y < -8 || ms.Pos.Y > world.FieldH+8
	if out || ms.Age > MaxAge {
		m.explode(w, ms, nil)
	}
}

func (m *Manager) explode(w *world.World, ms *Missile, e world.Enemy) {
	ms.Done = true
	killed := 0
	if e != nil && !e.Removed() {
		e.Eat(w)
		killed = 1
	}
	pos := world.Vec{X: math.Max(0, math.Min(world.FieldW, ms.Pos.X)), Y: math.Max(0, math.Min(world.FieldH, ms.Pos.Y))}
	w.Emit(world.Event{Kind: world.EvMissileHit, Pos: pos, Index: killed})
}

// Tail returns the point at the back of the missile (for the smoke trail).
func (ms *Missile) Tail() world.Vec {
	return ms.Pos.Add(world.Vec{X: -math.Cos(ms.Angle) * 7, Y: -math.Sin(ms.Angle) * 7})
}

func center(e world.Enemy) world.Vec {
	h := e.Hitbox()
	return world.Vec{X: h.X + h.W/2, Y: h.Y + h.H/2}
}

func dist(a, b world.Vec) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }

// wrap maps an angle difference into (-π, π].
func wrap(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a <= -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
