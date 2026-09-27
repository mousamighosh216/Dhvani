package matcher

import (
	"Dhvani/recognition/fingerprint"
	"fmt"
)

type Fingerprint struct {
	Hash         uint32
	SongID       int
	AbsoluteTime uint32 // t_query
}

type DBMatch struct {
	SongID         int
	DatabaseOffset uint32 // t_db
}

type MatchResult struct {
	SongID     int
	Confidence float64
	OffsetSec  float64
}

// interface used for dynamic usecase formation
type FingerprintStore interface {
	QueryHashes(hashes []uint32) (map[uint32][]DBMatch, error)
}

func MatchQuery(queryFps []fingerprint.Fingerprint, store FingerprintStore, sampleRate int, hopSize int) (*MatchResult, error) {
	if len(queryFps) == 0 {
		return nil, fmt.Errorf("empty query fingerprint slice.")
	}

	hashList := make([]uint32, len(queryFps))
	queryTimeMap := make(map[uint32][]uint32)

	for _, qfp := range queryFps {
		hashList = append(hashList, qfp.Hash)
		queryTimeMap[qfp.Hash] = append(queryTimeMap[qfp.Hash], qfp.AbsoluteTime)
	}

	dbMatchesMap, err := store.QueryHashes(hashList)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if len(dbMatchesMap) == 0 {
		return &MatchResult{SongID: 0, Confidence: 0.0}, nil
	}

	votes := make(map[int]map[int32]int)
	totalMatchesFound := 0

	for hash, dbMatches := range dbMatchesMap {
		queryOffsets := queryTimeMap[hash]

		// O(matches*offset) may explode for large storage
		for _, dbM := range dbMatches {
			for _, qOffset := range queryOffsets {
				offsetDiff := int32(dbM.DatabaseOffset) - int32(qOffset)
				if _, exists := votes[dbM.SongID]; !exists {
					votes[dbM.SongID] = make(map[int32]int)
				}
				votes[dbM.SongID][offsetDiff]++
				totalMatchesFound++
			}
		}
	}

	topSongID := 0
	topVotes := 0
	winningOffset := int32(0)

	for songID, offsetMap := range votes {
		for offsetDiff, voteCount := range offsetMap {
			if voteCount > topVotes {
				topVotes = voteCount
				topSongID = songID
				winningOffset = offsetDiff
			}
		}
	}

	// 5. Calculate Confidence Score
	// Confidence is the ratio of top cluster votes relative to total query fingerprints
	confidence := float64(topVotes) / float64(len(queryFps))
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Convert winning frame offset to seconds
	offsetSeconds := float64(winningOffset*int32(hopSize)) / float64(sampleRate)

	return &MatchResult{
		SongID:     topSongID,
		Confidence: confidence,
		OffsetSec:  offsetSeconds,
	}, nil
}
