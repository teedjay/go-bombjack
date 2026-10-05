package art

import "image/color"

// Animations returns every sprite animation in the game, keyed by name.
// Names are a stable contract used by the game code.
func Animations() []Anim {
	blue := [3]color.RGBA{NavyBlue, Blue, LightBlue}
	red := [3]color.RGBA{DarkRed, Red, Pink}
	green := [3]color.RGBA{DarkGreen, Green, rgb(0xd8f8a0)}
	purple := [3]color.RGBA{DarkPurp, Purple, Pink}
	gold := [3]color.RGBA{GoldDark, Gold, Yellow}
	cyan := [3]color.RGBA{DarkCyan, Cyan, White}

	anims := []Anim{
		{Name: "jack_idle", Frames: jackIdle(), FPS: 3, Loop: true},
		{Name: "jack_run", Frames: jackRun(), FPS: 10, Loop: true},
		{Name: "jack_jump", Frames: jackJump(), FPS: 1, Loop: true},
		{Name: "jack_float", Frames: jackFloat(), FPS: 6, Loop: true},
		{Name: "jack_land", Frames: jackLand(), FPS: 1, Loop: false},
		{Name: "jack_turn", Frames: jackTurn(), FPS: 1, Loop: false},
		{Name: "jack_die", Frames: jackDie(), FPS: 10, Loop: true},
		{Name: "mummy_walk", Frames: mummyWalk(), FPS: 4, Loop: true},
		{Name: "bird_fly", Frames: birdFly(), FPS: 8, Loop: true},
		{Name: "saucer", Frames: saucerSpin(), FPS: 8, Loop: true},
		{Name: "orb", Frames: orbSpin(), FPS: 6, Loop: true},
		{Name: "coin", Frames: coinSpin(), FPS: 8, Loop: true},
	}
	anims = append(anims, bombAnims()...)
	anims = append(anims,
		Anim{Name: "power_p", Frames: powerBall('P', blue, cyan), FPS: 6, Loop: true},
		Anim{Name: "power_b", Frames: powerBall('B', red, purple), FPS: 6, Loop: true},
		Anim{Name: "power_e", Frames: powerBall('E', green, cyan), FPS: 6, Loop: true},
		Anim{Name: "power_s", Frames: powerBall('S', gold, red), FPS: 6, Loop: true},
		Anim{Name: "sparkle", Frames: sparkle(), FPS: 15, Loop: false},
		Anim{Name: "twinkle", Frames: twinkle(), FPS: 14, Loop: false},
		Anim{Name: "poof", Frames: poof(), FPS: 10, Loop: false},
		Anim{Name: "explosion", Frames: explosion(), FPS: 12, Loop: false},
	)
	for _, t := range Themes {
		anims = append(anims, Anim{Name: "platform_" + t.Name, Frames: PlatformTiles(t), FPS: 0, Loop: false})
	}
	return anims
}
