package matcher

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"testing"

	"dhwani/recognition/fingerprint"
)

type fakeStore struct {
	rows []DBMatch
	err  error
}

func (f *fakeStore) QueryHashes(_ context.Context, hashes []uint32) ([]DBMatch, error) {
	if f.err != nil {
		return nil, f.err
	}
	want := map[uint32]bool{}
	for _, h := range hashes {
		want[h] = true
	}
	var out []DBMatch
	for _, r := range f.rows {
		if want[r.Hash] {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeStore) SaveFingerprints(context.Context, []fingerprint.Fingerprint) error { return nil }

// makeScenario: song 1 contains the query shifted by `shift` frames; song 2
// has the same hashes at unrelated offsets.
func makeScenario(n, shift int) ([]fingerprint.Fingerprint, *fakeStore) {
	rng := rand.New(rand.NewSource(1))
	var q []fingerprint.Fingerprint
	st := &fakeStore{}
	for i := 0; i < n; i++ {
		h := rng.Uint32()
		qo := uint32(i * 3)
		q = append(q, fingerprint.Fingerprint{Hash: h, SongOffset: qo})
		st.rows = append(st.rows,
			DBMatch{Hash: h, SongID: 1, SongOffset: qo + uint32(shift)},
			DBMatch{Hash: h, SongID: 2, SongOffset: uint32(rng.Intn(100000))})
	}
	return q, st
}

func TestVotingPicksAlignedSong(t *testing.T) {
	q, st := makeScenario(60, 5000)
	res, err := MatchQuery(context.Background(), q, st)
	if err != nil || res == nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if res.SongID != 1 || res.OffsetFrames != 5000 {
		t.Errorf("got %+v", res)
	}
	wantSec := 5000 * 512.0 / 44100
	if math.Abs(res.OffsetSec-wantSec) > 1e-9 {
		t.Errorf("offset %v want %v", res.OffsetSec, wantSec)
	}
	if res.Votes != 60 || res.Confidence <= 0 || res.Confidence > 1 {
		t.Errorf("votes/confidence: %+v", res)
	}
}

func TestNoMatchCases(t *testing.T) {
	ctx := context.Background()
	if r, err := MatchQuery(ctx, nil, &fakeStore{}); r != nil || err != nil {
		t.Error("empty query should give nil, nil")
	}
	q, _ := makeScenario(60, 0)
	if r, _ := MatchQuery(ctx, q, &fakeStore{}); r != nil {
		t.Error("no rows should give nil")
	}
	// Only random, unaligned collisions: below MinVotes and unaligned.
	rng := rand.New(rand.NewSource(2))
	st := &fakeStore{}
	for _, f := range q {
		st.rows = append(st.rows, DBMatch{Hash: f.Hash, SongID: 9, SongOffset: uint32(rng.Intn(1 << 20))})
	}
	if r, _ := MatchQuery(ctx, q, st); r != nil {
		t.Errorf("random collisions matched: %+v", r)
	}
}

func TestAmbiguousSongsRejected(t *testing.T) {
	q, st := makeScenario(40, 100)
	for _, f := range q { // song 3 aligned equally well -> no clear winner
		st.rows = append(st.rows, DBMatch{Hash: f.Hash, SongID: 3, SongOffset: f.SongOffset + 777})
	}
	if r, _ := MatchQuery(context.Background(), q, st); r != nil {
		t.Errorf("ambiguous result accepted: %+v", r)
	}
}

func TestStoreErrorPropagates(t *testing.T) {
	q, _ := makeScenario(5, 0)
	boom := errors.New("db down")
	if _, err := MatchQuery(context.Background(), q, &fakeStore{err: boom}); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

func TestOneFrameJitterTolerated(t *testing.T) {
	q, st := makeScenario(40, 300)
	for i := range st.rows { // jitter song 1 offsets by +-1 frame
		if st.rows[i].SongID == 1 && i%4 == 0 {
			st.rows[i].SongOffset++
		}
	}
	res, _ := MatchQuery(context.Background(), q, st)
	if res == nil || res.SongID != 1 || res.Votes != 40 {
		t.Fatalf("got %+v", res)
	}
}

func TestCalculateConfidence(t *testing.T) {
	if CalculateConfidence(0, 10) != 0 || CalculateConfidence(5, 0) != 0 {
		t.Error("degenerate inputs should be 0")
	}
	lo, hi := CalculateConfidence(10, 100), CalculateConfidence(10, 20)
	if !(hi > lo) {
		t.Errorf("denser cluster should score higher: %v %v", lo, hi)
	}
	if c := CalculateConfidence(500, 500); c < 0.99 || c > 1 {
		t.Errorf("perfect cluster = %v", c)
	}
	if CalculateConfidence(2, 2) >= CalculateConfidence(100, 100) {
		t.Error("tiny clusters must be damped")
	}
}
