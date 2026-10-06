package art

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------- Volcano ----

// VolcanoCrater is the crater centre in playfield coordinates at zero
// parallax (the game puffs smoke from here).
var VolcanoCrater = struct{ X, Y float64 }{128, 62}

func bgVolcano() Layers {
	far := NewCanvas(farW, FieldH)
	far.VGradientSmooth(0, 190, rgb(0x120406), rgb(0x3a0a0c), rgb(0x7a1a10), rgb(0xc04818), rgb(0xe07020))
	cx, top := VolcanoCrater.X+m, VolcanoCrater.Y
	// glow of the crater lighting the sky
	for y := 0; y < 120; y++ {
		for x := 0; x < farW; x++ {
			d := math.Hypot(float64(x)-cx, (float64(y)-top)*1.4)
			if d < 70 {
				far.Px(x, y, mix(far.At2(x, y), rgb(0xf89030), (70-d)/70*0.55))
			}
		}
	}
	// the cone: concave flanks from a 44px crater rim to a wide base
	base := 200.0
	for y := int(top); y < FieldH; y++ {
		t := (float64(y) - top) / (base - top)
		half := 22 + 150*math.Pow(math.Min(t, 1.2), 1.6)
		for x := int(cx - half); x <= int(cx+half); x++ {
			// lit from the left by the sunset sky, darker to the right
			side := (float64(x) - cx) / half
			col := mix(rgb(0x4a2a24), rgb(0x160a0a), 0.5+0.5*side)
			col = mix(col, rgb(0x0c0606), t*0.5)
			far.Px(x, y, col)
		}
	}
	// lava rivers winding down the flanks, with a hot core and glowing edge
	for i, r := range [][3]float64{{-12, -0.55, 1}, {6, 0.35, 2}, {14, 0.85, 3}, {-4, -0.15, 4}} {
		x0, drift, seed := r[0], r[1], r[2]
		for y := int(top) + 2; y < FieldH; y++ {
			dy := float64(y) - top
			x := cx + x0 + drift*dy + 3*math.Sin(dy/9+seed) + 2*math.Sin(dy/23+seed*2)
			w := 1.2 + dy/90 + float64(i%2)*0.5
			for dx := -w - 3; dx <= w+3; dx++ {
				px := int(x + dx)
				ad := math.Abs(dx)
				switch {
				case ad <= w*0.4:
					far.Px(px, y, rgb(0xf8e070))
				case ad <= w:
					far.Px(px, y, rgb(0xf87818))
				default: // glow on the rock either side
					far.Px(px, y, mix(far.At2(px, y), rgb(0xc03010), (w+3-ad)/3*0.6))
				}
			}
		}
	}
	// molten crater
	for y := int(top) - 4; y <= int(top)+4; y++ {
		for x := int(cx) - 24; x <= int(cx)+24; x++ {
			dx, dy := (float64(x)-cx)/24, (float64(y)-top)/4
			if d := dx*dx + dy*dy; d <= 1 {
				far.Px(x, y, mix(rgb(0xf8f0a0), rgb(0xf86010), d))
			}
		}
	}

	// drifting ash plumes, lit orange underneath by the lava
	clouds := NewCanvas(FieldW, FieldH)
	for _, p := range [][3]float64{{40, 28, 1.1}, {150, 18, 1.3}, {225, 44, 0.9}, {95, 50, 0.7}} {
		cloud(clouds, p[0], p[1], p[2], rgb(0x4a3434), rgb(0x8a4a30))
	}

	// foreground: jagged black rock with orange rims and a lava pool
	c := NewCanvas(FieldW, FieldH)
	ridge(c, func(x int) float64 {
		return 196 + 5*math.Abs(math.Sin(float64(x)/7)) - 6*math.Sin(float64(x)/19)
	}, FieldH, func(x, y int) color.RGBA {
		return mix(rgb(0x241414), rgb(0x0a0606), float64(y-188)/36)
	})
	for x := 0; x < FieldW; x++ { // hot rims along the rock tops
		for y := 186; y < FieldH; y++ {
			if c.At2(x, y).A != 0 {
				c.Px(x, y, rgb(0xd05018))
				break
			}
		}
	}
	for y := 208; y < FieldH; y++ { // lava pool
		for x := 70; x < 186; x++ {
			dx := (float64(x) - 128) / 58
			if dx*dx+math.Pow(float64(y-216)/9, 2) <= 1 {
				col := mix(rgb(0xf8d060), rgb(0xe05010), math.Abs(dx))
				if hash(x/4, y/2, 41) < 0.18 {
					col = rgb(0x5a1a0a) // cooling crust
				}
				c.Px(x, y, col)
			}
		}
	}
	return Layers{far.RGBA, clouds.RGBA, c.RGBA}
}

// ---------------------------------------------------------- Iceland ----

func bgIceland() Layers {
	far := NewCanvas(farW, FieldH)
	far.VGradientSmooth(0, 180, rgb(0x081028), rgb(0x183868), rgb(0x3878a0), rgb(0x9cc8e0))
	stars(far, 70, 37, 0.01)
	// aurora curtains: wavy green/teal bands blended into the sky
	for x := 0; x < farW; x++ {
		fx := float64(x)
		for _, b := range [][3]float64{{28, 0.6, 0}, {44, 0.45, 2.1}} {
			yc := b[0] + 9*math.Sin(fx/31+b[2]) + 4*math.Sin(fx/11+b[2]*2)
			for y := int(yc) - 14; y < int(yc)+10; y++ {
				d := (float64(y) - yc) / 14
				if d < -1 || d > 0.7 {
					continue
				}
				a := b[1] * (1 - math.Abs(d)) * (0.6 + 0.4*math.Sin(fx/7+b[2]))
				col := mix(rgb(0x40f0a0), rgb(0x40c0f0), 0.5+0.5*d)
				far.Px(x, y, mix(far.At2(x, y), col, a))
			}
		}
	}
	// the big ice mountain (plus a smaller peak behind)
	mountain := func(cx, peak, base, halfw float64, lit, shade, snow color.RGBA) {
		for y := int(peak); y < FieldH; y++ {
			t := (float64(y) - peak) / (base - peak)
			half := halfw * math.Min(t, 1.15) * (1 + 0.08*math.Sin(float64(y)/5))
			for x := int(cx - half); x <= int(cx+half); x++ {
				col := lit
				if float64(x) > cx+3*math.Sin(float64(y)/6) {
					col = shade
				}
				if t < 0.35+0.06*math.Sin(float64(x)/4) { // snow cap
					col = snow
					if float64(x) > cx {
						col = mix(snow, shade, 0.35)
					}
				}
				col = mix(col, rgb(0x9cc8e0), t*0.25) // aerial haze toward the base
				far.Px(x, y, col)
			}
		}
	}
	mountain(70+m, 70, 190, 110, rgb(0x6890b8), rgb(0x40608c), rgb(0xd8e8f8))
	mountain(170+m, 34, 190, 140, rgb(0x98c0e0), rgb(0x5878a8), White)
	// glacier streaks down the big face
	for i := 0; i < 5; i++ {
		x0 := 160 + m + float64(i*7)
		for y := 70; y < 180; y++ {
			x := x0 + float64(y-70)*(0.15*float64(i)-0.3) + 2*math.Sin(float64(y)/8+float64(i))
			far.Px(int(x), y, rgb(0xb8e0f8))
		}
	}

	// drifting mist banks
	clouds := NewCanvas(FieldW, FieldH)
	for _, p := range [][4]float64{{50, 96, 40, 0}, {170, 120, 52, 1}, {110, 150, 36, 0}, {230, 80, 30, 1}} {
		streak(clouds, p[0], p[1], p[2], rgb(0xa8c8e0), rgb(0xe8f4fc))
	}

	// foreground: snow dunes, ice spikes and snowy pines
	c := NewCanvas(FieldW, FieldH)
	for _, t := range []float64{14, 40, 214, 240} { // pines
		for y := 0; y < 34; y++ {
			half := float64(y) * 0.32
			for x := int(t - half); x <= int(t+half); x++ {
				col := rgb(0x183828)
				if y%8 < 2 {
					col = White // snow on the branches
				}
				c.Px(x, y+156, col)
			}
		}
	}
	for _, s := range [][3]float64{{70, 190, 14}, {84, 196, 9}, {176, 188, 16}, {190, 196, 10}} { // ice spikes
		c.Tri(s[0]-4, s[1]+12, s[0], s[1]-s[2], s[0]+4, s[1]+12, rgb(0x88d8f0))
		c.Tri(s[0], s[1]+12, s[0], s[1]-s[2], s[0]+4, s[1]+12, rgb(0x48a0d0))
	}
	ridge(c, func(x int) float64 {
		return 198 + 4*math.Sin(float64(x)/17) + 2*math.Sin(float64(x)/6)
	}, FieldH, func(x, y int) color.RGBA {
		return mix(White, rgb(0x98c0e0), float64(y-194)/30)
	})
	return Layers{far.RGBA, clouds.RGBA, c.RGBA}
}
