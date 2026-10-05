package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/world"
)

// Title is the attract screen.
type Title struct{ t int }

func NewTitle(g *Game) *Title {
	g.Audio.PlayMusic(-1)
	return &Title{}
}

func (t *Title) Update(g *Game, c world.Controls) Scene {
	t.t++
	if t.t > 20 && (c.Start || c.JumpPressed) {
		return NewSession(g)
	}
	return t
}

func (t *Title) Draw(g *Game, screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, HUDH)
	screen.DrawImage(g.Sheet.Background((t.t/300)%4), op)
	logo := g.Sheet.Logo()
	lw := logo.Bounds().Dx()
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(ScreenW-lw*2)/2, 40)
	screen.DrawImage(logo, op)
	g.centerText(screen, fmt.Sprintf("HI %07d", g.HiScore), 4, colYellow)
	if t.t/30%2 == 0 {
		g.centerText(screen, "PRESS ENTER", 130, colWhite)
	}
	g.centerText(screen, "ARROWS MOVE  Z JUMP", 170, colCyan)
	g.centerText(screen, "JUMP IN AIR TO FLOAT", 182, colCyan)
	g.centerText(screen, "M MUTE  P PAUSE", 194, colCyan)
	// parade of sprites
	names := []string{"jack_run", "mummy_walk", "bird_fly", "saucer", "orb", "bomb_lit"}
	for i, n := range names {
		x := float64((t.t/2+i*40)%(ScreenW+32)) - 16
		g.Sheet.Draw(screen, n, t.t, x, 212, false)
	}
}

// GameOver shows the final score, then returns to the title.
type GameOver struct {
	score int
	t     int
	best  bool
}

func NewGameOver(g *Game, score int) *GameOver {
	g.Audio.StopMusic()
	best := score >= g.HiScore
	if score > g.HiScore {
		g.HiScore = score
	}
	g.saveHiScore()
	return &GameOver{score: score, best: best}
}

func (o *GameOver) Update(g *Game, c world.Controls) Scene {
	o.t++
	if o.t > 600 || (o.t > 90 && (c.Start || c.JumpPressed)) {
		return NewTitle(g)
	}
	return o
}

func (o *GameOver) Draw(g *Game, screen *ebiten.Image) {
	screen.Fill(colBlack)
	g.centerText(screen, "GAME OVER", 90, colRed)
	g.centerText(screen, fmt.Sprintf("SCORE %07d", o.score), 116, colWhite)
	if o.best && o.t/20%2 == 0 {
		g.centerText(screen, "NEW HIGH SCORE!", 140, colYellow)
	}
}
