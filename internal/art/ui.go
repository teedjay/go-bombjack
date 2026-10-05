package art

import (
	"image"
	"strings"
	"sync"
)

// FontCell is the size of one font glyph cell in pixels.
const FontCell = 8

// glyphs are 5x7 bit rows ('#' set), drawn at the top-left of an 8x8 cell so
// a 1px drop shadow still fits inside the cell.
var fontGlyphs = map[rune]string{
	'0': ".###.|#...#|#..##|#.#.#|##..#|#...#|.###.",
	'1': "..#..|.##..|..#..|..#..|..#..|..#..|.###.",
	'2': ".###.|#...#|....#|...#.|..#..|.#...|#####",
	'3': "####.|....#|....#|.###.|....#|....#|####.",
	'4': "...#.|..##.|.#.#.|#..#.|#####|...#.|...#.",
	'5': "#####|#....|####.|....#|....#|#...#|.###.",
	'6': ".###.|#....|#....|####.|#...#|#...#|.###.",
	'7': "#####|....#|...#.|..#..|.#...|.#...|.#...",
	'8': ".###.|#...#|#...#|.###.|#...#|#...#|.###.",
	'9': ".###.|#...#|#...#|.####|....#|....#|.###.",
	'A': ".###.|#...#|#...#|#####|#...#|#...#|#...#",
	'B': "####.|#...#|#...#|####.|#...#|#...#|####.",
	'C': ".###.|#...#|#....|#....|#....|#...#|.###.",
	'D': "####.|#...#|#...#|#...#|#...#|#...#|####.",
	'E': "#####|#....|#....|####.|#....|#....|#####",
	'F': "#####|#....|#....|####.|#....|#....|#....",
	'G': ".###.|#...#|#....|#.###|#...#|#...#|.###.",
	'H': "#...#|#...#|#...#|#####|#...#|#...#|#...#",
	'I': ".###.|..#..|..#..|..#..|..#..|..#..|.###.",
	'J': "..###|...#.|...#.|...#.|...#.|#..#.|.##..",
	'K': "#...#|#..#.|#.#..|##...|#.#..|#..#.|#...#",
	'L': "#....|#....|#....|#....|#....|#....|#####",
	'M': "#...#|##.##|#.#.#|#.#.#|#...#|#...#|#...#",
	'N': "#...#|##..#|#.#.#|#..##|#...#|#...#|#...#",
	'O': ".###.|#...#|#...#|#...#|#...#|#...#|.###.",
	'P': "####.|#...#|#...#|####.|#....|#....|#....",
	'Q': ".###.|#...#|#...#|#...#|#.#.#|#..#.|.##.#",
	'R': "####.|#...#|#...#|####.|#.#..|#..#.|#...#",
	'S': ".####|#....|#....|.###.|....#|....#|####.",
	'T': "#####|..#..|..#..|..#..|..#..|..#..|..#..",
	'U': "#...#|#...#|#...#|#...#|#...#|#...#|.###.",
	'V': "#...#|#...#|#...#|#...#|#...#|.#.#.|..#..",
	'W': "#...#|#...#|#...#|#.#.#|#.#.#|##.##|#...#",
	'X': "#...#|#...#|.#.#.|..#..|.#.#.|#...#|#...#",
	'Y': "#...#|#...#|.#.#.|..#..|..#..|..#..|..#..",
	'Z': "#####|....#|...#.|..#..|.#...|#....|#####",
	' ': ".....|.....|.....|.....|.....|.....|.....",
	'-': ".....|.....|.....|#####|.....|.....|.....",
	'!': "..#..|..#..|..#..|..#..|..#..|.....|..#..",
	'.': ".....|.....|.....|.....|.....|.##..|.##..",
	':': ".....|.##..|.##..|.....|.##..|.##..|.....",
	'x': ".....|.....|#...#|.#.#.|..#..|.#.#.|#...#",
	'©': ".###.|#...#|#.##.|#.#..|#.##.|#...#|.###.",
}

var (
	fontOnce   sync.Once
	fontWhite  map[rune]*image.RGBA
	fontShadow map[rune]*image.RGBA
)

func buildFonts() {
	fontWhite = map[rune]*image.RGBA{}
	fontShadow = map[rune]*image.RGBA{}
	for r, def := range fontGlyphs {
		w := image.NewRGBA(image.Rect(0, 0, FontCell, FontCell))
		for y, row := range strings.Split(def, "|") {
			for x, ch := range row {
				if ch == '#' {
					w.SetRGBA(x, y, White)
				}
			}
		}
		s := image.NewRGBA(image.Rect(0, 0, FontCell, FontCell))
		for y := 0; y < FontCell; y++ {
			for x := 0; x < FontCell; x++ {
				if w.RGBAAt(x, y).A != 0 {
					if x+1 < FontCell && y+1 < FontCell {
						s.SetRGBA(x+1, y+1, Ink)
					}
				}
			}
		}
		for y := 0; y < FontCell; y++ {
			for x := 0; x < FontCell; x++ {
				if p := w.RGBAAt(x, y); p.A != 0 {
					s.SetRGBA(x, y, p)
				}
			}
		}
		fontWhite[r] = w
		fontShadow[r] = s
	}
}

// Font returns white 8x8 glyphs (0-9 A-Z space - ! . : x ©). Shared; do not mutate.
func Font() map[rune]*image.RGBA {
	fontOnce.Do(buildFonts)
	return fontWhite
}

// FontShadow returns the same glyphs with a 1px Ink drop shadow baked in.
func FontShadow() map[rune]*image.RGBA {
	fontOnce.Do(buildFonts)
	return fontShadow
}

// Logo returns the "BOMB JACK" title logo: 2x scaled letters with a
// yellow-to-orange gradient, a dark red under-shadow and a 2px Ink outline.
func Logo() *image.RGBA {
	const text = "BOMB JACK"
	const adv = 12
	w, h := len(text)*adv+8, 28
	fill := NewCanvas(w, h)
	f := Font()
	for i, r := range text {
		g := f[r]
		for y := 0; y < 7; y++ {
			for x := 0; x < 5; x++ {
				if g.RGBAAt(x, y).A != 0 {
					fill.Rect(4+i*adv+x*2, 4+y*2, 2, 2, White)
				}
			}
		}
	}
	// gradient fill + thick bottom shade
	out := NewCanvas(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if fill.RGBAAt(x, y).A == 0 {
				continue
			}
			t := float64(y-4) / 14
			col := Lerp(Yellow, Orange, t)
			if y >= 4+12 {
				col = Lerp(Orange, Red, 0.5)
			}
			if y == 4 {
				col = White
			}
			out.SetRGBA(x, y, col)
		}
	}
	// under-shadow
	sh := NewCanvas(w, h)
	for y := h - 2; y >= 2; y-- {
		for x := 0; x < w; x++ {
			if out.RGBAAt(x, y-2).A != 0 && out.RGBAAt(x, y).A == 0 {
				sh.SetRGBA(x, y, DarkRed)
			}
		}
	}
	out.Blit(sh.RGBA, 0, 0)
	out.Outline(Ink)
	out.Outline(Ink)
	return out.RGBA
}

// Icons returns small 8x8 HUD icons: "life", "meter_empty", "meter_full".
func Icons() map[string]*image.RGBA {
	e := func(rows ...string) *image.RGBA { return FromASCII(8, 8, rows...) }
	return map[string]*image.RGBA{
		"life": e(
			".KKKKK..",
			"KbbbbbK.",
			"KbSSSbK.",
			"KbWSWbK.",
			".KSSSK..",
			"KRRRRRK.",
			"KRKKKRK.",
			".KK.KK.."),
		"meter_empty": e(
			"KKKKKKKK",
			"KggggggK",
			"KggggggK",
			"KggggggK",
			"KggggggK",
			"KggggggK",
			"KggggggK",
			"KKKKKKKK"),
		"meter_full": e(
			"KKKKKKKK",
			"KWYYYYYK",
			"KYYYYYOK",
			"KYYYYOOK",
			"KYYYOOOK",
			"KYYOOOaK",
			"KOOOOaaK",
			"KKKKKKKK"),
	}
}

func jackLand() []*image.RGBA {
	return []*image.RGBA{sprite16(`
		................
		................
		................
		....K.....K.....
		...KbKKKKKbK....
		..KbbbbbbbbbbK..
		..KbWKbbbWKbbK..
		..KbSSSSSSSSbK..
		...KSSSSrSSSK...
		..RKBBBBBBBBKR..
		.RRKBYYYYYYBKRR.
		.rRKBBBBBBBBKRr.
		..rKRRRKKRRRKr..
		..KKRRRRRRRRKK..
		..KKKKKKKKKKKK..
		................`)}
}

func jackTurn() []*image.RGBA {
	return []*image.RGBA{sprite16(`
		....K.....K.....
		...KbK...KbK....
		...KbbKKKbbK....
		...KbbbbbbbbK...
		...KbWKbbWKbK...
		...KbSSSSSSSK...
		....KSSSrSSK....
		.....KKKKKK.....
		....RKBBBBKR....
		....RKBYYBKR....
		....rKBBBBKr....
		.....KBBBBK.....
		.....KBKKBK.....
		.....KRK.KRK....
		....KRRK.KRRK...
		....KKKK.KKKK...`)}
}
