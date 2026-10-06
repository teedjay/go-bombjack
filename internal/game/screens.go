package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/audio"
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
	// coming from the high-score screens the tune fades out first
	g.Audio.FadeTo(audio.TrackTitle, 90)
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

// The title panel cycles through four pages: controls, high scores,
// enemies and bonus items.
const (
	titlePageTicks = 360 // 6 seconds per page
	titlePanelX    = 24
)

// drawEnemyPage introduces the four enemies, two per row.
func drawEnemyPage(g *Game, screen *ebiten.Image, tick int) {
	g.centerText(screen, "MEET THE ENEMIES", 121, colYellow)
	enemies := []struct{ anim, name, hint string }{
		{"mummy_walk", "MUMMY", "MORPHS"},
		{"bird_fly", "BIRD", "HOMES IN"},
		{"saucer", "SAUCER", "BOUNCES"},
		{"orb", "ORB", "DRIFTS"},
	}
	for i, e := range enemies {
		x := float64(titlePanelX + 12 + (i%2)*100)
		y := float64(138 + (i/2)*30)
		g.Sheet.Draw(screen, e.anim, tick, x, y, false)
		g.Sheet.Text(screen, e.name, x+22, y, colWhite)
		g.Sheet.Text(screen, e.hint, x+22, y+9, colCyan)
	}
}

// drawPickupPage shows every collectable with its value, in a 3x3 grid.
func drawPickupPage(g *Game, screen *ebiten.Image, tick int) {
	g.centerText(screen, "BONUS ITEMS", 121, colYellow)
	items := []struct {
		anim, label string
		dy          float64 // vertical nudge for sprites shorter than 16px
	}{
		{"bomb", "100", 0},
		{"bomb_lit", "200", 0},
		{"coin", "100+", 0},
		{"power_p", "COINS", 0},
		{"power_b", "X+1", 0},
		{"power_e", "1UP", 0},
		{"power_s", "100K", 0},
		{"power_m", "+3", 0},
		{"missile", "500", 4},
	}
	for i, it := range items {
		x := float64(titlePanelX + 8 + (i%3)*68)
		y := float64(134 + (i/3)*22)
		g.Sheet.Draw(screen, it.anim, tick, x, y+it.dy, false)
		g.Sheet.Text(screen, it.label, x+19, y+4, colWhite)
	}
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
	panel(screen, titlePanelX, 114, ScreenW-2*titlePanelX, 90)
	switch t.t / titlePageTicks % 4 {
	case 0:
		g.centerText(screen, "ARROWS MOVE  Z JUMP", 137, colCyan)
		g.centerText(screen, "TAP Z IN AIR TO FLY", 149, colCyan)
		g.centerText(screen, "X FIRE HOMING MISSILE", 161, colCyan)
		g.centerText(screen, "M MUTE  P PAUSE", 173, colCyan)
	case 1:
		drawTable(g, screen, 121, "", -1)
	case 2:
		drawEnemyPage(g, screen, t.t)
	case 3:
		drawPickupPage(g, screen, t.t)
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
			// the SID tune plays through initials entry and the table
			g.Audio.PlayMusic(audio.TrackHiScore)
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

// nameEntryTicks is the time limit for entering initials; when it runs out
// the letters currently on screen are saved.
const nameEntryTicks = 30 * 60

const nameChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!. "

// NameEntry lets the player enter 3 initials: the arrows only change the
// current letter (right/up next, left/down previous), Z confirms it and
// moves on (ending the entry after the last), X goes back a letter. After
// 30 seconds the letters on screen are saved as they are.
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
	switch {
	case c.Left || c.Down:
		dir = -1
	case c.Right || c.Up:
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
	if n.t >= nameEntryTicks { // time's up: keep what's on screen
		g.Insert(string(n.name), n.score)
		n.done = 1
		return n
	}
	if c.Fire && n.pos > 0 { // X: back to the previous letter
		n.pos--
		return n
	}
	if n.t > 10 && c.Confirm {
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
	left := (nameEntryTicks - n.t + 59) / 60
	if left > 5 || n.t/10%2 == 0 { // blink during the last 5 seconds
		col := color.Color(colWhite)
		if left <= 5 {
			col = colRed
		}
		g.centerText(screen, fmt.Sprintf("TIME %02d", left), 150, col)
	}
	g.centerText(screen, "ARROWS=LETTER", 170, colCyan)
	g.centerText(screen, "Z=OK  X=BACK", 182, colCyan)
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
		g.centerText(screen, fmt.Sprintf("%d  %-3s  %07d", i+1, e.Name, e.Score), y+17+float64(i*11), col)
	}
}
