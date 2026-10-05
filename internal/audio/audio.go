// Package audio is the procedural chiptune synth: SFX and music generated
// in code, no asset files. Owned by WS5.
package audio

import (
	"bytes"
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"bombjack/internal/world"
)

const (
	voices      = 2
	musicVolume = 0.55
	comboReset  = 1500 * time.Millisecond
)

var ctx *eaudio.Context

// Player plays SFX for world events and per-level music.
type Player struct {
	Muted bool

	pool    [][voices]*eaudio.Player
	next    []int
	pcm     map[int]music // rendered tracks, lazily
	music   *eaudio.Player
	track   int
	playing bool
	// fade-out in progress: volume ramps to 0, then fadeNext starts
	fadeLeft, fadeTotal int
	fadeNext            int
	coinCombo           int
	lastCoin            time.Time
}

// New renders all SFX and creates the audio context (once per process).
func New() *Player {
	if ctx == nil {
		ctx = eaudio.NewContext(SampleRate)
	}
	p := &Player{pcm: map[int]music{}}
	for _, mono := range renderSFX() {
		pcm := ToPCM(mono)
		var vs [voices]*eaudio.Player
		for i := range vs {
			vs[i] = ctx.NewPlayerFromBytes(pcm)
			vs[i].SetVolume(0.7)
		}
		p.pool = append(p.pool, vs)
	}
	p.next = make([]int, len(p.pool))
	return p
}

func (p *Player) play(id int) {
	if p.Muted {
		return
	}
	vs := &p.pool[id]
	for i := range vs {
		if !vs[i].IsPlaying() {
			vs[i].Rewind()
			vs[i].Play()
			return
		}
	}
	// all busy: restart the oldest
	i := p.next[id]
	p.next[id] = (i + 1) % voices
	vs[i].Rewind()
	vs[i].Play()
}

// Handle plays sounds for this tick's events.
func (p *Player) Handle(events []world.Event) {
	for _, e := range events {
		switch e.Kind {
		case world.EvJump:
			p.play(sfxJump)
		case world.EvFloat:
			p.play(sfxFloat)
		case world.EvBombTaken:
			if e.Lit {
				p.play(sfxBombLit)
			} else {
				p.play(sfxBomb)
			}
		case world.EvPickupTaken:
			p.play(sfxPickup)
		case world.EvCoinEaten:
			now := time.Now()
			if now.Sub(p.lastCoin) > comboReset {
				p.coinCombo = 0
			}
			p.lastCoin = now
			p.play(sfxCoin0 + min(p.coinCombo, coinSteps-1))
			p.coinCombo++
		case world.EvPlayerHit:
			p.play(sfxHit)
		case world.EvPlayerDied:
			p.play(sfxDied)
		case world.EvRoundClear:
			p.play(sfxClear)
		case world.EvExtraLife:
			p.play(sfxExtra)
		case world.EvMissileFired:
			p.play(sfxLaunch)
		case world.EvMissileHit:
			p.play(sfxBlast)
		}
	}
}

// Music tracks: 0..3 are the levels.
const (
	TrackTitle   = -1
	TrackHiScore = -2 // SID-style high-score tune
	TrackNone    = -99
)

// music is a rendered track: intro plays once, loop repeats.
type music struct{ intro, loop []byte }

func renderTrack(track int) music {
	switch track {
	case TrackTitle:
		return music{loop: ToPCM(Title.Render())}
	case TrackHiScore:
		intro, loop := HiScoreSong()
		return music{intro: ToPCM(intro), loop: ToPCM(loop)}
	}
	return music{loop: ToPCM(Tracks[track%len(Tracks)].Render())}
}

// PlayMusic starts a level loop (0..3), the title (TrackTitle) or the
// high-score tune (TrackHiScore). It cancels any fade in progress.
func (p *Player) PlayMusic(track int) {
	if track >= 0 {
		track %= len(Tracks)
	}
	p.fadeLeft = 0
	if p.music != nil && p.playing && p.track == track {
		p.music.SetVolume(musicVolume)
		return
	}
	p.StopMusic()
	mu, ok := p.pcm[track]
	if !ok {
		mu = renderTrack(track)
		p.pcm[track] = mu
	}
	all := append(append([]byte(nil), mu.intro...), mu.loop...)
	src := eaudio.NewInfiniteLoopWithIntro(bytes.NewReader(all), int64(len(mu.intro)), int64(len(mu.loop)))
	m, err := ctx.NewPlayer(src)
	if err != nil {
		return
	}
	p.music, p.track, p.playing = m, track, true
	m.SetVolume(musicVolume)
	if !p.Muted {
		m.Play()
	}
}

// FadeTo fades the current music out over ticks (60/s) and then starts
// track. With nothing playing it starts track at once; if track is already
// playing it keeps going.
func (p *Player) FadeTo(track, ticks int) {
	if p.music == nil || !p.playing {
		p.PlayMusic(track)
		return
	}
	if p.track == track && p.fadeLeft == 0 {
		return
	}
	p.fadeLeft, p.fadeTotal, p.fadeNext = max(ticks, 1), max(ticks, 1), track
}

// Update advances fades; call once per tick.
func (p *Player) Update() {
	if p.fadeLeft <= 0 {
		return
	}
	p.fadeLeft--
	if p.music != nil {
		p.music.SetVolume(musicVolume * float64(p.fadeLeft) / float64(p.fadeTotal))
	}
	if p.fadeLeft == 0 {
		p.StopMusic()
		if p.fadeNext != TrackNone {
			p.PlayMusic(p.fadeNext)
		}
	}
}

func (p *Player) StopMusic() {
	if p.music != nil {
		p.music.Pause()
		p.music.Close()
		p.music = nil
	}
	p.playing = false
}

// ToggleMute flips mute; music is paused/resumed accordingly.
func (p *Player) ToggleMute() {
	p.Muted = !p.Muted
	if p.music == nil {
		return
	}
	if p.Muted {
		p.music.Pause()
	} else {
		p.music.Play()
	}
}
