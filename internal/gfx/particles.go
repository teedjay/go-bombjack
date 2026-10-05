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
}

// Burst emits particles centred on (x,y) in playfield coordinates.
func (f *FX) Burst(x, y float64, s BurstSpec) {
	for i := 0; i < s.N; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := s.Speed * (0.3 + 0.7*rand.Float64())
		if s.Ring {
			a, sp = float64(i)/float64(s.N)*2*math.Pi, s.Speed
		}
		size := float32(1)
		if rand.Float64() < s.Big {
			size = 2
		}
		drag := s.Drag
		if drag == 0 {
			drag = 1
		}
		f.parts = append(f.parts, particle{
			x: x, y: y,
			vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - s.UpBias,
			gravity: s.Gravity, drag: drag,
			life: int(float64(s.Life) * (0.6 + 0.4*rand.Float64())),
			from: s.Colors[rand.IntN(len(s.Colors))], to: s.Fade,
			size: size, floor: s.Floor,
		})
	}
}

func (f *FX) updateParticles() {
	n := 0
	for i := range f.parts {
		p := &f.parts[i]
		p.t++
		p.vx *= p.drag
		p.vy = p.vy*p.drag + p.gravity
		p.x += p.vx
		p.y += p.vy
		if p.floor > 0 && p.y > p.floor && p.vy > 0 {
			p.y, p.vy, p.vx = p.floor, -p.vy*0.45, p.vx*0.7
		}
		if p.t < p.life {
			f.parts[n] = *p
			n++
		}
	}
	f.parts = f.parts[:n]
}

func lerp8(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }

func (f *FX) drawParticles(dst *ebiten.Image, offsetY float64) {
	for _, p := range f.parts {
		t := float64(p.t) / float64(p.life)
		alpha := 1.0
		if t > 0.7 {
			alpha = 1 - (t-0.7)/0.3
		}
		c := color.RGBA{
			lerp8(p.from.R, p.to.R, t), lerp8(p.from.G, p.to.G, t), lerp8(p.from.B, p.to.B, t), 255,
		}
		// premultiplied alpha for ebiten
		c.R, c.G, c.B, c.A = uint8(float64(c.R)*alpha), uint8(float64(c.G)*alpha), uint8(float64(c.B)*alpha), uint8(255*alpha)
		vector.FillRect(dst, float32(math.Round(p.x)), float32(math.Round(p.y+offsetY)), p.size, p.size, c, false)
	}
}
