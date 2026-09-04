package fft

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/lucasb-eyer/go-colorful"
)

// SaveSpectrogramPNG exports the STFT magnitude matrix to a PNG heatmap image.
func SaveSpectrogramPNG(spec *Spectrogram, outputPath string) error {
	numFrames := len(spec.Magnitudes)
	if numFrames == 0 {
		return fmt.Errorf("empty spectrogram matrix")
	}

	numBins := len(spec.Magnitudes[0])

	// We flip the Y-axis so low frequencies are at the bottom and high frequencies at the top
	img := image.NewRGBA(image.Rect(0, 0, numFrames, numBins))

	// Find max magnitude for dB scaling/normalization
	maxMag := 0.0
	for t := 0; t < numFrames; t++ {
		for f := 0; f < numBins; f++ {
			if spec.Magnitudes[t][f] > maxMag {
				maxMag = spec.Magnitudes[t][f]
			}
		}
	}

	if maxMag == 0 {
		maxMag = 1.0
	}

	// Define color gradient stops (Dark Blue -> Purple -> Red -> Yellow -> White)
	gradient := []colorful.Color{
		{R: 0.0, G: 0.0, B: 0.1}, // Silence / Deep Background
		{R: 0.2, G: 0.0, B: 0.5}, // Low Energy
		{R: 0.8, G: 0.1, B: 0.2}, // Medium Energy
		{R: 1.0, G: 0.7, B: 0.0}, // High Energy
		{R: 1.0, G: 1.0, B: 1.0}, // Peak Energy
	}

	// Map magnitudes to pixel colors
	for t := 0; t < numFrames; t++ {
		for f := 0; f < numBins; f++ {
			mag := spec.Magnitudes[t][f]

			// Convert linear magnitude to Decibels (dB scale) for human visual perception
			// dB = 20 * log10(mag / maxMag)
			db := 20.0 * math.Log10((mag+1e-9)/maxMag)

			// Normalize dB from range [-80dB, 0dB] to [0.0, 1.0]
			normVal := (db + 80.0) / 80.0
			if normVal < 0 {
				normVal = 0
			} else if normVal > 1.0 {
				normVal = 1.0
			}

			// Interpolate color from gradient
			c := getGradientColor(gradient, normVal)

			// Invert Y-axis: f = 0 (0 Hz) at bottom, f = numBins-1 (Nyquist) at top
			img.Set(t, numBins-1-f, c)
		}
	}

	// Create output PNG file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer outFile.Close()

	return png.Encode(outFile, img)
}

// getGradientColor calculates smooth color transitions across multi-stop gradients
func getGradientColor(stops []colorful.Color, val float64) color.Color {
	if val <= 0 {
		return stops[0]
	}
	if val >= 1.0 {
		return stops[len(stops)-1]
	}

	scaled := val * float64(len(stops)-1)
	idx := int(scaled)
	frac := scaled - float64(idx)

	c1 := stops[idx]
	c2 := stops[idx+1]

	blended := c1.BlendHsv(c2, frac)
	r, g, b := blended.RGB255()
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
