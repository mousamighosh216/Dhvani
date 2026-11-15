// Package fingerprint pairs spectral peaks into compact 32-bit hashes.
package fingerprint

import (
	"sort"

	"dhwani/recognition/peaks"
)

// Hash layout (32 bits): | f1: 10 bits | f2: 10 bits | delta t: 12 bits |
// 10 bits hold all 1024 bins of a 2048-point FFT; 12 bits allow a frame gap
// up to 4095 (~47 s), far above the target zone used here.
const (
	FreqBits  = 10
	DeltaBits = 12
	freqMask  = 1<<FreqBits - 1
	deltaMask = 1<<DeltaBits - 1
)

// Fingerprint is one hash anchored at SongOffset (a frame index).
type Fingerprint struct {
	Hash       uint32
	SongID     int
	SongOffset uint32
}

// Params define the target zone behind each anchor peak.
type Params struct {
	FanOut    int // max targets paired per anchor
	MinDeltaT int // frames, inclusive
	MaxDeltaT int // frames, inclusive
}

// DefaultParams pairs each anchor with up to 10 following peaks that are
// 1..96 frames (~1.1 s) ahead.
func DefaultParams() Params {
	return Params{FanOut: 10, MinDeltaT: 1, MaxDeltaT: 96}
}

// CreateHash packs (f1, f2, deltaTime) into 10+10+12 bits.
func CreateHash(f1 int, f2 int, deltaTime int) uint32 {
	return uint32(f1&freqMask)<<(FreqBits+DeltaBits) |
		uint32(f2&freqMask)<<DeltaBits |
		uint32(deltaTime&deltaMask)
}

// UnpackHash is the inverse of CreateHash.
func UnpackHash(h uint32) (f1, f2, deltaTime int) {
	return int(h >> (FreqBits + DeltaBits) & freqMask),
		int(h >> DeltaBits & freqMask),
		int(h & deltaMask)
}

// GenerateFingerprints pairs anchors with targets using DefaultParams.
func GenerateFingerprints(pk []peaks.Peak, songID int) []Fingerprint {
	return GenerateWith(pk, songID, DefaultParams())
}

// GenerateWith pairs each anchor peak with the next FanOut peaks inside its
// target zone and hashes every pair.
func GenerateWith(pk []peaks.Peak, songID int, p Params) []Fingerprint {
	sorted := make([]peaks.Peak, len(pk))
	copy(sorted, pk)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].TimeFrame != sorted[j].TimeFrame {
			return sorted[i].TimeFrame < sorted[j].TimeFrame
		}
		return sorted[i].FreqBin < sorted[j].FreqBin
	})
	var out []Fingerprint
	for i, a := range sorted {
		paired := 0
		for j := i + 1; j < len(sorted) && paired < p.FanOut; j++ {
			dt := sorted[j].TimeFrame - a.TimeFrame
			if dt > p.MaxDeltaT {
				break
			}
			if dt < p.MinDeltaT {
				continue
			}
			out = append(out, Fingerprint{
				Hash:       CreateHash(a.FreqBin, sorted[j].FreqBin, dt),
				SongID:     songID,
				SongOffset: uint32(a.TimeFrame),
			})
			paired++
		}
	}
	return out
}
