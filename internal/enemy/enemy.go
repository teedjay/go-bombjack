// Package enemy implements mummies, birds, saucers and orbs plus the
// spawner. Owned by WS3.
package enemy

import "bombjack/internal/world"

type Kind int

const (
	Mummy Kind = iota
	Bird
	Saucer
	Orb
)

// AnimName maps a kind to its art animation.
func (k Kind) AnimName() string {
	return [...]string{"mummy_walk", "bird_fly", "saucer", "orb"}[k]
}

// Config is the per-level enemy setup, filled from level.Def.
type Config struct {
	Spawns     []world.Vec
	Mix        [3]int // transform weights: bird, saucer, orb
	MaxEnemies int
	StartCount int
	SpawnEvery int     // ticks between spawns
	Speed      float64 // base speed multiplier (1.0 = level 1)
}
