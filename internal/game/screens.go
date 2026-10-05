package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/gfx"
	"bombjack/internal/world"
)

// Title is the attract screen.
type Title struct {
	t  int
	fx *gfx.FX
}

const (
	logoScale = 2
	logoY     = 40
)

func NewTitle(g *Game) *Title {
	g.Audio.PlayMusic(-1)
	return &Title{fx: gfx.NewFX(g.Sheet)}
}

// logoOrigin is the screen position of the scaled logo's top-left.
func logoOrigin(g *Game) (float64, float64) {
	return float64(ScreenW-g.Sheet.Logo().Bounds().Dx()*logoScale) / 2, logoY
}

func (t *Title) Update(g *Game, c world.Controls) Scene {
	t.t++
	t.fx.Update()
	// sparkling stars on the logo outline, plus a little falling star dust
	if edge := g.Sheet.LogoEdge(); len(edge) > 0 && t.t%5 == 0 {
		lx, ly := logoOrigin(g)
		pt := edge[rand.IntN(len(edge))]
		x, y := lx+float64(pt.X*logoScale), ly+float64(pt.Y*logoScale)
		t.fx.Spawn("twinkle", x-4, y-4)
		if t.t%15 == 0 {
			t.fx.Burst(x, y, gfx.BurstSpec{N: 4, Colors: []color.RGBA{colYellow, colWhite},
				Fade: colOrange, Speed: 0.5, Gravity: 0.03, Life: 50})
		}
	}
	if t.t > 20 && (c.Start || c.JumpPressed) {
		return NewSession(g)
	}
	return t
}

var titleCycle = []color.Color{colYellow, colOrange, colRed, colPink, colPurple, colCyan, colWhite}

func (t *Title) Draw(g *Game, screen *ebiten.Image) {
	screen.Fill(colBlack)
	level := (t.t / 300) % 4
	g.Sheet.DrawBackground(screen, level, HUDH, math.Sin(float64(t.t)/120), t.t*3)

	logo := g.Sheet.Logo()
	lx, ly := logoOrigin(g)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(logoScale, logoScale)
	op.GeoM.Translate(lx, ly)
	screen.DrawImage(logo, op)
	// shine: a bright band sweeping across the logo every 4 seconds
	lw, lh := logo.Bounds().Dx(), logo.Bounds().Dy()
	if bx := t.t%240*2 - 20; bx > -6 && bx < lw {
		x0, x1 := max(bx, 0), min(bx+6, lw)
		band := logo.SubImage(image.Rect(x0, 0, x1, lh)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
		op.ColorScale.ScaleAlpha(0.7)
		op.GeoM.Scale(logoScale, logoScale)
		op.GeoM.Translate(lx+float64(x0*logoScale), ly)
		screen.DrawImage(band, op)
	}
	t.fx.Draw(screen, 0)

	g.centerText(screen, fmt.Sprintf("HI %07d", g.HiScore), 4, colYellow)
	g.centerText(screen, "PRESS ENTER", 100, titleCycle[t.t/6%len(titleCycle)])
	panel(screen, 36, 114, ScreenW-72, 84)
	if t.t/360%2 == 0 {
		g.centerText(screen, "ARROWS MOVE  Z JUMP", 140, colCyan)
		g.centerText(screen, "JUMP IN AIR TO FLOAT", 152, colCyan)
		g.centerText(screen, "M MUTE  P PAUSE", 164, colCyan)
	} else {
		drawTable(g, screen, 122, "", -1)
	}
	// parade of sprites
	names := []string{"jack_run", "mummy_walk", "bird_fly", "saucer", "orb", "bomb_lit"}
	for i, n := range names {
		x := float64((t.t/2+i*40)%(ScreenW+32)) - 16
		g.Sheet.Draw(screen, n, t.t, x, 212, false)
	}
}

// GameOver shows the final score, then name entry (if it qualifies) or the
// title screen.
type GameOver struct {
	score int
	t     int
}

func NewGameOver(g *Game, score int) *GameOver {
	g.Audio.StopMusic()
	return &GameOver{score: score}
}

func (o *GameOver) Update(g *Game, c world.Controls) Scene {
	o.t++
	if o.t > 240 || (o.t > 60 && (c.Start || c.JumpPressed)) {
		if g.Qualifies(o.score) {
			return &NameEntry{score: o.score, name: []byte("AAA")}
		}
		return NewTitle(g)
	}
	return o
}

func (o *GameOver) Draw(g *Game, screen *ebiten.Image) {
	screen.Fill(colBlack)
	g.centerText(screen, "GAME OVER", 90, colRed)
	g.centerText(screen, fmt.Sprintf("SCORE %07d", o.score), 116, colWhite)
}

const nameChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!. "

// NameEntry lets the player enter 3 initials: left/right cycles the letter,
// jump or enter confirms it.
type NameEntry struct {
	score  int
	name   []byte
	pos    int
	t      int
	repeat int
	done   int // ticks since the last letter was confirmed (0 = still entering)
}

func (n *NameEntry) Update(g *Game, c world.Controls) Scene {
	n.t++
	if n.done > 0 {
		n.done++
		if n.done > 150 || (n.done > 30 && (c.Start || c.JumpPressed)) {
			return NewTitle(g)
		}
		return n
	}
	dir := 0
	if c.Left {
		dir = -1
	} else if c.Right {
		dir = 1
	}
	if dir == 0 {
		n.repeat = 0
	} else {
		// step on press, then auto-repeat after a short delay
		if n.repeat == 0 || (n.repeat > 18 && n.repeat%5 == 0) {
			i := strings.IndexByte(nameChars, n.name[n.pos])
			i = (i + dir + len(nameChars)) % len(nameChars)
			n.name[n.pos] = nameChars[i]
		}
		n.repeat++
	}
	if n.t > 10 && (c.JumpPressed || c.Start) {
		n.pos++
		if n.pos == len(n.name) {
			g.Insert(string(n.name), n.score)
			n.done = 1
		}
	}
	return n
}

func (n *NameEntry) Draw(g *Game, screen *ebiten.Image) {
	screen.Fill(colBlack)
	if n.done > 0 {
		drawTable(g, screen, 70, string(n.name), n.score)
		return
	}
	g.centerText(screen, "NEW HIGH SCORE!", 60, colYellow)
	g.centerText(screen, fmt.Sprintf("%07d", n.score), 76, colWhite)
	g.centerText(screen, "ENTER YOUR INITIALS", 104, colCyan)
	x := float64(ScreenW-3*16) / 2
	for i, ch := range n.name {
		col := color.Color(colWhite)
		if i == n.pos && n.t/8%2 == 0 {
			col = colYellow
		}
		g.Sheet.Text(screen, string(ch), x+float64(i*16)+4, 124, col)
		if i == n.pos {
			g.Sheet.Text(screen, "-", x+float64(i*16)+4, 134, colYellow)
		}
	}
	g.centerText(screen, "LEFT/RIGHT  JUMP=OK", 170, colCyan)
}

// drawTable renders the high-score table; the row matching (name, score)
// is highlighted.
func drawTable(g *Game, screen *ebiten.Image, y float64, name string, score int) {
	g.centerText(screen, "BEST BOMBERS", y, colYellow)
	for i, e := range g.Table {
		col := color.Color(colWhite)
		if e.Name == name && e.Score == score {
			col = colYellow
		}
		g.centerText(screen, fmt.Sprintf("%d  %-3s  %07d", i+1, e.Name, e.Score), y+20+float64(i*12), col)
	}
}
