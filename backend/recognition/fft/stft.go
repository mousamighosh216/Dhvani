package fft

import (
	"math"
	"math/cmplx"

	godsp "github.com/madelynnblue/go-dsp/fft"
)

// spectrogram is a 2d matrix of magnitude time_frame*frequency_bin
type Spectrogram struct {
	Magnitudes [][]float64 // magnitude[t][f]
	SampleRate int
	WindowSize int
	HopSize    int
}

// computestft performs short time fourier transform on normalized mono pcm samples
func ComputeSTFT(samples []float64, sampleRate int, windowSize int, hopSize int) *Spectrogram {
	numSamples := len(samples)
	if numSamples < windowSize {
		return &Spectrogram{Magnitudes: [][]float64{}, SampleRate: sampleRate}
	}

	// calculate total time frames
	numFrames := (numSamples - windowSize) / hopSize

	// hann window curve
	hannWindow := make([]float64, windowSize)
	for i := 0; i < windowSize; i++ {
		hannWindow[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(windowSize-1)))
	}

	// Spectrogram matrix storage (numFrames x windowSize/2)
	numBins := windowSize / 2
	spectrogram := make([][]float64, numFrames)

	// Sliding window over audio samples
	for t := 0; t < numFrames; t++ {
		startIndex := t * hopSize
		windowBuffer := make([]float64, windowSize)

		// 1. Apply Hann Window to slice
		for n := 0; n < windowSize; n++ {
			windowBuffer[n] = samples[startIndex+n] * hannWindow[n]
		}

		// 2. Compute FFT
		fftOutput := godsp.FFTReal(windowBuffer)

		// 3. Extract magnitude for positive frequency bins (0 to N/2)
		spectrogram[t] = make([]float64, numBins)
		for f := range numBins {
			// Magnitude = sqrt(real^2 + imag^2)
			spectrogram[t][f] = cmplx.Abs(fftOutput[f])
		}
	}

	return &Spectrogram{
		Magnitudes: spectrogram,
		SampleRate: sampleRate,
		WindowSize: windowSize,
		HopSize:    hopSize,
	}
}
