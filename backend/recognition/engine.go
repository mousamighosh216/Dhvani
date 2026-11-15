// Package recognition wires the DSP pipeline into one Engine.
package recognition

import (
	"context"
	"errors"

	"dhwani/recognition/audio"
	"dhwani/recognition/fft"
	"dhwani/recognition/fingerprint"
	"dhwani/recognition/matcher"
	"dhwani/recognition/peaks"
)

// ErrTooShort is returned for audio shorter than one STFT window.
var ErrTooShort = errors.New("recognition: audio is too short")

// Engine is the contract used by the API layer.
type Engine interface {
	// Recognize returns (nil, nil) when no song matches.
	Recognize(ctx context.Context, a *audio.AudioData) (*matcher.MatchResult, error)
	FingerprintSong(songID int, a *audio.AudioData) ([]fingerprint.Fingerprint, error)
}

// DefaultEngine: decode -> resample -> STFT -> peaks -> hashes -> vote.
type DefaultEngine struct {
	store matcher.FingerprintStore
}

// NewEngine builds an engine over a fingerprint store.
func NewEngine(store matcher.FingerprintStore) *DefaultEngine {
	return &DefaultEngine{store: store}
}

func fingerprintAudio(a *audio.AudioData, songID int) ([]fingerprint.Fingerprint, error) {
	if a == nil {
		return nil, ErrTooShort
	}
	samples := audio.Resample(a.Samples, a.SampleRate, fft.TargetSampleRate)
	if len(samples) < fft.WindowSize {
		return nil, ErrTooShort
	}
	spec := fft.ComputeSTFT(samples, fft.TargetSampleRate, fft.WindowSize, fft.HopSize)
	pk := peaks.ExtractPeaks(spec, peaks.DefaultThresholdDB)
	return fingerprint.GenerateFingerprints(pk, songID), nil
}

// FingerprintSong fingerprints a full track for the catalog.
func (e *DefaultEngine) FingerprintSong(songID int, a *audio.AudioData) ([]fingerprint.Fingerprint, error) {
	return fingerprintAudio(a, songID)
}

// Recognize fingerprints the clip and votes against the store.
func (e *DefaultEngine) Recognize(ctx context.Context, a *audio.AudioData) (*matcher.MatchResult, error) {
	fps, err := fingerprintAudio(a, 0)
	if err != nil {
		return nil, err
	}
	return matcher.MatchQuery(ctx, fps, e.store)
}
