package player

import (
	"testing"

	"bombjack/internal/world"
)

func newWorld(p ...world.Platform) (*world.World, *Jack) {
	w := world.New(1)
	w.Platforms = p
	j := New(world.Vec{X: 120, Y: world.FloorY})
	w.Player = j
	return w, j
}

func count(w *world.World, k world.EventKind) int {
	n := 0
	for _, e := range w.Events {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// apex jumps and returns the max height risen above the start.
func apex(hold int) float64 {
	w, j := newWorld()
	w.Step(world.Controls{}) // settle
	start := j.Pos().Y
	min := start
	for i := 0; i < 200; i++ {
		c := world.Controls{Jump: i < hold, JumpPressed: i == 0}
		w.Step(c)
		if j.Pos().Y < min {
			min = j.Pos().Y
		}
	}
	return start - min
}

func TestJumpApex(t *testing.T) {
	full, tap := apex(14), apex(1)
	if full < 120 || full > 190 {
		t.Errorf("full jump apex %v out of range", full)
	}
	if tap >= full-30 || tap < 10 {
		t.Errorf("tap %v should be well below full %v", tap, full)
	}
}

func TestFloatCap(t *testing.T) {
	w, j := newWorld()
	w.Step(world.Controls{})
	w.Step(world.Controls{Jump: true, JumpPressed: true})
	for i := 0; i < 20; i++ {
		w.Step(world.Controls{Jump: i < 5})
	}
	w.Step(world.Controls{JumpPressed: true})
	if j.State() != Float {
		t.Fatalf("state %v, want Float", j.State())
	}
	prev := j.Pos().Y
	for i := 0; i < j.T.FloatHold-1; i++ {
		w.Step(world.Controls{})
		if d := j.Pos().Y - prev; d > j.T.FloatMaxFall+1e-9 {
			t.Fatalf("float descent %v exceeds cap during hold", d)
		}
		prev = j.Pos().Y
	}
	// after the hold, gravity takes over again and the float ends
	for i := 0; i < j.T.FloatDecay+2; i++ {
		w.Step(world.Controls{})
	}
	if j.State() == Float || j.vel.Y <= j.T.FloatMaxFall {
		t.Fatalf("float should have expired: state %v vy %v", j.State(), j.vel.Y)
	}
	// Down cancels the float.
	w, j = newWorld()
	w.Step(world.Controls{})
	w.Step(world.Controls{JumpPressed: true, Jump: true})
	for i := 0; i < 30; i++ {
		w.Step(world.Controls{})
	}
	w.Step(world.Controls{JumpPressed: true})
	for i := 0; i < 10; i++ {
		w.Step(world.Controls{Down: true})
	}
	if j.vel.Y <= j.T.FloatMaxFall {
		t.Errorf("down did not cancel float, vy=%v", j.vel.Y)
	}
}

func TestPlatforms(t *testing.T) {
	plat := world.Platform{TX: 10, TY: 16, Len: 10} // top y=128, x 80..160
	top := float64(plat.TY * world.Tile)

	t.Run("land", func(t *testing.T) {
		w, j := newWorld(plat)
		j.pos = world.Vec{X: 120, Y: 60}
		landed := false
		for i := 0; i < 100; i++ {
			w.Step(world.Controls{})
			landed = landed || count(w, world.EvLand) > 0
		}
		if got := j.Hitbox().Y + j.Hitbox().H; got != top {
			t.Errorf("bottom %v, want %v", got, top)
		}
		if !landed || j.State() != Idle {
			t.Errorf("landed=%v state=%v", landed, j.State())
		}
	})
	t.Run("up through", func(t *testing.T) {
		w, j := newWorld(plat)
		j.pos = world.Vec{X: 120, Y: world.FloorY}
		w.Step(world.Controls{})
		for i := 0; i < 80; i++ {
			w.Step(world.Controls{Jump: true, JumpPressed: i == 0})
		}
		// rose through the platform, then landed on top of it
		if j.Pos().Y != top-16 {
			t.Errorf("did not end standing on platform: y=%v", j.Pos().Y)
		}
	})
	t.Run("no tunnel at max fall", func(t *testing.T) {
		for _, y0 := range []float64{100, 100.3, 101.1, 102.7} {
			w, j := newWorld(plat)
			j.pos = world.Vec{X: 120, Y: y0}
			j.vel.Y = j.T.MaxFall
			for i := 0; i < 60; i++ {
				w.Step(world.Controls{})
			}
			if b := j.Hitbox().Y + j.Hitbox().H; b != top {
				t.Errorf("y0=%v bottom %v, want %v", y0, b, top)
			}
		}
	})
	t.Run("edge", func(t *testing.T) {
		w, j := newWorld(plat)
		// hitbox x = pos+3; right edge of hitbox just inside platform end (160)
		j.pos = world.Vec{X: 160 - 3 - 1, Y: 100}
		for i := 0; i < 40; i++ {
			w.Step(world.Controls{})
		}
		if b := j.Hitbox().Y + j.Hitbox().H; b != top {
			t.Errorf("did not land on edge, bottom %v", b)
		}
	})
}

func TestWalls(t *testing.T) {
	w, j := newWorld()
	for i := 0; i < 300; i++ {
		w.Step(world.Controls{Left: true})
	}
	if hb := j.Hitbox(); hb.X != 0 {
		t.Errorf("left wall: x=%v", hb.X)
	}
	for i := 0; i < 300; i++ {
		w.Step(world.Controls{Right: true})
	}
	if hb := j.Hitbox(); hb.X+hb.W != world.FieldW {
		t.Errorf("right wall: x=%v", hb.X+hb.W)
	}
	// ceiling
	for i := 0; i < 100; i++ {
		w.Step(world.Controls{Jump: true, JumpPressed: i == 0})
	}
	if j.Hitbox().Y < 0 {
		t.Errorf("ceiling breached")
	}
}

func TestDeath(t *testing.T) {
	w, j := newWorld()
	w.Step(world.Controls{})
	j.Kill(w)
	if j.Alive() || j.Anim() != "jack_die" {
		t.Fatal("not dying")
	}
	died := 0
	for i := 0; i < 200; i++ {
		w.Step(world.Controls{Left: true})
		died += count(w, world.EvPlayerDied)
		if i < DeathFreeze+DeathSpin-2 && j.Done() {
			t.Fatalf("done too early at %d", i)
		}
	}
	if died != 1 || !j.Done() {
		t.Errorf("died events=%d done=%v", died, j.Done())
	}
	j.Respawn(world.Vec{X: 10, Y: world.FloorY})
	if !j.Alive() || j.Done() {
		t.Error("respawn failed")
	}
}

// Repeated taps keep Jack aloft much longer than a single float.
func TestFloatRepeatedTaps(t *testing.T) {
	airtime := func(tapEvery int) int {
		w, j := newWorld()
		w.Step(world.Controls{})
		w.Step(world.Controls{Jump: true, JumpPressed: true})
		for i := 1; i < 1000; i++ {
			press := tapEvery > 0 && i >= 20 && (i-20)%tapEvery == 0
			if i == 20 {
				press = true
			}
			w.Step(world.Controls{JumpPressed: press, Jump: press})
			if j.State() == Idle || j.State() == Run {
				return i
			}
		}
		return 1000
	}
	once, tapping := airtime(0), airtime(12)
	if tapping < once*3 {
		t.Fatalf("tapping airtime %d should far exceed single float %d", tapping, once)
	}
	if every := airtime(0); every > 140 {
		t.Fatalf("a single float keeps Jack up too long: %d ticks", every)
	}
}
