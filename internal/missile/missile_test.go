package missile

import (
	"math"
	"testing"

	"bombjack/internal/world"
)

// dummy is a test enemy; move (optional) updates its position per tick.
type dummy struct {
	pos     world.Vec
	removed bool
	move    func(tick int) world.Vec
	tick    int
}

func (d *dummy) Update(*world.World) {
	d.tick++
	if d.move != nil {
		d.pos = d.move(d.tick)
	}
}
func (d *dummy) Hitbox() world.Rect { return world.Inset(d.pos, 12, 12) }
func (d *dummy) Pos() world.Vec     { return d.pos }
func (d *dummy) Anim() string       { return "orb" }
func (d *dummy) FacingLeft() bool   { return false }
func (d *dummy) Harmful() bool      { return true }
func (d *dummy) Eat(*world.World)   { d.removed = true }
func (d *dummy) Removed() bool      { return d.removed }

func setup(enemies ...*dummy) (*world.World, *Manager) {
	w := world.New(1)
	m := &Manager{}
	w.Systems = []world.System{m}
	for _, e := range enemies {
		w.Enemies = append(w.Enemies, e)
	}
	return w, m
}

// fly steps the world until all missiles are done; it returns the ticks
// taken and the hit events seen.
func fly(t *testing.T, w *world.World, m *Manager, limit int, each func(*Missile)) (int, []world.Event) {
	t.Helper()
	var hits []world.Event
	for i := 1; i <= limit; i++ {
		w.Step(world.Controls{})
		for _, ev := range w.Events {
			if ev.Kind == world.EvMissileHit {
				hits = append(hits, ev)
			}
		}
		for _, ms := range m.Missiles {
			if math.IsNaN(ms.Pos.X) || math.IsNaN(ms.Pos.Y) || math.IsNaN(ms.Angle) {
				t.Fatalf("NaN at tick %d", i)
			}
			if each != nil {
				each(ms)
			}
		}
		if len(m.Missiles) == 0 {
			return i, hits
		}
	}
	t.Fatalf("missiles still flying after %d ticks", limit)
	return 0, nil
}

func TestFireNeedsTarget(t *testing.T) {
	w, m := setup()
	if m.Fire(w, world.Vec{X: 120, Y: 200}, false) {
		t.Fatal("fired with no enemies")
	}
}

func TestHitsStationaryTargetOnACurve(t *testing.T) {
	e := &dummy{pos: world.Vec{X: 200, Y: 40}}
	w, m := setup(e)
	start := world.Vec{X: 40, Y: 200}
	if !m.Fire(w, start, false) {
		t.Fatal("did not fire")
	}
	// track deviation from the straight line start->target, angle steps
	// and speed
	goal := world.Vec{X: 208, Y: 48}
	maxDev, maxTurn := 0.0, 0.0
	prevAngle, prevSpeed := m.Missiles[0].Angle, 0.0
	accelerated := false
	_, hits := fly(t, w, m, 400, func(ms *Missile) {
		dx, dy := goal.X-start.X, goal.Y-start.Y
		dev := math.Abs((ms.Pos.X-start.X)*dy-(ms.Pos.Y-start.Y)*dx) / math.Hypot(dx, dy)
		maxDev = math.Max(maxDev, dev)
		maxTurn = math.Max(maxTurn, math.Abs(wrap(ms.Angle-prevAngle)))
		prevAngle = ms.Angle
		if ms.Speed > prevSpeed+1e-9 && prevSpeed > 0 {
			accelerated = true
		}
		prevSpeed = ms.Speed
	})
	if len(hits) != 1 || hits[0].Index != 1 || !e.removed {
		t.Fatalf("hits %+v removed %v", hits, e.removed)
	}
	if maxDev < 20 {
		t.Errorf("path too straight: max deviation %.1fpx", maxDev)
	}
	if maxTurn > 0.6 {
		t.Errorf("rotation not smooth: max %.2f rad in one tick", maxTurn)
	}
	if !accelerated {
		t.Error("missile never accelerated")
	}
}

func TestRetargetsWhenTargetLost(t *testing.T) {
	a := &dummy{pos: world.Vec{X: 60, Y: 60}}
	b := &dummy{pos: world.Vec{X: 220, Y: 180}}
	w, m := setup(a, b)
	m.Fire(w, world.Vec{X: 70, Y: 200}, false) // a is closer
	for i := 0; i < 10; i++ {
		w.Step(world.Controls{})
	}
	a.removed = true // eaten by something else mid-flight
	_, hits := fly(t, w, m, 400, nil)
	if len(hits) != 1 || !b.removed {
		t.Fatalf("expected the missile to hit b; hits=%+v", hits)
	}
}

func TestChasesMovingTarget(t *testing.T) {
	e := &dummy{move: func(tick int) world.Vec {
		a := float64(tick) * 0.03
		return world.Vec{X: 120 + 70*math.Cos(a), Y: 80 + 40*math.Sin(a)}
	}}
	e.pos = e.move(0)
	w, m := setup(e)
	m.Fire(w, world.Vec{X: 20, Y: 200}, false)
	ticks, hits := fly(t, w, m, 400, nil)
	if len(hits) != 1 || !e.removed {
		t.Fatalf("missed a moving target (ticks %d, hits %+v)", ticks, hits)
	}
}

func TestSalvoSpreadsAcrossEnemies(t *testing.T) {
	a := &dummy{pos: world.Vec{X: 40, Y: 40}}
	b := &dummy{pos: world.Vec{X: 200, Y: 40}}
	w, m := setup(a, b)
	m.Fire(w, world.Vec{X: 60, Y: 200}, false)
	m.Fire(w, world.Vec{X: 60, Y: 200}, false)
	if m.Missiles[0].target == m.Missiles[1].target {
		t.Fatal("both missiles chase the same enemy")
	}
	fly(t, w, m, 400, nil)
	if !a.removed || !b.removed {
		t.Fatalf("salvo should destroy both: a=%v b=%v", a.removed, b.removed)
	}
}
