package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/world"
)

// Autoshot is a dev/QA aid. With BOMBJACK_SHOTS="dir:round:t1,t2,..." the
// game starts round `round` directly, plays with scripted input, saves a
// screenshot to dir at each listed tick and exits after the last one.
type Autoshot struct {
	dir   string
	ticks []int
	idle  bool // no input (always for round < 0, which stays on the title)
}

func autoshotFromEnv(g *Game) *Autoshot {
	v := os.Getenv("BOMBJACK_SHOTS")
	if v == "" {
		return nil
	}
	parts := strings.SplitN(v, ":", 3)
	if len(parts) != 3 {
		return nil
	}
	// "3i" = round 3 with no input (idle), e.g. to see the hint flash
	idle := strings.HasSuffix(parts[1], "i")
	round, _ := strconv.Atoi(strings.TrimSuffix(parts[1], "i"))
	a := &Autoshot{dir: parts[0], idle: idle}
	for _, s := range strings.Split(parts[2], ",") {
		if n, err := strconv.Atoi(s); err == nil {
			a.ticks = append(a.ticks, n)
		}
	}
	_ = os.MkdirAll(a.dir, 0o755)
	g.Audio.ToggleMute()
	if round < 0 { // -1 title, -2 menu, -3 sound test; no input
		a.idle = true
		switch round {
		case -2:
			g.scene = NewMenu(g)
		case -3:
			g.scene = NewSoundTest(g)
		}
		return a
	}
	d := NewSession(g)
	for range round {
		d.Rules.NextRound(nil)
	}
	pl := newRound(g, d.Rules)
	pl.World.Invincible = true // autopilot never dies, so later events get captured
	g.scene = pl
	return a
}

// controls is a simple scripted pilot: wander, jump, float.
func (a *Autoshot) controls(tick int) world.Controls {
	if a.idle {
		return world.Controls{}
	}
	phase := tick % 240
	return world.Controls{
		Left:        phase < 100,
		Right:       phase >= 120 && phase < 220,
		Jump:        tick%90 < 20,
		JumpPressed: tick%90 == 0 || tick%90 == 45,
		Fire:        tick%240 == 200,
	}
}

func (a *Autoshot) capture(g *Game, screen *ebiten.Image) error {
	if len(a.ticks) == 0 || g.Tick < a.ticks[0] {
		return nil
	}
	a.ticks = a.ticks[1:]
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	f, err := os.Create(filepath.Join(a.dir, fmt.Sprintf("shot_%05d.png", g.Tick)))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	if len(a.ticks) == 0 {
		return ErrQuit
	}
	return nil
}
