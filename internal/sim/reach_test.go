package sim

import (
	"testing"

	"bombjack/internal/level"
	"bombjack/internal/player"
	"bombjack/internal/world"
)

// TestBombsReachable simulates Jack alone jumping/floating/drifting from
// every standing surface and checks that every bomb is touched.
func TestBombsReachable(t *testing.T) {
	for li, d := range level.Levels {
		w := level.Build(li, 1)
		type surf struct{ x0, x1, y float64 }
		surfs := []surf{{0, world.FieldW - 16, world.FloorY}}
		for _, p := range d.Platforms {
			r := p.Rect()
			surfs = append(surfs, surf{r.X - 6, r.X + r.W - 10, r.Y - 16})
		}
		touched := make([]bool, len(w.Bombs))
		reached := make([]bool, len(surfs))
		reached[0] = true
		done := make([]bool, len(surfs))
		for progress := true; progress; {
			progress = false
			for si, s := range surfs {
				if !reached[si] || done[si] {
					continue
				}
				done[si], progress = true, true
				scan(w, s.x0, s.x1, s.y, touched, func(lx, ly float64) {
					for k, o := range surfs {
						if ly == o.y && lx >= o.x0-1 && lx <= o.x1+1 {
							reached[k] = true
						}
					}
				})
			}
		}
		for k, ok := range reached {
			if !ok {
				t.Errorf("level %d: platform %d not reachable from the floor", li+1, k-1)
			}
		}
		for i, ok := range touched {
			if !ok {
				t.Errorf("level %d (%s): bomb %d at %v unreachable", li+1, d.Name, i, w.Bombs[i].Pos)
			}
		}
	}
}

func scan(w *world.World, x0, x1, y float64, touched []bool, land func(x, y float64)) {
	{
		{
			for x := x0; x <= x1; x += 4 {
				for hold := 1; hold <= player.Default.JumpHoldMax; hold++ {
					for dir := -1; dir <= 1; dir++ {
						for fl := -1; fl <= 60; fl += 5 {
							simJump(w, world.Vec{X: x, Y: y}, hold, dir, fl, touched, land)
						}
					}
				}
			}
		}
	}
}

func simJump(w *world.World, start world.Vec, hold, dir, floatAt int, touched []bool, land func(x, y float64)) {
	j := player.New(start)
	w.Player = j
	check := func() {
		hb := j.Hitbox()
		for i, b := range w.Bombs {
			if !touched[i] && hb.Overlaps(b.Hitbox()) {
				touched[i] = true
			}
		}
	}
	for t := 0; t < 300; t++ {
		c := world.Controls{Left: dir < 0, Right: dir > 0}
		if t == 0 {
			c.JumpPressed = true
		}
		c.Jump = t < hold
		if floatAt >= 0 && t == floatAt+1 {
			c.JumpPressed = true
		}
		j.Update(w, c)
		check()
		if t > 2 && (j.State() == player.Idle || j.State() == player.Run) {
			land(j.Pos().X, j.Pos().Y)
			break
		}
	}
}
