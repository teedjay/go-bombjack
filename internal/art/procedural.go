package art

import (
	"image"
	"image/color"
	"math"
)

// shadedBall draws a lit sphere with a dithered terminator and a highlight.
func shadedBall(c *Canvas, cx, cy, r float64, dark, mid, light color.RGBA) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy > r*r {
				continue
			}
			// light from top-left
			l := (-dx*0.6 - dy*0.8) / r // -1..1
			col := mid
			switch {
			case l > 0.45:
				col = Dither(x, y, mid, light, math.Min(1, (l-0.45)*3))
			case l < -0.25:
				col = Dither(x, y, mid, dark, math.Min(1, (-l-0.25)*2.5))
			}
			c.Px(x, y, col)
		}
	}
	c.Px(int(cx-r*0.45), int(cy-r*0.5), White)
}

// glyphs is a bold 4x6 micro font for the power-up letters.
var glyphs = map[rune][]string{
	'P': {"###.", "##.#", "##.#", "###.", "##..", "##.."},
	'B': {"###.", "##.#", "###.", "##.#", "##.#", "###."},
	'E': {"####", "##..", "###.", "##..", "##..", "####"},
	'S': {".###", "##..", ".##.", "..##", "..##", "###."},
}

func drawGlyph(c *Canvas, r rune, x, y int, col color.RGBA) {
	for j, row := range glyphs[r] {
		for i, ch := range row {
			if ch == '#' {
				c.Px(x+i, y+j, col)
			}
		}
	}
}

func bomb(lit bool, frame int) *image.RGBA {
	c := NewCanvas(16, 16)
	shadedBall(c, 7.5, 9.5, 5.5, DarkRed, Red, Pink)
	// cap + fuse
	c.Rect(6, 3, 4, 1, Grey)
	c.Rect(7, 2, 2, 1, LightGrey)
	c.Px(9, 1, Brown)
	c.Px(10, 0, Brown)
	c.Outline(Ink)
	if lit {
		// crackling spark at the fuse tip
		sparks := [][][2]int{
			{{10, 0}, {11, 0}, {11, 1}, {12, 0}},
			{{10, 0}, {11, 1}, {12, 2}, {11, 0}, {12, 0}, {13, 1}},
			{{10, 0}, {9, 0}, {11, 0}, {11, 1}, {13, 0}, {12, 2}},
		}
		for i, p := range sparks[frame%3] {
			col := Yellow
			if i == 0 {
				col = White
			} else if (i+frame)%2 == 0 {
				col = Orange
			}
			c.Px(p[0], p[1], col)
		}
		// glowing rim when lit
		if frame%2 == 0 {
			c.Px(2, 9, Yellow)
			c.Px(13, 9, Yellow)
		}
	}
	return c.RGBA
}

func bombAnims() []Anim {
	return []Anim{
		{Name: "bomb", Frames: []*image.RGBA{bomb(false, 0)}, FPS: 1, Loop: true},
		{Name: "bomb_lit", Frames: []*image.RGBA{bomb(true, 0), bomb(true, 1), bomb(true, 2)}, FPS: 12, Loop: true},
	}
}

func orbSpin() []*image.RGBA {
	var out []*image.RGBA
	for f := 0; f < 4; f++ {
		c := NewCanvas(16, 16)
		shadedBall(c, 8, 8, 6, DarkGreen, Green, rgb(0xd8f8a0))
		// rotating equatorial band
		for x := 2; x < 14; x++ {
			dx := float64(x) + 0.5 - 8
			y := 8 + int(math.Round(dx*dx/14)) // tilted ring seen from above
			col := Orange
			if (x+f)%4 < 2 {
				col = Yellow
			}
			if c.At2(x, y).A != 0 {
				c.Px(x, y, col)
			}
		}
		c.Outline(Ink)
		out = append(out, c.RGBA)
	}
	return out
}

func coinSpin() []*image.RGBA {
	widths := []float64{6, 4.5, 1.5, 4.5}
	var out []*image.RGBA
	for _, w := range widths {
		c := NewCanvas(16, 16)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				dx, dy := (float64(x)+0.5-8)/w, (float64(y)+0.5-8)/6
				if dx*dx+dy*dy <= 1 {
					col := Gold
					if dx < -0.4 {
						col = Yellow
					} else if dx > 0.45 {
						col = GoldDark
					}
					c.Px(x, y, col)
				}
			}
		}
		if w > 4 {
			c.Rect(7, 5, 2, 6, GoldDark)
			c.Rect(7, 5, 1, 6, Yellow)
		}
		c.Outline(Ink)
		out = append(out, c.RGBA)
	}
	return out
}

// powerBall: bouncing letter ball. Flashes between two colour sets.
func powerBall(letter rune, a, b [3]color.RGBA) []*image.RGBA {
	var out []*image.RGBA
	for f, set := range [][3]color.RGBA{a, b} {
		c := NewCanvas(16, 16)
		shadedBall(c, 8, 8, 6.5, set[0], set[1], set[2])
		c.Outline(Ink)
		col := White
		if f == 1 {
			col = Yellow
		}
		drawGlyph(c, letter, 7, 6, Ink)
		drawGlyph(c, letter, 6, 5, col)
		out = append(out, c.RGBA)
	}
	return out
}

func sparkle() []*image.RGBA {
	var out []*image.RGBA
	for f := 0; f < 5; f++ {
		c := NewCanvas(16, 16)
		r := float64(f*2 + 1)
		for a := 0; a < 8; a++ {
			ang := float64(a) * math.Pi / 4
			x := 8 + math.Cos(ang)*r
			y := 8 + math.Sin(ang)*r
			col := Yellow
			if a%2 == 1 {
				col = White
			}
			if f >= 3 {
				col = Orange
			}
			c.Px(int(x), int(y), col)
			if f < 3 {
				c.Px(int(8+math.Cos(ang)*(r-1)), int(8+math.Sin(ang)*(r-1)), White)
			}
		}
		if f < 2 {
			c.Circle(8, 8, float64(2-f), White)
		}
		out = append(out, c.RGBA)
	}
	return out
}

// explosion used when Jack dies or an enemy spawns.
func explosion() []*image.RGBA {
	var out []*image.RGBA
	for f := 0; f < 5; f++ {
		c := NewCanvas(16, 16)
		r := 2.0 + float64(f)*1.6
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				dx, dy := float64(x)+0.5-8, float64(y)+0.5-8
				d := math.Sqrt(dx*dx+dy*dy) + hash(x, y, f)*2
				if d > r {
					continue
				}
				col := White
				switch {
				case d > r-1.5:
					col = DarkRed
				case d > r-3:
					col = Orange
				case d > r-4.5:
					col = Yellow
				}
				if f >= 3 && hash(x, y, 99+f) < float64(f-2)*0.35 {
					continue // smoke breaking up
				}
				c.Px(x, y, col)
			}
		}
		out = append(out, c.RGBA)
	}
	return out
}

// ------------------------------------------------------------- tiles ----

// LevelTheme holds the colours for a level's platforms.
type LevelTheme struct {
	Name                 string
	Light, Mid, Dark, Hi color.RGBA
}

var Themes = []LevelTheme{
	{"egypt", rgb(0xe8b860), rgb(0xc08838), rgb(0x704818), rgb(0xf8e0a0)},
	{"greece", rgb(0xd8d8e8), rgb(0xa0a8c0), rgb(0x585878), White},
	{"castle", rgb(0x9878c8), rgb(0x6848a0), rgb(0x302060), rgb(0xc8b0f0)},
	{"city", rgb(0x50c8d8), rgb(0x2888b0), rgb(0x104060), rgb(0xa0f0f8)},
}

// PlatformTile is an 8x8 tile; platforms are rows of them (left cap, middle, right cap).
func PlatformTiles(t LevelTheme) []*image.RGBA {
	var out []*image.RGBA
	for kind := 0; kind < 3; kind++ {
		c := NewCanvas(8, 8)
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				col := t.Mid
				switch {
				case y == 0 || y == 7:
					col = Ink
				case y == 1:
					col = t.Hi
				case y == 2:
					col = t.Light
				case y >= 5:
					col = Dither(x, y, t.Mid, t.Dark, float64(y-4)/3)
				}
				// brick seam
				if kind == 1 && x == 3 && y > 1 && y < 7 {
					col = t.Dark
				}
				if kind == 0 && x == 0 && y > 0 && y < 7 {
					col = Ink
				}
				if kind == 2 && x == 7 && y > 0 && y < 7 {
					col = Ink
				}
				c.Px(x, y, col)
			}
		}
		out = append(out, c.RGBA)
	}
	return out
}
