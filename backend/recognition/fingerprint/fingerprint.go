package fingerprint

import "Dhvani/recognition/peaks"

// Fingerprint holds the generated hash, song identifier, and absolute timestamp offset
type Fingerprint struct {
	Hash         uint32 // Packed (f1, f2, delta_t) hash key
	SongID       int    // Associated song ID in database
	AbsoluteTime uint32 // Frame index t1 where the anchor peak occurred
}

// TargetZone defines the search bounding box relative to an anchor peak
type TargetZone struct {
	MinDeltaTime int // Minimum time frame offset ahead (e.g., 5 frames)
	MaxDeltaTime int // Maximum time frame offset ahead (e.g., 65 frames ~1.5s)
	MinFreqDiff  int // Freq bin lower bound shift
	MaxFreqDiff  int // Freq bin upper bound shift
}

// DefaultTargetZone creates a standard Shazam-style target search box
func DefaultTargetZone() TargetZone {
	return TargetZone{
		MinDeltaTime: 3,  // Start searching 3 frames ahead
		MaxDeltaTime: 50, // Search up to 50 frames (~1.1 seconds) ahead
		MinFreqDiff:  -30,
		MaxFreqDiff:  30,
	}
}

// iterates over extracted peaks and pairs anchor peaks with target peaks
func GenerateFingerprints(peakList []peaks.Peak, songID int, zone TargetZone) []Fingerprint {
	numPeaks := len(peakList)
	var fingerprints []Fingerprint

	// Iterate over each peak as a potential Anchor Peak
	for i := 0; i < numPeaks; i++ {
		anchor := peakList[i]

		// Search subsequent peaks as potential Target Peaks
		for j := i + 1; j < numPeaks; j++ {
			target := peakList[j]

			deltaTime := target.TimeFrame - anchor.TimeFrame

			// If target peak is beyond the maximum time range, break early
			if deltaTime > zone.MaxDeltaTime {
				break
			}

			// Check if target peak falls within the target zone criteria
			if deltaTime >= zone.MinDeltaTime {
				freqDiff := target.FreqBin - anchor.FreqBin
				if freqDiff >= zone.MinFreqDiff && freqDiff <= zone.MaxFreqDiff {
					// Quantize values to fit bit allocation
					f1 := uint32(anchor.FreqBin & 0xFFFF) // 16 bits
					f2 := uint32(target.FreqBin & 0xFF)   // 8 bits
					dt := uint32(deltaTime & 0xFF)        // 8 bits

					// Pack into 32-bit Hash: (f1 << 16) | (f2 << 8) | dt
					hash := (f1 << 16) | (f2 << 8) | dt

					fingerprints = append(fingerprints, Fingerprint{
						Hash:         hash,
						SongID:       songID,
						AbsoluteTime: uint32(anchor.TimeFrame),
					})
				}
			}
		}
	}

	return fingerprints
}
