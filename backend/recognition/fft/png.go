package fft

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// DefaultGradient goes black -> purple -> red -> orange -> yellow -> white.
var DefaultGradient = []color.RGBA{
	{0, 0, 0, 255},
	{60, 10, 110, 255},
	{190, 30, 80, 255},
	{245, 130, 20, 255},
	{255, 230, 80, 255},
	{255, 255, 255, 255},
}

// getGradientColor interpolates between gradient stops for val in [0, 1].
func getGradientColor(stops []color.RGBA, val float64) color.Color {
	if val <= 0 {
		return stops[0]
	}
	if val >= 1 {
		return stops[len(stops)-1]
	}
	pos := val * float64(len(stops)-1)
	i := int(pos)
	t := pos - float64(i)
	a, b := stops[i], stops[i+1]
	lerp := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255}
}

// SaveSpectrogramPNG writes a heatmap (time on x, frequency on y with low
// frequencies at the bottom) of the dB magnitudes with an 80 dB range.
func SaveSpectrogramPNG(spec *Spectrogram, outputPath string) error {
	if spec == nil || spec.NumFrames() == 0 {
		return fmt.Errorf("fft: empty spectrogram")
	}
	const rangeDB = 80.0
	frames, bins := spec.NumFrames(), spec.NumBins()
	maxDB := -math.MaxFloat64
	for _, row := range spec.Magnitudes {
		for _, m := range row {
			if db := 20 * math.Log10(m+1e-9); db > maxDB {
				maxDB = db
			}
		}
	}
	stride := 1
	for frames/stride > 4000 { // keep the image a sane width
		stride++
	}
	w := frames / stride
	img := image.NewRGBA(image.Rect(0, 0, w, bins))
	for x := 0; x < w; x++ {
		row := spec.Magnitudes[x*stride]
		for k := 0; k < bins; k++ {
			db := 20 * math.Log10(row[k]+1e-9)
			img.Set(x, bins-1-k, getGradientColor(DefaultGradient, (db-(maxDB-rangeDB))/rangeDB))
		}
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
