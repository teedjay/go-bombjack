package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"bombjack/internal/enemy"
	"bombjack/internal/gfx"
	"bombjack/internal/level"
	"bombjack/internal/player"
	"bombjack/internal/rules"
	"bombjack/internal/world"
)

var (
	colBlack     = color.RGBA{0x08, 0x06, 0x10, 0xff}
	colOrange    = color.RGBA{0xf8, 0x88, 0x20, 0xff}
	colRedDark   = color.RGBA{0x88, 0x18, 0x28, 0xff}
	colBlue      = color.RGBA{0x28, 0x50, 0xd8, 0xff}
	colGold      = color.RGBA{0xf8, 0xc0, 0x20, 0xff}
	colSmoke     = color.RGBA{0x88, 0x90, 0xa8, 0xff}
	colPurple    = color.RGBA{0xc0, 0x48, 0xd8, 0xff}
	colMagicHi   = color.RGBA{0xe8, 0xb0, 0xff, 0xff} // tint for the poof's upper cloud
	colGreen     = color.RGBA{0x70, 0xd8, 0x50, 0xff}
	colGreenDark = color.RGBA{0x28, 0x78, 0x38, 0xff}
	colBandage   = color.RGBA{0xe8, 0xdc, 0xb0, 0xff}
	colBandShade = color.RGBA{0xa8, 0x98, 0x68, 0xff}
)

type phase int

const (
	phaseIntro phase = iota // "ROUND N" banner, world frozen
	phasePlay
	phaseClear // round clear banner + bonus
)

const (
	introTicks = 100
	clearTicks = 200
	landTicks  = 6
	turnTicks  = 5
)

// Play runs the game session: one Play per round, rules carried across.
type Play struct {
	World  *world.World
	Rules  *rules.State
	Jack   *player.Jack
	Def    level.Def
	fx     *gfx.FX
	phase  phase
	t      int // ticks in current phase
	paused bool

	landT, turnT int
	lastLeft     bool
	parallax     float64 // smoothed -1..1 from Jack's x position
}

// NewSession starts a new game at round 0.
func NewSession(g *Game) *Play {
	d := level.Levels[0]
	return newRound(g, rules.NewForLevel(d.LitOrder, 0))
}

// NewPlay builds round `round` with fresh rules (used by tests).
func NewPlay(round int) *Play {
	d := level.Levels[round%len(level.Levels)]
	return newRound(nil, rules.NewForLevel(d.LitOrder, round))
}

func newRound(g *Game, r *rules.State) *Play {
	d := level.Levels[r.Round%len(level.Levels)]
	w := level.Build(r.Round, uint64(r.Round)*7919+1)
	j := player.New(d.PlayerStart)
	w.Player = j
	w.Systems = append(w.Systems,
		enemy.NewSpawner(rules.Scale(d.Enemies, r.Loop())),
		rules.PickupMover{})
	p := &Play{World: w, Rules: r, Jack: j, Def: d}
	if g != nil {
		r.HiScore = g.HiScore
		p.fx = gfx.NewFX(g.Sheet)
		g.Audio.PlayMusic(w.Level)
	} else {
		p.phase = phasePlay
	}
	return p
}

func (p *Play) Update(g *Game, c world.Controls) Scene {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if p.paused && g != nil {
			g.Audio.StopMusic()
			return NewTitle(g) // Esc twice: abandon the game
		}
		p.paused = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		p.paused = !p.paused
	}
	if p.paused {
		return p
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		p.World.Invincible = !p.World.Invincible
	}
	target := (p.Jack.Pos().X + 8 - world.FieldW/2) / (world.FieldW / 2)
	p.parallax += (target - p.parallax) * 0.04
	p.t++
	if p.fx != nil {
		p.fx.Update()
	}
	switch p.phase {
	case phaseIntro:
		if p.t >= introTicks {
			p.phase, p.t = phasePlay, 0
		}
		return p
	case phaseClear:
		if p.t >= clearTicks {
			p.Rules.NextRound(level.Levels[(p.Rules.Round+1)%len(level.Levels)].LitOrder)
			return newRound(g, p.Rules)
		}
		return p
	}

	w := p.World
	if p.Jack.Alive() {
		w.Step(c)
	} else {
		// everything freezes while Jack's death plays out
		w.Events = w.Events[:0]
		w.Tick++
		p.Jack.Update(w, world.Controls{})
	}
	p.Rules.Apply(w)
	if g != nil {
		g.Audio.Handle(w.Events)
		if p.Rules.HiScore > g.HiScore {
			g.HiScore = p.Rules.HiScore
		}
	}
	p.handleEvents()

	if p.Jack.FacingLeft() != p.lastLeft {
		p.turnT, p.lastLeft = turnTicks, p.Jack.FacingLeft()
	}
	p.landT = max(p.landT-1, 0)
	p.turnT = max(p.turnT-1, 0)

	for _, e := range w.Events {
		switch e.Kind {
		case world.EvRoundClear:
			p.phase, p.t = phaseClear, 0
		case world.EvPlayerDied:
			if p.Rules.Lives <= 0 {
				if g == nil {
					return p
				}
				return NewGameOver(g, p.Rules.Score)
			}
			p.respawn()
		}
	}
	return p
}

// respawn puts Jack back at the start and clears the screen of enemies.
func (p *Play) respawn() {
	w := p.World
	p.Jack.Respawn(p.Def.PlayerStart)
	w.Enemies = w.Enemies[:0]
	w.FrightTicks = 0
	p.phase, p.t = phaseIntro, introTicks/2
}

func (p *Play) handleEvents() {
	for _, e := range p.World.Events {
		switch e.Kind {
		case world.EvLand:
			p.landT = landTicks
		}
		if p.fx == nil {
			continue
		}
		cx, cy := e.Pos.X+8, e.Pos.Y+8
		switch e.Kind {
		case world.EvBombTaken:
			p.fx.Spawn("sparkle", e.Pos.X, e.Pos.Y)
			spec := gfx.BurstSpec{N: 10, Colors: []color.RGBA{colYellow, colOrange, colWhite},
				Fade: colRedDark, Speed: 1.6, Gravity: 0.05, Drag: 0.95, Life: 28}
			if e.Lit {
				spec.N, spec.Speed, spec.Big = 22, 2.4, 0.3
			}
			p.fx.Burst(cx, cy, spec)
		case world.EvPickupTaken:
			p.fx.Spawn("sparkle", e.Pos.X, e.Pos.Y)
			p.fx.Burst(cx, cy, gfx.BurstSpec{N: 24, Colors: []color.RGBA{colCyan, colWhite, colYellow},
				Fade: colBlue, Speed: 2.2, Drag: 0.94, Life: 40, Big: 0.25})
		case world.EvCoinEaten:
			p.fx.Burst(cx, cy, gfx.BurstSpec{N: 14, Colors: []color.RGBA{colYellow, colGold},
				Fade: colOrange, Speed: 1.8, Gravity: 0.08, Drag: 0.96, Life: 30, UpBias: 1})
		case world.EvEnemyTransformed:
			p.poof(cx, cy)
		case world.EvEnemySpawned:
			p.fx.Burst(cx, cy, gfx.BurstSpec{N: 10, Colors: []color.RGBA{colSmoke, colPurple},
				Fade: colBlack, Speed: 0.7, Gravity: -0.03, Drag: 0.95, Life: 40, Big: 0.6})
		case world.EvPlayerHit:
			p.fx.Spawn("explosion", e.Pos.X, e.Pos.Y)
			p.fx.Burst(cx, cy, gfx.BurstSpec{N: 30, Colors: []color.RGBA{colRed, colOrange, colWhite},
				Fade: colRedDark, Speed: 2.6, Gravity: 0.06, Drag: 0.95, Life: 45, Big: 0.3})
		case world.EvScore:
			p.fx.Popup(e.Value, e.Pos.X, e.Pos.Y)
		}
	}
}

// poof is the mummy transformation: a magic-coloured cloud (a palette-
// tinted explosion), a sparkling shockwave ring, bandage scraps bouncing on
// the floor and purple smoke curling upwards.
func (p *Play) poof(cx, cy float64) {
	floor := float64(world.FieldH - 1)
	p.fx.Spawn("poof", cx-8, cy-8)
	p.fx.SpawnTinted("explosion", cx-8, cy-20, colMagicHi)
	p.fx.Burst(cx, cy-4, gfx.BurstSpec{N: 24, Colors: []color.RGBA{colGreen, colWhite, colMagicHi},
		Fade: colGreenDark, Speed: 3.2, Drag: 0.9, Life: 26, Ring: true})
	p.fx.Burst(cx, cy-4, gfx.BurstSpec{N: 26, Colors: []color.RGBA{colBandage, colBandShade},
		Fade: colBandShade, Speed: 2.6, Gravity: 0.14, Drag: 0.98, Life: 75, Big: 0.5, UpBias: 2.2, Floor: floor})
	p.fx.Burst(cx, cy-6, gfx.BurstSpec{N: 26, Colors: []color.RGBA{colPurple, colMagicHi, colPink},
		Fade: colBlack, Speed: 1.0, Gravity: -0.05, Drag: 0.95, Life: 80, Big: 0.9, UpBias: 0.6})
}

func (p *Play) Draw(g *Game, screen *ebiten.Image) {
	w, s := p.World, g.Sheet
	screen.Fill(colBlack)
	s.DrawBackground(screen, w.Level, HUDH, p.parallax, w.Tick)
	for _, pl := range w.Platforms {
		s.DrawPlatform(screen, w.Level, float64(pl.TX*world.Tile), float64(pl.TY*world.Tile+HUDH), pl.Len)
	}
	for _, b := range w.Bombs {
		if b.Taken {
			continue
		}
		name := "bomb"
		if b.Lit {
			name = "bomb_lit"
		}
		s.Draw(screen, name, w.Tick, b.Pos.X, b.Pos.Y+HUDH, false)
	}
	pickupAnim := [...]string{"power_p", "power_b", "power_e", "power_s"}
	for _, pk := range w.Pickups {
		s.Draw(screen, pickupAnim[pk.Kind], w.Tick, pk.Pos.X, pk.Pos.Y+HUDH, false)
	}
	for _, e := range w.Enemies {
		name := e.Anim()
		if w.Frightened() {
			// blink back to the enemy in the last second as a warning
			if w.FrightTicks > 60 || w.FrightTicks/6%2 == 0 {
				name = "coin"
			}
		}
		if !e.Harmful() && !w.Frightened() && w.Tick/3%2 == 0 {
			continue // spawning: flicker
		}
		s.Draw(screen, name, w.Tick, e.Pos().X, e.Pos().Y+HUDH, e.FacingLeft())
	}
	p.drawJack(g, screen)
	if p.fx != nil {
		p.fx.Draw(screen, HUDH)
	}
	p.drawHUD(g, screen)

	switch {
	case p.paused, p.phase == phaseIntro:
		panel(screen, 48, 90, ScreenW-96, 34)
	case p.phase == phaseClear:
		panel(screen, 40, 84, ScreenW-80, 48)
	}
	switch {
	case p.paused:
		g.centerText(screen, "PAUSED", 96, colYellow)
		g.centerText(screen, "ESC AGAIN = QUIT", 110, colWhite)
	case p.phase == phaseIntro:
		g.centerText(screen, fmt.Sprintf("ROUND %d", p.Rules.Round+1), 96, colYellow)
		g.centerText(screen, "GET READY!", 112, colWhite)
	case p.phase == phaseClear:
		g.centerText(screen, "ROUND CLEAR!", 90, colYellow)
		g.centerText(screen, fmt.Sprintf("LIT BOMBS %d", p.Rules.LitCount), 108, colWhite)
		if b := p.Rules.RoundBonus(); b > 0 {
			g.centerText(screen, fmt.Sprintf("BONUS %d", b), 120, colCyan)
		}
	}
}

func (p *Play) drawJack(g *Game, screen *ebiten.Image) {
	j := p.Jack
	name := j.Anim()
	if j.Alive() {
		switch {
		case p.turnT > 0 && (name == "jack_run" || name == "jack_idle"):
			name = "jack_turn"
		case p.landT > 0 && name == "jack_idle":
			name = "jack_land"
		}
	}
	// blink during the "get ready" pause after a respawn
	if p.phase == phaseIntro && p.World.Tick > 0 && p.t/4%2 == 0 {
		return
	}
	pos := j.Pos()
	if p.World.Invincible && j.Alive() {
		shimmer := []color.Color{colCyan, colWhite, colYellow, colWhite}[p.World.Tick/4%4]
		g.Sheet.DrawTinted(screen, name, p.World.Tick, pos.X, pos.Y+HUDH, j.FacingLeft(), shimmer)
		return
	}
	g.Sheet.Draw(screen, name, p.World.Tick, pos.X, pos.Y+HUDH, j.FacingLeft())
}

func (p *Play) drawHUD(g *Game, screen *ebiten.Image) {
	s, r := g.Sheet, p.Rules
	s.Text(screen, fmt.Sprintf("SCORE %07d", r.Score), 2, 0, colWhite)
	hi := fmt.Sprintf("HI %07d", max(r.HiScore, g.HiScore))
	s.Text(screen, hi, float64(ScreenW-2-s.TextWidth(hi)), 0, colYellow)
	// bonus meter: 10 cells
	for i := 0; i < 10; i++ {
		img := s.MeterEmpty()
		if r.Meter >= (i+1)*10 {
			img = s.MeterFull()
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(2+i*8), 8)
		screen.DrawImage(img, op)
	}
	s.Text(screen, fmt.Sprintf("x%d", r.Multiplier), 88, 8, colCyan)
	if p.World.Invincible && p.World.Tick/20%2 == 0 {
		s.Text(screen, "INVINCIBLE", 112, 8, colRed)
	}
	for i := 0; i < min(r.Lives-1, 6); i++ { // spare lives
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(ScreenW-10-i*9), 8)
		screen.DrawImage(s.LifeIcon(), op)
	}
}
