// Package world holds the shared game state and the contracts between the
// gameplay packages (player, enemy, level, rules). It must not import ebiten
// so all game logic stays headless and unit-testable.
//
// Coordinates are playfield pixels: origin top-left, 256x224, y grows down.
// Entity positions are the top-left of their 16x16 sprite.
package world

import "math/rand/v2"

const (
	FieldW = 256
	FieldH = 224
	Tile   = 8
	// FloorY is the y position of a 16px-tall entity standing on the floor.
	FloorY = FieldH - 16
)

type Vec struct{ X, Y float64 }

func (v Vec) Add(o Vec) Vec       { return Vec{v.X + o.X, v.Y + o.Y} }
func (v Vec) Scale(f float64) Vec { return Vec{v.X * f, v.Y * f} }

type Rect struct{ X, Y, W, H float64 }

func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H
}

// Inset returns the hitbox for a 16x16 sprite at pos with a w x h box
// centred horizontally and aligned to the sprite bottom.
func Inset(pos Vec, w, h float64) Rect {
	return Rect{pos.X + (16-w)/2, pos.Y + 16 - h, w, h}
}

// Controls is one tick of player input, produced by internal/input.
type Controls struct {
	Left, Right, Down bool
	Jump              bool // held
	JumpPressed       bool // went down this tick
	Start             bool // pressed this tick
	Fire              bool // missile button pressed this tick
}

// Platform is a horizontal run of Len tiles at tile coords (TX,TY).
// Platforms are solid from above only.
type Platform struct{ TX, TY, Len int }

func (p Platform) Rect() Rect {
	return Rect{float64(p.TX * Tile), float64(p.TY * Tile), float64(p.Len * Tile), Tile}
}

type Bomb struct {
	Pos   Vec
	Lit   bool
	Taken bool
}

func (b *Bomb) Hitbox() Rect { return Inset(b.Pos, 10, 11) }

type PickupKind int

const (
	PickupP PickupKind = iota // power: frighten enemies
	PickupB                   // bonus multiplier +1
	PickupE                   // extra life
	PickupS                   // special
	PickupM                   // +1 homing missile
)

type Pickup struct {
	Kind  PickupKind
	Pos   Vec
	Vel   Vec
	Taken bool
}

func (p *Pickup) Hitbox() Rect { return Inset(p.Pos, 12, 12) }

// Player is implemented by internal/player.
type Player interface {
	Update(w *World, c Controls)
	Hitbox() Rect
	Pos() Vec
	Anim() string // art animation name, e.g. "jack_run"
	FacingLeft() bool
	Alive() bool   // false once hit (dying animation may still play)
	Done() bool    // death sequence finished
	Kill(w *World) // called by World on enemy contact
}

// Enemy is implemented by internal/enemy.
type Enemy interface {
	Update(w *World)
	Hitbox() Rect
	Pos() Vec
	Anim() string
	FacingLeft() bool
	Harmful() bool // false while spawning/transforming
	Eat(w *World)  // touched by Jack while frightened (rendered as "coin")
	Removed() bool // world drops it at end of tick
}

// System is anything updated once per tick after the player (enemy spawner,
// pickup movement, ...).
type System interface{ Update(w *World) }

type World struct {
	Tick      int
	Rng       *rand.Rand
	Level     int // 0..3 theme/level index
	Platforms []Platform
	Player    Player
	Enemies   []Enemy
	Bombs     []*Bomb
	Pickups   []*Pickup
	Systems   []System
	// FrightTicks > 0 while P is active: enemies are harmless coins.
	FrightTicks int
	// Invincible (cheat) makes enemy contact harmless to Jack.
	Invincible bool
	// Events emitted this tick; cleared at the start of each Step.
	Events []Event
}

func New(seed uint64) *World {
	return &World{Rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

func (w *World) Emit(e Event) { w.Events = append(w.Events, e) }

func (w *World) Frightened() bool { return w.FrightTicks > 0 }

// Step advances the world one tick: player, systems, enemies, then contacts.
// Game rules (scoring, lit chain, round clear) are applied afterwards by
// internal/rules reading w.Events.
func (w *World) Step(c Controls) {
	w.Events = w.Events[:0]
	w.Tick++
	if w.FrightTicks > 0 {
		w.FrightTicks--
	}
	if w.Player != nil {
		w.Player.Update(w, c)
	}
	for _, s := range w.Systems {
		s.Update(w)
	}
	for _, e := range w.Enemies {
		e.Update(w)
	}
	w.contacts()
	w.Enemies = removeIf(w.Enemies, Enemy.Removed)
}

// BombsLeft reports how many bombs are not yet collected.
func (w *World) BombsLeft() int {
	n := 0
	for _, b := range w.Bombs {
		if !b.Taken {
			n++
		}
	}
	return n
}

func removeIf[T any](s []T, f func(T) bool) []T {
	out := s[:0]
	for _, v := range s {
		if !f(v) {
			out = append(out, v)
		}
	}
	return out
}
