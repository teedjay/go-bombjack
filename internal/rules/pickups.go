package rules

import "bombjack/internal/world"

// PickupMover is a world.System that bounces pickups around the field at
// constant speed, reflecting off walls, ceiling, floor and platform tops.
type PickupMover struct{}

func (PickupMover) Update(w *world.World) {
	for _, p := range w.Pickups {
		prevBottom := p.Pos.Y + 16
		p.Pos = p.Pos.Add(p.Vel)
		if p.Pos.X < 0 {
			p.Pos.X, p.Vel.X = -p.Pos.X, abs(p.Vel.X)
		} else if p.Pos.X > world.FieldW-16 {
			p.Pos.X, p.Vel.X = 2*(world.FieldW-16)-p.Pos.X, -abs(p.Vel.X)
		}
		if p.Pos.Y < 0 {
			p.Pos.Y, p.Vel.Y = -p.Pos.Y, abs(p.Vel.Y)
		} else if p.Pos.Y > world.FloorY {
			p.Pos.Y, p.Vel.Y = 2*world.FloorY-p.Pos.Y, -abs(p.Vel.Y)
		}
		if p.Vel.Y > 0 {
			bottom := p.Pos.Y + 16
			for _, pl := range w.Platforms {
				r := pl.Rect()
				if p.Pos.X+14 > r.X && p.Pos.X+2 < r.X+r.W && prevBottom <= r.Y && bottom > r.Y {
					p.Pos.Y = r.Y - 16
					p.Vel.Y = -p.Vel.Y
					break
				}
			}
		}
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
