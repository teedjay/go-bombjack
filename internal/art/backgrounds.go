package art

import (
	"image"
	"image/color"
	"math"
)

// Playfield size in pixels (the original arcade used 256x224).
const (
	FieldW = 256
	FieldH = 224
	// ParallaxMargin is how far (px) the far layer may shift either way; the
	// far layer is FieldW + 2*ParallaxMargin wide.
	ParallaxMargin = 16
	farW           = FieldW + 2*ParallaxMargin
	m              = ParallaxMargin // x offset into the far layer
)

// Layers is one level backdrop split for parallax:
//   - Far: sky, sun/moon, stars and distant scenery, farW wide
//   - Clouds: optional (may be nil), FieldW wide, scrolled with wrap-around
//   - Near: foreground scenery with a transparent sky, FieldW wide, static
type Layers struct {
	Far, Clouds, Near *image.RGBA
}

// BackgroundLayers returns the four level backdrops in level order.
func BackgroundLayers() []Layers {
	return []Layers{bgEgypt(), bgGreece(), bgCastle(), bgCity(), bgVolcano(), bgIceland()}
}

// Backgrounds returns each backdrop flattened at zero parallax (previews).
func Backgrounds() []*image.RGBA {
	var out []*image.RGBA
	for _, l := range BackgroundLayers() {
		c := NewCanvas(FieldW, FieldH)
		c.Blit(l.Far.SubImage(image.Rect(m, 0, m+FieldW, FieldH)).(*image.RGBA), 0, 0)
		if l.Clouds != nil {
			c.Blit(l.Clouds, 0, 0)
		}
		c.Blit(l.Near, 0, 0)
		out = append(out, c.RGBA)
	}
	return out
}

// mix is the backdrop blend: smooth instead of the sprites' ordered dither.
func mix(a, b color.RGBA, t float64) color.RGBA { return Lerp(a, b, t) }

func stars(c *Canvas, maxY, seed int, density float64) {
	for y := 0; y < maxY; y++ {
		for x := 0; x < c.W; x++ {
			h := hash(x, y, seed)
			if h < density {
				col := LightGrey
				if h < density/4 {
					col = White
				}
				c.Px(x, y, col)
			}
		}
	}
}

// ridge draws a filled silhouette along a function of x down to the bottom.
func ridge(c *Canvas, top func(x int) float64, bottom int, fill func(x, y int) color.RGBA) {
	for x := 0; x < c.W; x++ {
		for y := int(top(x)); y < bottom; y++ {
			c.Px(x, y, fill(x, y))
		}
	}
}

func cloud(c *Canvas, cx, cy, s float64, top, under color.RGBA) {
	for _, p := range [][3]float64{{0, 0, 8}, {-10, 3, 6}, {10, 3, 6}, {-4, -4, 6}, {6, -3, 5}} {
		c.Circle(cx+p[0]*s, cy+p[1]*s, p[2]*s, top)
	}
	// shaded underside
	for y := int(cy + 2*s); y < int(cy+10*s); y++ {
		for x := int(cx - 20*s); x < int(cx+20*s); x++ {
			if c.At2(x, y) == top {
				c.Px(x, y, mix(top, under, float64(y-int(cy+2*s))/(8*s)))
			}
		}
	}
}

// streak is a long thin sunset cloud with a lit top edge.
func streak(c *Canvas, cx, cy, w float64, col, hi color.RGBA) {
	for y := int(cy - 3); y <= int(cy+3); y++ {
		for x := int(cx - w); x <= int(cx+w); x++ {
			dx, dy := (float64(x)-cx)/w, (float64(y)-cy)/3
			if dx*dx+dy*dy <= 1 {
				c.Px(x, y, mix(hi, col, (dy+1)/2))
			}
		}
	}
}

// ------------------------------------------------------------ Egypt ----

func bgEgypt() Layers {
	far := NewCanvas(farW, FieldH)
	far.VGradientSmooth(0, 170, rgb(0x281850), rgb(0x682868), rgb(0xd04848), rgb(0xf89040), rgb(0xf8d070))
	// sun with a soft glow
	for y := 0; y < 170; y++ {
		for x := 0; x < farW; x++ {
			dx, dy := float64(x)-(180+m), float64(y)-128
			d := math.Sqrt(dx*dx + dy*dy)
			if d < 22 {
				far.Px(x, y, mix(rgb(0xf8e088), rgb(0xf8f8d0), (22-d)/22))
			} else if d < 40 {
				far.Px(x, y, mix(far.At2(x, y), rgb(0xf8d080), (40-d)/18*0.6))
			}
		}
	}
	clouds := NewCanvas(FieldW, FieldH)
	streak(clouds, 60, 52, 34, rgb(0xd06078), rgb(0xf8b8b0))
	streak(clouds, 74, 47, 18, rgb(0xe07888), rgb(0xf8d0c0))
	streak(clouds, 200, 84, 28, rgb(0xf09070), rgb(0xf8e0b0))
	streak(clouds, 140, 28, 22, rgb(0x985088), rgb(0xd890b0))

	c := NewCanvas(FieldW, FieldH)
	// pyramids: lit face left, shaded face right, brick courses
	pyr := func(cx, base, h float64) {
		light, shade, dark := rgb(0xe0a050), rgb(0xa86030), rgb(0x683820)
		for y := int(base - h); y < int(base); y++ {
			half := (float64(y) - (base - h)) * 1.15
			for x := int(cx - half); x <= int(cx+half); x++ {
				col := light
				if float64(x) > cx {
					col = mix(shade, dark, (float64(x)-cx)/half*0.6)
				}
				if (y-int(base))%5 == 0 {
					col = mix(col, dark, 0.3)
				}
				c.Px(x, y, col)
			}
		}
	}
	pyr(70, 168, 70)
	pyr(140, 168, 48)
	pyr(30, 168, 34)
	// sphinx silhouette
	sph := rgb(0x784020)
	c.Rect(186, 150, 48, 14, sph)
	c.Rect(222, 136, 14, 16, sph)
	c.Rect(219, 134, 18, 6, rgb(0x904c28))
	c.Rect(180, 158, 12, 6, sph)
	c.Px(229, 141, Ink)
	// dunes
	ridge(c, func(x int) float64 { return 164 + 4*math.Sin(float64(x)/23) + 3*math.Sin(float64(x)/7) },
		FieldH, func(x, y int) color.RGBA {
			return mix(rgb(0xe8b868), rgb(0xb07838), float64(y-160)/64)
		})
	return Layers{far.RGBA, clouds.RGBA, c.RGBA}
}

// ----------------------------------------------------------- Greece ----

func bgGreece() Layers {
	far := NewCanvas(farW, FieldH)
	far.VGradientSmooth(0, 150, rgb(0x2050c0), rgb(0x4888e8), rgb(0x90c8f8))
	// distant islands on the horizon
	ridge(far, func(x int) float64 {
		return 140 - 8*math.Max(0, math.Sin(float64(x)/30)) - 3*math.Sin(float64(x)/9)
	}, 150, func(x, y int) color.RGBA { return rgb(0x6890c0) })

	clouds := NewCanvas(FieldW, FieldH)
	under := rgb(0xb8c8e8)
	cloud(clouds, 50, 40, 1.2, White, under)
	cloud(clouds, 190, 25, 0.9, White, under)
	cloud(clouds, 140, 70, 0.7, White, under)

	c := NewCanvas(FieldW, FieldH)
	// sea
	for y := 140; y < 160; y++ {
		for x := 0; x < FieldW; x++ {
			col := mix(rgb(0x1868b0), rgb(0x104888), float64(y-140)/20)
			if hash(x/3, y, 7) < 0.06 {
				col = rgb(0x90d0f8)
			}
			c.Px(x, y, col)
		}
	}
	// hill
	ridge(c, func(x int) float64 {
		return 128 + 18*math.Cos((float64(x)-128)/70)*-1 + 4*math.Sin(float64(x)/11)
	}, FieldH, func(x, y int) color.RGBA {
		return mix(rgb(0x88a040), rgb(0x506828), float64(y-110)/110)
	})
	// Parthenon
	marble, shade, dark := rgb(0xf0e8d8), rgb(0xc0b8a8), rgb(0x807868)
	px, py := 70, 92 // top-left of entablature
	w := 116
	c.Tri(float64(px-2), float64(py), float64(px+w/2), float64(py-14), float64(px+w+2), float64(py), marble)
	c.Tri(float64(px+8), float64(py-1), float64(px+w/2), float64(py-10), float64(px+w-8), float64(py-1), shade)
	c.Rect(px-3, py, w+6, 6, marble)
	c.Rect(px-3, py+5, w+6, 1, dark)
	for i := 0; i < 9; i++ {
		x := px + 2 + i*14
		if i == 6 {
			c.Rect(x, py+22, 6, 18, marble) // broken column
			c.Rect(x+1, py+20, 3, 2, marble)
			continue
		}
		c.Rect(x, py+6, 6, 34, marble)
		c.Rect(x+4, py+6, 2, 34, shade)
		c.Rect(x+2, py+6, 1, 34, shade)
	}
	c.Rect(px-6, py+40, w+12, 3, marble)
	c.Rect(px-8, py+43, w+16, 3, shade)
	c.Rect(px-8, py+46, w+16, 1, dark)
	// olive trees
	for _, tx := range []int{28, 220, 238} {
		c.Rect(tx, 140, 2, 12, Brown)
		c.Circle(float64(tx)+1, 136, 7, rgb(0x406830))
		c.Circle(float64(tx)-1, 134, 4, rgb(0x689048))
	}
	return Layers{far.RGBA, clouds.RGBA, c.RGBA}
}

// ----------------------------------------------------------- Castle ----

func bgCastle() Layers {
	far := NewCanvas(farW, FieldH)
	far.VGradientSmooth(0, 180, rgb(0x080828), rgb(0x282068), rgb(0x684890), rgb(0xd07898))
	stars(far, 90, 11, 0.012)
	far.Circle(40+m, 36, 13, rgb(0xf8f0c8))
	far.Circle(46+m, 32, 12, rgb(0x181040)) // crescent cut
	// distant mountains with snow caps
	for _, mt := range [][3]float64{{-10, 160, 60}, {30, 160, 70}, {110, 160, 85}, {200, 160, 60}, {250, 160, 75}, {300, 160, 55}} {
		cx, base, h := mt[0]+m, mt[1], mt[2]
		far.Tri(cx-h*1.1, base, cx, base-h, cx+h*1.1, base, rgb(0x383870))
		far.Tri(cx, base-h, cx+h*1.1, base, cx+h*0.3, base, rgb(0x282858))
		far.Tri(cx-h*0.25, base-h*0.75, cx, base-h, cx+h*0.25, base-h*0.75, rgb(0xe0e0f8))
	}
	far.Rect(0, 160, farW, FieldH-160, rgb(0x282858))

	clouds := NewCanvas(FieldW, FieldH)
	streak(clouds, 90, 70, 30, rgb(0x504888), rgb(0x9888c8))
	streak(clouds, 210, 100, 24, rgb(0x806098), rgb(0xc0a0d8))

	c := NewCanvas(FieldW, FieldH)
	stone, sh, roof := rgb(0xc8c0d8), rgb(0x8880a8), rgb(0x3858a8)
	tower := func(x, y, w, h int) {
		c.Rect(x, y, w, h, stone)
		c.Rect(x+w-w/3, y, w/3, h, sh)
		c.Tri(float64(x-2), float64(y), float64(x)+float64(w)/2, float64(y-w*2), float64(x+w+2), float64(y), roof)
		for wy := y + 4; wy < y+h-6; wy += 9 {
			c.Rect(x+w/2-1, wy, 2, 4, rgb(0xf8d860))
		}
	}
	c.Rect(110, 110, 70, 60, stone)
	c.Rect(150, 110, 30, 60, sh)
	for x := 110; x < 180; x += 6 {
		c.Rect(x, 106, 3, 4, stone) // battlements
	}
	tower(98, 90, 14, 80)
	tower(132, 70, 16, 60)
	tower(176, 96, 12, 74)
	tower(156, 84, 10, 30)
	for i := 0; i < 5; i++ {
		c.Rect(116+i*12, 124, 3, 6, rgb(0xf8d860))
	}
	c.Rect(138, 150, 12, 20, rgb(0x302040)) // gate
	// forest silhouette
	ridge(c, func(x int) float64 {
		return 168 - 6*math.Abs(math.Sin(float64(x)/5)) - 4*math.Sin(float64(x)/17)
	}, FieldH, func(x, y int) color.RGBA {
		return mix(rgb(0x183828), rgb(0x0c1c18), float64(y-160)/50)
	})
	return Layers{far.RGBA, clouds.RGBA, c.RGBA}
}

// ------------------------------------------------------------- City ----

func bgCity() Layers {
	far := NewCanvas(farW, FieldH)
	far.VGradientSmooth(0, 200, rgb(0x050510), rgb(0x101838), rgb(0x283068), rgb(0x684878))
	stars(far, 110, 23, 0.015)
	far.Circle(200+m, 40, 16, rgb(0xf8f0d0))
	far.Circle(195+m, 36, 3, rgb(0xd8d0b0))
	far.Circle(206+m, 46, 2, rgb(0xd8d0b0))
	// distant skyline
	x := 0
	for i := 0; x < farW; i++ {
		w := 14 + int(hash(i, 1, 5)*18)
		h := 50 + int(hash(i, 2, 5)*50)
		far.Rect(x, 190-h, w, h+40, rgb(0x283050))
		for wy := 194 - h; wy < 188; wy += 8 {
			if hash(x, wy, 31) < 0.25 {
				far.Rect(x+w/2, wy, 1, 2, rgb(0x8890b0))
			}
		}
		x += w + 1
	}

	c := NewCanvas(FieldW, FieldH)
	x = -4
	for i := 0; x < FieldW; i++ {
		w := 20 + int(hash(i, 3, 9)*22)
		h := 40 + int(hash(i, 4, 9)*80)
		top := 200 - h
		body, edge := rgb(0x182040), rgb(0x303868)
		c.Rect(x, top, w, h, body)
		c.Rect(x, top, 1, h, edge)
		c.Rect(x, top, w, 1, edge)
		for wy := top + 4; wy < 196; wy += 6 {
			for wx := x + 3; wx < x+w-3; wx += 5 {
				if hash(wx, wy, 13) < 0.45 {
					col := rgb(0xf8d860)
					if hash(wx, wy, 14) < 0.3 {
						col = rgb(0x88e0f8)
					}
					c.Rect(wx, wy, 2, 3, col)
				}
			}
		}
		if i == 3 { // antenna tower
			c.Rect(x+w/2, top-24, 2, 24, edge)
			c.Px(x+w/2, top-25, Red)
			c.Px(x+w/2+1, top-25, Red)
		}
		x += w + 2
	}
	// street
	c.Rect(0, 200, FieldW, 24, rgb(0x101018))
	for sx := 0; sx < FieldW; sx += 16 {
		c.Rect(sx, 211, 8, 2, rgb(0x707080))
	}
	// no cloud layer at night in the city
	return Layers{Far: far.RGBA, Near: c.RGBA}
}
