package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/audio"
	"bombjack/internal/world"
)

// Difficulty is chosen in the menu. It is stored for later use; for now
// every difficulty plays the regular game.
type Difficulty int

const (
	Easy Difficulty = iota
	Normal
	Hard
)

// menuTimeout returns to the title after this long without input.
const menuTimeout = 30 * 60

// cursor turns held up/down (or left/right) input into single steps with
// auto-repeat, for menu navigation.
type cursor struct{ held int }

// step returns -1, 0 or +1 for this tick.
func (k *cursor) step(back, fwd bool) int {
	dir := 0
	switch {
	case back:
		dir = -1
	case fwd:
		dir = 1
	}
	if dir == 0 {
		k.held = 0
		return 0
	}
	k.held++
	if k.held == 1 || (k.held > 18 && k.held%6 == 0) {
		return dir
	}
	return 0
}

func anyInput(c world.Controls) bool {
	return c.Left || c.Right || c.Up || c.Down || c.Jump || c.Start || c.Confirm || c.Fire
}

// ------------------------------------------------------------------ menu --

var menuItems = []string{"EASY", "NORMAL", "HARD", "SOUND & FX"}

// Menu picks a difficulty (starting a game) or the sound test.
type Menu struct {
	sel  int
	t    int
	idle int
	nav  cursor
}

// NewMenu opens the menu with Normal preselected.
func NewMenu(g *Game) *Menu { return &Menu{sel: int(g.Difficulty)} }

func (m *Menu) Update(g *Game, c world.Controls) Scene {
	m.t++
	m.idle++
	if anyInput(c) {
		m.idle = 0
	}
	if m.idle >= menuTimeout {
		return NewTitle(g)
	}
	m.sel = (m.sel + m.nav.step(c.Up, c.Down) + len(menuItems)) % len(menuItems)
	if m.t > 10 && (c.Start || c.Confirm) {
		if m.sel == len(menuItems)-1 {
			return NewSoundTest(g)
		}
		g.Difficulty = Difficulty(m.sel)
		return NewSession(g)
	}
	return m
}

func (m *Menu) Draw(g *Game, screen *ebiten.Image) {
	drawTitleBackdrop(g, screen, g.Tick)
	g.centerText(screen, fmt.Sprintf("HI %07d", g.HiScore), 4, colYellow)
	panel(screen, 56, 96, ScreenW-112, 104)
	g.centerText(screen, "SELECT GAME", 104, colYellow)
	for i, item := range menuItems {
		y := float64(124 + i*14)
		if i == m.sel {
			g.centerText(screen, item, y, titleCycle[m.t/6%len(titleCycle)])
			w := float64(g.Sheet.TextWidth(item))
			if m.t/12%2 == 0 {
				g.Sheet.Text(screen, ">", (ScreenW-w)/2-14, y, colYellow)
			}
			// Jack runs beside the selection
			g.Sheet.Draw(screen, "jack_run", m.t, (ScreenW+w)/2+6, y-5, true)
		} else {
			g.centerText(screen, item, y, colWhite)
		}
	}
	g.centerText(screen, "ARROWS + ENTER", 186, colCyan)
}

// ------------------------------------------------------------ sound test --

var musicItems = []struct {
	name  string
	track int
}{
	{"TITLE", audio.TrackTitle},
	{"EGYPT", 0},
	{"GREECE", 1},
	{"CASTLE", 2},
	{"CITY", 3},
	{"VOLCANO", 4},
	{"ICELAND", 5},
	{"HI-SCORE", audio.TrackHiScore},
	{"STOP", audio.TrackNone},
}

// SoundTest lets the player audition every tune and sound effect. Esc goes
// back to the title (handled in Game.Update).
type SoundTest struct {
	col, row [2]int // col 0: music, 1: sound fx; row per column
	cur      int    // active column
	t        int
	nav, lr  cursor
	flash    int // ticks to highlight the item just played
}

func NewSoundTest(g *Game) *SoundTest { return &SoundTest{} }

func (st *SoundTest) Update(g *Game, c world.Controls) Scene {
	st.t++
	st.flash = max(st.flash-1, 0)
	if d := st.lr.step(c.Left, c.Right); d != 0 {
		st.cur = (st.cur + d + 2) % 2
	}
	n := len(musicItems)
	if st.cur == 1 {
		n = len(audio.SoundTest)
	}
	st.row[st.cur] = (st.row[st.cur] + st.nav.step(c.Up, c.Down) + n) % n
	if st.t > 10 && (c.Start || c.Confirm) {
		st.flash = 20
		if st.cur == 0 {
			if tr := musicItems[st.row[0]].track; tr == audio.TrackNone {
				g.Audio.StopMusic()
			} else {
				g.Audio.PlayMusic(tr)
			}
		} else {
			g.Audio.PlayTest(st.row[1])
		}
	}
	return st
}

func (st *SoundTest) Draw(g *Game, screen *ebiten.Image) {
	screen.Fill(colBlack)
	g.Sheet.DrawBackground(screen, 3, HUDH, 0, st.t*3)
	panel(screen, 8, 6, ScreenW-16, ScreenH-12)
	g.centerText(screen, "SOUND TEST", 14, colYellow)

	cols := []struct {
		title string
		x     float64
		items []string
	}{
		{"MUSIC", 20, nil},
		{"SOUND FX", 132, nil},
	}
	for _, m := range musicItems {
		cols[0].items = append(cols[0].items, m.name)
	}
	for _, s := range audio.SoundTest {
		cols[1].items = append(cols[1].items, s.Name)
	}
	playing := g.Audio.Track()
	for ci, col := range cols {
		hcol := color.Color(colCyan)
		if ci == st.cur {
			hcol = colYellow
		}
		g.Sheet.Text(screen, col.title, col.x, 34, hcol)
		for i, name := range col.items {
			y := float64(50 + i*13)
			c := color.Color(colWhite)
			if ci == 0 && musicItems[i].track == playing && playing != audio.TrackNone {
				c = colGreen // currently playing tune
				if st.t/15%2 == 0 {
					g.Sheet.Text(screen, ":", col.x+float64(g.Sheet.TextWidth(name))+4, y, colGreen)
				}
			}
			if ci == st.cur && i == st.row[ci] {
				c = colYellow
				if st.flash > 0 && st.flash/3%2 == 0 {
					c = colWhite
				}
				g.Sheet.Text(screen, ">", col.x-10, y, colYellow)
			}
			g.Sheet.Text(screen, name, col.x, y, c)
		}
	}
	if g.Audio.Muted {
		g.centerText(screen, "MUTED - PRESS M", 206, colRed)
	}
	g.centerText(screen, "ARROWS  ENTER=PLAY", 218, colCyan)
	g.centerText(screen, "ESC BACK TO TITLE", 228, colCyan)
}
