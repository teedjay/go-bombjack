// Package player implements Jack. Owned by WS2.
package player

import "bombjack/internal/world"

type State int

const (
	Idle State = iota
	Run
	Jump
	Float
	Fall
	Dying
	Dead
)

// Death sequence timings (ticks).
const (
	DeathFreeze = 30
	DeathSpin   = 60
)

// Tuning holds every physics constant in one place (pixels / tick).
type Tuning struct {
	RunSpeed     float64
	JumpImpulse  float64
	JumpHoldMax  int
	Gravity      float64
	MaxFall      float64
	FloatMaxFall float64
	// A float lasts FloatHold ticks at FloatMaxFall, then over FloatDecay
	// ticks the fall cap ramps back up to MaxFall. Each new jump press in
	// the air restarts it, so staying aloft takes repeated tapping.
	FloatHold  int
	FloatDecay int
	FloatLift  float64 // vertical velocity set by each float tap (negative = up)
}

var Default = Tuning{
	RunSpeed: 1.5, JumpImpulse: -6.0, JumpHoldMax: 14,
	Gravity: 0.25, MaxFall: 3.5, FloatMaxFall: 0.6,
	FloatHold: 18, FloatDecay: 24, FloatLift: -1.1,
}

// Jack implements world.Player.
type Jack struct {
	T          Tuning
	pos        world.Vec
	vel        world.Vec
	state      State
	left       bool
	onFloor    bool
	jumping    bool // still in the held-jump rise phase
	holdTicks  int
	floating   bool
	floatTicks int // ticks since the last float press
	deathTicks int
}

// New places Jack standing at start (levels start him on the floor).
func New(start world.Vec) *Jack { return &Jack{T: Default, pos: start, onFloor: true} }

// Respawn resets Jack after a lost life.
func (j *Jack) Respawn(pos world.Vec) {
	t := j.T
	*j = Jack{T: t, pos: pos, onFloor: true}
}

func (j *Jack) Update(w *world.World, c world.Controls) {
	switch j.state {
	case Dead:
		return
	case Dying:
		j.deathTicks++
		if j.deathTicks >= DeathFreeze+DeathSpin {
			j.state = Dead
			w.Emit(world.Event{Kind: world.EvPlayerDied, Pos: j.pos})
		}
		return
	}

	j.vel.X = 0
	if c.Left {
		j.vel.X, j.left = -j.T.RunSpeed, true
	}
	if c.Right {
		j.vel.X, j.left = j.T.RunSpeed, false
	}

	if c.JumpPressed {
		if j.onFloor {
			j.vel.Y = j.T.JumpImpulse
			j.jumping, j.holdTicks = true, 0
			j.floating = false
			j.onFloor = false
			w.Emit(world.Event{Kind: world.EvJump, Pos: j.pos})
		} else {
			j.vel.Y = j.T.FloatLift
			j.jumping, j.floating, j.floatTicks = false, true, 0
			w.Emit(world.Event{Kind: world.EvFloat, Pos: j.pos})
		}
	}

	if j.jumping && c.Jump && j.holdTicks < j.T.JumpHoldMax {
		j.holdTicks++ // keep impulse velocity, no gravity
	} else {
		j.jumping = false
		j.vel.Y += j.T.Gravity
	}
	if j.floating && c.Down {
		j.floating = false
	}
	limit := j.T.MaxFall
	if j.floating {
		j.floatTicks++
		limit = j.floatLimit()
		if limit >= j.T.MaxFall {
			j.floating = false // float expired: back to a normal fall
		}
	}
	if j.vel.Y > limit {
		j.vel.Y = limit
	}

	hb := j.Hitbox()
	r, v, ground := w.MoveAndCollide(hb, j.vel, true)
	j.pos = j.pos.Add(world.Vec{X: r.X - hb.X, Y: r.Y - hb.Y})
	j.vel = v
	if v.Y == 0 && !ground && j.jumping { // bumped the ceiling
		j.jumping = false
	}
	if ground {
		if !j.onFloor {
			w.Emit(world.Event{Kind: world.EvLand, Pos: j.pos})
		}
		j.jumping, j.floating = false, false
	}
	j.onFloor = ground

	switch {
	case ground && j.vel.X != 0:
		j.state = Run
	case ground:
		j.state = Idle
	case j.floating:
		j.state = Float
	case j.vel.Y < 0:
		j.state = Jump
	default:
		j.state = Fall
	}
}

func (j *Jack) Hitbox() world.Rect { return world.Inset(j.pos, 10, 14) }
func (j *Jack) Pos() world.Vec     { return j.pos }
func (j *Jack) FacingLeft() bool   { return j.left }
func (j *Jack) Alive() bool        { return j.state != Dying && j.state != Dead }
func (j *Jack) Done() bool         { return j.state == Dead }
func (j *Jack) State() State       { return j.state }

// Kill starts the death sequence (idempotent).
func (j *Jack) Kill(w *world.World) {
	if !j.Alive() {
		return
	}
	j.state = Dying
	j.deathTicks = 0
	j.vel = world.Vec{}
}

// DeathTicks reports ticks elapsed in the death sequence; spin starts after DeathFreeze.
func (j *Jack) DeathTicks() int { return j.deathTicks }

func (j *Jack) Anim() string {
	switch j.state {
	case Run:
		return "jack_run"
	case Jump:
		return "jack_jump"
	case Float, Fall:
		return "jack_float"
	case Dying, Dead:
		return "jack_die"
	}
	return "jack_idle"
}

// floatLimit is the current fall-speed cap while floating: FloatMaxFall
// for FloatHold ticks, then ramping linearly to MaxFall over FloatDecay.
func (j *Jack) floatLimit() float64 {
	t := j.floatTicks - j.T.FloatHold
	if t <= 0 {
		return j.T.FloatMaxFall
	}
	f := min(float64(t)/float64(max(j.T.FloatDecay, 1)), 1)
	return j.T.FloatMaxFall + (j.T.MaxFall-j.T.FloatMaxFall)*f
}
