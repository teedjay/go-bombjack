package gfx

import (
	"image/color"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/art"
)

// TextWidth is the pixel width of s (8 px per glyph).
func TextWidth(s string) int { return art.FontCell * len([]rune(s)) }

// TextWidth is the pixel width of s (8 px per glyph).
func (s *Sheet) TextWidth(str string) int { return TextWidth(str) }

// Text draws s with the 8x8 bitmap font, tinted col, with a 1px Ink drop
// shadow. Lowercase is upper-cased; unknown runes are skipped.
func (s *Sheet) Text(dst *ebiten.Image, str string, x, y float64, col color.Color) {
	str = strings.Map(unicode.ToUpper, str)
	cr, cg, cb, ca := col.RGBA()
	for pass := 0; pass < 2; pass++ {
		px := x
		for _, r := range str {
			g, ok := s.font[r]
			if r == 'X' && !ok {
				g = s.font['x']
			}
			if g != nil && r != ' ' {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(px, y)
				if pass == 0 {
					op.GeoM.Translate(1, 1)
					op.ColorScale.Scale(0.08, 0.06, 0.12, float32(ca)/0xffff)
				} else {
					a := float32(ca) / 0xffff
					if ca > 0 {
						op.ColorScale.Scale(float32(cr)/float32(ca), float32(cg)/float32(ca), float32(cb)/float32(ca), a)
					}
				}
				dst.DrawImage(g, op)
			}
			px += float64(art.FontCell)
		}
	}
}

// Logo returns the title logo image.
func (s *Sheet) Logo() *ebiten.Image { return s.logo }

// LifeIcon, MeterEmpty and MeterFull return 8x8 HUD icons.
func (s *Sheet) LifeIcon() *ebiten.Image   { return s.icons["life"] }
func (s *Sheet) MeterEmpty() *ebiten.Image { return s.icons["meter_empty"] }
func (s *Sheet) MeterFull() *ebiten.Image  { return s.icons["meter_full"] }
