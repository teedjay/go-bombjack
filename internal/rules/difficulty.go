package rules

import (
	"math"

	"bombjack/internal/enemy"
)

// Difficulty is chosen in the menu. Easy is the original tuning; Normal and
// Hard scale each level's enemy config and the power-up generosity on top
// of the normal per-level progression and per-loop scaling.
type Difficulty int

const (
	Easy Difficulty = iota
	Normal
	Hard
)

func (d Difficulty) String() string {
	return [...]string{"EASY", "NORMAL", "HARD"}[d.clamp()]
}

func (d Difficulty) clamp() Difficulty { return max(Easy, min(Hard, d)) }

// Profile holds every difficulty-dependent setting.
type Profile struct {
	Speed      float64 // enemy speed multiplier
	Spawn      float64 // spawn interval multiplier (< 1 = enemies arrive faster)
	ExtraMax   int     // added to the level's enemy cap...
	LoopMax    int     // ...plus this much per loop (capped at +3 in total)
	ExtraStart int     // added to the enemies present at round start
	FirstDelay int     // ticks of quiet before the first enemy
	FrightSec  int     // P (coins) duration in seconds at loop 0; -1 s per loop, min 2
	LoopSpeed  float64 // speed increase per loop (0.10 = +10% per loop)
	LoopSpawn  float64 // spawn-interval factor per loop (compounding)
	Missiles   int     // missiles at the start of each life
	CrateEvery int     // ticks between missile crates
}

// Profiles: Easy reproduces the original tuning exactly.
var Profiles = [...]Profile{
	Easy: {Speed: 1, Spawn: 1, FirstDelay: 200, FrightSec: 5, LoopSpeed: 0.10, LoopSpawn: 0.85,
		Missiles: StartMissiles, CrateEvery: MissileEvery},
	Normal: {Speed: 1.15, Spawn: 0.8, ExtraMax: 1, ExtraStart: 1, FirstDelay: 150, FrightSec: 4,
		LoopSpeed: 0.12, LoopSpawn: 0.85, Missiles: StartMissiles, CrateEvery: 720},
	Hard: {Speed: 1.3, Spawn: 0.65, ExtraMax: 2, LoopMax: 1, ExtraStart: 1, FirstDelay: 100, FrightSec: 3,
		LoopSpeed: 0.15, LoopSpawn: 0.80, Missiles: 2, CrateEvery: 900},
}

// Profile returns the settings for d.
func (d Difficulty) Profile() Profile { return Profiles[d.clamp()] }

// SetDifficulty selects the difficulty for a new game and gives the
// matching starting missile stock.
func (s *State) SetDifficulty(d Difficulty) {
	s.Diff = d.clamp()
	s.Missiles = s.Diff.Profile().Missiles
}

// FrightDuration returns the P duration in ticks for a loop.
func FrightDuration(loop int, d Difficulty) int {
	return max(d.Profile().FrightSec-loop, 2) * 60
}

// Scale returns a level's enemy config adjusted for the loop and the
// difficulty: faster enemies, shorter spawn intervals, a higher cap and
// more enemies at the start.
func Scale(cfg enemy.Config, loop int, d Difficulty) enemy.Config {
	p := d.Profile()
	cfg.Speed *= p.Speed * (1 + p.LoopSpeed*float64(loop))
	cfg.SpawnEvery = int(math.Round(float64(cfg.SpawnEvery) * p.Spawn * math.Pow(p.LoopSpawn, float64(loop))))
	cfg.MaxEnemies += p.ExtraMax + min(p.LoopMax*loop, 3)
	cfg.StartCount = min(cfg.StartCount+p.ExtraStart, cfg.MaxEnemies)
	cfg.FirstDelay = p.FirstDelay
	return cfg
}
