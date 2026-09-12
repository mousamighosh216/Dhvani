package fingerprint

// Fingerprint holds the generated hash, song identifier, and absolute timestamp offset
type Fingerprint struct {
	Hash       uint32 // Packed (f1, f2, delta_t) hash key
	SongID     int    // Associated song ID in database
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

func GenerateFingerprints(peakList []peaks.Peak, songID int, zone TargetZone) [] {

}