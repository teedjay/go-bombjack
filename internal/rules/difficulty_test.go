package rules

import (
	"testing"

	"bombjack/internal/enemy"
	"bombjack/internal/world"
)

var testCfg = enemy.Config{Speed: 1, SpawnEvery: 400, MaxEnemies: 5, StartCount: 2}

// Easy must reproduce the original tuning exactly.
func TestEasyIsOriginalTuning(t *testing.T) {
	c := Scale(testCfg, 0, Easy)
	if c.Speed != 1 || c.SpawnEvery != 400 || c.MaxEnemies != 5 || c.StartCount != 2 || c.FirstDelay != 200 {
		t.Fatalf("easy loop 0 changed the level: %+v", c)
	}
	c = Scale(testCfg, 2, Easy)
	if c.Speed != 1.2 || c.SpawnEvery != 289 || c.MaxEnemies != 5 {
		t.Fatalf("easy loop 2: %+v", c)
	}
	if FrightDuration(0, Easy) != 5*60 || Easy.Profile().Missiles != StartMissiles || Easy.Profile().CrateEvery != MissileEvery {
		t.Fatal("easy power-ups changed")
	}
}

// Each step up is strictly harder on every axis.
func TestDifficultyOrdering(t *testing.T) {
	for loop := 0; loop < 4; loop++ {
		e, n, h := Scale(testCfg, loop, Easy), Scale(testCfg, loop, Normal), Scale(testCfg, loop, Hard)
		if !(e.Speed < n.Speed && n.Speed < h.Speed) {
			t.Errorf("loop %d speed %v %v %v", loop, e.Speed, n.Speed, h.Speed)
		}
		if !(e.SpawnEvery > n.SpawnEvery && n.SpawnEvery > h.SpawnEvery) {
			t.Errorf("loop %d spawn %v %v %v", loop, e.SpawnEvery, n.SpawnEvery, h.SpawnEvery)
		}
		if !(e.MaxEnemies < n.MaxEnemies && n.MaxEnemies < h.MaxEnemies) {
			t.Errorf("loop %d cap %v %v %v", loop, e.MaxEnemies, n.MaxEnemies, h.MaxEnemies)
		}
		if !(e.FirstDelay > n.FirstDelay && n.FirstDelay > h.FirstDelay) {
			t.Errorf("first delay %v %v %v", e.FirstDelay, n.FirstDelay, h.FirstDelay)
		}
		if FrightDuration(loop, Easy) < FrightDuration(loop, Hard) {
			t.Errorf("loop %d: hard P lasts longer than easy", loop)
		}
	}
	// Hard's cap grows per loop but never by more than +3 over its base
	if got := Scale(testCfg, 10, Hard).MaxEnemies; got != testCfg.MaxEnemies+Hard.Profile().ExtraMax+3 {
		t.Errorf("hard cap at loop 10 = %d", got)
	}
}

func TestDifficultyStockAndCrates(t *testing.T) {
	w, order := newWorld()
	s := NewForLevel(order, 0)
	s.SetDifficulty(Hard)
	if s.Missiles != 2 {
		t.Fatalf("hard starts with %d missiles", s.Missiles)
	}
	s.Missiles = 7
	w.Events = []world.Event{{Kind: world.EvPlayerDied}}
	s.Apply(w)
	if s.Missiles != 2 {
		t.Fatalf("hard life restarts with %d missiles", s.Missiles)
	}
	crateAt := func(d Difficulty) int {
		w, order := newWorld()
		s := NewForLevel(order, 0)
		s.SetDifficulty(d)
		for i := 1; i < 2000; i++ {
			w.Events = w.Events[:0]
			s.Apply(w)
			for _, p := range w.Pickups {
				if p.Kind == world.PickupM {
					return i
				}
			}
		}
		return -1
	}
	if e, h := crateAt(Easy), crateAt(Hard); e != MissileEvery || h != Hard.Profile().CrateEvery {
		t.Fatalf("first crate easy %d hard %d", e, h)
	}
}
