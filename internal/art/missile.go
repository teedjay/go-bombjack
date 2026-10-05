package art

import "image"

// missileBody is the rocket facing right, 16x9; columns 0-1 hold the
// exhaust flame, which is filled in per frame.
var missileBody = []string{
	"..KK............",
	"..KRK...........",
	"..KRRKKKKKKKK...",
	"..KWWWWCWWWRRK..",
	"..KwWWWWWWWRRRRK",
	"..KwwwwwwwwRRK..",
	"..KRRKKKKKKKK...",
	"..KRK...........",
	"..KK............",
}

// missileFrames returns the cartoon homing missile with a flickering flame.
func missileFrames() []*image.RGBA {
	flames := [][3]string{
		{"OY", "YW", "OY"},
		{".O", "OY", ".O"},
		{"YO", "WY", "YO"},
		{"O.", "YY", "O."},
	}
	var out []*image.RGBA
	for _, f := range flames {
		rows := append([]string(nil), missileBody...)
		for i := 0; i < 3; i++ {
			rows[3+i] = f[i] + rows[3+i][2:]
		}
		out = append(out, FromASCII(16, 9, rows...))
	}
	return out
}

// missileIcon is the 8x8 HUD / pickup icon.
func missileIcon() *image.RGBA {
	return FromASCII(8, 8,
		"........",
		"K.......",
		"KRKKKK..",
		"KWWCWRRK",
		"KwwwwRRK",
		"KRKKKK..",
		"K.......",
		"........")
}

// missileCrate is the pickup: an army crate with hazard stripes and a
// missile stencil; the second frame bobs the icon and adds a glint.
func missileCrate() []*image.RGBA {
	var out []*image.RGBA
	for f := 0; f < 2; f++ {
		c := NewCanvas(16, 16)
		c.Rect(1, 2, 14, 13, DarkGreen)
		c.Rect(1, 2, 14, 2, Green)
		for x := 1; x < 15; x++ { // hazard stripes on the lid
			if (x/2)%2 == f {
				c.Px(x, 4, Yellow)
			} else {
				c.Px(x, 4, Ink)
			}
		}
		c.Rect(2, 14, 13, 1, rgb(0x184820))
		c.Blit(missileIcon(), 4, 6-f)
		if f == 1 {
			c.Px(13, 3, White)
			c.Px(12, 2, Yellow)
		}
		c.Outline(Ink)
		out = append(out, c.RGBA)
	}
	return out
}
