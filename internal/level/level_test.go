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

// Groups light one by one in layout order: rows left to right, columns top
// to bottom, and a group is never interleaved with another.
func TestGroupOrder(t *testing.T) {
	gs := groups(
		row(150, 100, 3, 16), // right, lower
		col(4, 20, 3, 20),    // left column, top
		row(40, 60, 2, 16),   // left, middle
	)
	bombs, order := groupOrder(gs)
	if len(bombs) != 8 || len(order) != 8 {
		t.Fatalf("got %d bombs, %d order", len(bombs), len(order))
	}
	// zig-zag by first bomb alternates sides: left col (y20), right row
	// (y100), then left row (y60)
	want := []int{3, 4, 5, 0, 1, 2, 6, 7}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	for i := 1; i < 3; i++ { // column lights top to bottom
		if bombs[order[i]].Y <= bombs[order[i-1]].Y {
			t.Fatalf("column not top to bottom: %v", order)
		}
	}
}
