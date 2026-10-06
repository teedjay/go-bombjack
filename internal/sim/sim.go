// Package sim runs headless rounds with a simple bot. It is a measuring tool
// for balance and soak tests and mirrors how internal/game/play.go steps a
// round (it must not import internal/game, which pulls in ebiten).
package sim

import (
	"math"
	"math/rand/v2"

	"bombjack/internal/enemy"
	"bombjack/internal/level"
	"bombjack/internal/player"
	"bombjack/internal/rules"
	"bombjack/internal/world"
)

// Round bundles a headless round.
type Round struct {
	World *world.World
	Rules *rules.State
	Jack  *player.Jack
	Def   level.Def
}

// NewRound builds round `round` (0-based, loops through the levels).
func NewRound(round int, seed uint64) *Round { return NewRoundDiff(round, seed, rules.Easy) }

// NewRoundDiff is NewRound at a given difficulty.
func NewRoundDiff(round int, seed uint64, diff rules.Difficulty) *Round {
	d := level.Levels[round%len(level.Levels)]
	r := rules.NewForLevel(d.LitOrder, round)
	r.SetDifficulty(diff)
	w := level.Build(round, seed)
	j := player.New(d.PlayerStart)
	w.Player = j
	w.Systems = append(w.Systems,
		enemy.NewSpawner(rules.Scale(d.Enemies, r.Loop(), r.Diff)),
		rules.PickupMover{})
	return &Round{World: w, Rules: r, Jack: j, Def: d}
}

// Step advances one tick like Play.Update. It reports whether Jack died
// (death sequence finished) and whether the round was cleared this tick.
func (r *Round) Step(c world.Controls) (died, cleared bool) {
	w := r.World
	if r.Jack.Alive() {
		w.Step(c)
	} else {
		w.Events = w.Events[:0]
		w.Tick++
		r.Jack.Update(w, world.Controls{})
	}
	r.Rules.Apply(w)
	for _, e := range w.Events {
		switch e.Kind {
		case world.EvRoundClear:
			cleared = true
		case world.EvPlayerDied:
			died = true
			if r.Rules.Lives > 0 {
				r.Jack.Respawn(r.Def.PlayerStart)
				w.Enemies = w.Enemies[:0]
				w.FrightTicks = 0
			}
		}
	}
	return
}

// Result summarises a bot run.
type Result struct {
	TicksAlive     int // total ticks played until game over / clear / maxTicks
	FirstDeathTick int // tick of the first death (== TicksAlive if none)
	Deaths         int
	BombsTaken     int
	Cleared        bool
	Score          int
}

// Run plays one round with the bot until clear, game over or maxTicks.
func Run(round int, seed uint64, maxTicks int) Result {
	return RunDiff(round, seed, maxTicks, rules.Easy)
}

// RunDiff is Run at a given difficulty.
func RunDiff(round int, seed uint64, maxTicks int, diff rules.Difficulty) Result {
	r := NewRoundDiff(round, seed, diff)
	b := NewBotSeeded(seed)
	res := Result{FirstDeathTick: -1}
	for t := 0; t < maxTicks; t++ {
		died, cleared := r.Step(b.Controls(r.World, r.Jack))
		res.TicksAlive = t + 1
		if died {
			res.Deaths++
			if res.FirstDeathTick < 0 {
				res.FirstDeathTick = t + 1
			}
			if r.Rules.Lives <= 0 {
				break
			}
		}
		if cleared {
			res.Cleared = true
			break
		}
	}
	if res.FirstDeathTick < 0 {
		res.FirstDeathTick = res.TicksAlive
	}
	res.BombsTaken = len(r.World.Bombs) - r.World.BombsLeft()
	res.Score = r.Rules.Score
	return res
}

// Bot is a simple heuristic player.
type Bot struct {
	prevY    float64
	prevHeld bool
	holdFor  int
	held     int
	cur      *world.Bomb
	since    int
	skip     map[*world.Bomb]int
	rng      *rand.Rand
	pause    int
	wx       float64
	wvalid   bool
}

func NewBot() *Bot { return NewBotSeeded(1) }

// NewBotSeeded returns a bot whose human-like hesitation depends on seed.
func NewBotSeeded(seed uint64) *Bot {
	return &Bot{rng: rand.New(rand.NewPCG(seed, 0xb07))}
}

func center(r world.Rect) (float64, float64) { return r.X + r.W/2, r.Y + r.H/2 }

// Controls decides this tick's input.
func (b *Bot) Controls(w *world.World, j *player.Jack) world.Controls {
	var c world.Controls
	if !j.Alive() {
		return c
	}
	// human-like hesitation: occasionally stand still for a moment
	if b.pause > 0 {
		b.pause--
		if j.State() == player.Idle || j.State() == player.Run {
			return c
		}
	} else if b.rng != nil && b.rng.IntN(150) == 0 {
		b.pause = 10 + b.rng.IntN(30)
	}
	hb := j.Hitbox()
	jx, jy := center(hb)
	vy := j.Pos().Y - b.prevY
	b.prevY = j.Pos().Y
	grounded := j.State() == player.Idle || j.State() == player.Run

	// Pick the target bomb: lit first, else nearest.
	var tgt *world.Bomb
	best := math.Inf(1)
	for _, bm := range w.Bombs {
		if bm.Taken || w.Tick < b.skip[bm] {
			continue
		}
		bx, by := center(bm.Hitbox())
		d := math.Hypot(bx-jx, (by-jy)*1.5)
		if bm.Lit {
			d -= 400
		}
		if d < best {
			best, tgt = d, bm
		}
	}

	// Danger: nearest harmful enemy within 28 px.
	var danger world.Enemy
	dd := 28.0
	if !w.Frightened() {
		for _, e := range w.Enemies {
			if !e.Harmful() {
				continue
			}
			ex, ey := center(e.Hitbox())
			if d := math.Hypot(ex-jx, ey-jy); d < dd {
				dd, danger = d, e
			}
		}
	}

	jumpWant := false
	if danger != nil {
		ex, ey := center(danger.Hitbox())
		if ex >= jx {
			c.Left = true
		} else {
			c.Right = true
		}
		if grounded {
			jumpWant = ey >= jy-4
		} else if ey < jy && !c.Down {
			c.Down = true // cancel float / drop away from enemies above
		}
		if !grounded && ey >= jy {
			jumpWant = true // float / glide away
		}
		b.apply(&c, jumpWant, grounded, vy, 14)
		return c
	}
	if tgt == nil {
		return c
	}
	if tgt != b.cur {
		b.cur, b.since = tgt, w.Tick
	} else if w.Tick-b.since > 360 { // stuck: give up on this bomb for a while
		if b.skip == nil {
			b.skip = map[*world.Bomb]int{}
		}
		b.skip[tgt] = w.Tick + 900
		b.cur = nil
	}
	bx, by := center(tgt.Hitbox())
	dx, dy := bx-jx, by-jy
	feet := hb.Y + hb.H
	if grounded {
		b.wvalid = false
		if by < feet-140 { // too high for one jump: hop via a platform
			bestScore := math.Inf(1)
			for _, p := range w.Platforms {
				r := p.Rect()
				if r.Y > feet-16 || r.Y < feet-125 {
					continue
				}
				x := math.Max(r.X+4, math.Min(r.X+r.W-4, bx))
				if sc := math.Abs(x-jx) + math.Abs(by-r.Y)*0.5; sc < bestScore {
					bestScore, b.wx, b.wvalid = sc, x, true
				}
			}
		}
	}
	if b.wvalid {
		bx = b.wx
		dx = bx - jx
		if math.Abs(dx) <= 3 && grounded {
			dy = -100
		} else if grounded {
			dy = 0
		}
	}
	edge := false
	if grounded && dy > 10 && feet < world.FloorY+15 {
		// bomb is below: if it lies under our platform, walk off the nearest edge to it
		for _, p := range w.Platforms {
			r := p.Rect()
			if math.Abs(r.Y-feet) < 1.5 && jx >= r.X-5 && jx <= r.X+r.W+5 && bx >= r.X-4 && bx <= r.X+r.W+4 {
				edge = true
				if bx < r.X+r.W/2 {
					c.Left = true
				} else {
					c.Right = true
				}
			}
		}
	}
	if edge {
	} else if dx < -2 {
		c.Left = true
	} else if dx > 2 {
		c.Right = true
	}
	hold := 14
	if grounded {
		// jump when the target is above us (or far above on a platform edge)
		if dy < -8 && (math.Abs(dx) < 14 || dy > -60 || feet < world.FloorY+15 == false) {
			jumpWant = true
			// shorter hold for nearer targets
			hold = int(math.Min(14, math.Max(4, -dy/6.5)))
			b.holdFor = hold
		}
	} else {
		if vy > 0 && dy < 10 && b.held >= 0 && math.Abs(dx) > 6 {
			jumpWant = true // float to get across
		}
		if vy > 0 && dy > 4 {
			c.Down = true
		}
	}
	b.apply(&c, jumpWant, grounded, vy, hold)
	return c
}

// apply turns the wish to jump into held/pressed signals.
func (b *Bot) apply(c *world.Controls, want, grounded bool, vy float64, hold int) {
	if grounded {
		if want {
			c.Jump, c.JumpPressed = true, true
			b.held = 1
			if hold > 0 {
				b.holdFor = hold
			}
		} else {
			b.held = 0
		}
		return
	}
	if b.held > 0 && b.held < b.holdFor && vy <= 0 {
		c.Jump = true
		b.held++
		return
	}
	if want && vy > 0 && !b.prevHeld {
		c.Jump, c.JumpPressed = true, true
		b.prevHeld = true
		return
	}
	if vy <= 0 {
		b.prevHeld = false
	}
}
