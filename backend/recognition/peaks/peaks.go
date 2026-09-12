package peaks

import "Dhvani/recognition/fft"

// strong spectral landmark in the 2d spectrogram
type Peak struct {
	TimeFrame int
	FreqBin   int
	Magnitude float64
}

func ExtractPeaks(spec **fft.Spectrogram, neighbourhoodTime int, neighbourhoodFreq int, minMagnitude float64) []Peak {
	numFrames := len(spec.Magnitudes)
	if numFrames == 0 {
		return nil
	}
	numBins := len(spec.Magnitudes[0])
	var peaks []Peak
}
