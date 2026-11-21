// Package matcher votes on (db_time - query_time) offsets to find the best song.
package matcher

import (
	"sort"

	"dhwani/metadata"
	"dhwani/recognition/fingerprint"
)

const (
	MinVotes      = 8
	MinRatio      = 1.5 // winner must beat the runner-up song by this factor
	MinOtherVotes = 4   // weaker candidates reported as "other likely matches"
	MaxOthers     = 4
)

type Candidate struct{ SongID, Votes int }

type Result struct {
	Matched      bool
	SongID       int
	Votes        int     // aligned hashes in the winning cluster (±1 frame)
	OffsetFrames int     // where the clip starts in the song
	Total        int     // all hash hits, aligned or not
	Confidence   float64 // aligned / all hits, damped for tiny clusters
	Others       []Candidate
}

// Confidence is the share of hash hits that line up, damped when there are few votes.
func Confidence(votes, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(votes) / float64(total) * min(1.0, float64(votes)/20)
}

func Match(query []fingerprint.Fingerprint, hits map[uint32][]metadata.Entry) Result {
	type key struct{ song, off int }
	counts := map[key]int{}
	total := 0
	for _, q := range query {
		for _, e := range hits[q.Hash] {
			counts[key{e.SongID, e.T - q.T}]++
			total++
		}
	}
	type best struct{ votes, off int }
	perSong := map[int]best{}
	for k := range counts {
		v := counts[k] + counts[key{k.song, k.off - 1}] + counts[key{k.song, k.off + 1}]
		if b := perSong[k.song]; v > b.votes {
			perSong[k.song] = best{v, k.off}
		}
	}

	ranked := make([]Candidate, 0, len(perSong))
	for id, b := range perSong {
		ranked = append(ranked, Candidate{id, b.votes})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Votes != ranked[j].Votes {
			return ranked[i].Votes > ranked[j].Votes
		}
		return ranked[i].SongID < ranked[j].SongID
	})

	res := Result{Total: total}
	if len(ranked) == 0 {
		return res
	}
	top, second := ranked[0].Votes, 0
	res.SongID, res.Votes, res.OffsetFrames = ranked[0].SongID, top, perSong[ranked[0].SongID].off
	res.Confidence = Confidence(top, total)
	if len(ranked) > 1 {
		second = ranked[1].Votes
	}
	for _, c := range ranked[1:] {
		if c.Votes < MinOtherVotes || len(res.Others) == MaxOthers {
			break
		}
		res.Others = append(res.Others, c)
	}
	res.Matched = top >= MinVotes && float64(top) >= MinRatio*float64(second)
	return res
}
