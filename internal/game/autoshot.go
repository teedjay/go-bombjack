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

	"bombjack/internal/enemy"
	"bombjack/internal/rules"
	"bombjack/internal/world"
)

// Autoshot is a dev/QA aid. With BOMBJACK_SHOTS="dir:round:t1,t2,..." the
// game starts round `round` directly, plays with scripted input, saves a
// screenshot to dir at each listed tick and exits after the last one.
type Autoshot struct {
	dir   string
	ticks []int
	idle  bool // no input (always for round < 0, which stays on the title)
	show  bool // showcase: crowded screen and missile salvoes (README shots)
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
	// "3i" = round 3 with no input (idle), e.g. to see the hint flash;
	// "4x" = round 4 as a showcase: lots of enemies and missile salvoes
	idle := strings.HasSuffix(parts[1], "i")
	show := strings.HasSuffix(parts[1], "x")
	round, _ := strconv.Atoi(strings.TrimRight(parts[1], "ix"))
	a := &Autoshot{dir: parts[0], idle: idle, show: show}
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
	pl.still = show            // showcase stills: no shake or white flash
	g.scene = pl
	return a
}

// controls is a simple scripted pilot: wander, jump, float.
func (a *Autoshot) controls(tick int) world.Controls {
	if a.idle {
		return world.Controls{}
	}
	if a.show { // stand still at the bottom and fire bursts of 4 missiles
		return world.Controls{Fire: tick > 240 && tick%120 < 24 && tick%6 == 0}
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

// showcase keeps the screen busy for promotional shots: a crowd of every
// enemy type and a full missile stock.
func (a *Autoshot) showcase(g *Game) {
	p, ok := g.scene.(*Play)
	if !a.show || !ok {
		return
	}
	p.Rules.Missiles = rules.MaxMissiles
	w := p.World
	// keep the crowd high up (enemies that come down are replaced up top),
	// so missiles fly long arcs from Jack at the bottom
	kept := w.Enemies[:0]
	for _, e := range w.Enemies {
		if e.Pos().Y <= 110 {
			kept = append(kept, e)
		}
	}
	w.Enemies = kept
	for len(w.Enemies) < 10 {
		// the crowd lives in the upper half, so missiles fly long arcs
		pos := world.Vec{X: float64(16 + w.Rng.IntN(world.FieldW-48)), Y: float64(8 + w.Rng.IntN(80))}
		var e world.Enemy
		switch len(w.Enemies) % 4 {
		case 0:
			e = enemy.NewBird(w, pos, 1.2)
		case 1:
			e = enemy.NewSaucer(w, pos, 1.2)
		case 2:
			e = enemy.NewOrb(w, pos, 1.2)
		default:
			e = enemy.NewMummy(w, pos, 1.2, [3]int{1, 1, 1})
		}
		w.Enemies = append(w.Enemies, e)
	}
}
