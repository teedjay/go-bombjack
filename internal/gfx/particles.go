package gfx

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// particle is a single pixel (or 2x2 block) with simple ballistic motion.
type particle struct {
	x, y, vx, vy float64
	gravity      float64
	drag         float64
	t, life      int
	from, to     color.RGBA // colour at birth -> colour at death
	size         float32
	floor        float64 // bounce at this y (0 = no floor)
	trail        float64 // draw a streak this many ticks of velocity long
	grow         float32 // size increase per tick (smoke puffs swell)
	round        bool    // draw as a disc instead of a square
	wobble       float64 // sideways sine drift amplitude (steam)
	phase        float64
	outline      color.RGBA // cartoon outline for round puffs (A=0: none)
	pop          int        // sparks spawned when it dies (firework crackle)
}

// BurstSpec describes a radial particle burst.
type BurstSpec struct {
	N       int          // particle count
	Colors  []color.RGBA // birth colours, picked at random
	Fade    color.RGBA   // colour particles fade towards
	Speed   float64      // max initial speed (px/tick)
	Gravity float64      // + falls, - rises (smoke)
	Drag    float64      // velocity multiplier per tick (e.g. 0.96)
	Life    int          // ticks, each particle gets 60..100% of it
	Big     float64      // fraction of particles drawn 2x2
	UpBias  float64      // extra upward initial velocity
	Ring    bool         // evenly spaced angles at full speed (shockwave)
	Floor   float64      // particles bounce at this y (0 = no floor)
	Size    float32      // base size in px (default 1; Big doubles it)
	Trail   float64      // flare streak length in ticks of velocity
	Grow    float32      // size growth per tick
	Round   bool         // discs instead of squares (smoke)
	Wobble  float64      // sideways drift amplitude (steam)
	Outline color.RGBA   // outline colour for Round puffs; overlapping puffs merge into one cartoon cloud
	Pop     int          // each particle bursts into this many sparks when it dies
	VX, VY  float64      // base velocity added to every particle (drift, fall)
}

// Burst emits particles centred on (x,y) in playfield coordinates.
func (f *FX) Burst(x, y float64, s BurstSpec) {
	for i := 0; i < s.N; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := s.Speed * (0.3 + 0.7*rand.Float64())
		if s.Ring {
			a, sp = float64(i)/float64(s.N)*2*math.Pi, s.Speed
		}
		size := max(s.Size, 1)
		if rand.Float64() < s.Big {
			size *= 2
		}
		drag := s.Drag
		if drag == 0 {
			drag = 1
		}
		f.parts = append(f.parts, particle{
			x: x, y: y,
			vx: math.Cos(a)*sp + s.VX, vy: math.Sin(a)*sp - s.UpBias + s.VY,
			gravity: s.Gravity, drag: drag,
			life: int(float64(s.Life) * (0.6 + 0.4*rand.Float64())),
			from: s.Colors[rand.IntN(len(s.Colors))], to: s.Fade,
			size: size, floor: s.Floor,
			trail: s.Trail, grow: s.Grow, round: s.Round,
			wobble: s.Wobble, phase: rand.Float64() * 2 * math.Pi,
			outline: s.Outline, pop: s.Pop,
		})
	}
}

func (f *FX) updateParticles() {
	var pops []particle
	n := 0
	for i := range f.parts {
		p := &f.parts[i]
		p.t++
		p.vx *= p.drag
		p.vy = p.vy*p.drag + p.gravity
		p.x += p.vx + math.Sin(p.phase+float64(p.t)*0.15)*p.wobble*0.1
		p.y += p.vy
		p.size += p.grow
		if p.floor > 0 && p.y > p.floor && p.vy > 0 {
			p.y, p.vy, p.vx = p.floor, -p.vy*0.45, p.vx*0.7
		}
		if p.t < p.life {
			f.parts[n] = *p
			n++
		} else if p.pop > 0 {
			pops = append(pops, *p)
		}
	}
	f.parts = f.parts[:n]
	for _, p := range pops {
		f.Burst(p.x, p.y, BurstSpec{N: p.pop, Colors: []color.RGBA{{0xf8, 0xf8, 0xf8, 0xff}, p.from},
			Fade: p.to, Speed: 1.2, Gravity: 0.05, Drag: 0.94, Life: 16})
	}
}

func lerp8(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }

func (f *FX) drawParticles(dst *ebiten.Image, offsetY float64) {
	// outlines first, so overlapping outlined puffs merge into one cloud
	for _, p := range f.parts {
		if !p.round || p.outline.A == 0 {
			continue
		}
		a := particleAlpha(p)
		c := color.RGBA{uint8(float64(p.outline.R) * a), uint8(float64(p.outline.G) * a), uint8(float64(p.outline.B) * a), uint8(255 * a)}
		vector.FillCircle(dst, float32(math.Round(p.x)), float32(math.Round(p.y+offsetY)), p.size/2+1.5, c, false)
	}
	for _, p := range f.parts {
		t := float64(p.t) / float64(p.life)
		alpha := particleAlpha(p)
		c := color.RGBA{
			lerp8(p.from.R, p.to.R, t), lerp8(p.from.G, p.to.G, t), lerp8(p.from.B, p.to.B, t), 255,
		}
		// premultiplied alpha for ebiten
		c.R, c.G, c.B, c.A = uint8(float64(c.R)*alpha), uint8(float64(c.G)*alpha), uint8(float64(c.B)*alpha), uint8(255*alpha)
		x, y := float32(math.Round(p.x)), float32(math.Round(p.y+offsetY))
		switch {
		case p.trail > 0:
			tx, ty := float32(p.x-p.vx*p.trail), float32(p.y+offsetY-p.vy*p.trail)
			vector.StrokeLine(dst, tx, ty, x, y, p.size, c, false)
		case p.round:
			vector.FillCircle(dst, x, y, p.size/2, c, false)
		default:
			vector.FillRect(dst, x, y, p.size, p.size, c, false)
		}
	}
}

// particleAlpha fades a particle out over the last 30% of its life.
func particleAlpha(p particle) float64 {
	t := float64(p.t) / float64(p.life)
	if t > 0.7 {
		return 1 - (t-0.7)/0.3
	}
	return 1
}
