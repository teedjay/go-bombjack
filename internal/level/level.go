// Package level holds the four level definitions. Owned by WS4.
package level

import (
	"sort"

	"bombjack/internal/enemy"
	"bombjack/internal/world"
)

// Def describes one screen. Platforms are in tile coords (32x28 grid);
// everything else is in playfield pixels (top-left of a 16x16 sprite).
type Def struct {
	Name        string // also the art theme name: egypt, greece, castle, city
	Platforms   []world.Platform
	Bombs       []world.Vec // exactly 24
	LitOrder    []int       // permutation of 0..23: order fuses light up
	PlayerStart world.Vec
	Enemies     enemy.Config
}

// row returns n bomb positions starting at (x,y) spaced step px apart.
func row(x, y float64, n int, step float64) []world.Vec {
	out := make([]world.Vec, n)
	for i := range out {
		out[i] = world.Vec{X: x + float64(i)*step, Y: y}
	}
	return out
}

// col returns n bomb positions starting at (x,y) spaced step px apart downwards.
func col(x, y float64, n int, step float64) []world.Vec {
	out := make([]world.Vec, n)
	for i := range out {
		out[i] = world.Vec{X: x, Y: y + float64(i)*step}
	}
	return out
}

// groups bundles bomb groups (each a row or column) for mk.
func groups(parts ...[]world.Vec) [][]world.Vec { return parts }

// zigzag orders positions top to bottom, alternating between the left and right
// halves of the field, so the lit fuse jumps across the screen.
func zigzag(bombs []world.Vec) []int {
	idx := make([]int, len(bombs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		pa, pb := bombs[idx[a]], bombs[idx[b]]
		if pa.Y != pb.Y {
			return pa.Y < pb.Y
		}
		return pa.X < pb.X
	})
	var l, r []int
	for _, i := range idx {
		if bombs[i].X < world.FieldW/2-8 {
			l = append(l, i)
		} else {
			r = append(r, i)
		}
	}
	out := make([]int, 0, len(bombs))
	for i := 0; i < len(l) || i < len(r); i++ {
		if i < len(l) {
			out = append(out, l[i])
		}
		if i < len(r) {
			out = append(out, r[i])
		}
	}
	return out
}

// groupOrder builds the lit order from bomb groups: groups are visited in
// zig-zag order of their first bomb, and within a group bombs light in the
// order they were laid out (row: left to right, col: top to bottom), so a
// whole group can be cleared in one run or one drop.
func groupOrder(gs [][]world.Vec) (bombs []world.Vec, order []int) {
	firsts := make([]world.Vec, len(gs))
	start := make([]int, len(gs))
	for i, g := range gs {
		firsts[i] = g[0]
		start[i] = len(bombs)
		bombs = append(bombs, g...)
	}
	for _, gi := range zigzag(firsts) {
		for k := range gs[gi] {
			order = append(order, start[gi]+k)
		}
	}
	return bombs, order
}

func mk(name string, plats []world.Platform, gs [][]world.Vec, cfg enemy.Config) Def {
	bombs, order := groupOrder(gs)
	return Def{Name: name, Platforms: plats, Bombs: bombs, LitOrder: order,
		PlayerStart: world.Vec{X: 120, Y: world.FloorY}, Enemies: cfg}
}

var topSpawns = []world.Vec{{X: 24, Y: 0}, {X: 120, Y: 0}, {X: 216, Y: 0}}

// Levels in play order; after the last one the game loops with higher
// difficulty (see internal/rules).
var Levels = []Def{
	mk("egypt",
		[]world.Platform{P(3, 7, 8), P(21, 7, 8), P(11, 13, 10), P(2, 19, 6), P(24, 19, 6)},
		groups(row(32, 38, 3, 16), row(176, 38, 3, 16), row(100, 86, 4, 16),
			row(24, 134, 3, 16), row(192, 134, 3, 16),
			col(4, 70, 3, 18), col(236, 70, 3, 18), row(112, 190, 2, 16)),
		enemy.Config{Spawns: topSpawns, Mix: [3]int{4, 2, 1}, MaxEnemies: 3, StartCount: 1, SpawnEvery: 720, Speed: 0.8}),
	mk("greece",
		[]world.Platform{P(12, 6, 8), P(2, 11, 7), P(23, 11, 7), P(12, 17, 8)},
		groups(row(100, 30, 4, 16), row(20, 70, 3, 16), row(188, 70, 3, 16), row(100, 118, 4, 16),
			row(20, 190, 3, 16), row(188, 190, 3, 16),
			col(4, 130, 2, 20), col(236, 130, 2, 20)),
		enemy.Config{Spawns: topSpawns, Mix: [3]int{3, 3, 2}, MaxEnemies: 5, StartCount: 2, SpawnEvery: 520, Speed: 0.95}),
	mk("castle",
		[]world.Platform{P(2, 6, 6), P(24, 6, 6), P(8, 12, 16), P(2, 19, 5), P(25, 19, 5)},
		groups(row(20, 30, 3, 16), row(196, 30, 3, 16), row(84, 78, 4, 24),
			row(20, 134, 3, 16), row(204, 134, 3, 16),
			col(4, 70, 3, 20), col(236, 70, 3, 20), row(116, 190, 2, 16)),
		enemy.Config{Spawns: topSpawns, Mix: [3]int{2, 4, 2}, MaxEnemies: 6, StartCount: 2, SpawnEvery: 400, Speed: 1.15}),
	mk("city",
		[]world.Platform{P(5, 8, 6), P(21, 8, 6), P(13, 14, 6)},
		groups(row(44, 46, 3, 16), row(172, 46, 3, 16), row(108, 94, 3, 16), row(96, 20, 4, 16),
			col(4, 40, 4, 30), col(236, 40, 4, 30), row(60, 190, 1, 0), row(116, 190, 1, 0), row(172, 190, 1, 0)),
		enemy.Config{Spawns: topSpawns, Mix: [3]int{2, 3, 4}, MaxEnemies: 7, StartCount: 3, SpawnEvery: 280, Speed: 1.4}),
}

// Build creates the static part of a world for level index i (0-based,
// wraps around). The caller adds the Player and Systems.
func Build(i int, seed uint64) *world.World {
	d := Levels[i%len(Levels)]
	w := world.New(seed)
	w.Level = i % len(Levels)
	w.Platforms = append([]world.Platform(nil), d.Platforms...)
	for _, p := range d.Bombs {
		w.Bombs = append(w.Bombs, &world.Bomb{Pos: p})
	}
	return w
}

// P is shorthand for a platform at tile (tx,ty) n tiles long.
func P(tx, ty, n int) world.Platform { return world.Platform{TX: tx, TY: ty, Len: n} }
