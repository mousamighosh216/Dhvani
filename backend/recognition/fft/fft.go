// Package fft: radix-2 FFT, Hann window and a dB-magnitude STFT.
package fft

import (
	"math"
	"math/bits"
)

type Plan struct {
	n        int
	cos, sin []float64
	rev      []int
}

func NewPlan(n int) *Plan {
	p := &Plan{n: n, cos: make([]float64, n/2), sin: make([]float64, n/2), rev: make([]int, n)}
	for i := 0; i < n/2; i++ {
		a := -2 * math.Pi * float64(i) / float64(n)
		p.cos[i], p.sin[i] = math.Cos(a), math.Sin(a)
	}
	nb := bits.Len(uint(n)) - 1
	for i := range p.rev {
		p.rev[i] = int(bits.Reverse64(uint64(i)) >> (64 - nb))
	}
	return p
}

// Transform runs an in-place forward FFT. len(re) == len(im) == n (power of two).
func (p *Plan) Transform(re, im []float64) {
	n := p.n
	for i := 0; i < n; i++ {
		if j := p.rev[i]; j > i {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		half, step := size/2, n/size
		for start := 0; start < n; start += size {
			for k := 0; k < half; k++ {
				wr, wi := p.cos[k*step], p.sin[k*step]
				a, b := start+k, start+k+half
				tr := re[b]*wr - im[b]*wi
				ti := re[b]*wi + im[b]*wr
				re[b], im[b] = re[a]-tr, im[a]-ti
				re[a], im[a] = re[a]+tr, im[a]+ti
			}
		}
	}
}

func Hann(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n))
	}
	return w
}

// Spectrogram returns frames x (n/2) magnitudes in dB.
func Spectrogram(x []float32, n, hop int) [][]float32 {
	if len(x) < n {
		return nil
	}
	frames := (len(x)-n)/hop + 1
	win, plan := Hann(n), NewPlan(n)
	re, im := make([]float64, n), make([]float64, n)
	out := make([][]float32, frames)
	for f := range out {
		off := f * hop
		for i := 0; i < n; i++ {
			re[i], im[i] = float64(x[off+i])*win[i], 0
		}
		plan.Transform(re, im)
		row := make([]float32, n/2)
		for k := range row {
			row[k] = float32(20 * math.Log10(math.Hypot(re[k], im[k])+1e-6))
		}
		out[f] = row
	}
	return out
}
