// Package recognition ties decoding, STFT, peaks, hashing and matching together.
package recognition

import (
	"context"

	"dhwani/metadata"
	"dhwani/recognition/fft"
	"dhwani/recognition/fingerprint"
	"dhwani/recognition/matcher"
	"dhwani/recognition/peaks"
)

const (
	WindowSize = 2048
	Hop        = 512 // 75% overlap
	PeakDT     = 12  // 25-frame neighbourhood
	PeakDF     = 15  // 31-bin neighbourhood
	RangeDB    = 30
)

func Fingerprint(x []float32) []fingerprint.Fingerprint {
	spec := fft.Spectrogram(x, WindowSize, Hop)
	return fingerprint.Generate(peaks.Find(spec, PeakDT, PeakDF, RangeDB))
}

func Recognize(ctx context.Context, st metadata.Store, x []float32) (matcher.Result, int, error) {
	fps := Fingerprint(x)
	seen := map[uint32]bool{}
	var hashes []uint32
	for _, f := range fps {
		if !seen[f.Hash] {
			seen[f.Hash] = true
			hashes = append(hashes, f.Hash)
		}
	}
	hits, err := st.Lookup(ctx, hashes)
	if err != nil {
		return matcher.Result{}, len(fps), err
	}
	return matcher.Match(fps, hits), len(fps), nil
}
