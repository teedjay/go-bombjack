package world

// MoveAndCollide moves hitbox r by vel against the playfield walls, the
// floor, and (if solidFromAbove) the one-way platforms. It returns the moved
// rect, the velocity after collision (zeroed on blocked axes) and whether the
// rect is standing on something.
//
// TODO(WS2): implement. Requirements:
//   - side walls at x=0 and x=FieldW and the ceiling at y=0 are solid
//   - floor: rect bottom may not exceed FieldH
//   - platforms block only downward motion whose previous bottom was at or
//     above the platform top (jump up through them; land on them)
//   - no tunnelling at |vel| up to 8 px/tick
func (w *World) MoveAndCollide(r Rect, vel Vec, solidFromAbove bool) (Rect, Vec, bool) {
	const eps = 1e-6
	// X axis: walls only.
	r.X += vel.X
	if r.X < 0 {
		r.X, vel.X = 0, 0
	}
	if r.X+r.W > FieldW {
		r.X, vel.X = FieldW-r.W, 0
	}
	// Y axis: ceiling, floor, one-way platforms (swept, so no tunnelling).
	prevBottom := r.Y + r.H
	r.Y += vel.Y
	if r.Y < 0 {
		r.Y, vel.Y = 0, 0
	}
	onGround := false
	if vel.Y >= 0 {
		land := float64(FieldH)
		hit := r.Y+r.H >= FieldH-eps
		if solidFromAbove {
			newBottom := r.Y + r.H
			for _, p := range w.Platforms {
				pr := p.Rect()
				if r.X+r.W <= pr.X || pr.X+pr.W <= r.X {
					continue
				}
				if prevBottom <= pr.Y+eps && newBottom >= pr.Y-eps && pr.Y < land {
					land, hit = pr.Y, true
				}
			}
		}
		if hit {
			r.Y, vel.Y, onGround = land-r.H, 0, true
		}
	}
	return r, vel, onGround
}

// contacts resolves Jack against bombs, pickups and enemies.
func (w *World) contacts() {
	if w.Player == nil || !w.Player.Alive() {
		return
	}
	hb := w.Player.Hitbox()
	for i, b := range w.Bombs {
		if !b.Taken && hb.Overlaps(b.Hitbox()) {
			b.Taken = true
			w.Emit(Event{Kind: EvBombTaken, Pos: b.Pos, Index: i, Lit: b.Lit})
		}
	}
	for _, p := range w.Pickups {
		if !p.Taken && hb.Overlaps(p.Hitbox()) {
			p.Taken = true
			w.Emit(Event{Kind: EvPickupTaken, Pos: p.Pos, Pickup: p.Kind})
		}
	}
	w.Pickups = removeIf(w.Pickups, func(p *Pickup) bool { return p.Taken })
	for _, e := range w.Enemies {
		if e.Removed() || !hb.Overlaps(e.Hitbox()) {
			continue
		}
		if w.Frightened() {
			e.Eat(w)
			w.Emit(Event{Kind: EvCoinEaten, Pos: e.Pos()})
		} else if e.Harmful() && !w.Invincible {
			w.Player.Kill(w)
			w.Emit(Event{Kind: EvPlayerHit, Pos: w.Player.Pos()})
			return
		}
	}
}
