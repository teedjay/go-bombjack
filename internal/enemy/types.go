package enemy

import (
	"math"

	"bombjack/internal/world"
)

const (
	spawnInTicks = 90
	morphTicks   = 20
	maxX         = world.FieldW - 16
	maxY         = world.FieldH - 16
)

type base struct {
	kind    Kind
	pos     world.Vec
	left    bool
	removed bool
	grace   int // ticks during which the enemy is not harmful
	speed   float64
}

func (b *base) Hitbox() world.Rect { return world.Inset(b.pos, 12, 12) }
func (b *base) Pos() world.Vec     { return b.pos }
func (b *base) Anim() string       { return b.kind.AnimName() }
func (b *base) FacingLeft() bool   { return b.left }
func (b *base) Removed() bool      { return b.removed }
func (b *base) Eat(w *world.World) { b.removed = true }
func (b *base) Harmful() bool      { return !b.removed && b.grace <= 0 }

// frozen reports whether the enemy must not act this tick, also ticking the
// grace timer when it may.
func (b *base) frozen(w *world.World) bool {
	if w.Frightened() {
		return true
	}
	if b.grace > 0 {
		b.grace--
	}
	return false
}

func (b *base) clamp() {
	b.pos.X = math.Max(0, math.Min(maxX, b.pos.X))
	b.pos.Y = math.Max(0, math.Min(maxY, b.pos.Y))
}

func (b *base) toward(w *world.World) world.Vec {
	if w.Player == nil {
		return world.Vec{X: 128, Y: 112}
	}
	return w.Player.Pos()
}

// ---------------------------------------------------------------- Mummy

type MummyEnemy struct {
	base
	vy      float64
	dir     float64
	decided bool
	fright  bool
	mix     [3]int
	level   float64
}

func NewMummy(w *world.World, pos world.Vec, speed float64, mix [3]int) *MummyEnemy {
	if speed <= 0 {
		speed = 1
	}
	m := &MummyEnemy{base: base{kind: Mummy, pos: pos, grace: spawnInTicks, speed: 0.6 * speed}, dir: 1, mix: mix, level: speed}
	if w.Rng.IntN(2) == 0 {
		m.dir = -1
	}
	m.left = m.dir < 0
	return m
}

func (m *MummyEnemy) Harmful() bool { return m.base.Harmful() && !m.fright }

// supported reports whether solid ground exists below point x at foot level y.
func supported(w *world.World, x, y float64) bool {
	if y >= world.FieldH-0.5 {
		return true
	}
	for _, p := range w.Platforms {
		r := p.Rect()
		if x >= r.X && x < r.X+r.W && y >= r.Y-1 && y <= r.Y+2 {
			return true
		}
	}
	return false
}

func (m *MummyEnemy) Update(w *world.World) {
	m.fright = w.Frightened()
	if m.frozen(w) {
		return
	}
	if m.grace > 0 {
		return // spawning in: stay put
	}
	hb := m.Hitbox()
	const g, maxFall = 0.15, 3.0
	vel := world.Vec{X: m.dir * m.speed, Y: m.vy + g}
	r, nv, onGround := w.MoveAndCollide(hb, vel, true)
	if !onGround {
		// slower drift in the air
		r.X = hb.X + (r.X-hb.X)*0.5
	}
	m.vy = math.Min(nv.Y, maxFall)
	if onGround {
		m.vy = 0
	}
	if vel.X != 0 && nv.X == 0 { // wall
		m.dir = -m.dir
	}
	m.pos = world.Vec{X: r.X - 2, Y: r.Y - 4}
	m.left = m.dir < 0
	if !onGround {
		m.decided = false
		return
	}
	if r.Y+r.H >= world.FieldH-0.5 {
		m.transform(w)
		return
	}
	ahead := r.X + r.W/2 + m.dir*(r.W/2+1)
	if supported(w, ahead, r.Y+r.H) {
		m.decided = false
	} else if !m.decided {
		m.decided = true
		if w.Rng.IntN(2) == 0 {
			m.dir = -m.dir
			m.left = m.dir < 0
		}
	}
}

func (m *MummyEnemy) transform(w *world.World) {
	m.removed = true
	k := pickKind(w, m.mix, m.level)
	w.Enemies = append(w.Enemies, k(m.pos))
	w.Emit(world.Event{Kind: world.EvEnemyTransformed, Pos: m.pos})
}

// pickKind chooses bird/saucer/orb by the weights; equal if all zero.
func pickKind(w *world.World, mix [3]int, speed float64) func(world.Vec) world.Enemy {
	total := mix[0] + mix[1] + mix[2]
	if total <= 0 {
		mix, total = [3]int{1, 1, 1}, 3
	}
	n := w.Rng.IntN(total)
	switch {
	case n < mix[0]:
		return func(p world.Vec) world.Enemy { return NewBird(w, p, speed) }
	case n < mix[0]+mix[1]:
		return func(p world.Vec) world.Enemy { return NewSaucer(w, p, speed) }
	}
	return func(p world.Vec) world.Enemy { return NewOrb(w, p, speed) }
}

// ---------------------------------------------------------------- Bird

type BirdEnemy struct {
	base
	angle  float64 // radians
	fright bool
}

const maxTurn = 2 * math.Pi / 180

func NewBird(w *world.World, pos world.Vec, speed float64) *BirdEnemy {
	if speed <= 0 {
		speed = 1
	}
	b := &BirdEnemy{base: base{kind: Bird, pos: pos, grace: morphTicks, speed: (1.0 + 0.6*w.Rng.Float64()) * speed}}
	b.angle = -math.Pi / 2
	return b
}

func (b *BirdEnemy) Harmful() bool { return b.base.Harmful() && !b.fright }

func (b *BirdEnemy) Update(w *world.World) {
	b.fright = w.Frightened()
	if b.frozen(w) {
		return
	}
	t := b.toward(w)
	want := math.Atan2(t.Y-b.pos.Y, t.X-b.pos.X)
	d := math.Remainder(want-b.angle, 2*math.Pi)
	d = math.Max(-maxTurn, math.Min(maxTurn, d))
	b.angle += d
	vx, vy := math.Cos(b.angle)*b.speed, math.Sin(b.angle)*b.speed
	b.pos.X += vx
	b.pos.Y += vy
	b.left = vx < 0
	b.clamp()
}

// ---------------------------------------------------------------- Saucer

type SaucerEnemy struct {
	base
	vel    world.Vec
	fright bool
}

func NewSaucer(w *world.World, pos world.Vec, speed float64) *SaucerEnemy {
	if speed <= 0 {
		speed = 1
	}
	s := &SaucerEnemy{base: base{kind: Saucer, pos: pos, grace: morphTicks, speed: 1.3 * speed}}
	c := s.speed / math.Sqrt2
	s.vel = world.Vec{X: c, Y: -c}
	if w.Rng.IntN(2) == 0 {
		s.vel.X = -c
	}
	return s
}

func (s *SaucerEnemy) Harmful() bool { return s.base.Harmful() && !s.fright }

func (s *SaucerEnemy) Update(w *world.World) {
	s.fright = w.Frightened()
	if s.frozen(w) {
		return
	}
	s.pos = s.pos.Add(s.vel)
	if s.pos.X < 0 {
		s.pos.X, s.vel.X = -s.pos.X, math.Abs(s.vel.X)
	} else if s.pos.X > maxX {
		s.pos.X, s.vel.X = 2*maxX-s.pos.X, -math.Abs(s.vel.X)
	}
	if s.pos.Y < 0 {
		s.pos.Y, s.vel.Y = -s.pos.Y, math.Abs(s.vel.Y)
	} else if s.pos.Y > maxY {
		s.pos.Y, s.vel.Y = 2*maxY-s.pos.Y, -math.Abs(s.vel.Y)
	}
	s.left = s.vel.X < 0
}

// ---------------------------------------------------------------- Orb

type OrbEnemy struct {
	base
	dir    world.Vec // unit vector towards the current target
	phase  float64
	retime int
	fright bool
}

const orbRetarget = 120

func NewOrb(w *world.World, pos world.Vec, speed float64) *OrbEnemy {
	if speed <= 0 {
		speed = 1
	}
	o := &OrbEnemy{base: base{kind: Orb, pos: pos, grace: morphTicks, speed: 0.7 * speed}}
	o.phase = w.Rng.Float64() * 2 * math.Pi
	o.retarget(w)
	return o
}

func (o *OrbEnemy) Harmful() bool { return o.base.Harmful() && !o.fright }

func (o *OrbEnemy) retarget(w *world.World) {
	t := o.toward(w)
	dx, dy := t.X-o.pos.X, t.Y-o.pos.Y
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		o.dir = world.Vec{X: 1}
	} else {
		o.dir = world.Vec{X: dx / l, Y: dy / l}
	}
	o.retime = orbRetarget
}

func (o *OrbEnemy) Update(w *world.World) {
	o.fright = w.Frightened()
	if o.frozen(w) {
		return
	}
	if o.retime--; o.retime <= 0 {
		o.retarget(w)
	}
	o.phase += 0.07
	wob := math.Sin(o.phase) * 0.9
	vx := o.dir.X*o.speed - o.dir.Y*wob
	vy := o.dir.Y*o.speed + o.dir.X*wob
	o.pos.X += vx
	o.pos.Y += vy
	o.left = vx < 0
	o.clamp()
}
