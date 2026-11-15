// Package synth generates deterministic synthetic "songs" and degraded clips
// for tests and manual experiments (no copyrighted audio needed).
package synth

import (
	"encoding/binary"
	"io"
	"math"
	"math/rand"
)

// pentatonic semitone offsets used to build melodies.
var scale = []int{0, 2, 4, 7, 9, 12, 14, 16, 19, 21}

func midiHz(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

// Song renders a melody + bass + percussion mix. The same seed always yields
// the same audio and different seeds give unrelated note sequences.
func Song(seed int64, seconds float64, sampleRate int) []float64 {
	rng := rand.New(rand.NewSource(seed))
	n := int(seconds * float64(sampleRate))
	out := make([]float64, n)
	root := 48 + rng.Intn(12)

	// Melody: random notes with 3 harmonics and a decaying envelope.
	for t := 0.0; t < seconds; {
		dur := 0.15 + rng.Float64()*0.35
		if rng.Float64() < 0.12 { // occasional rest
			t += dur
			continue
		}
		note := float64(root + 12 + scale[rng.Intn(len(scale))])
		addTone(out, sampleRate, t, dur, midiHz(note), 0.30, 3)
		t += dur
	}
	// Bass line.
	for t := 0.0; t < seconds; {
		dur := 0.4 + rng.Float64()*0.6
		note := float64(root - 12 + scale[rng.Intn(5)])
		addTone(out, sampleRate, t, dur, midiHz(note), 0.35, 2)
		t += dur
	}
	// Percussion: short decaying noise bursts.
	for t := 0.0; t < seconds; t += 0.25 + rng.Float64()*0.25 {
		start := int(t * float64(sampleRate))
		for i := 0; i < sampleRate/20 && start+i < n; i++ {
			env := math.Exp(-float64(i) / float64(sampleRate) * 90)
			out[start+i] += (rng.Float64()*2 - 1) * 0.10 * env
		}
	}
	return normalize(out, 0.8)
}

func addTone(out []float64, sr int, start, dur, freq, amp float64, harmonics int) {
	s0 := int(start * float64(sr))
	length := int(dur * float64(sr))
	for i := 0; i < length && s0+i < len(out); i++ {
		tt := float64(i) / float64(sr)
		env := math.Min(1, tt*80) * math.Exp(-tt*3.0) * math.Min(1, float64(length-i)/float64(sr)*60)
		var v float64
		for h := 1; h <= harmonics; h++ {
			v += math.Sin(2*math.Pi*freq*float64(h)*tt) / float64(h)
		}
		out[s0+i] += amp * env * v
	}
}

func normalize(x []float64, peak float64) []float64 {
	var m float64
	for _, v := range x {
		m = math.Max(m, math.Abs(v))
	}
	if m == 0 {
		return x
	}
	for i := range x {
		x[i] = x[i] / m * peak
	}
	return x
}

// Clip returns a copy of x[startSec : startSec+durSec].
func Clip(x []float64, sampleRate int, startSec, durSec float64) []float64 {
	a := int(startSec * float64(sampleRate))
	b := a + int(durSec*float64(sampleRate))
	if a < 0 {
		a = 0
	}
	if b > len(x) {
		b = len(x)
	}
	out := make([]float64, b-a)
	copy(out, x[a:b])
	return out
}

// AddNoise mixes white noise at the given signal-to-noise ratio (dB).
func AddNoise(x []float64, snrDB float64, seed int64) []float64 {
	rng := rand.New(rand.NewSource(seed))
	var p float64
	for _, v := range x {
		p += v * v
	}
	p /= float64(len(x))
	sigma := math.Sqrt(p / math.Pow(10, snrDB/10))
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = v + rng.NormFloat64()*sigma
	}
	return out
}

// Gain scales x and hard-clips to [-1, 1] like a real recorder would.
func Gain(x []float64, g float64) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = math.Max(-1, math.Min(1, v*g))
	}
	return out
}

// WriteWAV writes mono 16-bit PCM.
func WriteWAV(w io.Writer, x []float64, sampleRate int) error {
	dataLen := uint32(len(x) * 2)
	hdr := make([]byte, 44)
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], 36+dataLen)
	copy(hdr[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1) // PCM
	binary.LittleEndian.PutUint16(hdr[22:], 1) // mono
	binary.LittleEndian.PutUint32(hdr[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(hdr[28:], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(hdr[32:], 2)
	binary.LittleEndian.PutUint16(hdr[34:], 16)
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], dataLen)
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	buf := make([]byte, len(x)*2)
	for i, v := range x {
		v = math.Max(-1, math.Min(1, v))
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(int16(math.Round(v*32767))))
	}
	_, err := w.Write(buf)
	return err
}
