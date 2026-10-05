package game

import (
	"path/filepath"
	"testing"
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
