package world

// EventKind identifies something that happened during a tick. Gameplay code
// emits events; rules, audio and FX consume them. This is the decoupling
// layer between workstreams.
type EventKind int

const (
	EvBombTaken   EventKind = iota // Index = bomb index, Lit = was lit
	EvPickupTaken                  // Pickup = kind
	EvCoinEaten                    // enemy eaten while frightened
	EvPlayerHit                    // Jack touched a harmful enemy
	EvPlayerDied                   // death sequence finished
	EvJump                         // Jack started a jump
	EvFloat                        // Jack started floating
	EvLand                         // Jack landed
	EvEnemySpawned
	EvEnemyTransformed // mummy reached floor and changed form
	EvRoundClear       // emitted by rules when all bombs are taken
	EvExtraLife        // emitted by rules
	EvScore            // emitted by rules: Value points at Pos (score popup)
)

type Event struct {
	Kind   EventKind
	Pos    Vec
	Index  int
	Lit    bool
	Pickup PickupKind
	Value  int
}
