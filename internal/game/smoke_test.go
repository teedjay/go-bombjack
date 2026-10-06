package game

import (
	"testing"

	"bombjack/internal/level"
	"bombjack/internal/world"
)

// TestHeadlessRounds steps every level's world without ebiten rendering.
func TestHeadlessRounds(t *testing.T) {
	for round := 0; round < len(level.Levels); round++ {
		p := NewPlay(round)
		for i := 0; i < 600; i++ {
			c := world.Controls{Left: i%200 < 100, Right: i%200 >= 100, Jump: i%60 < 20, JumpPressed: i%60 == 0}
			p.World.Step(c)
			p.Rules.Apply(p.World)
			hb := p.World.Player.Hitbox()
			if hb.X < 0 || hb.X+hb.W > world.FieldW || hb.Y < 0 || hb.Y+hb.H > world.FieldH {
				t.Fatalf("round %d tick %d: player hitbox out of field: %+v", round, i, hb)
			}
		}
	}
}
