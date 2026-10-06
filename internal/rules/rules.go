// Package rules implements scoring, the lit-bomb chain, the bonus meter,
// power-ups, lives and difficulty. Owned by WS4.
package rules

import (
	"bombjack/internal/level"
	"bombjack/internal/world"
)

type State struct {
	Score      int
	HiScore    int
	Lives      int
	Multiplier int // 1..5
	Meter      int // 0..100; full spawns a P
	LitChain   int // index into level LitOrder of the currently lit bomb
	LitCount   int // lit bombs taken this round
	Round      int // 0-based, increases forever
	CoinCombo  int // index into coin value table during one P
	Missiles   int // homing missiles in stock (0..MaxMissiles)
	Diff       Difficulty

	// LitOrder is the level's lit-bomb sequence (set by NewForLevel/NextRound).
	LitOrder []int

	started  bool // first bomb of the round taken
	cleared  bool // round clear already emitted
	nextB    int  // next score threshold for a B pickup
	eGiven   [2]bool
	bonusPts int // last round-clear bonus awarded
	mTimer   int // ticks since the last missile crate spawned
}

var coinValues = [...]int{100, 200, 300, 500, 800, 1200, 2000}

// Missile stock and scoring.
const (
	StartMissiles  = 3 // per life (Easy/Normal; see Profiles)
	MaxMissiles    = 9
	MissileEvery   = 600  // ticks between missile crate spawns (Easy; see Profiles)
	MissileKillPts = 500  // per enemy destroyed (× multiplier)
	MissileFullPts = 1000 // crate picked up while already holding MaxMissiles
	MissilesPerBox = 3    // missiles in one crate (capped at MaxMissiles)
)

const (
	bStep = 30000
	eAt0  = 50000
	eAt1  = 150000
)

func New() *State {
	return &State{Lives: 3, Multiplier: 1, Missiles: StartMissiles, nextB: bStep}
}

// UseMissile takes one missile from stock; false when empty.
func (s *State) UseMissile() bool {
	if s.Missiles <= 0 {
		return false
	}
	s.Missiles--
	return true
}

// NewForLevel creates a fresh game state for round (0-based) using the level's
// LitOrder.
func NewForLevel(litOrder []int, round int) *State {
	s := New()
	s.Round = round
	s.LitOrder = litOrder
	return s
}

// NextRound resets per-round state (lit chain, clear flag) keeping score,
// lives, multiplier and meter, and installs the next level's LitOrder.
func (s *State) NextRound(litOrder []int) {
	s.Round++
	s.LitOrder = litOrder
	s.LitChain, s.LitCount, s.CoinCombo = 0, 0, 0
	s.started, s.cleared = false, false
}

// Loop is how many times every level has been completed.
func (s *State) Loop() int { return s.Round / len(level.Levels) }

// RoundBonus returns the lit-bomb bonus last awarded at round clear.
func (s *State) RoundBonus() int { return s.bonusPts }

// LitBonus is the round-end bonus for the number of lit bombs taken.
func LitBonus(lit int) int {
	switch {
	case lit >= 23:
		return 50000
	case lit == 22:
		return 30000
	case lit == 21:
		return 20000
	case lit == 20:
		return 10000
	}
	return 0
}

func (s *State) add(w *world.World, pts int, pos world.Vec) {
	s.Score += pts
	w.Emit(world.Event{Kind: world.EvScore, Pos: pos, Value: pts})
}

func (s *State) gainMeter(w *world.World, n int) {
	s.Meter += n
	if s.Meter >= 100 {
		s.Meter = 0
		spawnPickup(w, world.PickupP)
	}
}

// HintBomb returns the bomb index that starts the lit chain while no bomb
// has been lit yet this round (taking it first lights its whole group in
// sequence), or -1 once the chain has started.
func (s *State) HintBomb(w *world.World) int {
	if s.started {
		return -1
	}
	for _, idx := range s.LitOrder {
		if idx >= 0 && idx < len(w.Bombs) && !w.Bombs[idx].Taken {
			return idx
		}
	}
	return -1
}

// lightNext lights the next not-taken bomb after bomb `from` in LitOrder,
// wrapping around. Because groups are consecutive in LitOrder, taking any
// bomb of a row or column lights its neighbour, so a group can be cleared
// in one run.
func (s *State) lightNext(w *world.World, from int) {
	n := len(s.LitOrder)
	start := 0
	for i, idx := range s.LitOrder {
		if idx == from {
			start = i + 1
			break
		}
	}
	for k := 0; k < n; k++ {
		i := (start + k) % n
		idx := s.LitOrder[i]
		if idx >= 0 && idx < len(w.Bombs) && !w.Bombs[idx].Taken {
			w.Bombs[idx].Lit = true
			s.LitChain = i
			return
		}
	}
}

func (s *State) extraLife(w *world.World, pos world.Vec) {
	s.Lives++
	w.Emit(world.Event{Kind: world.EvExtraLife, Pos: pos})
}

// Apply consumes w.Events for this tick and updates score/state, lights the
// next bomb, spawns pickups and sets w.FrightTicks. It appends follow-up
// events (EvScore, EvRoundClear, EvExtraLife) to w.Events.
func (s *State) Apply(w *world.World) {
	n := len(w.Events) // only process events from before Apply
	for i := 0; i < n; i++ {
		e := w.Events[i]
		switch e.Kind {
		case world.EvBombTaken:
			if e.Lit {
				s.add(w, 200*s.Multiplier, e.Pos)
				s.LitCount++
				s.gainMeter(w, 8)
			} else {
				s.add(w, 100*s.Multiplier, e.Pos)
				s.gainMeter(w, 4)
			}
			if !s.started || e.Lit {
				s.started = true
				s.lightNext(w, e.Index)
			}
		case world.EvPickupTaken:
			switch e.Pickup {
			case world.PickupP:
				w.FrightTicks = FrightDuration(s.Loop(), s.Diff)
				s.CoinCombo = 0
				if w.Rng.IntN(40) == 0 {
					spawnPickup(w, world.PickupS)
				}
			case world.PickupB:
				s.Multiplier = min(s.Multiplier+1, 5)
			case world.PickupE:
				s.extraLife(w, e.Pos)
			case world.PickupS:
				s.add(w, 100000, e.Pos)
				s.extraLife(w, e.Pos)
			case world.PickupM:
				if s.Missiles < MaxMissiles {
					s.Missiles = min(s.Missiles+MissilesPerBox, MaxMissiles)
				} else {
					s.add(w, MissileFullPts, e.Pos)
				}
			}
		case world.EvMissileHit:
			if e.Index == 1 {
				s.add(w, MissileKillPts*s.Multiplier, e.Pos)
				s.gainMeter(w, 4)
			}
		case world.EvCoinEaten:
			v := coinValues[min(s.CoinCombo, len(coinValues)-1)]
			s.CoinCombo++
			s.add(w, v, e.Pos)
			s.gainMeter(w, 2)
		case world.EvPlayerDied:
			if s.Lives > 0 {
				s.Lives--
			}
			s.Missiles = s.Diff.Profile().Missiles // every new life starts with a fresh stock
		}
	}
	// a missile crate drops in regularly (one on the field at a time)
	s.mTimer++
	if s.mTimer >= s.Diff.Profile().CrateEvery {
		s.mTimer = 0
		onField := false
		for _, p := range w.Pickups {
			if p.Kind == world.PickupM {
				onField = true
			}
		}
		if !onField {
			spawnPickup(w, world.PickupM)
		}
	}
	for s.Score >= s.nextB {
		s.nextB += bStep
		spawnPickup(w, world.PickupB)
	}
	if !s.eGiven[0] && s.Score >= eAt0 {
		s.eGiven[0] = true
		spawnPickup(w, world.PickupE)
	}
	if !s.eGiven[1] && s.Score >= eAt1 {
		s.eGiven[1] = true
		spawnPickup(w, world.PickupE)
	}
	if !s.cleared && len(w.Bombs) > 0 && w.BombsLeft() == 0 {
		s.cleared = true
		s.bonusPts = LitBonus(s.LitCount)
		pos := world.Vec{X: 120, Y: 104}
		if w.Player != nil {
			pos = w.Player.Pos()
		}
		w.Emit(world.Event{Kind: world.EvRoundClear, Pos: pos, Value: s.bonusPts})
		if s.bonusPts > 0 {
			s.add(w, s.bonusPts, pos)
		}
	}
	if s.Score > s.HiScore {
		s.HiScore = s.Score
	}
}

// spawnPickup adds a bouncing pickup near the top of the field.
func spawnPickup(w *world.World, k world.PickupKind) {
	vx := 0.8
	if w.Rng.IntN(2) == 0 {
		vx = -vx
	}
	w.Pickups = append(w.Pickups, &world.Pickup{
		Kind: k,
		Pos:  world.Vec{X: float64(32 + w.Rng.IntN(world.FieldW-64)), Y: 16},
		Vel:  world.Vec{X: vx, Y: 0.8},
	})
}
