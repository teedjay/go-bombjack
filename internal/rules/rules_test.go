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
	if lit != order[0] {
		t.Fatalf("lit %d want %d", lit, order[0])
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
	if s.LitCount != 22 { // 23 remaining after first, but 5 was skipped in chain => 23 lit
		// chain covers all 23 other bombs
		if s.LitCount != 23 {
			t.Fatalf("litcount %d", s.LitCount)
		}
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
