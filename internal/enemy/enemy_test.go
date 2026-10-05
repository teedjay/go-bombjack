package enemy

import (
	"math"
	"testing"

	"bombjack/internal/world"
)

func TestMummyTransformsOnFloor(t *testing.T) {
	w := world.New(1)
	m := NewMummy(w, world.Vec{X: 100, Y: 190}, 1, [3]int{0, 1, 0})
	w.Enemies = append(w.Enemies, m)
	if m.Harmful() {
		t.Fatal("harmful while spawning")
	}
	got := false
	for i := 0; i < 300 && !got; i++ {
		w.Step(world.Controls{})
		for _, e := range w.Events {
			if e.Kind == world.EvEnemyTransformed {
				got = true
			}
		}
	}
	if !got || !m.Removed() {
		t.Fatal("mummy did not transform")
	}
	if _, ok := w.Enemies[len(w.Enemies)-1].(*SaucerEnemy); !ok {
		t.Fatalf("want saucer, got %T", w.Enemies[len(w.Enemies)-1])
	}
}

func TestSaucerBounce(t *testing.T) {
	w := world.New(1)
	s := NewSaucer(w, world.Vec{X: 0.5, Y: 100}, 1)
	s.vel = world.Vec{X: -s.speed / math.Sqrt2, Y: s.speed / math.Sqrt2}
	s.grace = 0
	s.Update(w)
	if s.vel.X <= 0 || s.vel.Y <= 0 {
		t.Fatalf("vel %v", s.vel)
	}
	s.pos = world.Vec{X: 100, Y: maxY - 0.5}
	s.Update(w)
	if s.vel.Y >= 0 {
		t.Fatalf("no floor bounce %v", s.vel)
	}
	if math.Abs(math.Hypot(s.vel.X, s.vel.Y)-s.speed) > 1e-9 {
		t.Fatal("speed changed")
	}
}

type fakePlayer struct{ pos world.Vec }

func (f *fakePlayer) Update(*world.World, world.Controls) {}
func (f *fakePlayer) Hitbox() world.Rect                  { return world.Inset(f.pos, 10, 14) }
func (f *fakePlayer) Pos() world.Vec                      { return f.pos }
func (f *fakePlayer) Anim() string                        { return "" }
func (f *fakePlayer) FacingLeft() bool                    { return false }
func (f *fakePlayer) Alive() bool                         { return true }
func (f *fakePlayer) Done() bool                          { return false }
func (f *fakePlayer) Kill(*world.World)                   {}

func TestBirdTurnLimitAndHoming(t *testing.T) {
	w := world.New(1)
	w.Player = &fakePlayer{pos: world.Vec{X: 200, Y: 200}}
	b := NewBird(w, world.Vec{X: 100, Y: 100}, 1)
	b.grace = 0
	prev := b.angle
	for i := 0; i < 200; i++ {
		b.Update(w)
		if d := math.Abs(math.Remainder(b.angle-prev, 2*math.Pi)); d > maxTurn+1e-9 {
			t.Fatalf("turn %v too large", d)
		}
		prev = b.angle
	}
	if math.Hypot(b.pos.X-200, b.pos.Y-200) > 20 {
		t.Fatalf("bird did not home: %v", b.pos)
	}
}

func TestFrightenedFreezes(t *testing.T) {
	w := world.New(1)
	w.Player = &fakePlayer{pos: world.Vec{X: 200, Y: 200}}
	b := NewBird(w, world.Vec{X: 100, Y: 100}, 1)
	b.grace = 0
	w.FrightTicks = 10
	p := b.pos
	b.Update(w)
	if b.pos != p || b.Harmful() {
		t.Fatal("frightened bird moved or harmful")
	}
	b.Eat(w)
	if !b.Removed() {
		t.Fatal("not removed")
	}
}

func TestSpawnerCapAndReplace(t *testing.T) {
	w := world.New(2)
	cfg := Config{Spawns: []world.Vec{{X: 10, Y: 0}}, MaxEnemies: 4, StartCount: 2, SpawnEvery: 100, Speed: 1}
	sp := NewSpawner(cfg)
	for i := 0; i < 3000; i++ {
		w.Tick++
		sp.Update(w)
		if len(w.Enemies) > cfg.MaxEnemies {
			t.Fatal("cap exceeded")
		}
	}
	if len(w.Enemies) != 4 {
		t.Fatalf("got %d", len(w.Enemies))
	}
	w.Enemies[0].Eat(w)
	w.Enemies = w.Enemies[1:]
	for i := 0; i < 200; i++ {
		w.Tick++
		sp.Update(w)
	}
	if len(w.Enemies) != 4 {
		t.Fatalf("not replaced: %d", len(w.Enemies))
	}
}

func TestEdgeSupport(t *testing.T) {
	w := world.New(1)
	w.Platforms = []world.Platform{{TX: 2, TY: 10, Len: 4}}
	if !supported(w, 20, 80) || supported(w, 60, 80) {
		t.Fatal("support probe wrong")
	}
}
