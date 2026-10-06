package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bombjack/internal/art"
	"bombjack/internal/enemy"
	"bombjack/internal/gfx"
	"bombjack/internal/level"
	"bombjack/internal/missile"
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
	colSteam     = color.RGBA{0xd8, 0xdc, 0xe8, 0xff}
	colSmokeDark = color.RGBA{0x48, 0x4c, 0x60, 0xff}
	colTrailInk  = color.RGBA{0x40, 0x42, 0x58, 0xff} // cartoon outline on smoke
	colAsh       = color.RGBA{0x5a, 0x48, 0x48, 0xff} // volcano smoke
	colAshLit    = color.RGBA{0x8a, 0x58, 0x40, 0xff}
	colAshDark   = color.RGBA{0x2a, 0x1e, 0x20, 0xff}
	colAshInk    = color.RGBA{0x1a, 0x10, 0x10, 0xff}
	colSnow      = color.RGBA{0xc0, 0xe0, 0xf8, 0xff}
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
	hintDelay  = 180 // ticks before the chain-start bomb starts flashing
)

// Play runs the game session: one Play per round, rules carried across.
type Play struct {
	World  *world.World
	Rules  *rules.State
	Jack   *player.Jack
	Def    level.Def
	fx     *gfx.FX
	bgfx   *gfx.FX // ambient particles between the backdrop and the platforms
	phase  phase
	t      int // ticks in current phase
	paused bool

	landT, turnT int
	lastLeft     bool
	parallax     float64 // smoothed -1..1 from Jack's x position
	shake        int     // ticks of screen shake left
	shakeAmp     float64
	flash        int // ticks of white screen flash left
	missiles     *missile.Manager
	lastTail     map[*missile.Missile]world.Vec // trail interpolation
	canvas       *ebiten.Image                  // offscreen frame, drawn shaken to the screen
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
	ms := &missile.Manager{}
	w.Systems = append(w.Systems,
		enemy.NewSpawner(rules.Scale(d.Enemies, r.Loop())),
		rules.PickupMover{},
		ms)
	p := &Play{World: w, Rules: r, Jack: j, Def: d, missiles: ms}
	if g != nil {
		r.HiScore = g.HiScore
		p.fx = gfx.NewFX(g.Sheet)
		p.bgfx = gfx.NewFX(g.Sheet)
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
	p.shake = max(p.shake-1, 0)
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		p.World.Invincible = !p.World.Invincible
	}
	target := (p.Jack.Pos().X + 8 - world.FieldW/2) / (world.FieldW / 2)
	p.parallax += (target - p.parallax) * 0.04
	p.t++
	if p.fx != nil {
		p.fx.Update()
		p.bgfx.Update()
		p.ambient()
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
		if c.Fire && p.Rules.Missiles > 0 {
			jp := p.Jack.Pos()
			if p.missiles.Fire(w, world.Vec{X: jp.X + 8, Y: jp.Y + 6}, p.Jack.FacingLeft()) {
				p.Rules.UseMissile()
			}
		}
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
	p.missileTrails()
	p.flash = max(p.flash-1, 0)

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

// ambient spawns each level's living-backdrop particles: smoke from the
// volcano's crater and embers from its lava pool; snowfall in Iceland, in
// two depths (small slow flakes behind everything, big fast ones in front).
func (p *Play) ambient() {
	tick := p.World.Tick + p.t
	switch p.Def.Name {
	case "volcano":
		if tick%5 == 0 {
			// the crater sits on the far layer, which shifts with parallax
			cx := art.VolcanoCrater.X - p.parallax*art.ParallaxMargin
			p.bgfx.Burst(cx+rand.Float64()*24-12, art.VolcanoCrater.Y-2, gfx.BurstSpec{N: 1,
				Colors: []color.RGBA{colAsh, colAshLit}, Fade: colAshDark, Speed: 0.25,
				VX: 0.15, VY: -0.35, Drag: 0.995, Life: 170, Size: 5, Grow: 0.09,
				Round: true, Outline: colAshInk, Wobble: 2})
		}
		if tick%3 == 0 {
			p.bgfx.Burst(70+rand.Float64()*116, world.FieldH-10, gfx.BurstSpec{N: 1,
				Colors: []color.RGBA{colYellow, colOrange, colWhite}, Fade: colRedDark,
				Speed: 0.4, VY: -1.1, Gravity: 0.012, Life: 75, Wobble: 2, Big: 0.25})
		}
	case "iceland":
		p.bgfx.Burst(rand.Float64()*(world.FieldW+40)-20, -2, gfx.BurstSpec{N: 1,
			Colors: []color.RGBA{colWhite, colSnow}, Fade: colSnow, Speed: 0.1,
			VX: 0.12, VY: 0.45, Life: 520, Wobble: 3})
		if tick%7 == 0 {
			p.fx.Burst(rand.Float64()*(world.FieldW+60)-30, -4, gfx.BurstSpec{N: 1,
				Colors: []color.RGBA{colWhite}, Fade: colSnow, Speed: 0.1,
				VX: 0.3, VY: 1.0, Life: 240, Size: 2, Wobble: 4})
		}
	}
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
			p.bombBoom(cx, cy, e.Lit)
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
		case world.EvMissileFired:
			p.fx.Burst(e.Pos.X, e.Pos.Y, gfx.BurstSpec{N: 10, Colors: []color.RGBA{colWhite, colSteam},
				Fade: colSmoke, Speed: 1.4, Gravity: -0.02, Drag: 0.9, Life: 40,
				Size: 4, Grow: 0.12, Round: true, Outline: colTrailInk})
		case world.EvMissileHit:
			p.firework(e.Pos.X, e.Pos.Y)
		}
	}
}

// missileTrails lays a fat cartoon smoke trail behind every missile:
// outlined puffs (which merge into one cloud) spaced every few pixels along
// the path the tail travelled this tick, so a fast missile leaves no gaps,
// plus flame sparks at the nozzle.
func (p *Play) missileTrails() {
	if p.fx == nil {
		return
	}
	if p.lastTail == nil {
		p.lastTail = map[*missile.Missile]world.Vec{}
	}
	live := map[*missile.Missile]bool{}
	for _, ms := range p.missiles.Missiles {
		live[ms] = true
		t := ms.Tail()
		from, ok := p.lastTail[ms]
		if !ok {
			from = t
		}
		p.lastTail[ms] = t
		d := math.Hypot(t.X-from.X, t.Y-from.Y)
		n := max(1, int(d/3))
		for i := 1; i <= n; i++ {
			f := float64(i) / float64(n)
			x, y := from.X+(t.X-from.X)*f, from.Y+(t.Y-from.Y)*f
			p.fx.Burst(x, y, gfx.BurstSpec{N: 1, Colors: []color.RGBA{colWhite, colSteam},
				Fade: colSmoke, Speed: 0.3, Gravity: -0.012, Drag: 0.93, Life: 55,
				Size: 5, Grow: 0.2, Round: true, Outline: colTrailInk, Wobble: 1.5})
		}
		p.fx.Burst(t.X, t.Y, gfx.BurstSpec{N: 2, Colors: []color.RGBA{colYellow, colOrange, colWhite},
			Fade: colRedDark, Speed: 0.8, Drag: 0.9, Life: 12, Size: 2})
	}
	for ms := range p.lastTail {
		if !live[ms] {
			delete(p.lastTail, ms)
		}
	}
}

// firework is the missile impact: a double-size boom ringed by smaller
// ones, a rainbow ring and a spray of trailing sparks that crackle into
// more sparks, embers bouncing on the floor, billowing outlined smoke, a
// white flash and a big screen shake.
func (p *Play) firework(cx, cy float64) {
	if p.fx == nil {
		return
	}
	floor := float64(world.FieldH - 1)
	rainbow := []color.RGBA{colRed, colYellow, colCyan, colPink, colGreen, colWhite, colPurple}
	p.fx.SpawnCentered("boom", cx, cy, 2)
	for i := 0; i < 3; i++ {
		a := rand.Float64() * 2 * math.Pi
		p.fx.SpawnCentered("boom", cx+math.Cos(a)*18, cy+math.Sin(a)*18, 1)
	}
	p.fx.Burst(cx, cy, gfx.BurstSpec{N: 36, Colors: rainbow, Fade: colOrange, Speed: 4.2,
		Gravity: 0.05, Drag: 0.95, Life: 42, Size: 2, Trail: 2.5, Ring: true, Pop: 4})
	p.fx.Burst(cx, cy, gfx.BurstSpec{N: 44, Colors: rainbow, Fade: colRedDark, Speed: 6,
		Gravity: 0.09, Drag: 0.96, Life: 55, Trail: 2, Big: 0.3, Pop: 3})
	p.fx.Burst(cx, cy, gfx.BurstSpec{N: 18, Colors: []color.RGBA{colOrange, colYellow},
		Fade: colRedDark, Speed: 3, Gravity: 0.14, Drag: 0.98, Life: 70, Size: 2, UpBias: 2, Floor: floor})
	p.fx.Burst(cx, cy, gfx.BurstSpec{N: 16, Colors: []color.RGBA{colSteam, colWhite, colSmoke},
		Fade: colSmokeDark, Speed: 1.6, Gravity: -0.04, Drag: 0.94, Life: 90,
		Size: 5, Grow: 0.14, Round: true, Outline: colTrailInk, Wobble: 3})
	p.shake = max(p.shake, 16)
	p.shakeAmp = 3
	p.flash = 6
}

// bombBoom is the exaggerated cartoon explosion for collecting a bomb: a
// big spiky "boom", flares streaking out on arcs, bouncing embers, steam
// puffs swelling as they rise, and a short screen shake. Lit bombs go bigger.
func (p *Play) bombBoom(cx, cy float64, lit bool) {
	floor := float64(world.FieldH - 1)
	scale := 1.0
	if lit {
		scale = 1.6
	}
	n := func(base int) int { return int(float64(base) * scale) }
	p.fx.Spawn("boom", cx-16, cy-16)
	p.fx.Burst(cx, cy, gfx.BurstSpec{N: n(14), Colors: []color.RGBA{colWhite, colYellow},
		Fade: colOrange, Speed: 3.8 * scale, Gravity: 0.1, Drag: 0.965, Life: 40,
		Size: 2, Trail: 3, UpBias: 1})
	p.fx.Burst(cx, cy, gfx.BurstSpec{N: n(10), Colors: []color.RGBA{colOrange, colYellow},
		Fade: colRedDark, Speed: 2.2, Gravity: 0.12, Drag: 0.98, Life: 55, UpBias: 1.5, Floor: floor,
		Size: 2})
	p.fx.Burst(cx, cy-2, gfx.BurstSpec{N: n(9), Colors: []color.RGBA{colSteam, colWhite},
		Fade: colSmokeDark, Speed: 0.8, Gravity: -0.045, Drag: 0.96, Life: 75,
		Size: 3, Grow: 0.09, Round: true, UpBias: 0.7, Wobble: 3})
	p.shake = max(p.shake, n(8))
	p.shakeAmp = 1 + float64(n(1))
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

func (p *Play) Draw(g *Game, dst *ebiten.Image) {
	if p.canvas == nil {
		p.canvas = ebiten.NewImage(ScreenW, ScreenH)
	}
	p.drawFrame(g, p.canvas)
	dst.Fill(colBlack)
	op := &ebiten.DrawImageOptions{}
	if p.shake > 0 && !p.paused {
		// jitter alternates direction each tick and eases out
		a := p.shakeAmp * float64(p.shake) / 8
		dx := math.Round(a * float64(1-2*(p.World.Tick%2)))
		dy := math.Round(a * 0.6 * float64(1-2*((p.World.Tick/2)%2)))
		op.GeoM.Translate(dx, dy)
	}
	dst.DrawImage(p.canvas, op)
}

func (p *Play) drawFrame(g *Game, screen *ebiten.Image) {
	w, s := p.World, g.Sheet
	screen.Fill(colBlack)
	s.DrawBackground(screen, w.Level, HUDH, p.parallax, w.Tick)
	if p.bgfx != nil {
		p.bgfx.Draw(screen, HUDH)
	}
	for _, pl := range w.Platforms {
		s.DrawPlatform(screen, w.Level, float64(pl.TX*world.Tile), float64(pl.TY*world.Tile+HUDH), pl.Len)
	}
	hint := -1
	if w.Tick > hintDelay {
		hint = p.Rules.HintBomb(w)
	}
	for i, b := range w.Bombs {
		if b.Taken {
			continue
		}
		name := "bomb"
		switch {
		case b.Lit:
			name = "bomb_lit"
		case i == hint:
			name = "bomb_flash"
		}
		s.Draw(screen, name, w.Tick, b.Pos.X, b.Pos.Y+HUDH, false)
	}
	pickupAnim := [...]string{"power_p", "power_b", "power_e", "power_s", "power_m"}
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
	for _, ms := range p.missiles.Missiles {
		s.DrawRotated(screen, "missile", w.Tick, ms.Pos.X, ms.Pos.Y+HUDH, ms.Angle)
	}
	if p.flash > 0 {
		a := float32(p.flash) / 6 * 0.45
		vector.FillRect(screen, 0, HUDH, ScreenW, world.FieldH, color.RGBA{uint8(255 * a), uint8(255 * a), uint8(255 * a), uint8(255 * a)}, false)
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
	// missile stock
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(114, 0)
	screen.DrawImage(s.MissileIcon(), op)
	s.Text(screen, fmt.Sprintf("x%d", r.Missiles), 124, 0, colWhite)
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
