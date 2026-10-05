package level

import (
	"testing"

	"bombjack/internal/world"
)

func TestLevels(t *testing.T) {
	for _, d := range Levels {
		if len(d.Bombs) != 24 || len(d.LitOrder) != 24 {
			t.Fatalf("%s: %d bombs, %d lit", d.Name, len(d.Bombs), len(d.LitOrder))
		}
		seen := map[int]bool{}
		for _, i := range d.LitOrder {
			if i < 0 || i > 23 || seen[i] {
				t.Fatalf("%s: LitOrder not a permutation", d.Name)
			}
			seen[i] = true
		}
		for i, b := range d.Bombs {
			r := world.Rect{X: b.X, Y: b.Y, W: 16, H: 16}
			if b.X < 0 || b.Y < 0 || b.X+16 > world.FieldW || b.Y+16 > world.FieldH {
				t.Errorf("%s bomb %d outside field", d.Name, i)
			}
			for _, p := range d.Platforms {
				if r.Overlaps(p.Rect()) {
					t.Errorf("%s bomb %d overlaps platform %v", d.Name, i, p)
				}
			}
			for j := 0; j < i; j++ {
				o := d.Bombs[j]
				if r.Overlaps(world.Rect{X: o.X, Y: o.Y, W: 16, H: 16}) {
					t.Errorf("%s bombs %d and %d overlap", d.Name, i, j)
				}
			}
		}
		if len(d.Enemies.Spawns) == 0 || d.Enemies.MaxEnemies < 3 {
			t.Errorf("%s: bad enemy config", d.Name)
		}
	}
}
