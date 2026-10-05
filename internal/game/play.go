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

var colBlack = color.RGBA{0x08, 0x06, 0x10, 0xff}

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
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		p.paused = !p.paused
	}
	if p.paused {
		return p
	}
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
		switch e.Kind {
		case world.EvBombTaken, world.EvPickupTaken:
			p.fx.Spawn("sparkle", e.Pos.X, e.Pos.Y)
		case world.EvEnemyTransformed, world.EvPlayerHit:
			p.fx.Spawn("explosion", e.Pos.X, e.Pos.Y)
		case world.EvScore:
			p.fx.Popup(e.Value, e.Pos.X, e.Pos.Y)
		}
	}
}

func (p *Play) Draw(g *Game, screen *ebiten.Image) {
	w, s := p.World, g.Sheet
	screen.Fill(colBlack)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, HUDH)
	screen.DrawImage(s.Background(w.Level), op)
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
	case p.paused:
		g.centerText(screen, "PAUSED", 110, colYellow)
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
	for i := 0; i < min(r.Lives-1, 6); i++ { // spare lives
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(ScreenW-10-i*9), 8)
		screen.DrawImage(s.LifeIcon(), op)
	}
}
