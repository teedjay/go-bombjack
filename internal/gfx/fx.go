package gfx

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Animator plays a named animation from tick 0. Zero value is usable after
// setting Name.
type Animator struct {
	Name string
	T    int
}

func (a *Animator) Update() { a.T++ }

// Draw draws the current frame at (x,y).
func (a *Animator) Draw(s *Sheet, dst *ebiten.Image, x, y float64, flip bool) {
	s.Draw(dst, a.Name, a.T, x, y, flip)
}

// Done reports whether a non-looping animation has shown its last frame.
func (a *Animator) Done(s *Sheet) bool {
	an, ok := s.anims[a.Name]
	if !ok || an.loop || an.fps == 0 {
		return false
	}
	return int(float64(a.T)*an.fps/60) >= len(an.frames)
}

const popupLife = 45

type fxItem struct {
	Animator
	x, y float64
	tint *ebiten.ColorScale // nil = untinted
}

type popup struct {
	text string
	x, y float64
	t    int
}

// FX holds one-shot animations and floating score popups.
type FX struct {
	sheet  *Sheet
	items  []fxItem
	popups []popup
	parts  []particle
}

// NewFX creates an effect list drawing from sheet.
func NewFX(s *Sheet) *FX { return &FX{sheet: s} }

// Spawn starts a one-shot animation with its top-left at (x,y).
func (f *FX) Spawn(name string, x, y float64) {
	f.items = append(f.items, fxItem{Animator: Animator{Name: name}, x: x, y: y})
}

// SpawnTinted starts a one-shot animation multiplied by colour c.
func (f *FX) SpawnTinted(name string, x, y float64, c color.Color) {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	f.items = append(f.items, fxItem{Animator: Animator{Name: name}, x: x, y: y, tint: &cs})
}

// Popup starts a floating score popup centred on (x,y).
func (f *FX) Popup(value int, x, y float64) {
	f.popups = append(f.popups, popup{text: fmt.Sprint(value), x: x, y: y})
}

// Update advances effects by one tick and drops finished ones.
func (f *FX) Update() {
	n := 0
	for i := range f.items {
		f.items[i].Update()
		if !f.items[i].Done(f.sheet) {
			f.items[n] = f.items[i]
			n++
		}
	}
	f.items = f.items[:n]
	n = 0
	for i := range f.popups {
		f.popups[i].t++
		if f.popups[i].t < popupLife {
			f.popups[n] = f.popups[i]
			n++
		}
	}
	f.popups = f.popups[:n]
	f.updateParticles()
}

// Draw renders all effects; offsetY shifts them down (e.g. below the HUD).
func (f *FX) Draw(dst *ebiten.Image, offsetY float64) {
	for _, it := range f.items {
		if it.tint != nil {
			f.sheet.drawScaled(dst, it.Name, it.T, it.x, it.y+offsetY, false, *it.tint)
		} else {
			it.Draw(f.sheet, dst, it.x, it.y+offsetY, false)
		}
	}
	f.drawParticles(dst, offsetY)
	for _, p := range f.popups {
		a := 1.0
		if p.t > popupLife/2 {
			a = 1 - float64(p.t-popupLife/2)/float64(popupLife-popupLife/2)
		}
		col := color.NRGBA{255, 216, 48, uint8(255 * a)}
		w := float64(TextWidth(p.text))
		f.sheet.Text(dst, p.text, p.x-w/2, p.y+offsetY-float64(p.t)*0.4, col)
	}
}

// Empty reports whether no effects are active.
func (f *FX) Empty() bool { return len(f.items) == 0 && len(f.popups) == 0 && len(f.parts) == 0 }
