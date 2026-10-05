package audio

// A small SID-style (Commodore 64) music engine used for the high-score
// tune. Like a C64 tracker it runs on 50 Hz PAL frames: every frame the
// sequencer updates each voice's frequency, waveform, pulse width and gate,
// then the frame's 882 samples are synthesised. Classic SID tricks used:
//
//   - pulse-width modulation (PWM) sweeps on bass, chords and lead
//   - 50 Hz arpeggios: one voice cycles chord notes every frame to fake chords
//   - resonant low-pass filter on the bass, cutoff swept down per note ("wah")
//   - drums stolen from melodic voices: the kick is a fast pitch drop at the
//     start of a bass note, the snare a noise burst then a pitch drop in the
//     chord voice
//   - hard restart: envelopes snap to zero on each new note for a crisp attack
//   - portamento slides and delayed vibrato on the lead
//   - echo: a delayed, quieter copy of the lead (normally done with a voice)
//
// The tune is an original laid-back funk piece in A minor.

import (
	"math"
	"strings"
)

const (
	sidFrameRate   = 50
	samplesPerFrm  = SampleRate / sidFrameRate // 882
	framesPerStep  = 7                         // one 16th note (~107 BPM)
	stepsPerBar    = 16
	echoDelaySteps = 3
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
}

func newVoice(a, d, s, r float64) *sidVoice {
	// a: attack time (s); d, r: time constants (s); s: sustain level
	return &sidVoice{
		attack: 1 / (a*SampleRate + 1), decay: math.Exp(-1 / (d * SampleRate)),
		sustain: s, release: math.Exp(-1 / (r * SampleRate)), lfsr: 0x7ffff8, pw: 0.5,
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

// ------------------------------------------------------------- song ----

// bar parses 16 tokens: a note ("A2", "G#4"), "-" to hold, "." to release.
func bar(s string) []string {
	t := strings.Fields(s)
	if len(t) != stepsPerBar {
		panic("audio: bar needs 16 steps: " + s)
	}
	return t
}

type sidBar struct {
	bass, lead []string
	chord      []string // arpeggio notes
}

func mkBar(bass, lead string, chord ...string) sidBar {
	return sidBar{bar(bass), bar(lead), chord}
}

const rest = ". . . . . . . . . . . . . . . ."

var (
	amBass  = "A2 - A3 . G2 A2 - C3 . A2 E3 - G3 . E3 D3"
	fBass   = "F2 - F3 . E2 F2 - A2 . F2 C3 - E3 . C3 A2"
	dmBass  = "D2 - D3 . C3 D2 - F2 . D2 A2 - C3 . A2 F2"
	e7Bass  = "E2 - E3 . D3 E2 . G#2 B2 - D3 . E3 D3 B2 G#2"
	cBass   = "C3 - C4 . B2 C3 - E3 . C3 G3 - B3 . G3 E3"
	bmBass  = "B2 - B3 . A2 B2 - D3 . B2 F3 - A3 . F3 D3"
	am2Bass = "A2 - A3 . G2 A2 - C3 . E3 . A2 . G2 . E2"

	am7   = []string{"A3", "C4", "E4", "G4"}
	fmaj7 = []string{"F3", "A3", "C4", "E4"}
	dm7   = []string{"D4", "F4", "A4", "C5"}
	e7    = []string{"E3", "G#3", "B3", "D4"}
	cmaj7 = []string{"C4", "E4", "G4", "B4"}
	bm7b5 = []string{"B3", "D4", "F4", "A4"}
)

// hiScoreIntro plays once (groove only), hiScoreLoop repeats forever.
var hiScoreIntro = []sidBar{
	mkBar(amBass, rest, am7...),
	mkBar(fBass, rest, fmaj7...),
	mkBar(dmBass, rest, dm7...),
	mkBar(e7Bass, ". . . . . . . . . . . . E4 - G#4 -", e7...),
}

var hiScoreLoop = []sidBar{
	// A
	mkBar(amBass, "E5 - - - - - - - D5 - C5 - D5 - - -", am7...),
	mkBar(fBass, "C5 - - - - - A4 - C5 - - - E5 - - -", fmaj7...),
	mkBar(dmBass, "F5 - - - - - - - E5 - D5 - C5 - - -", dm7...),
	mkBar(e7Bass, "B4 - - - - - - - - - - - G#4 - - -", e7...),
	// B
	mkBar(cBass, "G5 - - - E5 - - - C5 - - - E5 - - -", cmaj7...),
	mkBar(bmBass, "F5 - - - - - - - D5 - - - B4 - - -", bm7b5...),
	mkBar(am2Bass, "E5 - - - C5 - - - A4 - - - - - - .", am7...),
	mkBar(e7Bass, "G#4 - - - - - - - B4 - - - D5 - E5 -", e7...),
	// A'
	mkBar(amBass, "A5 - - - G5 - E5 - - - D5 - E5 - - -", am7...),
	mkBar(fBass, "C5 - - - - - A4 - C5 - D5 - E5 - - -", fmaj7...),
	mkBar(dmBass, "F5 - - - A5 - - - G5 - F5 - E5 - D5 -", dm7...),
	mkBar(e7Bass, "B4 - - - C5 - B4 - A4 - - - - - - .", e7...),
	// B' (lead answers an octave lower, then climbs back)
	mkBar(cBass, "E5 - - - G5 - - - B5 - - - G5 - - -", cmaj7...),
	mkBar(bmBass, "A5 - - - F5 - - - D5 - - - F5 - - -", bm7b5...),
	mkBar(am2Bass, "E5 - - - - - - - C5 - B4 - A4 - - -", am7...),
	mkBar(e7Bass, "G#4 - - - B4 - - - D5 - - - G#5 - - .", e7...),
}

// drum pattern per step: 'k' kick (bass voice), 's' snare (chord voice)
const drums = "k........k..s..." // kick on 1 and the "and" of 3, snare on 4
// chord stabs: off-beat skank, two steps long
const stabs = "..x...x...x...x."

// renderSID renders bars to mono samples.
func renderSID(bars []sidBar) []float32 {
	bass := newVoice(0.002, 0.18, 0.55, 0.05)
	chord := newVoice(0.003, 0.12, 0.35, 0.04)
	lead := newVoice(0.02, 0.4, 0.75, 0.12)
	var filt svf

	total := len(bars) * stepsPerBar * framesPerStep * samplesPerFrm
	out := make([]float32, total)
	echoLen := echoDelaySteps * framesPerStep * samplesPerFrm
	echo := make([]float64, echoLen)
	ei := 0

	var cutoff, leadTarget, leadFrom float64
	leadFrames, slideFrames := 0, 0
	frame := 0
	snareFrames, kickFrames := -1, -1
	bassNote := 0.0

	for _, br := range bars {
		for st := 0; st < stepsPerBar; st++ {
			// ---- step events
			if tok := br.bass[st]; tok != "-" && tok != "." {
				bassNote = Freq(tok)
				bass.noteOn()
				cutoff = 2400
			} else if tok == "." {
				bass.gate = false
			}
			if drums[st] == 'k' {
				kickFrames = 0
				bass.noteOn()
			}
			if drums[st] == 's' {
				snareFrames = 0
				chord.noteOn()
			}
			if stabs[st] == 'x' {
				chord.noteOn()
			} else if st > 0 && stabs[st-1] == 'x' {
				// hold through the second step
			} else if snareFrames < 0 || snareFrames > 5 {
				chord.gate = false
			}
			if tok := br.lead[st]; tok != "-" && tok != "." {
				f := Freq(tok)
				// slide into the note if the lead was already sounding
				if lead.gate && leadTarget > 0 {
					leadFrom, slideFrames = leadTarget, 4
				} else {
					leadFrom, slideFrames = f, 0
				}
				leadTarget, leadFrames = f, 0
				lead.noteOn()
			} else if tok == "." {
				lead.gate = false
			}

			for fr := 0; fr < framesPerStep; fr++ {
				t := float64(frame) / sidFrameRate
				// ---- bass: PWM + kick pitch drop + filter sweep
				bass.wave = sidPulse
				bass.pw = 0.32 + 0.12*math.Sin(t*2*math.Pi*0.35)
				bass.freq = bassNote
				if kickFrames >= 0 && kickFrames < 3 {
					bass.wave = sidTri
					bass.freq = []float64{220, 130, 80}[kickFrames]
					kickFrames++
				} else {
					kickFrames = -1
				}
				cutoff = math.Max(380, cutoff*0.86)

				// ---- chords: 50 Hz arpeggio, snare steals the voice
				chord.wave = sidPulse
				chord.pw = 0.12 + 0.36*(0.5+0.5*math.Sin(t*2*math.Pi*0.22))
				chord.freq = Freq(br.chord[frame%len(br.chord)])
				if snareFrames >= 0 && snareFrames < 6 {
					if snareFrames < 2 {
						chord.wave, chord.freq = sidNoise, 7000
					} else {
						chord.wave = sidPulse
						chord.freq = 330 - 40*float64(snareFrames)
						if snareFrames >= 4 {
							chord.wave, chord.freq = sidNoise, 4000
						}
					}
					snareFrames++
				} else {
					snareFrames = -1
				}

				// ---- lead: portamento, delayed vibrato, slow PWM
				lead.wave = sidPulse
				lead.pw = 0.42 - 0.14*math.Sin(t*2*math.Pi*0.5)
				f := leadTarget
				if slideFrames > 0 && leadFrames < slideFrames {
					k := float64(leadFrames+1) / float64(slideFrames+1)
					f = leadFrom * math.Pow(leadTarget/leadFrom, k)
				}
				if leadFrames > 12 {
					depth := math.Min(1, float64(leadFrames-12)/20) * 0.008
					f *= 1 + depth*math.Sin(float64(leadFrames)*2*math.Pi*5.5/sidFrameRate)
				}
				lead.freq = f
				leadFrames++

				// ---- synthesise this frame
				base := frame * samplesPerFrm
				for i := 0; i < samplesPerFrm; i++ {
					bs := filt.lowpass(bass.sample(), cutoff, 0.35)
					cs := chord.sample()
					ls := lead.sample()
					e := echo[ei]
					echo[ei] = ls
					ei = (ei + 1) % echoLen
					mix := 0.62*bs + 0.2*cs + 0.26*ls + 0.1*e
					out[base+i] = float32(math.Tanh(mix * 0.36)) // gentle saturation, level matched to the other tracks
				}
				frame++
			}
		}
	}
	return out
}

// HiScoreSong renders the high-score tune: an intro that plays once and a
// loop that repeats. Both are mono samples at SampleRate.
func HiScoreSong() (intro, loop []float32) {
	all := renderSID(append(append([]sidBar(nil), hiScoreIntro...), hiScoreLoop...))
	n := len(hiScoreIntro) * stepsPerBar * framesPerStep * samplesPerFrm
	return all[:n], all[n:]
}
