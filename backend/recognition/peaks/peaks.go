// Package peaks extracts spectral landmarks from a spectrogram.
package peaks

import (
	"math"
	"sort"

	"dhwani/recognition/fft"
)

// Peak is a spectral landmark. Magnitude is in dB.
type Peak struct {
	TimeFrame int
	FreqBin   int
	Magnitude float64
}

// Params control peak picking.
type Params struct {
	TimeRadius int   // half-height of the local-maximum neighbourhood, in frames
	FreqRadius int   // half-width of the neighbourhood, in bins
	MinFreqBin int   // lowest bin considered (inclusive)
	MaxFreqBin int   // highest bin considered (exclusive)
	BandEdges  []int // ascending band boundaries for filterFrequencyBands
}

// DefaultParams are tuned for a 2048-point FFT at 44.1 kHz (21.5 Hz per bin).
func DefaultParams() Params {
	return Params{
		TimeRadius: 12,
		FreqRadius: 15,
		MinFreqBin: 4,   // ~86 Hz, skips rumble
		MaxFreqBin: 512, // ~11 kHz
		BandEdges:  []int{4, 12, 24, 48, 96, 192, 512},
	}
}

// DefaultThresholdDB: a peak must be within this many dB of the loudest
// bin in the clip.
const DefaultThresholdDB = 40.0

// ExtractPeaks returns local maxima of the spectrogram (in a
// (2*TimeRadius+1) x (2*FreqRadius+1) neighbourhood) that are no more than
// thresholdDB below the loudest point, then balances them across bands.
func ExtractPeaks(spec *fft.Spectrogram, thresholdDB float64) []Peak {
	return ExtractPeaksWith(spec, thresholdDB, DefaultParams())
}

// ExtractPeaksWith is ExtractPeaks with explicit parameters.
func ExtractPeaksWith(spec *fft.Spectrogram, thresholdDB float64, p Params) []Peak {
	if spec == nil || spec.NumFrames() == 0 {
		return nil
	}
	lo, hi := p.MinFreqBin, p.MaxFreqBin
	if hi > spec.NumBins() {
		hi = spec.NumBins()
	}
	if lo >= hi {
		return nil
	}
	frames, width := spec.NumFrames(), hi-lo

	// dB matrix restricted to [lo, hi).
	db := make([][]float32, frames)
	globalMax := float32(-math.MaxFloat32)
	for t := 0; t < frames; t++ {
		row := make([]float32, width)
		for k := 0; k < width; k++ {
			v := float32(20 * math.Log10(spec.Magnitudes[t][lo+k]+1e-9))
			row[k] = v
			if v > globalMax {
				globalMax = v
			}
		}
		db[t] = row
	}
	if globalMax < -80 { // digital silence
		return nil
	}
	floor := globalMax - float32(thresholdDB)

	// Separable max filter: along frequency, then along time.
	freqMax := make([][]float32, frames)
	for t := 0; t < frames; t++ {
		out := make([]float32, width)
		for k := 0; k < width; k++ {
			a, b := max(0, k-p.FreqRadius), min(width-1, k+p.FreqRadius)
			m := db[t][a]
			for j := a + 1; j <= b; j++ {
				if db[t][j] > m {
					m = db[t][j]
				}
			}
			out[k] = m
		}
		freqMax[t] = out
	}

	var found []Peak
	for t := 0; t < frames; t++ {
		a, b := max(0, t-p.TimeRadius), min(frames-1, t+p.TimeRadius)
		for k := 0; k < width; k++ {
			v := db[t][k]
			if v < floor {
				continue
			}
			isMax := true
			for j := a; j <= b; j++ {
				if freqMax[j][k] > v {
					isMax = false
					break
				}
			}
			if isMax {
				found = append(found, Peak{TimeFrame: t, FreqBin: lo + k, Magnitude: float64(v)})
			}
		}
	}
	return filterFrequencyBands(found, p.BandEdges)
}

// filterFrequencyBands keeps, inside every frequency band, only peaks at or
// above that band's mean peak level, so loud low frequencies cannot crowd out
// quieter mid and high landmarks.
func filterFrequencyBands(found []Peak, edges []int) []Peak {
	if len(edges) < 2 {
		return found
	}
	band := func(bin int) int {
		for i := 0; i+1 < len(edges); i++ {
			if bin >= edges[i] && bin < edges[i+1] {
				return i
			}
		}
		return -1
	}
	n := len(edges) - 1
	sum := make([]float64, n)
	cnt := make([]int, n)
	for _, pk := range found {
		if b := band(pk.FreqBin); b >= 0 {
			sum[b] += pk.Magnitude
			cnt[b]++
		}
	}
	var kept []Peak
	for _, pk := range found {
		b := band(pk.FreqBin)
		if b < 0 {
			continue
		}
		if pk.Magnitude >= sum[b]/float64(cnt[b]) {
			kept = append(kept, pk)
		}
	}
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].TimeFrame != kept[j].TimeFrame {
			return kept[i].TimeFrame < kept[j].TimeFrame
		}
		return kept[i].FreqBin < kept[j].FreqBin
	})
	return kept
}
