package audio

// A small SID-style (Commodore 64) music engine used for all the game's
// music. Like a C64 tracker it runs on 50 Hz PAL frames: every frame the
// sequencer updates each voice's frequency, waveform, pulse width and gate,
// then the frame's 882 samples are synthesised from three voices (bass,
// chord, lead), just like the SID's three oscillators. Trademark SID tricks,
// switched on per instrument in each song:
//
//   - pulse-width modulation (PWM) sweeps
//   - 50 Hz arpeggios: one voice cycles chord notes every frame to fake chords
//   - "fat" octave arpeggio: the bass alternates note and octave every frame
//   - resonant low-pass filter, swept per note ("wah") or by an LFO
//   - drums stolen from melodic voices: the kick is a fast pitch drop at the
//     start of a bass note; snare and hi-hats briefly switch the chord voice
//     to noise
//   - noise-attack instruments: the first frame of a note is a noise click
//   - hard restart: envelopes snap to zero on each new note
//   - portamento slides and delayed vibrato
//   - ring modulation for metallic bell tones
//   - echo: a delayed, quieter copy of the lead

import (
	"math"
	"strings"
)

const (
	sidFrameRate  = 50
	samplesPerFrm = SampleRate / sidFrameRate // 882
	stepsPerBar   = 16
	targetRMS     = 0.19 // all songs are normalised to this loudness
)

type sidWave int

const (
	sidPulse sidWave = iota
	sidSaw
	sidTri
	sidNoise
)

// sidVoice is one oscillator + envelope, roughly like a SID voice.
type sidVoice struct {
	freq, pw  float64
	wave      sidWave
	gate      bool
	phase     float64
	env       float64
	attack    float64 // per-sample increment
	attacking bool
	decay     float64
	sustain   float64
	release   float64
	lfsr      uint32
	noise     float64
	ringPhase float64 // second oscillator for ring modulation
	ring      float64 // ring modulator frequency ratio (0 = off)
}

func newVoice(in Instr) *sidVoice {
	// A: attack time (s); D, R: time constants (s); S: sustain level
	return &sidVoice{
		attack: 1 / (in.A*SampleRate + 1), decay: math.Exp(-1 / (in.D * SampleRate)),
		sustain: in.S, release: math.Exp(-1 / (in.R * SampleRate)), lfsr: 0x7ffff8, pw: 0.5,
	}
}

// noteOn restarts the envelope from zero (SID "hard restart") and re-arms
// the attack phase.
func (v *sidVoice) noteOn() {
	v.gate, v.env, v.attacking = true, 0, true
}

func (v *sidVoice) sample() float64 {
	prev := v.phase
	v.phase += v.freq / SampleRate
	v.phase -= math.Floor(v.phase)
	var out float64
	switch v.wave {
	case sidPulse:
		if v.phase < v.pw {
			out = 1
		} else {
			out = -1
		}
	case sidSaw:
		out = 2*v.phase - 1
	case sidTri:
		out = 4*math.Abs(v.phase-0.5) - 1
	case sidNoise:
		// 23-bit LFSR clocked on phase wrap, like the SID noise generator
		if v.phase < prev {
			bit := ((v.lfsr >> 22) ^ (v.lfsr >> 17)) & 1
			v.lfsr = (v.lfsr<<1 | bit) & 0x7fffff
			v.noise = float64(v.lfsr&0xff)/127.5 - 1
		}
		out = v.noise
	}
	if v.ring > 0 && v.wave == sidTri {
		// ring modulation: the triangle is multiplied by another
		// oscillator's square wave, giving a metallic, bell-like tone
		v.ringPhase += v.freq * v.ring / SampleRate
		v.ringPhase -= math.Floor(v.ringPhase)
		if v.ringPhase >= 0.5 {
			out = -out
		}
	}
	// envelope: linear attack, exponential decay to sustain, exp release
	if v.gate {
		if v.attacking {
			v.env += v.attack
			if v.env >= 1 {
				v.env, v.attacking = 1, false
			}
		} else {
			v.env = v.sustain + (v.env-v.sustain)*v.decay
		}
	} else {
		v.env *= v.release
	}
	return out * v.env
}

// svf is a resonant state-variable low-pass filter (SID-ish).
type svf struct{ low, band float64 }

func (f *svf) lowpass(in, cutoff, q float64) float64 {
	k := 2 * math.Sin(math.Pi*math.Min(cutoff, SampleRate/6)/SampleRate)
	f.low += k * f.band
	high := in - f.low - q*f.band
	f.band += k * high
	return f.low
}

// ----------------------------------------------------------- song data ----

// Instr is one voice's instrument: waveform, envelope and SID tricks.
type Instr struct {
	Wave              sidWave
	PW                float64 // base pulse width (0..1)
	PWMDepth, PWMRate float64 // pulse-width sweep depth and speed (Hz)
	A, D, S, R        float64 // envelope (seconds / sustain level)
	NoiseAttack       bool    // first frame of each note is a noise click
	Filter            bool    // resonant low-pass on this voice
	Cutoff, CutoffEnd float64 // per-note sweep Cutoff -> CutoffEnd (Hz)...
	FilterLFO         float64 // ...or, if > 0, an LFO between them (Hz)
	Q                 float64 // filter damping (lower = more resonance)
	OctaveArp         bool    // alternate note and octave every frame
	Vibrato           float64 // vibrato depth as a fraction of the pitch
	VibDelay          int     // frames before vibrato fades in
	Slide             int     // portamento frames between legato notes
	Ring              float64 // ring-mod ratio for a triangle wave (0 = off)
}

// Bar is one bar of music: 16 steps per part. Notes are like "A2" or
// "G#4"; "-" holds the previous note, "." releases it.
type Bar struct {
	Bass, Lead []string
	Chord      []string // chord notes for the 50 Hz arpeggio
	Drums      string   // optional per-bar override of Song.Drums
}

// Song is a complete tune: Intro plays once, Loop repeats.
type Song struct {
	Name        string
	Speed       int    // frames per 16th step (6 = 125 BPM, 7 = 107 BPM)
	Drums       string // per step: k kick, s snare, h hi-hat, . none
	Stabs       string // chord gate per step: x trigger, - hold, . off
	Bass, Chord Instr
	Lead        Instr
	Echo        float64 // lead echo level (0 = off)
	EchoSteps   int     // echo delay in steps
	Mix         [3]float64
	Intro, Loop []Bar
}

// B builds a bar from 16-step strings; chord is the arpeggio's notes.
func B(bass, lead string, chord ...string) Bar {
	return Bar{Bass: steps(bass), Lead: steps(lead), Chord: chord}
}

func steps(s string) []string {
	t := strings.Fields(s)
	if len(t) != stepsPerBar {
		panic("audio: bar needs 16 steps: " + s)
	}
	return t
}

// pattern checks a 16-character step pattern.
func pattern(s string) string {
	if len(s) != stepsPerBar {
		panic("audio: pattern needs 16 steps: " + s)
	}
	return s
}

// ------------------------------------------------------------- render ----

// RenderSong renders a song's intro and loop as mono samples.
func RenderSong(s *Song) (intro, loop []float32) {
	all := renderSID(s, append(append([]Bar(nil), s.Intro...), s.Loop...))
	n := len(s.Intro) * stepsPerBar * s.Speed * samplesPerFrm
	return all[:n], all[n:]
}

func (in Instr) pwAt(t float64) float64 {
	return math.Max(0.05, math.Min(0.95, in.PW+in.PWMDepth*math.Sin(2*math.Pi*in.PWMRate*t)))
}

func renderSID(s *Song, bars []Bar) []float32 {
	bass, chord, lead := newVoice(s.Bass), newVoice(s.Chord), newVoice(s.Lead)
	lead.ring = s.Lead.Ring
	var bassF, chordF svf

	total := len(bars) * stepsPerBar * s.Speed * samplesPerFrm
	raw := make([]float64, total)
	echoLen := max(1, s.EchoSteps*s.Speed*samplesPerFrm)
	echo := make([]float64, echoLen)
	ei := 0

	var bassCut, leadTarget, leadFrom, bassNote float64
	bassFrames, leadFrames, slideFrames := 0, 0, 0
	snareFrames, kickFrames, hatFrames := -1, -1, -1
	chordHeld := false
	frame := 0

	for _, br := range bars {
		drums := s.Drums
		if br.Drums != "" {
			drums = br.Drums
		}
		for st := 0; st < stepsPerBar; st++ {
			// ---- step events
			switch tok := br.Bass[st]; tok {
			case "-":
			case ".":
				bass.gate = false
			default:
				bassNote, bassFrames = Freq(tok), 0
				bass.noteOn()
				bassCut = s.Bass.Cutoff
			}
			switch s.Stabs[st] {
			case 'x':
				chord.noteOn()
				chordHeld = true
			case '.':
				chord.gate, chordHeld = false, false
			}
			switch drums[st] {
			case 'k':
				kickFrames = 0
				bass.noteOn()
			case 's':
				snareFrames = 0
				chord.noteOn()
			case 'h':
				hatFrames = 0
				if !chordHeld {
					chord.noteOn()
				}
			}
			switch tok := br.Lead[st]; tok {
			case "-":
			case ".":
				lead.gate = false
			default:
				f := Freq(tok)
				if lead.gate && leadTarget > 0 && s.Lead.Slide > 0 {
					leadFrom, slideFrames = leadTarget, s.Lead.Slide
				} else {
					leadFrom, slideFrames = f, 0
				}
				leadTarget, leadFrames = f, 0
				lead.noteOn()
			}

			for fr := 0; fr < s.Speed; fr++ {
				t := float64(frame) / sidFrameRate

				// ---- bass voice (+ kick drum)
				bass.wave, bass.pw = s.Bass.Wave, s.Bass.pwAt(t)
				bass.freq = bassNote
				if s.Bass.OctaveArp && frame%2 == 1 {
					bass.freq *= 2
				}
				if s.Bass.NoiseAttack && bassFrames == 0 {
					bass.wave, bass.freq = sidNoise, 3000
				}
				if kickFrames >= 0 && kickFrames < 3 {
					bass.wave = sidTri
					bass.freq = []float64{220, 130, 80}[kickFrames]
					kickFrames++
				} else {
					kickFrames = -1
				}
				bassFrames++
				bassCut = math.Max(s.Bass.CutoffEnd, bassCut*0.86)

				// ---- chord voice: 50 Hz arpeggio; snare and hats steal it
				chord.wave, chord.pw = s.Chord.Wave, s.Chord.pwAt(t)
				chord.freq = Freq(br.Chord[frame%len(br.Chord)])
				switch {
				case snareFrames >= 0 && snareFrames < 6:
					if snareFrames < 2 || snareFrames >= 4 {
						chord.wave, chord.freq = sidNoise, 7000-1000*float64(snareFrames)
					} else {
						chord.wave, chord.freq = sidPulse, 330-40*float64(snareFrames)
					}
					snareFrames++
					if snareFrames == 6 && !chordHeld {
						chord.gate = false
					}
				case hatFrames == 0:
					chord.wave, chord.freq = sidNoise, 9000
					hatFrames++
				default:
					if hatFrames == 1 && !chordHeld {
						chord.gate = false // a lone hat is just a click
					}
					hatFrames, snareFrames = -1, -1
				}

				// ---- lead voice: portamento, delayed vibrato, noise attack
				lead.wave, lead.pw = s.Lead.Wave, s.Lead.pwAt(t)
				f := leadTarget
				if slideFrames > 0 && leadFrames < slideFrames {
					k := float64(leadFrames+1) / float64(slideFrames+1)
					f = leadFrom * math.Pow(leadTarget/leadFrom, k)
				}
				if s.Lead.Vibrato > 0 && leadFrames > s.Lead.VibDelay {
					depth := math.Min(1, float64(leadFrames-s.Lead.VibDelay)/20) * s.Lead.Vibrato
					f *= 1 + depth*math.Sin(float64(leadFrames)*2*math.Pi*5.5/sidFrameRate)
				}
				lead.freq = f
				if s.Lead.NoiseAttack && leadFrames == 0 {
					lead.wave, lead.freq = sidNoise, 5000
				}
				leadFrames++

				chordCut := s.Chord.Cutoff
				if s.Chord.FilterLFO > 0 {
					m := 0.5 + 0.5*math.Sin(2*math.Pi*s.Chord.FilterLFO*t)
					chordCut = s.Chord.CutoffEnd + (s.Chord.Cutoff-s.Chord.CutoffEnd)*m
				}

				// ---- synthesise this frame
				base := frame * samplesPerFrm
				for i := 0; i < samplesPerFrm; i++ {
					bs := bass.sample()
					if s.Bass.Filter {
						bs = bassF.lowpass(bs, bassCut, s.Bass.Q)
					}
					cs := chord.sample()
					if s.Chord.Filter {
						cs = chordF.lowpass(cs, chordCut, s.Chord.Q)
					}
					ls := lead.sample()
					e := echo[ei]
					echo[ei] = ls
					ei = (ei + 1) % echoLen
					raw[base+i] = s.Mix[0]*bs + s.Mix[1]*cs + s.Mix[2]*ls + s.Echo*e
				}
				frame++
			}
		}
	}
	return normalise(raw)
}

// normalise scales to targetRMS and soft-clips, so every song sits at the
// same loudness.
func normalise(raw []float64) []float32 {
	sum := 0.0
	for _, v := range raw {
		sum += v * v
	}
	gain := 1.0
	if rms := math.Sqrt(sum / float64(max(1, len(raw)))); rms > 0 {
		gain = targetRMS / rms
	}
	out := make([]float32, len(raw))
	for i, v := range raw {
		out[i] = float32(math.Tanh(v * gain))
	}
	return out
}
