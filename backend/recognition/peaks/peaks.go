package peaks

import "Dhvani/recognition/fft"

// strong spectral landmark in the 2d spectrogram
type Peak struct {
	TimeFrame int
	FreqBin   int
	Magnitude float64
}

func ExtractPeaks(spec *fft.Spectrogram, neighbourhoodTime int, neighbourhoodFreq int, minMagnitude float64) []Peak {
	numFrames := len(spec.Magnitudes)
	if numFrames == 0 {
		return nil
	}
	numBins := len(spec.Magnitudes[0])
	var peaks []Peak

	// iterate through time and freq grid
	for t := 0; t < numFrames; t++ {
		for f := 0; f < numBins; f++ {
			mag := spec.Magnitudes[t][f]

			// background noice skipped
			if mag < minMagnitude {
				continue
			}
			// check if this point is the absolute min in its 2d neighbour
			if isLocalMax((*spec).Magnitudes, t, f, neighbourhoodTime, neighbourhoodFreq, numFrames, numBins) {
				peaks = append(peaks, Peak{
					TimeFrame: t,
					FreqBin:   f,
					Magnitude: mag,
				})
			}
		}
	}
	return peaks
}

// isLocalMax checks if the current cell (t, f) is strictly greater than all surrounding neighbors
func isLocalMax(matrix [][]float64, t int, f int, nTime int, nFreq int, maxT int, maxF int) bool {
	currentMag := matrix[t][f]

	tMin := max(0, t-nTime)
	tMax := min(maxT-1, t+nTime)
	fMin := max(0, f-nFreq)
	fMax := min(maxF-1, f+nFreq)

	for nt := tMin; nt <= tMax; nt++ {
		for nf := fMin; nf <= fMax; nf++ {
			// Skip comparing the cell with itself
			if nt == t && nf == f {
				continue
			}
			if matrix[nt][nf] >= currentMag {
				return false
			}
		}
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
