package sim

import (
	"math"
	"math/rand/v2"
	"testing"

	"bombjack/internal/level"
	"bombjack/internal/world"
)

func finite(v world.Vec) bool {
	return !math.IsNaN(v.X) && !math.IsNaN(v.Y) && !math.IsInf(v.X, 0) && !math.IsInf(v.Y, 0)
}

func inField(r world.Rect) bool {
	const tol = 1.0
	return r.X >= -tol && r.Y >= -tol && r.X+r.W <= world.FieldW+tol && r.Y+r.H <= world.FieldH+tol
}

// TestSoak plays 10 simulated minutes per level with random input.
func TestSoak(t *testing.T) {
	if testing.Short() {
		t.Skip("soak test skipped in short mode")
	}
	const ticks = 36000
	for li := range level.Levels {
		r := NewRound(li, uint64(li)+42)
		maxEn := r.Def.Enemies.MaxEnemies
		rng := rand.New(rand.NewPCG(uint64(li)+1, 99))
		var c world.Controls
		for i := 0; i < ticks; i++ {
			if i%15 == 0 {
				c = world.Controls{Left: rng.IntN(3) == 0, Right: rng.IntN(3) == 0, Down: rng.IntN(8) == 0}
			}
			c.Jump = rng.IntN(4) != 0
			c.JumpPressed = rng.IntN(25) == 0
			r.Step(c)
			if r.Rules.Lives <= 0 {
				r.Rules.Lives = 3 // keep playing
			}
			if r.World.BombsLeft() == 0 { // endless: refill bombs
				for _, b := range r.World.Bombs {
					b.Taken, b.Lit = false, false
				}
				r.Rules.NextRound(r.Def.LitOrder)
			}
			w := r.World
			if !finite(w.Player.Pos()) || !inField(w.Player.Hitbox()) {
				t.Fatalf("L%d tick %d: bad player %v", li+1, i, w.Player.Pos())
			}
			if len(w.Enemies) > maxEn+1 { // +1: a mummy is replaced by its transformed form within a tick
				t.Fatalf("L%d tick %d: %d enemies > max %d", li+1, i, len(w.Enemies), maxEn)
			}
			for _, e := range w.Enemies {
				if !finite(e.Pos()) || !inField(e.Hitbox()) {
					t.Fatalf("L%d tick %d: bad enemy %T at %v", li+1, i, e, e.Pos())
				}
			}
			for _, p := range w.Pickups {
				if !finite(p.Pos) {
					t.Fatalf("L%d tick %d: bad pickup %v", li+1, i, p.Pos)
				}
			}
		}
	}
}
