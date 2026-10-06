package audio

import (
	"math"
	"testing"
)

func TestSFXBuffers(t *testing.T) {
	for i, s := range renderSFX() {
		if len(s) < SampleRate/20 {
			t.Errorf("sfx %d too short: %d", i, len(s))
		}
		pcm := ToPCM(s)
		if len(pcm) != len(s)*4 {
			t.Errorf("sfx %d pcm len %d", i, len(pcm))
		}
	}
}

func TestToPCMClips(t *testing.T) {
	pcm := ToPCM([]float32{5, -5, 0})
	if got := int16(uint16(pcm[0]) | uint16(pcm[1])<<8); got != 32767 {
		t.Errorf("positive clip = %d", got)
	}
	if got := int16(uint16(pcm[4]) | uint16(pcm[5])<<8); got != -32767 {
		t.Errorf("negative clip = %d", got)
	}
}

func TestFreq(t *testing.T) {
	if f := Freq("A4"); math.Abs(f-440) > 0.01 {
		t.Errorf("A4 = %f", f)
	}
	if Freq("R") != 0 {
		t.Error("rest")
	}
}

// Every song renders whole bars at its tempo, without NaNs, at a level in
// line with the others, and with no constant clipping.
func TestSongs(t *testing.T) {
	songs := append([]*Song{TitleSong, HiScoreSong, GameOverSong}, LevelSongs[:]...)
	for _, s := range songs {
		intro, loop := RenderSong(s)
		if s.Jingle != (len(loop) == 0) {
			t.Errorf("%s: jingle %v but loop has %d samples", s.Name, s.Jingle, len(loop))
		}
		if s.Jingle {
			loop = intro // a jingle is all intro; measure that
		}
		bar := stepsPerBar * s.Speed * samplesPerFrm
		if len(intro) != len(s.Intro)*bar || (!s.Jingle && len(loop) != len(s.Loop)*bar) {
			t.Errorf("%s: lengths intro %d loop %d", s.Name, len(intro), len(loop))
		}
		var sum float64
		loud := 0
		for _, v := range loop {
			f := float64(v)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				t.Fatalf("%s: NaN/Inf sample", s.Name)
			}
			sum += f * f
			if math.Abs(f) > 0.98 {
				loud++
			}
		}
		rms := math.Sqrt(sum / float64(len(loop)))
		t.Logf("%-8s loop %4.1fs intro %4.1fs rms %.3f", s.Name, float64(len(loop))/SampleRate, float64(len(intro))/SampleRate, rms)
		if rms < 0.12 || rms > 0.3 {
			t.Errorf("%s: rms %.3f out of range", s.Name, rms)
		}
		if frac := float64(loud) / float64(len(loop)); frac > 0.01 {
			t.Errorf("%s: %.1f%% of samples near clipping", s.Name, frac*100)
		}
	}
}
