// Package game wires everything together: scenes, HUD, FX.
package game

import (
	"encoding/json"
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bombjack/internal/audio"
	"bombjack/internal/gfx"
	"bombjack/internal/input"
	"bombjack/internal/world"
)

const (
	ScreenW = world.FieldW
	ScreenH = world.FieldH + HUDH
	HUDH    = 16 // HUD strip above the playfield
)

// ErrQuit ends the game loop cleanly.
var ErrQuit = errors.New("quit")

var (
	colWhite  = color.RGBA{0xf8, 0xf8, 0xf8, 0xff}
	colYellow = color.RGBA{0xf8, 0xd8, 0x30, 0xff}
	colCyan   = color.RGBA{0x58, 0xe0, 0xf0, 0xff}
	colRed    = color.RGBA{0xf8, 0x58, 0x58, 0xff}
	colPink   = color.RGBA{0xf8, 0x78, 0xb8, 0xff}
)

// Scene is one screen of the game. Update returns the next scene (or itself).
type Scene interface {
	Update(g *Game, c world.Controls) Scene
	Draw(g *Game, screen *ebiten.Image)
}

// Game implements ebiten.Game.
type Game struct {
	Sheet      *gfx.Sheet
	Audio      *audio.Player
	HiScore    int
	Table      []HiEntry
	Difficulty Difficulty // chosen in the menu (not yet used by the rules)
	Tick       int
	scene      Scene
	auto       *Autoshot
	autoErr    error
}

func New() *Game {
	g := &Game{Sheet: gfx.Load(), Audio: audio.New(), Table: loadTable(), Difficulty: Normal}
	g.HiScore = g.Table[0].Score
	g.scene = NewTitle(g)
	g.auto = autoshotFromEnv(g)
	return g
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch g.scene.(type) {
		case *Title:
			return ErrQuit
		case *Play: // Play handles Esc itself (pause, then quit to title)
		default:
			g.scene = NewTitle(g)
			return nil
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.Audio.ToggleMute()
	}
	if g.autoErr != nil {
		return g.autoErr
	}
	g.Tick++
	g.Audio.Update()
	c := input.Read()
	if g.auto != nil {
		c = g.auto.controls(g.Tick)
		g.auto.showcase(g)
	}
	g.scene = g.scene.Update(g, c)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.scene.Draw(g, screen)
	if g.auto != nil && g.autoErr == nil {
		g.autoErr = g.auto.capture(g, screen)
	}
}

func (g *Game) Layout(int, int) (int, int) { return ScreenW, ScreenH }

// panel darkens a horizontal band so text stays readable over backdrops.
func panel(dst *ebiten.Image, x, y, w, h float32) {
	vector.FillRect(dst, x, y, w, h, color.RGBA{0x08, 0x06, 0x10, 0xb0}, false)
	vector.FillRect(dst, x, y, w, 1, color.RGBA{0xf8, 0xd8, 0x30, 0xff}, false)
	vector.FillRect(dst, x, y+h-1, w, 1, color.RGBA{0xf8, 0xd8, 0x30, 0xff}, false)
}

// centerText draws s horizontally centred at y.
func (g *Game) centerText(dst *ebiten.Image, s string, y float64, col color.Color) {
	g.Sheet.Text(dst, s, float64(ScreenW-g.Sheet.TextWidth(s))/2, y, col)
}

// --------------------------------------------------------------- hiscore --

// HiEntry is one row of the high-score table.
type HiEntry struct {
	Name  string
	Score int
}

const hiTableSize = 5

func defaultTable() []HiEntry {
	return []HiEntry{{"JCK", 50000}, {"TEH", 40000}, {"KAN", 30000}, {"BMB", 20000}, {"GO!", 10000}}
}

// hiScoreFile is a variable so tests can redirect it.
var hiScoreFile = hiScorePath

func hiScorePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "bombjack", "hiscores.json")
}

func loadTable() []HiEntry {
	var t []HiEntry
	if b, err := os.ReadFile(hiScoreFile()); err == nil {
		_ = json.Unmarshal(b, &t)
	}
	if len(t) == 0 {
		return defaultTable()
	}
	return t
}

func (g *Game) saveTable() {
	p := hiScoreFile()
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, _ := json.Marshal(g.Table)
	_ = os.WriteFile(p, b, 0o644)
}

// Qualifies reports whether score earns a place in the table.
func (g *Game) Qualifies(score int) bool {
	return score > 0 && (len(g.Table) < hiTableSize || score > g.Table[len(g.Table)-1].Score)
}

// Insert adds an entry, keeps the table sorted and trimmed, and saves it.
func (g *Game) Insert(name string, score int) {
	g.Table = append(g.Table, HiEntry{name, score})
	sort.SliceStable(g.Table, func(i, j int) bool { return g.Table[i].Score > g.Table[j].Score })
	g.Table = g.Table[:min(len(g.Table), hiTableSize)]
	g.HiScore = g.Table[0].Score
	g.saveTable()
}
