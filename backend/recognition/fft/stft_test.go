package fft

import (
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"testing"
)

func TestFFTMatchesNaiveDFT(t *testing.T) {
	n := 64
	re := make([]float64, n)
	im := make([]float64, n)
	x := make([]complex128, n)
	for i := range re {
		re[i] = math.Sin(float64(i)*0.3) + 0.5*math.Cos(float64(i)*1.1)
		x[i] = complex(re[i], 0)
	}
	FFT(re, im)
	for k := 0; k < n; k++ {
		var s complex128
		for j := 0; j < n; j++ {
			s += x[j] * cmplx.Exp(complex(0, -2*math.Pi*float64(j*k)/float64(n)))
		}
		if cmplx.Abs(s-complex(re[k], im[k])) > 1e-9 {
			t.Fatalf("bin %d: fft=%v dft=%v", k, complex(re[k], im[k]), s)
		}
	}
}

func TestHannWindow(t *testing.T) {
	w := HannWindow(2048)
	if w[0] != 0 || math.Abs(w[2047]) > 1e-12 {
		t.Errorf("edges should be zero: %v %v", w[0], w[2047])
	}
	if math.Abs(w[1023]-1) > 1e-5 {
		t.Errorf("centre ~1, got %v", w[1023])
	}
}

func TestSTFTShapeAndOverlap(t *testing.T) {
	samples := make([]float64, 44100) // 1 s
	s := ComputeSTFT(samples, 44100, 2048, 512)
	wantFrames := 1 + (44100-2048)/512
	if s.NumFrames() != wantFrames || s.NumBins() != 1024 {
		t.Fatalf("frames=%d bins=%d", s.NumFrames(), s.NumBins())
	}
	if len(s.Magnitudes[0]) != 1024 {
		t.Fatalf("row width %d", len(s.Magnitudes[0]))
	}
	// 75% overlap: 1536 of 2048 samples shared between consecutive frames.
	if 1-float64(s.HopSize)/float64(s.WindowSize) != 0.75 {
		t.Error("overlap should be 75%")
	}
	if ComputeSTFT(samples, 44100, 1000, 512) != nil {
		t.Error("non power-of-two window must return nil")
	}
	if got := ComputeSTFT(make([]float64, 100), 44100, 2048, 512); got.NumFrames() != 0 {
		t.Error("short input must give zero frames")
	}
}

func TestSTFTSinePeakBin(t *testing.T) {
	const sr = 44100
	freq := 1000.0
	samples := make([]float64, sr)
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * freq * float64(i) / sr)
	}
	s := ComputeSTFT(samples, sr, 2048, 512)
	wantBin := int(math.Round(freq / s.BinHz()))
	for _, f := range []int{0, 10, s.NumFrames() - 1} {
		best := 0
		for k, m := range s.Magnitudes[f] {
			if m > s.Magnitudes[f][best] {
				best = k
			}
		}
		if best != wantBin {
			t.Errorf("frame %d: peak bin %d, want %d", f, best, wantBin)
		}
	}
}

func TestSaveSpectrogramPNG(t *testing.T) {
	samples := make([]float64, 44100)
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * 2000 * float64(i) / 44100)
	}
	s := ComputeSTFT(samples, 44100, 2048, 512)
	path := filepath.Join(t.TempDir(), "spec.png")
	if err := SaveSpectrogramPNG(s, path); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(path); err != nil || st.Size() < 100 {
		t.Fatalf("png not written: %v", err)
	}
	if err := SaveSpectrogramPNG(&Spectrogram{}, path); err == nil {
		t.Error("empty spectrogram should error")
	}
}
