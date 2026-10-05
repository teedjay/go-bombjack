// Package gfx turns internal/art images into ebiten images and plays
// animations. Owned by WS1.
package gfx

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/art"
)

type anim struct {
	frames []*ebiten.Image
	fps    float64
	loop   bool
}

// Sheet caches every animation and background as GPU images.
type Sheet struct {
	anims map[string]anim
	bgs   []bgLayers
	font  map[rune]*ebiten.Image
	logo  *ebiten.Image
	// logoEdge lists opaque logo pixels next to transparency (for sparkles).
	logoEdge []image.Point
	icons    map[string]*ebiten.Image
}

// Load converts all art once; call at startup.
func Load() *Sheet {
	s := &Sheet{anims: map[string]anim{}}
	for _, a := range art.Animations() {
		var fr []*ebiten.Image
		for _, f := range a.Frames {
			fr = append(fr, ebiten.NewImageFromImage(f))
		}
		s.anims[a.Name] = anim{fr, a.FPS, a.Loop}
	}
	for _, l := range art.BackgroundLayers() {
		b := bgLayers{far: ebiten.NewImageFromImage(l.Far), near: ebiten.NewImageFromImage(l.Near)}
		if l.Clouds != nil {
			b.clouds = ebiten.NewImageFromImage(l.Clouds)
		}
		s.bgs = append(s.bgs, b)
	}
	s.font = map[rune]*ebiten.Image{}
	for r, g := range art.Font() {
		s.font[r] = ebiten.NewImageFromImage(g)
	}
	logo := art.Logo()
	s.logo = ebiten.NewImageFromImage(logo)
	b := logo.Bounds()
	opaque := func(x, y int) bool { return image.Pt(x, y).In(b) && logo.RGBAAt(x, y).A != 0 }
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if opaque(x, y) && (!opaque(x-1, y) || !opaque(x+1, y) || !opaque(x, y-1) || !opaque(x, y+1)) {
				s.logoEdge = append(s.logoEdge, image.Pt(x-b.Min.X, y-b.Min.Y))
			}
		}
	}
	s.icons = map[string]*ebiten.Image{}
	for n, im := range art.Icons() {
		s.icons[n] = ebiten.NewImageFromImage(im)
	}
	return s
}

type bgLayers struct{ far, clouds, near *ebiten.Image }

// DrawBackground draws a level backdrop at y offsetY. parallax in -1..1
// shifts the far layer (pass e.g. Jack's position relative to the centre);
// the cloud layer drifts slowly with tick and wraps around.
func (s *Sheet) DrawBackground(dst *ebiten.Image, level int, offsetY, parallax float64, tick int) {
	b := s.bgs[level%len(s.bgs)]
	parallax = math.Max(-1, math.Min(1, parallax))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(-art.ParallaxMargin-parallax*art.ParallaxMargin), offsetY)
	dst.DrawImage(b.far, op)
	if b.clouds != nil {
		w := float64(b.clouds.Bounds().Dx())
		x := -math.Mod(float64(tick)*0.08, w)
		for _, dx := range []float64{x, x + w} {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(math.Round(dx), offsetY)
			dst.DrawImage(b.clouds, op)
		}
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, offsetY)
	dst.DrawImage(b.near, op)
}

// Frame returns the frame of animation name to show after tick ticks
// (60 ticks per second). Unknown names panic: names are a contract.
func (s *Sheet) Frame(name string, tick int) *ebiten.Image {
	a, ok := s.anims[name]
	if !ok {
		panic("gfx: unknown animation " + name)
	}
	i := int(float64(tick) * a.fps / 60)
	if a.loop {
		i %= len(a.frames)
	} else {
		i = min(i, len(a.frames)-1)
	}
	return a.frames[i]
}

// FrameCount reports the number of frames in an animation.
func (s *Sheet) FrameCount(name string) int { return len(s.anims[name].frames) }

// Draw draws a frame at (x,y) in screen pixels, mirrored if flip.
func (s *Sheet) Draw(dst *ebiten.Image, name string, tick int, x, y float64, flip bool) {
	s.drawScaled(dst, name, tick, x, y, flip, ebiten.ColorScale{})
}

// DrawTinted is Draw with the frame multiplied by colour c.
func (s *Sheet) DrawTinted(dst *ebiten.Image, name string, tick int, x, y float64, flip bool, c color.Color) {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	s.drawScaled(dst, name, tick, x, y, flip, cs)
}

func (s *Sheet) drawScaled(dst *ebiten.Image, name string, tick int, x, y float64, flip bool, cs ebiten.ColorScale) {
	img := s.Frame(name, tick)
	op := &ebiten.DrawImageOptions{ColorScale: cs}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(img.Bounds().Dx()), 0)
	}
	op.GeoM.Translate(x, y)
	dst.DrawImage(img, op)
}

// DrawPlatform draws a run of n 8x8 tiles in the level's theme.
func (s *Sheet) DrawPlatform(dst *ebiten.Image, level int, x, y float64, n int) {
	tiles := s.anims["platform_"+art.Themes[level%len(art.Themes)].Name].frames
	for i := 0; i < n; i++ {
		t := tiles[1]
		if i == 0 {
			t = tiles[0]
		} else if i == n-1 {
			t = tiles[2]
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x+float64(i*8), y)
		dst.DrawImage(t, op)
	}
}

// LogoEdge returns logo outline pixels (logo-local coords) for sparkles.
func (s *Sheet) LogoEdge() []image.Point { return s.logoEdge }

// DrawRotated draws a frame centred on (cx,cy), rotated by angle radians
// (0 = as drawn, clockwise positive since y points down).
func (s *Sheet) DrawRotated(dst *ebiten.Image, name string, tick int, cx, cy, angle float64) {
	img := s.Frame(name, tick)
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(math.Round(cx), math.Round(cy))
	dst.DrawImage(img, op)
}

// MissileIcon is the 8x8 HUD missile.
func (s *Sheet) MissileIcon() *ebiten.Image { return s.icons["missile"] }
