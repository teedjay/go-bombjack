package art

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 0xff}
}

// Master palette. Sprites are authored as ASCII where each rune maps to one
// of these colours ('.' is transparent). Keeping a small shared palette is
// what gives the graphics a coherent 16-bit console look.
var (
	Clear     = color.RGBA{}
	Ink       = rgb(0x140f1e) // outline
	White     = rgb(0xf8f8f8)
	LightGrey = rgb(0xb8c0d0)
	Grey      = rgb(0x8890a8)
	DarkGrey  = rgb(0x4a5068)
	Blue      = rgb(0x2850d8)
	LightBlue = rgb(0x68a0f8)
	NavyBlue  = rgb(0x182878)
	Red       = rgb(0xe02838)
	DarkRed   = rgb(0x881828)
	Yellow    = rgb(0xf8d830)
	Orange    = rgb(0xf88820)
	Skin      = rgb(0xf8b890)
	SkinShade = rgb(0xc87858)
	Cyan      = rgb(0x58e0f0)
	DarkCyan  = rgb(0x2080a0)
	Purple    = rgb(0xc048d8)
	DarkPurp  = rgb(0x682088)
	Bandage   = rgb(0xe8dcb0)
	BandShade = rgb(0xa89868)
	Brown     = rgb(0x8a5a2a)
	Green     = rgb(0x70d850)
	DarkGreen = rgb(0x287838)
	Pink      = rgb(0xf878b8)
	Gold      = rgb(0xf8c020)
	GoldDark  = rgb(0xb07010)
)

var paletteRunes = map[rune]color.RGBA{
	'.': Clear, 'K': Ink, 'W': White, 'w': LightGrey, 'G': Grey, 'g': DarkGrey,
	'B': Blue, 'b': LightBlue, 'D': NavyBlue, 'R': Red, 'r': DarkRed,
	'Y': Yellow, 'O': Orange, 'S': Skin, 's': SkinShade, 'C': Cyan, 'c': DarkCyan,
	'P': Purple, 'p': DarkPurp, 'T': Bandage, 't': BandShade, 'N': Brown,
	'L': Green, 'E': DarkGreen, 'M': Pink, 'A': Gold, 'a': GoldDark,
}

// FromASCII builds an image from rows of palette runes. Short rows are padded
// with transparency; long rows or unknown runes panic (authoring errors).
func FromASCII(w, h int, rows ...string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if len(rows) > h {
		panic(fmt.Sprintf("art: %d rows > height %d", len(rows), h))
	}
	for y, row := range rows {
		if len([]rune(row)) > w {
			panic(fmt.Sprintf("art: row %d %q wider than %d", y, row, w))
		}
		for x, r := range []rune(row) {
			col, ok := paletteRunes[r]
			if !ok {
				panic(fmt.Sprintf("art: unknown palette rune %q in %q", r, row))
			}
			img.SetRGBA(x, y, col)
		}
	}
	return img
}

// lines splits a raw string literal into rows, dropping blank edge lines and
// indentation tabs so sprites can be written as readable blocks.
func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// sprite16 is shorthand for a 16x16 ASCII sprite in a raw string block.
func sprite16(s string) *image.RGBA { return FromASCII(16, 16, lines(s)...) }
