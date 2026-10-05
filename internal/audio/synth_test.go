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

func TestTracks(t *testing.T) {
	all := append([]Track{Title}, Tracks[:]...)
	for i, tr := range all {
		l, b := Units(Parse(tr.Lead)), Units(Parse(tr.Bass))
		if l != b {
			t.Errorf("track %d lead %d units != bass %d", i, l, b)
		}
		if l%8 != 0 {
			t.Errorf("track %d not whole bars: %d", i, l)
		}
		buf := tr.Render()
		want := int(float64(l) * 60 / tr.BPM / 2 * SampleRate)
		if d := len(buf) - want; d < 0 || d > 2 {
			t.Errorf("track %d len %d want ~%d", i, len(buf), want)
		}
		peak := 0.0
		for _, v := range buf {
			peak = math.Max(peak, math.Abs(float64(v)))
		}
		if peak > 1 || peak < 0.05 {
			t.Errorf("track %d peak %.2f out of range", i, peak)
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

func TestHiScoreSong(t *testing.T) {
	intro, loop := HiScoreSong()
	barSamples := stepsPerBar * framesPerStep * samplesPerFrm
	if len(intro) != len(hiScoreIntro)*barSamples || len(loop) != len(hiScoreLoop)*barSamples {
		t.Fatalf("lengths intro %d loop %d", len(intro), len(loop))
	}
	var sum, peak float64
	loud := 0
	for _, v := range loop {
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatal("NaN/Inf sample")
		}
		sum += f * f
		peak = math.Max(peak, math.Abs(f))
		if math.Abs(f) > 0.98 {
			loud++
		}
	}
	rms := math.Sqrt(sum / float64(len(loop)))
	t.Logf("loop %.1fs intro %.1fs rms %.3f peak %.3f", float64(len(loop))/SampleRate, float64(len(intro))/SampleRate, rms, peak)
	if rms < 0.12 || rms > 0.3 { // in line with the other tracks (~0.18)
		t.Errorf("rms %.3f out of range", rms)
	}
	if frac := float64(loud) / float64(len(loop)); frac > 0.01 {
		t.Errorf("%.1f%% of samples near clipping", frac*100)
	}
}
