package matcher

import (
	"Dhvani/recognition/fingerprint"
	"fmt"
)

type DBMatch struct {
	SongID         int
	DatabaseOffset uint32
}

type FingerprintStore interface {
	QueryHashes(hashes []uint32) (map[uint32][]DBMatch, error)
}

type MatchResult struct {
	SongID     int
	Confidence float64
	OffsetSet  float64
}

func MatchQuery(queryFps []fingerprint.Fingerprint, store FingerprintStore, sampleRate int, hopSize int) (*MatchResult, error) {
	if len(queryFps) == 0 {
		return nil, fmt.Errorf("empty query fingerprints slice")
	}

	hashList := make([]uint32, len(queryFps))
	queryTimeMap := make(map[uint32][]uint32)

	for _, qfp := range queryFps {

	}
}
