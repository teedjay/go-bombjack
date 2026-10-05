// Package art generates all game graphics in code: palette-indexed ASCII
// sprites, procedural sprites and procedural backgrounds. It depends only on
// the standard library so it can be used both by the game (converted to
// ebiten images at startup) and by cmd/spritegen (written to PNG files).
package art

import (
	"image"
	"image/color"
	"math"
)

// Canvas is a thin wrapper around image.RGBA with retro drawing helpers.
type Canvas struct {
	*image.RGBA
	W, H int
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{RGBA: image.NewRGBA(image.Rect(0, 0, w, h)), W: w, H: h}
}

func (c *Canvas) In(x, y int) bool { return x >= 0 && y >= 0 && x < c.W && y < c.H }

func (c *Canvas) Px(x, y int, col color.RGBA) {
	if c.In(x, y) {
		c.SetRGBA(x, y, col)
	}
}

func (c *Canvas) At2(x, y int) color.RGBA {
	if !c.In(x, y) {
		return color.RGBA{}
	}
	return c.RGBAAt(x, y)
}

func (c *Canvas) Rect(x, y, w, h int, col color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c.Px(i, j, col)
		}
	}
}

// Circle fills a disc. Using a .5 centre gives even-diameter circles.
func (c *Canvas) Circle(cx, cy, r float64, col color.RGBA) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy <= r*r {
				c.Px(x, y, col)
			}
		}
	}
}

// Tri fills a triangle.
func (c *Canvas) Tri(x0, y0, x1, y1, x2, y2 float64, col color.RGBA) {
	minX := int(math.Floor(math.Min(x0, math.Min(x1, x2))))
	maxX := int(math.Ceil(math.Max(x0, math.Max(x1, x2))))
	minY := int(math.Floor(math.Min(y0, math.Min(y1, y2))))
	maxY := int(math.Ceil(math.Max(y0, math.Max(y1, y2))))
	edge := func(ax, ay, bx, by, px, py float64) float64 { return (bx-ax)*(py-ay) - (by-ay)*(px-ax) }
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			a := edge(x0, y0, x1, y1, px, py)
			b := edge(x1, y1, x2, y2, px, py)
			d := edge(x2, y2, x0, y0, px, py)
			if (a >= 0 && b >= 0 && d >= 0) || (a <= 0 && b <= 0 && d <= 0) {
				c.Px(x, y, col)
			}
		}
	}
}

var bayer4 = [4][4]float64{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// Dither returns a or b depending on t (0..1) and the ordered-dither threshold
// at (x,y). This gives the classic 16-bit "checkerboard blend" look.
func Dither(x, y int, a, b color.RGBA, t float64) color.RGBA {
	if t > (bayer4[y&3][x&3]+0.5)/16 {
		return b
	}
	return a
}

// VGradient fills rows y0..y1 with a dithered blend through the given colours.
func (c *Canvas) VGradient(y0, y1 int, cols ...color.RGBA) {
	n := len(cols) - 1
	for y := y0; y < y1; y++ {
		f := float64(y-y0) / float64(y1-y0) * float64(n)
		i := int(f)
		if i >= n {
			i = n - 1
		}
		t := f - float64(i)
		// quantise t to 5 steps so bands stay crisp
		t = math.Round(t*4) / 4
		for x := 0; x < c.W; x++ {
			c.Px(x, y, Dither(x, y, cols[i], cols[i+1], t))
		}
	}
}

// Lerp blends two colours linearly (t clamped to 0..1).
func Lerp(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	m := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 0xff}
}

// VGradientSmooth fills rows y0..y1 with a smooth (undithered) blend through
// the given colours. Used for backdrops; sprites keep the dithered look.
func (c *Canvas) VGradientSmooth(y0, y1 int, cols ...color.RGBA) {
	n := len(cols) - 1
	for y := y0; y < y1; y++ {
		f := float64(y-y0) / float64(y1-y0) * float64(n)
		i := min(int(f), n-1)
		col := Lerp(cols[i], cols[i+1], f-float64(i))
		for x := 0; x < c.W; x++ {
			c.Px(x, y, col)
		}
	}
}

// Outline draws col on every transparent pixel that touches an opaque one.
func (c *Canvas) Outline(col color.RGBA) {
	var pts []image.Point
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if c.At2(x, y).A != 0 {
				continue
			}
			if c.At2(x+1, y).A != 0 || c.At2(x-1, y).A != 0 || c.At2(x, y+1).A != 0 || c.At2(x, y-1).A != 0 {
				pts = append(pts, image.Pt(x, y))
			}
		}
	}
	for _, p := range pts {
		c.SetRGBA(p.X, p.Y, col)
	}
}

// Blit copies src onto c at (dx,dy), skipping transparent pixels.
func (c *Canvas) Blit(src *image.RGBA, dx, dy int) {
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := src.RGBAAt(x, y)
			if p.A != 0 {
				c.Px(dx+x-b.Min.X, dy+y-b.Min.Y, p)
			}
		}
	}
}

// FlipH returns a horizontally mirrored copy.
func FlipH(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.SetRGBA(b.Max.X-1-(x-b.Min.X), y, src.RGBAAt(x, y))
		}
	}
	return dst
}

// Rot90 rotates a square image clockwise n quarter turns.
func Rot90(src *image.RGBA, n int) *image.RGBA {
	out := src
	for k := 0; k < ((n%4)+4)%4; k++ {
		b := out.Bounds()
		w, h := b.Dx(), b.Dy()
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetRGBA(h-1-y, x, out.RGBAAt(x, y))
			}
		}
		out = dst
	}
	return out
}

// hash gives a cheap deterministic pseudo random value in [0,1).
func hash(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffff) / 65536
}
