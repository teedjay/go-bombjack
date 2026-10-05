package audio

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
)

// SampleRate is the output rate in Hz.
const SampleRate = 44100

// Wave selects an oscillator shape.
type Wave int

const (
	Square Wave = iota
	Pulse25
	Triangle
	Noise
)

// ADSR is a simple envelope; times in seconds, Sustain is a level 0..1.
type ADSR struct{ A, D, S, R float64 }

func (e ADSR) at(t, dur float64) float64 {
	if t < 0 || t >= dur {
		return 0
	}
	rel := math.Min(e.R, dur*0.5)
	v := 1.0
	switch {
	case t < e.A:
		v = t / e.A
	case t < e.A+e.D:
		v = 1 - (1-e.S)*(t-e.A)/e.D
	default:
		v = e.S
	}
	if t > dur-rel {
		v *= (dur - t) / rel
	}
	return v
}

// osc returns the oscillator value for phase in [0,1).
func osc(w Wave, ph float64, rng *rand.Rand) float64 {
	switch w {
	case Square:
		if ph < 0.5 {
			return 1
		}
		return -1
	case Pulse25:
		if ph < 0.25 {
			return 1
		}
		return -1
	case Triangle:
		return 4*math.Abs(ph-0.5) - 1
	default:
		return rng.Float64()*2 - 1
	}
}

// Sweep renders a mono tone gliding from f0 to f1 Hz over dur seconds.
func Sweep(w Wave, f0, f1, dur, vol float64, env ADSR) []float32 {
	n := int(dur * SampleRate)
	out := make([]float32, n)
	rng := rand.New(rand.NewPCG(1, 2))
	ph := 0.0
	for i := range out {
		t := float64(i) / SampleRate
		f := f0 + (f1-f0)*t/dur
		ph += f / SampleRate
		ph -= math.Floor(ph)
		out[i] = float32(osc(w, ph, rng) * env.at(t, dur) * vol)
	}
	return out
}

// Mix adds src into dst starting at sample offset.
func Mix(dst []float32, src []float32, offset int) []float32 {
	if need := offset + len(src); need > len(dst) {
		dst = append(dst, make([]float32, need-len(dst))...)
	}
	for i, v := range src {
		dst[offset+i] += v
	}
	return dst
}

// ToPCM converts mono samples to 16-bit little-endian stereo bytes,
// hard-limiting to the int16 range.
func ToPCM(mono []float32) []byte {
	out := make([]byte, len(mono)*4)
	for i, v := range mono {
		s := int16(max(-1, min(1, float64(v))) * 32767)
		lo, hi := byte(s), byte(uint16(s)>>8)
		out[i*4], out[i*4+1], out[i*4+2], out[i*4+3] = lo, hi, lo, hi
	}
	return out
}

// ---------------------------------------------------------------- notes ----

var noteIndex = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// Freq parses a note name such as "C5", "F#4" or "Bb3" into Hz.
// "R" (rest) returns 0.
func Freq(name string) float64 {
	if name == "R" || name == "" {
		return 0
	}
	semi := noteIndex[name[0]]
	i := 1
	switch name[i] {
	case '#':
		semi++
		i++
	case 'b':
		semi--
		i++
	}
	oct, _ := strconv.Atoi(name[i:])
	midi := 12*(oct+1) + semi
	return 440 * math.Pow(2, float64(midi-69)/12)
}

// Step is one note: Freq 0 is a rest; Len is in eighth-note units.
type Step struct {
	Freq float64
	Len  int
}

// Parse reads "C5:2 E5:1 R:1" into steps ('|' bar separators are ignored).
func Parse(s string) []Step {
	var out []Step
	for _, tok := range strings.Fields(s) {
		if tok == "|" {
			continue
		}
		name, l, _ := strings.Cut(tok, ":")
		n, _ := strconv.Atoi(l)
		out = append(out, Step{Freq(name), n})
	}
	return out
}

// Units sums the length of steps.
func Units(steps []Step) int {
	n := 0
	for _, s := range steps {
		n += s.Len
	}
	return n
}

// RenderVoice renders a melody line; unit is the duration of one length unit
// in seconds.
func RenderVoice(steps []Step, unit float64, w Wave, vol float64, env ADSR) []float32 {
	out := make([]float32, int(float64(Units(steps))*unit*SampleRate)+1)
	pos := 0.0
	for _, st := range steps {
		d := float64(st.Len) * unit
		if st.Freq > 0 {
			note := Sweep(w, st.Freq, st.Freq, d*0.92, vol, env)
			Mix(out, note, int(pos*SampleRate))
		}
		pos += d
	}
	return out
}

// Track is a looping 2-voice tune.
type Track struct {
	BPM        float64
	Lead, Bass string
}

// Render mixes both voices into one mono buffer.
func (t Track) Render() []float32 {
	unit := 60 / t.BPM / 2 // eighth note
	lead := RenderVoice(Parse(t.Lead), unit, Pulse25, 0.2, ADSR{0.005, 0.06, 0.6, 0.04})
	bass := RenderVoice(Parse(t.Bass), unit, Triangle, 0.32, ADSR{0.005, 0.05, 0.8, 0.03})
	n := max(len(lead), len(bass))
	out := make([]float32, n)
	Mix(out, lead, 0)
	Mix(out, bass, 0)
	return out
}

// bounce builds an octave-bouncing bass bar (8 units) for each root.
func bounce(roots ...string) string {
	var b strings.Builder
	for _, r := range roots {
		up := r[:len(r)-1] + string(r[len(r)-1]+1)
		b.WriteString(r + ":2 " + up + ":2 " + r + ":2 " + up + ":2 ")
	}
	return b.String()
}

// Tracks holds the music. Index 0..3 are the levels; Title is separate.
var Tracks = [4]Track{
	{150, "E5:1 G5:1 C6:2 G5:1 E5:1 G5:2 | D5:1 F5:1 B5:2 F5:1 D5:1 F5:2 | C5:1 E5:1 A5:2 E5:1 C5:1 E5:2 | D5:1 G5:1 B5:2 G5:2 R:2 | " +
		"C6:2 B5:1 A5:1 G5:2 E5:2 | F5:2 A5:2 C6:4 | B5:1 A5:1 G5:1 F5:1 E5:2 D5:2 | C5:4 G4:2 C5:2",
		bounce("C3", "G2", "A2", "G2", "F3", "F3", "G2", "C3")},
	{140, "A4:2 C5:1 E5:1 A5:2 E5:2 | G4:2 B4:1 D5:1 G5:2 D5:2 | F4:2 A4:1 C5:1 F5:2 C5:2 | E4:2 G#4:1 B4:1 E5:4 | " +
		"E5:1 D5:1 C5:1 B4:1 A4:4 | D5:1 C5:1 B4:1 A4:1 G4:4 | A4:1 C5:1 F5:2 E5:2 C5:2 | B4:2 G#4:2 A4:4",
		bounce("A2", "G2", "F2", "E2", "A2", "G2", "F2", "E2")},
	{130, "D5:2 F5:2 A5:2 F5:2 | Bb4:2 D5:2 F5:2 D5:2 | C5:2 E5:2 G5:2 E5:2 | A4:2 C#5:2 E5:4 | " +
		"A5:1 G5:1 F5:1 E5:1 D5:4 | F5:1 E5:1 D5:1 C5:1 Bb4:4 | G4:2 Bb4:2 E5:2 G5:2 | A5:4 E5:2 C#5:2",
		bounce("D2", "Bb1", "C2", "A1", "D2", "Bb1", "G1", "A1")},
	{160, "E5:1 E5:1 B5:2 A5:1 G5:1 E5:2 | D5:1 D5:1 A5:2 G5:1 F#5:1 D5:2 | C5:1 C5:1 G5:2 F#5:1 E5:1 C5:2 | B4:2 D#5:2 F#5:2 B5:2 | " +
		"G5:2 F#5:2 E5:2 B4:2 | A5:2 G5:2 F#5:2 D5:2 | G5:1 F#5:1 E5:1 D#5:1 C5:2 A4:2 | B4:4 B4:1 D#5:1 F#5:2",
		bounce("E2", "D2", "C2", "B1", "E2", "D2", "C2", "B1")},
}

// Title is the title-screen jingle.
var Title = Track{120,
	"C5:2 E5:2 G5:2 C6:2 | B5:2 G5:2 A5:2 F5:2 | E5:2 G5:2 C6:2 E6:2 | D6:4 C6:4",
	bounce("C3", "G2", "C3", "G2")}

// ------------------------------------------------------------------ sfx ----

// SFX identifiers.
const (
	sfxJump = iota
	sfxFloat
	sfxBomb
	sfxBombLit
	sfxPickup
	sfxHit
	sfxDied
	sfxClear
	sfxExtra
	sfxLaunch // homing missile whoosh
	sfxBlast  // missile firework explosion
	sfxCoin0  // sfxCoin0 .. sfxCoin0+coinSteps-1
)

const coinSteps = 8

var sfxEnv = ADSR{0.003, 0.03, 0.7, 0.03}

func seq(parts ...[]float32) []float32 {
	var out []float32
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func arp(w Wave, vol, noteDur float64, notes ...string) []float32 {
	var out []float32
	for _, n := range notes {
		f := Freq(n)
		out = append(out, Sweep(w, f, f, noteDur, vol, sfxEnv)...)
	}
	return out
}

// renderSFX builds every sound effect as mono samples.
func renderSFX() [][]float32 {
	s := make([][]float32, sfxCoin0+coinSteps)
	s[sfxJump] = Sweep(Square, 300, 760, 0.13, 0.25, sfxEnv)
	s[sfxFloat] = Sweep(Triangle, 520, 380, 0.1, 0.3, sfxEnv)
	s[sfxBomb] = seq(Sweep(Pulse25, 880, 880, 0.04, 0.25, sfxEnv), Sweep(Pulse25, 1320, 1320, 0.07, 0.25, sfxEnv))
	s[sfxBombLit] = seq(arp(Square, 0.22, 0.04, "E6", "A6", "E7"), Sweep(Square, 2637, 3520, 0.06, 0.22, sfxEnv))
	s[sfxPickup] = arp(Pulse25, 0.25, 0.05, "C6", "E6", "G6", "C7", "E7")
	for i := 0; i < coinSteps; i++ {
		f := 700 * math.Pow(2, float64(i)*2/12)
		s[sfxCoin0+i] = seq(Sweep(Square, f, f*1.5, 0.06, 0.22, sfxEnv), Sweep(Square, f*1.5, f*1.5, 0.05, 0.22, sfxEnv))
	}
	hit := Sweep(Noise, 0, 0, 0.3, 0.35, ADSR{0.002, 0.25, 0.1, 0.05})
	Mix(hit, Sweep(Square, 400, 80, 0.3, 0.2, sfxEnv), 0)
	s[sfxHit] = hit
	died := Sweep(Square, 700, 60, 0.9, 0.25, ADSR{0.005, 0.5, 0.4, 0.2})
	Mix(died, Sweep(Noise, 0, 0, 0.9, 0.12, ADSR{0.005, 0.7, 0.1, 0.2}), 0)
	s[sfxDied] = died
	s[sfxClear] = seq(arp(Pulse25, 0.25, 0.09, "C5", "E5", "G5", "C6", "G5", "C6", "E6"), Sweep(Pulse25, 1568, 1568, 0.4, 0.25, ADSR{0.005, 0.1, 0.6, 0.15}))
	s[sfxExtra] = arp(Square, 0.2, 0.07, "G5", "B5", "D6", "G6", "B6", "D7")
	launch := Sweep(Noise, 0, 0, 0.35, 0.22, ADSR{0.01, 0.2, 0.4, 0.12})
	launch = Mix(launch, Sweep(Square, 180, 900, 0.35, 0.12, sfxEnv), 0)
	s[sfxLaunch] = launch
	blast := Sweep(Noise, 0, 0, 0.8, 0.4, ADSR{0.002, 0.6, 0.15, 0.2})
	blast = Mix(blast, Sweep(Square, 220, 40, 0.5, 0.22, ADSR{0.002, 0.4, 0.2, 0.1}), 0)
	for i, f := range []float64{1800, 2400, 1500, 2900, 2100} { // firework crackles
		blast = Mix(blast, Sweep(Pulse25, f, f*0.7, 0.04, 0.16, sfxEnv), int(float64(SampleRate)*(0.12+0.09*float64(i))))
	}
	s[sfxBlast] = blast
	return s
}
