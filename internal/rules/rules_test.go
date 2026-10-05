package rules

import (
	"testing"

	"bombjack/internal/enemy"
	"bombjack/internal/world"
)

func newWorld() (*world.World, []int) {
	w := world.New(1)
	order := make([]int, 24)
	for i := range order {
		w.Bombs = append(w.Bombs, &world.Bomb{Pos: world.Vec{X: float64(i * 8), Y: 50}})
		order[i] = (i * 7) % 24 // permutation
	}
	return w, order
}

// take marks bomb i taken, emits the event as the world would, and applies.
func take(s *State, w *world.World, i int) {
	w.Events = w.Events[:0]
	b := w.Bombs[i]
	b.Taken = true
	w.Emit(world.Event{Kind: world.EvBombTaken, Index: i, Lit: b.Lit, Pos: b.Pos})
	s.Apply(w)
}

func hasEv(w *world.World, k world.EventKind) int {
	n := 0
	for _, e := range w.Events {
		if e.Kind == k {
			n++
		}
	}
	return n
}

func TestFullLitChain(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	take(s, w, 5) // arbitrary first bomb
	lit := -1
	for i, b := range w.Bombs {
		if b.Lit {
			if lit >= 0 {
				t.Fatal("two lit")
			}
			lit = i
		}
	}
	// the bomb lit is the one after bomb 5 in the order
	pos := 0
	for i, idx := range order {
		if idx == 5 {
			pos = i
		}
	}
	if want := order[(pos+1)%len(order)]; lit != want {
		t.Fatalf("lit %d want %d", lit, want)
	}
	count := 0
	for lit >= 0 && count < 30 {
		take(s, w, lit)
		count++
		lit = -1
		for i, b := range w.Bombs {
			if b.Lit && !b.Taken {
				lit = i
			}
		}
	}
	if s.LitCount != 23 { // the chain covers all 23 bombs after the first
		t.Fatalf("litcount %d", s.LitCount)
	}
	if w.BombsLeft() != 0 {
		t.Fatalf("left %d", w.BombsLeft())
	}
}

func TestScoreAndMultiplierCap(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	for i := 0; i < 8; i++ {
		w.Events = []world.Event{{Kind: world.EvPickupTaken, Pickup: world.PickupB}}
		s.Apply(w)
	}
	if s.Multiplier != 5 {
		t.Fatalf("mult %d", s.Multiplier)
	}
	take(s, w, 0)
	if s.Score != 500 {
		t.Fatalf("score %d", s.Score)
	}
	if hasEv(w, world.EvScore) != 1 {
		t.Fatal("no score popup")
	}
}

func TestMeterSpawnsP(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	for i := 0; i < 25; i++ {
		w.Events = []world.Event{{Kind: world.EvBombTaken, Index: i % 24}}
		s.Apply(w)
	}
	if len(w.Pickups) != 1 || w.Pickups[0].Kind != world.PickupP || s.Meter != 0 {
		t.Fatalf("pickups %d meter %d", len(w.Pickups), s.Meter)
	}
	w.Events = []world.Event{{Kind: world.EvPickupTaken, Pickup: world.PickupP}}
	s.Apply(w)
	if w.FrightTicks != 5*60 {
		t.Fatalf("fright %d", w.FrightTicks)
	}
	w.Events = []world.Event{{Kind: world.EvCoinEaten}, {Kind: world.EvCoinEaten}}
	before := s.Score
	s.Apply(w)
	if s.Score-before != 300 {
		t.Fatalf("coins %d", s.Score-before)
	}
}

func TestRoundClearOnceAndBonus(t *testing.T) {
	for lit, want := range map[int]int{19: 0, 20: 10000, 21: 20000, 22: 30000, 23: 50000} {
		w, order := newWorld()
		s := NewForLevel(order, 0)
		s.LitCount = lit
		for _, b := range w.Bombs {
			b.Taken = true
		}
		w.Events = nil
		s.Apply(w)
		if hasEv(w, world.EvRoundClear) != 1 || s.Score != want {
			t.Fatalf("lit %d: score %d clears %d", lit, s.Score, hasEv(w, world.EvRoundClear))
		}
		w.Events = nil
		s.Apply(w)
		if hasEv(w, world.EvRoundClear) != 0 || s.Score != want {
			t.Fatal("cleared twice")
		}
	}
}

func TestLivesOnDeathAndExtra(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	w.Events = []world.Event{{Kind: world.EvPlayerDied}}
	s.Apply(w)
	if s.Lives != 2 {
		t.Fatal("lives")
	}
	w.Events = []world.Event{{Kind: world.EvPickupTaken, Pickup: world.PickupE}}
	s.Apply(w)
	if s.Lives != 3 || hasEv(w, world.EvExtraLife) != 1 {
		t.Fatal("extra life")
	}
}

func TestThresholdsAndScale(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	s.Score = 60000
	w.Events = nil
	s.Apply(w)
	kinds := map[world.PickupKind]int{}
	for _, p := range w.Pickups {
		kinds[p.Kind]++
	}
	if kinds[world.PickupB] != 2 || kinds[world.PickupE] != 1 {
		t.Fatalf("%v", kinds)
	}
	c := Scale(enemy.Config{Speed: 1, SpawnEvery: 400}, 2)
	if c.Speed != 1.2 || c.SpawnEvery != 289 {
		t.Fatalf("%+v", c)
	}
	if FrightDuration(9) != 120 {
		t.Fatal("min fright")
	}
}

func TestPickupMoverBounds(t *testing.T) {
	w := world.New(1)
	w.Platforms = []world.Platform{{TX: 10, TY: 10, Len: 8}}
	w.Pickups = []*world.Pickup{{Pos: world.Vec{X: 100, Y: 20}, Vel: world.Vec{X: 0.8, Y: 0.8}}}
	for i := 0; i < 5000; i++ {
		PickupMover{}.Update(w)
		p := w.Pickups[0]
		if p.Pos.X < -0.01 || p.Pos.X > 240.01 || p.Pos.Y < -0.01 || p.Pos.Y > world.FloorY+0.01 {
			t.Fatalf("out of bounds %v", p.Pos)
		}
	}
}

func TestHintBomb(t *testing.T) {
	w := world.New(1)
	for i := 0; i < 4; i++ {
		w.Bombs = append(w.Bombs, &world.Bomb{Pos: world.Vec{X: float64(i * 20)}})
	}
	s := NewForLevel([]int{2, 3, 0, 1}, 0)
	if got := s.HintBomb(w); got != 2 {
		t.Fatalf("hint = %d, want 2 (first in LitOrder)", got)
	}
	w.Bombs[2].Taken = true // taken out of the world by other means
	if got := s.HintBomb(w); got != 3 {
		t.Fatalf("hint = %d, want 3 (first not taken)", got)
	}
	w.Bombs[0].Taken = true
	w.Events = []world.Event{{Kind: world.EvBombTaken, Index: 0}}
	s.Apply(w)
	if got := s.HintBomb(w); got != -1 {
		t.Fatalf("hint = %d after the chain started, want -1", got)
	}
}

// Taking the first bomb of a group lights its neighbour, so running through
// the group takes every bomb lit.
func TestRunThroughGroup(t *testing.T) {
	w := world.New(1)
	for i := 0; i < 6; i++ {
		w.Bombs = append(w.Bombs, &world.Bomb{Pos: world.Vec{X: float64(i * 16)}})
	}
	s := NewForLevel([]int{3, 4, 5, 0, 1, 2}, 0) // two groups of three
	take(s, w, 0)                                // enter group 2 at its start
	for _, i := range []int{1, 2} {
		if !w.Bombs[i].Lit {
			t.Fatalf("bomb %d should be lit when reached", i)
		}
		take(s, w, i)
	}
	if !w.Bombs[3].Lit {
		t.Fatal("chain should wrap to the next group")
	}
}

func TestMissileStock(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	if s.Missiles != StartMissiles {
		t.Fatalf("start with %d missiles", s.Missiles)
	}
	// crates spawn regularly, one at a time
	for i := 0; i < MissileEvery*3; i++ {
		w.Events = w.Events[:0]
		s.Apply(w)
	}
	crates := 0
	for _, p := range w.Pickups {
		if p.Kind == world.PickupM {
			crates++
		}
	}
	if crates != 1 {
		t.Fatalf("%d crates on field, want 1", crates)
	}
	// a crate adds 3, capped at MaxMissiles; at the cap it pays points
	crate := func() {
		w.Events = []world.Event{{Kind: world.EvPickupTaken, Pickup: world.PickupM}}
		s.Apply(w)
	}
	crate()
	if s.Missiles != StartMissiles+MissilesPerBox {
		t.Fatalf("one crate: %d missiles", s.Missiles)
	}
	s.Missiles = MaxMissiles - 1
	crate() // only room for one more
	if s.Missiles != MaxMissiles || s.Score != 0 {
		t.Fatalf("near cap: missiles %d score %d", s.Missiles, s.Score)
	}
	crate()
	crate()
	if s.Missiles != MaxMissiles || s.Score != 2*MissileFullPts {
		t.Fatalf("at cap: missiles %d score %d", s.Missiles, s.Score)
	}
	for i := 0; i < MaxMissiles; i++ {
		if !s.UseMissile() {
			t.Fatal("ran out early")
		}
	}
	if s.UseMissile() {
		t.Fatal("fired with an empty stock")
	}
	// a kill scores; an explosion with no kill does not
	before := s.Score
	w.Events = []world.Event{{Kind: world.EvMissileHit, Index: 1}, {Kind: world.EvMissileHit, Index: 0}}
	s.Apply(w)
	if s.Score-before != MissileKillPts {
		t.Fatalf("kill scored %d", s.Score-before)
	}
}

func TestMissilesResetOnDeath(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	for _, n := range []int{7, 0} { // more than the start stock, and empty
		s.Missiles = n
		w.Events = []world.Event{{Kind: world.EvPlayerDied}}
		s.Apply(w)
		if s.Missiles != StartMissiles {
			t.Fatalf("after death with %d: %d missiles, want %d", n, s.Missiles, StartMissiles)
		}
	}
}
