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

	pool      [][voices]*eaudio.Player
	next      []int
	pcm       [4 + 1][]byte // music PCM, index track+1 (lazily rendered)
	music     *eaudio.Player
	track     int
	playing   bool
	coinCombo int
	lastCoin  time.Time
}

// New renders all SFX and creates the audio context (once per process).
func New() *Player {
	if ctx == nil {
		ctx = eaudio.NewContext(SampleRate)
	}
	p := &Player{}
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
		}
	}
}

func trackPCM(track int) []byte {
	if track < 0 {
		return ToPCM(Title.Render())
	}
	return ToPCM(Tracks[track%len(Tracks)].Render())
}

// PlayMusic starts the loop for a level (0..3) or the title (-1).
func (p *Player) PlayMusic(track int) {
	if track < -1 {
		track = -1
	}
	if track >= 0 {
		track %= len(Tracks)
	}
	if p.music != nil && p.playing && p.track == track {
		return
	}
	p.StopMusic()
	if p.pcm[track+1] == nil {
		p.pcm[track+1] = trackPCM(track)
	}
	b := p.pcm[track+1]
	loop := eaudio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b)))
	m, err := ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	p.music, p.track, p.playing = m, track, true
	m.SetVolume(musicVolume)
	if !p.Muted {
		m.Play()
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
