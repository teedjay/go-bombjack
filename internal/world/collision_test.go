package world

import "testing"

func TestMoveAndCollide(t *testing.T) {
	w := New(1)
	w.Platforms = []Platform{{TX: 10, TY: 10, Len: 4}} // top 80, x 80..112
	tests := []struct {
		name   string
		r      Rect
		v      Vec
		wantY  float64
		wantGr bool
	}{
		{"land fast", Rect{90, 60, 10, 14}, Vec{0, 8}, 66, true},
		{"rest exactly", Rect{90, 66, 10, 14}, Vec{0, 0}, 66, true},
		{"up through", Rect{90, 70, 10, 14}, Vec{0, -8}, 62, false},
		{"inside moving down passes", Rect{90, 70, 10, 14}, Vec{0, 3}, 73, false},
		{"beside platform", Rect{60, 60, 10, 14}, Vec{0, 8}, 68, false},
		{"floor", Rect{10, 205, 10, 14}, Vec{0, 8}, FieldH - 14, true},
		{"ceiling", Rect{10, 3, 10, 14}, Vec{0, -8}, 0, false},
	}
	for _, tc := range tests {
		r, _, g := w.MoveAndCollide(tc.r, tc.v, true)
		if r.Y != tc.wantY || g != tc.wantGr {
			t.Errorf("%s: y=%v ground=%v want %v %v", tc.name, r.Y, g, tc.wantY, tc.wantGr)
		}
	}
	r, _, g := w.MoveAndCollide(Rect{90, 60, 10, 14}, Vec{0, 8}, false)
	if g || r.Y != 68 {
		t.Errorf("non-solid platform blocked: %v %v", r.Y, g)
	}
	r, v, _ := w.MoveAndCollide(Rect{2, 100, 10, 14}, Vec{-5, 0}, true)
	if r.X != 0 || v.X != 0 {
		t.Errorf("left wall %v %v", r.X, v.X)
	}
}

// stubEnemy is a harmful enemy sitting at a fixed position.
type stubEnemy struct {
	pos   Vec
	eaten bool
}

func (e *stubEnemy) Update(*World)    {}
func (e *stubEnemy) Hitbox() Rect     { return Inset(e.pos, 12, 12) }
func (e *stubEnemy) Pos() Vec         { return e.pos }
func (e *stubEnemy) Anim() string     { return "orb" }
func (e *stubEnemy) FacingLeft() bool { return false }
func (e *stubEnemy) Harmful() bool    { return true }
func (e *stubEnemy) Eat(*World)       { e.eaten = true }
func (e *stubEnemy) Removed() bool    { return e.eaten }

type stubPlayer struct {
	pos    Vec
	killed bool
}

func (p *stubPlayer) Update(*World, Controls) {}
func (p *stubPlayer) Hitbox() Rect            { return Inset(p.pos, 10, 14) }
func (p *stubPlayer) Pos() Vec                { return p.pos }
func (p *stubPlayer) Anim() string            { return "jack_idle" }
func (p *stubPlayer) FacingLeft() bool        { return false }
func (p *stubPlayer) Alive() bool             { return !p.killed }
func (p *stubPlayer) Done() bool              { return p.killed }
func (p *stubPlayer) Kill(*World)             { p.killed = true }

func TestInvincibleIgnoresEnemies(t *testing.T) {
	for _, inv := range []bool{false, true} {
		w := New(1)
		pl := &stubPlayer{pos: Vec{X: 100, Y: FloorY}}
		w.Player = pl
		w.Enemies = []Enemy{&stubEnemy{pos: Vec{X: 100, Y: FloorY}}}
		w.Invincible = inv
		w.Step(Controls{})
		if pl.killed == inv {
			t.Fatalf("invincible=%v: killed=%v", inv, pl.killed)
		}
	}
}
