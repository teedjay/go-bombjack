package game

import (
	"path/filepath"
	"testing"

	"bombjack/internal/world"
)

func TestHiScoreTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hs.json")
	hiScoreFile = func() string { return path }
	defer func() { hiScoreFile = hiScorePath }()

	g := &Game{Table: loadTable()}
	if len(g.Table) != hiTableSize {
		t.Fatalf("default table size %d", len(g.Table))
	}
	if g.Qualifies(5000) {
		t.Fatal("5000 should not beat the default table")
	}
	if !g.Qualifies(35000) {
		t.Fatal("35000 should qualify")
	}
	g.Insert("ABC", 35000)
	if len(g.Table) != hiTableSize || g.Table[2].Name != "ABC" {
		t.Fatalf("insert placed wrongly: %+v", g.Table)
	}
	g.Insert("TOP", 99999)
	if g.HiScore != 99999 || g.Table[0].Name != "TOP" {
		t.Fatalf("top entry wrong: %+v", g.Table)
	}
	if got := loadTable(); got[0].Name != "TOP" || len(got) != hiTableSize {
		t.Fatalf("not persisted: %+v", got)
	}
}

func TestNameEntryBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hs.json")
	hiScoreFile = func() string { return path }
	defer func() { hiScoreFile = hiScorePath }()

	g := &Game{Table: loadTable()}
	n := &NameEntry{score: 99999, name: []byte("AAA"), t: 20}
	n.Update(g, world.Controls{Confirm: true}) // confirm letter 1
	n.Update(g, world.Controls{Fire: true})    // back to letter 1
	if n.pos != 0 {
		t.Fatalf("pos %d after X, want 0", n.pos)
	}
	n.Update(g, world.Controls{Fire: true}) // already at the first letter
	if n.pos != 0 {
		t.Fatalf("pos %d, X must not go before the first letter", n.pos)
	}
	n.Update(g, world.Controls{Right: true}) // A -> B
	n.Update(g, world.Controls{})
	for i := 0; i < 3; i++ {
		n.Update(g, world.Controls{Confirm: true})
	}
	if g.Table[0].Name != "BAA" {
		t.Fatalf("saved %q, want BAA", g.Table[0].Name)
	}
}

// Arrows (including up, which is also a jump key) and Enter never move the
// cursor; only Confirm (Z) does.
func TestNameEntryOnlyZAdvances(t *testing.T) {
	g := &Game{Table: defaultTable()}
	n := &NameEntry{score: 99999, name: []byte("AAA"), t: 20}
	for _, c := range []world.Controls{
		{Up: true, Jump: true, JumpPressed: true}, // up arrow
		{},
		{Down: true},
		{},
		{Start: true}, // enter
		{Right: true},
		{},
		{Left: true},
	} {
		n.Update(g, c)
		if n.pos != 0 {
			t.Fatalf("%+v moved the cursor to %d", c, n.pos)
		}
	}
	// up A->B, down B->A, right A->B, left B->A: the letter cycled back
	if string(n.name) != "AAA" {
		t.Fatalf("name %q, want AAA", n.name)
	}
	n.Update(g, world.Controls{Confirm: true})
	if n.pos != 1 {
		t.Fatalf("Z did not advance: pos %d", n.pos)
	}
}

func TestNameEntryTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hs.json")
	hiScoreFile = func() string { return path }
	defer func() { hiScoreFile = hiScorePath }()

	g := &Game{Table: loadTable()}
	n := &NameEntry{score: 99999, name: []byte("AAA")}
	n.Update(g, world.Controls{Right: true}) // first letter A -> B, then idle
	for i := 0; i < nameEntryTicks-2; i++ {
		n.Update(g, world.Controls{})
	}
	if n.done != 0 {
		t.Fatal("entry ended before the time limit")
	}
	n.Update(g, world.Controls{})
	if n.done == 0 || g.Table[0].Name != "BAA" || g.Table[0].Score != 99999 {
		t.Fatalf("timeout should save what is on screen: done=%d table=%+v", n.done, g.Table[0])
	}
}
