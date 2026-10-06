package game

import (
	"testing"

	"bombjack/internal/world"
)

// step runs one Update and returns the next scene.
func step(g *Game, s Scene, c world.Controls) Scene { return s.Update(g, c) }

func TestMenuNavigationAndStart(t *testing.T) {
	g := &Game{Table: defaultTable(), Difficulty: Normal}
	m := NewMenu(g)
	if m.sel != int(Normal) {
		t.Fatalf("menu should preselect the current difficulty, got %d", m.sel)
	}
	for i := 0; i < 12; i++ { // let the open-guard pass
		step(g, m, world.Controls{})
	}
	step(g, m, world.Controls{Up: true}) // Normal -> Easy
	step(g, m, world.Controls{})
	step(g, m, world.Controls{Up: true}) // Easy -> wraps to Sound & FX
	step(g, m, world.Controls{})
	if m.sel != len(menuItems)-1 {
		t.Fatalf("up should wrap to the last item, sel=%d", m.sel)
	}
	step(g, m, world.Controls{Down: true}) // wraps back to Easy
	step(g, m, world.Controls{})
	step(g, m, world.Controls{Down: true}) // Easy -> Normal
	step(g, m, world.Controls{})
	step(g, m, world.Controls{Down: true}) // Normal -> Hard
	if next := step(g, m, world.Controls{Start: true}); next == Scene(m) {
		t.Fatal("enter did not start a game")
	} else if _, ok := next.(*Play); !ok {
		t.Fatalf("expected a game, got %T", next)
	}
	if g.Difficulty != Hard {
		t.Fatalf("difficulty %d, want Hard", g.Difficulty)
	}
}

func TestMenuSoundTestAndTimeout(t *testing.T) {
	g := &Game{Table: defaultTable()}
	m := NewMenu(g)
	m.sel = len(menuItems) - 1
	m.t = 20
	if _, ok := step(g, m, world.Controls{Start: true}).(*SoundTest); !ok {
		t.Fatal("Sound & FX should open the sound test")
	}
	m = NewMenu(g)
	var s Scene = m
	for i := 0; i < menuTimeout-1; i++ {
		if i == menuTimeout/2 { // input resets the idle timer
			s = step(g, s, world.Controls{Down: true})
			continue
		}
		s = step(g, s, world.Controls{})
	}
	if _, ok := s.(*Menu); !ok {
		t.Fatalf("menu timed out too early: %T", s)
	}
	for i := 0; i < menuTimeout; i++ {
		s = step(g, s, world.Controls{})
		if _, ok := s.(*Title); ok {
			break // the title needs graphics; stop once we're back there
		}
	}
	if _, ok := s.(*Title); !ok {
		t.Fatalf("idle menu should return to the title, got %T", s)
	}
}
