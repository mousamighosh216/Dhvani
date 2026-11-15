// Package fft computes the Short-Time Fourier Transform spectrogram.
package fft

import (
	"math"
	"math/bits"
	"runtime"
	"sync"
)

// Pipeline constants (see the DSP notes): N=2048 at 44.1 kHz is ~46 ms,
// hop 512 is ~11.6 ms, i.e. 75% overlap.
const (
	TargetSampleRate = 44100
	WindowSize       = 2048
	HopSize          = 512
)

// Spectrogram holds a time-frequency magnitude matrix.
// Magnitudes[frame][bin], with WindowSize/2 bins per frame.
type Spectrogram struct {
	Magnitudes [][]float64
	SampleRate int
	WindowSize int
	HopSize    int
}

// NumFrames returns the number of time frames.
func (s *Spectrogram) NumFrames() int { return len(s.Magnitudes) }

// NumBins returns the number of frequency bins per frame.
func (s *Spectrogram) NumBins() int { return s.WindowSize / 2 }

// FrameSeconds is the time between consecutive frames.
func (s *Spectrogram) FrameSeconds() float64 {
	return float64(s.HopSize) / float64(s.SampleRate)
}

// BinHz is the width of one frequency bin.
func (s *Spectrogram) BinHz() float64 {
	return float64(s.SampleRate) / float64(s.WindowSize)
}

// plan holds precomputed tables for one power-of-two FFT size.
type plan struct {
	n        int
	rev      []int
	cos, sin []float64
}

var (
	plansMu sync.Mutex
	plans   = map[int]*plan{}
)

func getPlan(n int) *plan {
	plansMu.Lock()
	defer plansMu.Unlock()
	if p, ok := plans[n]; ok {
		return p
	}
	p := &plan{n: n, rev: make([]int, n), cos: make([]float64, n/2), sin: make([]float64, n/2)}
	shift := uint(bits.UintSize - bits.Len(uint(n-1)))
	for i := 0; i < n; i++ {
		p.rev[i] = int(bits.Reverse(uint(i)) >> shift)
	}
	for i := 0; i < n/2; i++ {
		a := -2 * math.Pi * float64(i) / float64(n)
		p.cos[i], p.sin[i] = math.Cos(a), math.Sin(a)
	}
	plans[n] = p
	return p
}

// FFT computes an in-place radix-2 complex FFT. len(re) must be a power of two.
func FFT(re, im []float64) {
	n := len(re)
	p := getPlan(n)
	p.transform(re, im)
}

func (p *plan) transform(re, im []float64) {
	n := p.n
	for i := 0; i < n; i++ {
		if j := p.rev[i]; j > i {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		half := size / 2
		step := n / size
		for start := 0; start < n; start += size {
			for k := 0; k < half; k++ {
				wr, wi := p.cos[k*step], p.sin[k*step]
				a, b := start+k, start+k+half
				tr := re[b]*wr - im[b]*wi
				ti := re[b]*wi + im[b]*wr
				re[b], im[b] = re[a]-tr, im[a]-ti
				re[a] += tr
				im[a] += ti
			}
		}
	}
}

// HannWindow returns w[n] = 0.5 * (1 - cos(2*pi*n/(N-1))).
func HannWindow(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return w
}

// ComputeSTFT applies a Hann window over sliding frames and runs FFTs,
// keeping the first windowSize/2 magnitude bins of each frame.
// windowSize must be a power of two. Returns nil on invalid parameters.
// Input shorter than one window yields a spectrogram with zero frames.
func ComputeSTFT(samples []float64, sampleRate int, windowSize int, hopSize int) *Spectrogram {
	if windowSize < 2 || windowSize&(windowSize-1) != 0 || hopSize < 1 || sampleRate < 1 {
		return nil
	}
	spec := &Spectrogram{SampleRate: sampleRate, WindowSize: windowSize, HopSize: hopSize}
	if len(samples) < windowSize {
		return spec
	}
	frames := 1 + (len(samples)-windowSize)/hopSize
	spec.Magnitudes = make([][]float64, frames)

	window := HannWindow(windowSize)
	p := getPlan(windowSize)
	bins := windowSize / 2

	workers := runtime.NumCPU()
	if workers > frames {
		workers = frames
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			re := make([]float64, windowSize)
			im := make([]float64, windowSize)
			for f := w; f < frames; f += workers {
				base := f * hopSize
				for i := 0; i < windowSize; i++ {
					re[i] = samples[base+i] * window[i]
					im[i] = 0
				}
				p.transform(re, im)
				mags := make([]float64, bins)
				for k := 0; k < bins; k++ {
					mags[k] = math.Hypot(re[k], im[k])
				}
				spec.Magnitudes[f] = mags
			}
		}(w)
	}
	wg.Wait()
	return spec
}
