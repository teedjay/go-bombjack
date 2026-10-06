package audio

import "strings"

// All tunes are original compositions written for this game.

// fig fills a bass figure: tokens R, O, F (root, octave, fifth) are
// replaced by the given notes; "-" and "." pass through.
func fig(pattern, root, oct, fifth string) string {
	r := strings.NewReplacer("R", root, "O", oct, "F", fifth)
	return r.Replace(pattern)
}

const rest = ". . . . . . . . . . . . . . . ."

// ------------------------------------------------------------------ title --

// Title: an upbeat heroic anthem in C. Running 50 Hz arpeggio chords under
// a slow resonant filter sweep, an octave-bouncing bass and a bright
// noise-attack saw lead with vibrato and echo.
var TitleSong = func() *Song {
	bounce := "R - O - R - O - R - O - F - O -"
	c := fig(bounce, "C2", "C3", "G2")
	am := fig(bounce, "A1", "A2", "E2")
	f := fig(bounce, "F1", "F2", "C2")
	g := fig(bounce, "G1", "G2", "D2")
	em := fig(bounce, "E2", "E3", "B2")
	C, Am, F, G, Em := []string{"C4", "E4", "G4"}, []string{"A3", "C4", "E4"},
		[]string{"F3", "A3", "C4"}, []string{"G3", "B3", "D4"}, []string{"E4", "G4", "B4"}
	return &Song{
		Name: "title", Speed: 6,
		Drums: pattern("k.h.s.h.k.k.s.h."),
		Stabs: pattern("x---------------"),
		Bass:  Instr{Wave: sidPulse, PW: 0.4, PWMDepth: 0.1, PWMRate: 0.3, A: 0.002, D: 0.1, S: 0.6, R: 0.04},
		Chord: Instr{Wave: sidPulse, PW: 0.3, PWMDepth: 0.2, PWMRate: 0.25, A: 0.003, D: 0.3, S: 0.7, R: 0.05,
			Filter: true, Cutoff: 3200, CutoffEnd: 700, FilterLFO: 0.12, Q: 0.3},
		Lead: Instr{Wave: sidSaw, A: 0.005, D: 0.3, S: 0.7, R: 0.08, NoiseAttack: true,
			Vibrato: 0.007, VibDelay: 10, Slide: 3},
		Echo: 0.12, EchoSteps: 3, Mix: [3]float64{0.5, 0.22, 0.3},
		Intro: []Bar{B(c, rest, C...), B(g, rest, G...)},
		Loop: []Bar{
			B(c, "E5 - - - G5 - - - C6 - - - - - B5 -", C...),
			B(am, "A5 - - - - - - - E5 - - - A5 - B5 -", Am...),
			B(f, "C6 - - - - - A5 - F5 - - - A5 - C6 -", F...),
			B(g, "B5 - - - - - - - G5 - - - - - - -", G...),
			B(c, "E5 - - - G5 - - - C6 - - - E6 - D6 -", C...),
			B(am, "C6 - - - B5 - A5 - - - - - E5 - - -", Am...),
			B(f, "F5 - A5 - C6 - F6 - - - E6 - D6 - C6 -", F...),
			B(g, "D6 - - - - - - - - - - - G5 - - -", G...),
			B(f, "A5 - - - C6 - - - A5 - - - F5 - - -", F...),
			B(g, "G5 - - - B5 - - - D6 - - - B5 - - -", G...),
			B(em, "E6 - - - D6 - B5 - - - G5 - - - - -", Em...),
			B(am, "A5 - - - - - - - C6 - - - E6 - - -", Am...),
			B(f, "F6 - - - E6 - - - C6 - - - A5 - - -", F...),
			B(g, "D6 - - - B5 - - - G5 - - - B5 - D6 -", G...),
			B(c, "E6 - - - D6 - - - C6 - - - - - - -", C...),
			B(g, "B5 - - - - - - - D6 - - - G5 - - .", G...),
		},
	}
}()

// ------------------------------------------------------------------ Egypt --

// Egypt: a mysterious snake-charmer tune in E Phrygian dominant. A droning
// octave bass, slow held arpeggios, and a nasal narrow-pulse lead with long
// portamento slides and deep vibrato.
var egyptSong = func() *Song {
	drone := "R - - - O - R - - - O - F - R -"
	e := fig(drone, "E2", "E3", "B2")
	f := fig(drone, "F2", "F3", "C3")
	dm := fig(drone, "D2", "D3", "A2")
	E, F, Dm := []string{"E4", "G#4", "B4"}, []string{"F4", "A4", "C5"}, []string{"D4", "F4", "A4"}
	return &Song{
		Name: "egypt", Speed: 7,
		Drums: pattern("k..h..s.k.h.s..h"),
		Stabs: pattern("x-------x-------"),
		Bass:  Instr{Wave: sidTri, A: 0.002, D: 0.25, S: 0.7, R: 0.06},
		Chord: Instr{Wave: sidPulse, PW: 0.25, PWMDepth: 0.15, PWMRate: 0.2, A: 0.02, D: 0.5, S: 0.5, R: 0.1},
		Lead: Instr{Wave: sidPulse, PW: 0.18, PWMDepth: 0.06, PWMRate: 0.4, A: 0.01, D: 0.4, S: 0.8, R: 0.1,
			Vibrato: 0.014, VibDelay: 8, Slide: 7},
		Echo: 0.14, EchoSteps: 4, Mix: [3]float64{0.55, 0.2, 0.32},
		Intro: []Bar{B(e, rest, E...), B(e, ". . . . . . . . . . . . B4 - - -", E...)},
		Loop: []Bar{
			B(e, "E5 - - - F5 - G#5 - - - - - F5 - E5 -", E...),
			B(f, "F5 - - - - - - - E5 - D5 - C5 - - -", F...),
			B(dm, "D5 - - - F5 - - - E5 - - - D5 - C5 -", Dm...),
			B(e, "B4 - - - - - - - - - - - - - - -", E...),
			B(e, "B5 - - - A5 - G#5 - A5 - - - G#5 - F5 -", E...),
			B(f, "E5 - - - - - - - F5 - G#5 - A5 - - -", F...),
			B(dm, "B5 - - - A5 - - - G#5 - - - F5 - - -", Dm...),
			B(e, "E5 - - - - - - - - - - - - - - .", E...),
			B(e, "G#5 - - - - - B5 - - - A5 - G#5 - - -", E...),
			B(f, "A5 - - - C6 - - - B5 - A5 - G#5 - - -", F...),
			B(dm, "F5 - - - - - - - A5 - G#5 - F5 - - -", Dm...),
			B(e, "E5 - - - F5 - - - G#5 - - - - - - -", E...),
			B(f, "C6 - - - B5 - - - A5 - - - G#5 - - -", F...),
			B(e, "B5 - - - - - - - G#5 - - - E5 - - -", E...),
			B(dm, "F5 - - - E5 - D5 - - - C5 - B4 - - -", Dm...),
			B(e, "E5 - - - - - - - - - - - - - - .", E...),
		},
	}
}()

// ----------------------------------------------------------------- Greece --

// Greece: bright and bouncy in D major. Staccato bouncing bass, off-beat
// arpeggio skank, hi-hats, and a pulse lead with a fast PWM sweep.
var greeceSong = func() *Song {
	hop := "R . O . F . O . R . O . F . O ."
	d := fig(hop, "D2", "D3", "A2")
	g := fig(hop, "G2", "G3", "D3")
	a := fig(hop, "A2", "A3", "E3")
	bm := fig(hop, "B2", "B3", "F#3")
	D, G, A, Bm := []string{"D4", "F#4", "A4"}, []string{"G4", "B4", "D5"},
		[]string{"A4", "C#5", "E5"}, []string{"B4", "D5", "F#5"}
	return &Song{
		Name: "greece", Speed: 6,
		Drums: pattern("k.h.s.h.k.k.s.h."),
		Stabs: pattern("..x-..x-..x-..x-"),
		Bass:  Instr{Wave: sidPulse, PW: 0.5, PWMDepth: 0.15, PWMRate: 0.5, A: 0.002, D: 0.08, S: 0.4, R: 0.03},
		Chord: Instr{Wave: sidPulse, PW: 0.35, PWMDepth: 0.25, PWMRate: 0.35, A: 0.002, D: 0.08, S: 0.4, R: 0.04},
		Lead: Instr{Wave: sidPulse, PW: 0.5, PWMDepth: 0.35, PWMRate: 1.4, A: 0.004, D: 0.25, S: 0.7, R: 0.06,
			Vibrato: 0.006, VibDelay: 14, Slide: 2},
		Echo: 0.1, EchoSteps: 3, Mix: [3]float64{0.5, 0.24, 0.3},
		Intro: []Bar{B(d, rest, D...), B(a, ". . . . . . . . . . . . A5 - - -", A...)},
		Loop: []Bar{
			B(d, "F#5 - A5 - D6 - A5 - F#5 - - - E5 - D5 -", D...),
			B(g, "G5 - - - B5 - - - D6 - - - B5 - - -", G...),
			B(a, "C#6 - - - A5 - - - E5 - - - A5 - C#6 -", A...),
			B(d, "D6 - - - - - - - - - - - A5 - - -", D...),
			B(bm, "B5 - - - D6 - - - F#6 - - - D6 - - -", Bm...),
			B(g, "G6 - - - - - F#6 - E6 - - - D6 - - -", G...),
			B(a, "E6 - - - C#6 - - - A5 - B5 - C#6 - - -", A...),
			B(d, "D6 - - - - - - - - - - - - - - .", D...),
			B(g, "B5 - D6 - B5 - G5 - - - B5 - D6 - - -", G...),
			B(d, "A5 - F#5 - A5 - D6 - - - - - A5 - - -", D...),
			B(g, "G5 - B5 - D6 - G6 - - - F#6 - E6 - - -", G...),
			B(a, "E6 - - - - - - - C#6 - - - A5 - - -", A...),
			B(bm, "F#6 - - - D6 - - - B5 - - - D6 - - -", Bm...),
			B(g, "E6 - - - D6 - B5 - G5 - - - B5 - - -", G...),
			B(a, "C#6 - - - E6 - - - A6 - - - G6 - E6 -", A...),
			B(d, "D6 - - - - - - - A5 - - - D6 - - .", D...),
		},
	}
}()

// ----------------------------------------------------------------- Castle --

// Castle: driving gothic D harmonic minor. Galloping filtered saw bass,
// fast running arpeggios through a resonant filter sweep, and a catchy
// pulse-wave lead (PWM, short slides, light vibrato) built on a repeated
// syncopated 3-3-2 hook with a higher answering phrase.
var castleSong = func() *Song {
	gallop := "R . R R O . R R R . R R O . R R"
	dm := fig(gallop, "D2", "D3", "A2")
	bb := fig(gallop, "Bb1", "Bb2", "F2")
	gm := fig(gallop, "G1", "G2", "D2")
	a := fig(gallop, "A1", "A2", "E2")
	Dm, Bb, Gm, A := []string{"D4", "F4", "A4"}, []string{"Bb3", "D4", "F4"},
		[]string{"G3", "Bb3", "D4"}, []string{"A3", "C#4", "E4"}
	hookDm := "A5 - - A5 - - D6 - C6 - A5 - - - F5 -"
	hookBb := "G5 - - G5 - - Bb5 - A5 - F5 - - - D5 -"
	return &Song{
		Name: "castle", Speed: 5,
		Drums: pattern("k.h.s.h.k.k.s.hk"),
		Stabs: pattern("x-------x-------"),
		Bass: Instr{Wave: sidSaw, A: 0.002, D: 0.07, S: 0.45, R: 0.03,
			Filter: true, Cutoff: 1800, CutoffEnd: 450, Q: 0.4},
		Chord: Instr{Wave: sidPulse, PW: 0.2, PWMDepth: 0.15, PWMRate: 0.4, A: 0.002, D: 0.3, S: 0.6, R: 0.05,
			Filter: true, Cutoff: 2800, CutoffEnd: 600, FilterLFO: 0.25, Q: 0.22},
		Lead: Instr{Wave: sidPulse, PW: 0.35, PWMDepth: 0.15, PWMRate: 0.8, A: 0.003, D: 0.25, S: 0.65, R: 0.06,
			Vibrato: 0.006, VibDelay: 10, Slide: 2},
		Echo: 0.14, EchoSteps: 3, Mix: [3]float64{0.52, 0.22, 0.36},
		Intro: []Bar{B(dm, rest, Dm...), B(a, ". . . . . . . . . . . . A4 - C#5 -", A...)},
		Loop: []Bar{
			// A: the hook, twice
			B(dm, hookDm, Dm...),
			B(bb, hookBb, Bb...),
			B(gm, "E5 - - E5 - - G5 - F5 - E5 - D5 - E5 -", Gm...),
			B(a, "C#5 - - - - - - - A4 - - - C#5 - E5 -", A...),
			B(dm, hookDm, Dm...),
			B(bb, hookBb, Bb...),
			B(gm, "Bb5 - - Bb5 - - A5 - G5 - F5 - E5 - F5 -", Gm...),
			B(a, "E5 - - - - - - - - - - - - - - .", A...),
			// B: the answer, climbing higher
			B(gm, "D6 - - - Bb5 - - - G5 - Bb5 - D6 - - -", Gm...),
			B(dm, "F6 - - - D6 - - - A5 - D6 - F6 - - -", Dm...),
			B(bb, "F6 - E6 - D6 - - - Bb5 - - - D6 - F6 -", Bb...),
			B(a, "E6 - - - - - C#6 - - - A5 - - - - -", A...),
			B(gm, "G6 - - - F6 - E6 - D6 - - - Bb5 - - -", Gm...),
			B(dm, "A5 - - A5 - - D6 - F6 - - - E6 - D6 -", Dm...),
			B(a, "C#6 - - C#6 - - E6 - A6 - - - G6 - E6 -", A...),
			B(dm, "D6 - - - - - - - - - - - - - - .", Dm...),
		},
	}
}()

// ------------------------------------------------------------------- City --

// City: night-time funk in E minor. A "fat" bass that octave-arpeggiates
// every frame through a filter, syncopated chord stabs and an echoed lead.
var citySong = func() *Song {
	E, C, Am, B7 := []string{"E4", "G4", "B4", "D5"}, []string{"C4", "E4", "G4", "B4"},
		[]string{"A3", "C4", "E4", "G4"}, []string{"B3", "D#4", "F#4", "A4"}
	em := "E2 - . E2 - . G2 - A2 - . B2 - D3 - ."
	c := "C2 - . C2 - . E2 - G2 - . A2 - B2 - ."
	am := "A1 - . A1 - . C2 - E2 - . G2 - A2 - ."
	b7 := "B1 - . B1 - . D#2 - F#2 - . A2 - B2 - ."
	return &Song{
		Name: "city", Speed: 7,
		Drums: pattern("k.h.s.hk.hk.s.h."),
		Stabs: pattern("...x-.x-...x-.x-"),
		Bass: Instr{Wave: sidPulse, PW: 0.3, PWMDepth: 0.12, PWMRate: 0.3, A: 0.002, D: 0.15, S: 0.6, R: 0.04,
			OctaveArp: true, Filter: true, Cutoff: 2200, CutoffEnd: 500, Q: 0.35},
		Chord: Instr{Wave: sidPulse, PW: 0.4, PWMDepth: 0.2, PWMRate: 0.2, A: 0.002, D: 0.1, S: 0.35, R: 0.05},
		Lead: Instr{Wave: sidPulse, PW: 0.4, PWMDepth: 0.12, PWMRate: 0.5, A: 0.01, D: 0.4, S: 0.75, R: 0.1,
			Vibrato: 0.008, VibDelay: 12, Slide: 4},
		Echo: 0.16, EchoSteps: 3, Mix: [3]float64{0.58, 0.2, 0.28},
		Intro: []Bar{B(em, rest, E...), B(b7, rest, B7...)},
		Loop: []Bar{
			B(em, "B4 - - - D5 - E5 - - - - - G5 - E5 -", E...),
			B(c, "G5 - - - - - E5 - D5 - - - B4 - - -", C...),
			B(am, "C5 - - - E5 - - - A5 - - - G5 - E5 -", Am...),
			B(b7, "F#5 - - - - - - - D#5 - - - B4 - - -", B7...),
			B(em, "E6 - - - D6 - B5 - - - G5 - A5 - B5 -", E...),
			B(c, "G5 - - - - - - - E5 - G5 - B5 - - -", C...),
			B(am, "C6 - - - B5 - A5 - - - G5 - E5 - - -", Am...),
			B(b7, "D#5 - - - F#5 - - - A5 - - - B5 - - .", B7...),
			B(c, "E5 - - - G5 - - - B5 - - - - - G5 -", C...),
			B(em, "B5 - - - - - - - E5 - - - D5 - E5 -", E...),
			B(am, "A5 - - - - - G5 - E5 - - - C5 - - -", Am...),
			B(b7, "B4 - - - D#5 - - - F#5 - - - - - - -", B7...),
			B(c, "G5 - - - E5 - - - C6 - - - B5 - - -", C...),
			B(em, "E6 - - - - - - - B5 - D6 - E6 - - -", E...),
			B(am, "C6 - - - - - B5 - A5 - - - E5 - - -", Am...),
			B(b7, "F#5 - - - A5 - - - D#5 - - - - - - .", B7...),
		},
	}
}()

// ---------------------------------------------------------------- Volcano --

// Volcano: intense E minor at 150 BPM. A pumping 16th-note bass through a
// swept filter, running arpeggios under a fast filter wobble, heavy drums
// and a screaming noise-attack saw lead with vibrato.
var volcanoSong = func() *Song {
	pump := "R R O R R R O R R R O R F F O F"
	em := fig(pump, "E2", "E3", "B2")
	c := fig(pump, "C2", "C3", "G2")
	d := fig(pump, "D2", "D3", "A2")
	am := fig(pump, "A1", "A2", "E2")
	b7 := fig(pump, "B1", "B2", "F#2")
	Em, C, D, Am, B7 := []string{"E4", "G4", "B4"}, []string{"C4", "E4", "G4"}, []string{"D4", "F#4", "A4"},
		[]string{"A3", "C4", "E4"}, []string{"B3", "D#4", "F#4", "A4"}
	return &Song{
		Name: "volcano", Speed: 5,
		Drums: pattern("k.h.s.hkk.h.s.hh"),
		Stabs: pattern("x-------x-------"),
		Bass: Instr{Wave: sidPulse, PW: 0.3, PWMDepth: 0.12, PWMRate: 0.6, A: 0.001, D: 0.06, S: 0.4, R: 0.02,
			Filter: true, Cutoff: 2600, CutoffEnd: 500, Q: 0.3},
		Chord: Instr{Wave: sidPulse, PW: 0.25, PWMDepth: 0.2, PWMRate: 0.5, A: 0.002, D: 0.3, S: 0.6, R: 0.04,
			Filter: true, Cutoff: 3200, CutoffEnd: 700, FilterLFO: 0.9, Q: 0.2},
		Lead: Instr{Wave: sidSaw, A: 0.003, D: 0.3, S: 0.7, R: 0.06, NoiseAttack: true,
			Vibrato: 0.009, VibDelay: 8, Slide: 2},
		Echo: 0.12, EchoSteps: 3, Mix: [3]float64{0.52, 0.22, 0.32},
		Intro: []Bar{B(em, rest, Em...), B(b7, ". . . . . . . . B4 - - - D#5 - F#5 -", B7...)},
		Loop: []Bar{
			B(em, "E5 - - - G5 - - - B5 - - - A5 - G5 -", Em...),
			B(c, "E5 - - - - - - - C5 - - - E5 - G5 -", C...),
			B(d, "F#5 - - - A5 - - - D6 - - - C6 - A5 -", D...),
			B(em, "B5 - - - - - - - - - - - - - - -", Em...),
			B(em, "E6 - - - D6 - B5 - - - G5 - A5 - B5 -", Em...),
			B(c, "C6 - - - B5 - G5 - - - E5 - G5 - C6 -", C...),
			B(d, "D6 - - - C6 - A5 - F#5 - - - A5 - D6 -", D...),
			B(b7, "D#6 - - - - - - - B5 - - - F#5 - - .", B7...),
			B(am, "A5 - - - C6 - - - E6 - - - D6 - C6 -", Am...),
			B(c, "G5 - - - - - E5 - G5 - - - C6 - - -", C...),
			B(b7, "B5 - - - A5 - - - F#5 - - - D#5 - - -", B7...),
			B(em, "E5 - - - - - - - G5 - - - B5 - - -", Em...),
			B(am, "C6 - - - B5 - A5 - - - E5 - - - - -", Am...),
			B(c, "E6 - - - D6 - C6 - - - G5 - - - - -", C...),
			B(b7, "F#6 - - - D#6 - - - B5 - - - A5 - F#5 -", B7...),
			B(em, "E6 - - - - - - - - - - - - - - .", Em...),
		},
	}
}()

// ---------------------------------------------------------------- Iceland --

// Iceland: calm and airy in D major (~94 BPM). Soft triangle bass, slow
// shimmering arpeggios under a gentle filter LFO, sparse drums and a
// floating triangle lead with long echo, slides and vibrato.
var icelandSong = func() *Song {
	float := "R - - - - - - - F - - - O - - -"
	d := fig(float, "D2", "D3", "A2")
	g := fig(float, "G1", "G2", "D2")
	em := fig(float, "E2", "E3", "B2")
	a := fig(float, "A1", "A2", "E2")
	bm := fig(float, "B1", "B2", "F#2")
	Dmaj7, Gmaj7, Em, A, Bm := []string{"D4", "F#4", "A4", "C#5"}, []string{"G3", "B3", "D4", "F#4"},
		[]string{"E4", "G4", "B4", "D5"}, []string{"A3", "C#4", "E4"}, []string{"B3", "D4", "F#4"}
	return &Song{
		Name: "iceland", Speed: 8,
		Drums: pattern("k.......h...h..."),
		Stabs: pattern("x---------------"),
		Bass:  Instr{Wave: sidTri, A: 0.01, D: 0.6, S: 0.6, R: 0.2},
		Chord: Instr{Wave: sidPulse, PW: 0.5, PWMDepth: 0.3, PWMRate: 0.15, A: 0.05, D: 0.8, S: 0.7, R: 0.3,
			Filter: true, Cutoff: 2400, CutoffEnd: 900, FilterLFO: 0.1, Q: 0.35},
		Lead: Instr{Wave: sidTri, A: 0.03, D: 0.6, S: 0.8, R: 0.3,
			Vibrato: 0.006, VibDelay: 14, Slide: 5},
		Echo: 0.22, EchoSteps: 4, Mix: [3]float64{0.45, 0.25, 0.4},
		Intro: []Bar{B(d, rest, Dmaj7...), B(a, ". . . . . . . . . . . . E5 - - -", A...)},
		Loop: []Bar{
			B(d, "F#5 - - - - - - - A5 - - - C#6 - - -", Dmaj7...),
			B(g, "B5 - - - - - - - - - - - A5 - - -", Gmaj7...),
			B(em, "G5 - - - - - B5 - - - - - E6 - - -", Em...),
			B(a, "C#6 - - - - - - - - - - - - - - -", A...),
			B(d, "D6 - - - - - C#6 - A5 - - - F#5 - - -", Dmaj7...),
			B(g, "G5 - - - B5 - - - D6 - - - F#6 - - -", Gmaj7...),
			B(em, "E6 - - - D6 - - - B5 - - - G5 - - -", Em...),
			B(a, "A5 - - - - - - - - - - - - - - .", A...),
			B(bm, "D6 - - - - - - - F#6 - - - - - - -", Bm...),
			B(g, "E6 - - - D6 - - - B5 - - - - - - -", Gmaj7...),
			B(d, "A5 - - - F#5 - - - A5 - - - D6 - - -", Dmaj7...),
			B(a, "C#6 - - - - - - - E6 - - - - - - -", A...),
			B(bm, "F#6 - - - E6 - D6 - - - B5 - - - - -", Bm...),
			B(g, "G6 - - - F#6 - - - D6 - - - B5 - - -", Gmaj7...),
			B(em, "E6 - - - - - - - G6 - - - F#6 - E6 -", Em...),
			B(a, "C#6 - - - - - - - - - - - - - - .", A...),
		},
	}
}()

// LevelSongs are the in-game tunes in level order.
var LevelSongs = [6]*Song{egyptSong, greeceSong, castleSong, citySong, volcanoSong, icelandSong}

// ---------------------------------------------------------------- hiscore --

// HiScoreSong: laid-back A-minor funk for the high-score screens. Filtered
// "wah" pluck bass, off-beat arpeggio skank and a sliding, echoed lead.
var HiScoreSong = func() *Song {
	amBass := "A2 - A3 . G2 A2 - C3 . A2 E3 - G3 . E3 D3"
	fBass := "F2 - F3 . E2 F2 - A2 . F2 C3 - E3 . C3 A2"
	dmBass := "D2 - D3 . C3 D2 - F2 . D2 A2 - C3 . A2 F2"
	e7Bass := "E2 - E3 . D3 E2 . G#2 B2 - D3 . E3 D3 B2 G#2"
	cBass := "C3 - C4 . B2 C3 - E3 . C3 G3 - B3 . G3 E3"
	bmBass := "B2 - B3 . A2 B2 - D3 . B2 F3 - A3 . F3 D3"
	am2Bass := "A2 - A3 . G2 A2 - C3 . E3 . A2 . G2 . E2"
	am7 := []string{"A3", "C4", "E4", "G4"}
	fmaj7 := []string{"F3", "A3", "C4", "E4"}
	dm7 := []string{"D4", "F4", "A4", "C5"}
	e7 := []string{"E3", "G#3", "B3", "D4"}
	cmaj7 := []string{"C4", "E4", "G4", "B4"}
	bm7b5 := []string{"B3", "D4", "F4", "A4"}
	return &Song{
		Name: "hiscore", Speed: 7,
		Drums: pattern("k........k..s..."),
		Stabs: pattern("..x-..x-..x-..x-"),
		Bass: Instr{Wave: sidPulse, PW: 0.32, PWMDepth: 0.12, PWMRate: 0.35, A: 0.002, D: 0.18, S: 0.55, R: 0.05,
			Filter: true, Cutoff: 2400, CutoffEnd: 380, Q: 0.35},
		Chord: Instr{Wave: sidPulse, PW: 0.3, PWMDepth: 0.18, PWMRate: 0.22, A: 0.003, D: 0.12, S: 0.35, R: 0.04},
		Lead: Instr{Wave: sidPulse, PW: 0.42, PWMDepth: 0.14, PWMRate: 0.5, A: 0.02, D: 0.4, S: 0.75, R: 0.12,
			Vibrato: 0.008, VibDelay: 12, Slide: 4},
		Echo: 0.1, EchoSteps: 3, Mix: [3]float64{0.62, 0.2, 0.26},
		Intro: []Bar{
			B(amBass, rest, am7...),
			B(fBass, rest, fmaj7...),
			B(dmBass, rest, dm7...),
			B(e7Bass, ". . . . . . . . . . . . E4 - G#4 -", e7...),
		},
		Loop: []Bar{
			// A
			B(amBass, "E5 - - - - - - - D5 - C5 - D5 - - -", am7...),
			B(fBass, "C5 - - - - - A4 - C5 - - - E5 - - -", fmaj7...),
			B(dmBass, "F5 - - - - - - - E5 - D5 - C5 - - -", dm7...),
			B(e7Bass, "B4 - - - - - - - - - - - G#4 - - -", e7...),
			// B
			B(cBass, "G5 - - - E5 - - - C5 - - - E5 - - -", cmaj7...),
			B(bmBass, "F5 - - - - - - - D5 - - - B4 - - -", bm7b5...),
			B(am2Bass, "E5 - - - C5 - - - A4 - - - - - - .", am7...),
			B(e7Bass, "G#4 - - - - - - - B4 - - - D5 - E5 -", e7...),
			// A'
			B(amBass, "A5 - - - G5 - E5 - - - D5 - E5 - - -", am7...),
			B(fBass, "C5 - - - - - A4 - C5 - D5 - E5 - - -", fmaj7...),
			B(dmBass, "F5 - - - A5 - - - G5 - F5 - E5 - D5 -", dm7...),
			B(e7Bass, "B4 - - - C5 - B4 - A4 - - - - - - .", e7...),
			// B'
			B(cBass, "E5 - - - G5 - - - B5 - - - G5 - - -", cmaj7...),
			B(bmBass, "A5 - - - F5 - - - D5 - - - F5 - - -", bm7b5...),
			B(am2Bass, "E5 - - - - - - - C5 - B4 - A4 - - -", am7...),
			B(e7Bass, "G#4 - - - B4 - - - D5 - - - G#5 - - .", e7...),
		},
	}
}()
