// Package gfx turns internal/art images into ebiten images and plays
// animations. Owned by WS1.
package gfx

import (
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
	bgs   []*ebiten.Image
	font  map[rune]*ebiten.Image
	logo  *ebiten.Image
	icons map[string]*ebiten.Image
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
	for _, bg := range art.Backgrounds() {
		s.bgs = append(s.bgs, ebiten.NewImageFromImage(bg))
	}
	s.font = map[rune]*ebiten.Image{}
	for r, g := range art.Font() {
		s.font[r] = ebiten.NewImageFromImage(g)
	}
	s.logo = ebiten.NewImageFromImage(art.Logo())
	s.icons = map[string]*ebiten.Image{}
	for n, im := range art.Icons() {
		s.icons[n] = ebiten.NewImageFromImage(im)
	}
	return s
}

func (s *Sheet) Background(level int) *ebiten.Image { return s.bgs[level%len(s.bgs)] }

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
	img := s.Frame(name, tick)
	op := &ebiten.DrawImageOptions{}
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
