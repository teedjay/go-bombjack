// Package game wires everything together: scenes, HUD, FX.
package game

import (
	"encoding/json"
	"errors"
	"image/color"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

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
)

// Scene is one screen of the game. Update returns the next scene (or itself).
type Scene interface {
	Update(g *Game, c world.Controls) Scene
	Draw(g *Game, screen *ebiten.Image)
}

// Game implements ebiten.Game.
type Game struct {
	Sheet   *gfx.Sheet
	Audio   *audio.Player
	HiScore int
	Tick    int
	scene   Scene
	auto    *Autoshot
	autoErr error
}

func New() *Game {
	g := &Game{Sheet: gfx.Load(), Audio: audio.New(), HiScore: loadHiScore()}
	g.scene = NewTitle(g)
	g.auto = autoshotFromEnv(g)
	return g
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ErrQuit
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.Audio.ToggleMute()
	}
	if g.autoErr != nil {
		return g.autoErr
	}
	g.Tick++
	c := input.Read()
	if g.auto != nil {
		c = g.auto.controls(g.Tick)
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

// centerText draws s horizontally centred at y.
func (g *Game) centerText(dst *ebiten.Image, s string, y float64, col color.Color) {
	g.Sheet.Text(dst, s, float64(ScreenW-g.Sheet.TextWidth(s))/2, y, col)
}

// --------------------------------------------------------------- hiscore --

func hiScorePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "bombjack", "hiscore.json")
}

func loadHiScore() int {
	var v struct{ HiScore int }
	if b, err := os.ReadFile(hiScorePath()); err == nil {
		_ = json.Unmarshal(b, &v)
	}
	return max(v.HiScore, 10000)
}

func (g *Game) saveHiScore() {
	p := hiScorePath()
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, _ := json.Marshal(struct{ HiScore int }{g.HiScore})
	_ = os.WriteFile(p, b, 0o644)
}
