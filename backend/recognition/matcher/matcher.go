// Package matcher looks up fingerprints and votes on time-offset alignment.
package matcher

import (
	"context"
	"math"

	"dhwani/recognition/fft"
	"dhwani/recognition/fingerprint"
)

// DBMatch is one stored fingerprint row returned for a queried hash.
type DBMatch struct {
	Hash       uint32
	SongID     int
	SongOffset uint32
}

// FingerprintStore is the persistence contract (implemented by metadata).
type FingerprintStore interface {
	// QueryHashes returns every stored row whose hash is in hashes.
	QueryHashes(ctx context.Context, hashes []uint32) ([]DBMatch, error)
	SaveFingerprints(ctx context.Context, fps []fingerprint.Fingerprint) error
}

// CandidateMatch pairs a stored offset with the query offset of the same hash.
type CandidateMatch struct {
	SongID         int
	DatabaseOffset uint32
	QueryOffset    uint32
}

// MatchResult describes the winning song.
type MatchResult struct {
	SongID       int
	Confidence   float64 // see CalculateConfidence
	OffsetSec    float64 // where in the song the query clip starts
	OffsetFrames int
	Votes        int // fingerprints agreeing on the winning alignment
}

// Config tunes acceptance.
type Config struct {
	MinVotes     int     // minimum aligned fingerprints to accept a match
	MinMargin    float64 // winner must have at least this many times the runner-up's votes
	FrameSeconds float64 // duration of one STFT frame
}

// DefaultConfig matches the default STFT parameters.
func DefaultConfig() Config {
	return Config{
		MinVotes:     8,
		MinMargin:    1.5,
		FrameSeconds: float64(fft.HopSize) / float64(fft.TargetSampleRate),
	}
}

// MatchQuery runs MatchQueryWith using DefaultConfig.
func MatchQuery(ctx context.Context, queryFps []fingerprint.Fingerprint, store FingerprintStore) (*MatchResult, error) {
	return MatchQueryWith(ctx, DefaultConfig(), queryFps, store)
}

// MatchQueryWith looks up all query hashes, computes
// offset = t_db - t_query for every hit, builds one offset histogram per
// song and returns the song with the densest offset cluster. It returns
// (nil, nil) when nothing is convincing.
func MatchQueryWith(ctx context.Context, cfg Config, queryFps []fingerprint.Fingerprint, store FingerprintStore) (*MatchResult, error) {
	if len(queryFps) == 0 {
		return nil, nil
	}
	byHash := make(map[uint32][]uint32, len(queryFps))
	for _, q := range queryFps {
		byHash[q.Hash] = append(byHash[q.Hash], q.SongOffset)
	}
	hashes := make([]uint32, 0, len(byHash))
	for h := range byHash {
		hashes = append(hashes, h)
	}
	rows, err := store.QueryHashes(ctx, hashes)
	if err != nil {
		return nil, err
	}

	hist := map[int]map[int]int{} // song -> offset delta -> count
	total := 0
	for _, row := range rows {
		for _, qo := range byHash[row.Hash] {
			c := CandidateMatch{SongID: row.SongID, DatabaseOffset: row.SongOffset, QueryOffset: qo}
			delta := int(c.DatabaseOffset) - int(c.QueryOffset)
			if hist[c.SongID] == nil {
				hist[c.SongID] = map[int]int{}
			}
			hist[c.SongID][delta]++
			total++
		}
	}
	if total == 0 {
		return nil, nil
	}

	type best struct{ song, delta, votes int }
	var winner, runnerUp best
	for song, h := range hist {
		b := best{song: song}
		for d := range h {
			// Tolerate +-1 frame of jitter between neighbouring bins.
			if v := h[d-1] + h[d] + h[d+1]; v > b.votes || (v == b.votes && d < b.delta) {
				b.delta, b.votes = d, v
			}
		}
		switch {
		case b.votes > winner.votes || (b.votes == winner.votes && b.song < winner.song):
			runnerUp, winner = winner, b
		case b.votes > runnerUp.votes:
			runnerUp = b
		}
	}

	if winner.votes < cfg.MinVotes {
		return nil, nil
	}
	if runnerUp.votes > 0 && float64(winner.votes) < cfg.MinMargin*float64(runnerUp.votes) {
		return nil, nil
	}
	offset := math.Max(0, float64(winner.delta)*cfg.FrameSeconds)
	return &MatchResult{
		SongID:       winner.song,
		Confidence:   CalculateConfidence(winner.votes, total),
		OffsetSec:    offset,
		OffsetFrames: winner.delta,
		Votes:        winner.votes,
	}, nil
}

// CalculateConfidence returns the share of all hash hits that agree on the
// winning alignment (cluster density), damped when the cluster is tiny.
func CalculateConfidence(topClusterSize int, totalMatches int) float64 {
	if totalMatches <= 0 || topClusterSize <= 0 {
		return 0
	}
	density := math.Min(1, float64(topClusterSize)/float64(totalMatches))
	return density * (1 - math.Exp(-float64(topClusterSize)/20))
}
