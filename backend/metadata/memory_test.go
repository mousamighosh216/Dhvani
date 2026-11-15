package metadata

import (
	"context"
	"testing"

	"dhwani/recognition/fingerprint"
)

func TestMemoryStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()
	id, err := m.IngestSong(ctx, &SongMetadata{Title: "T", Artist: Artist{Name: "A"}},
		[]fingerprint.Fingerprint{{Hash: 1, SongOffset: 5}, {Hash: 1, SongOffset: 5}, {Hash: 2, SongOffset: 6}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.GetSongByID(ctx, id)
	if err != nil || got.Title != "T" || got.ExternalLinks == nil {
		t.Fatalf("%+v %v", got, err)
	}
	rows, _ := m.QueryHashes(ctx, []uint32{1, 2, 3})
	if len(rows) != 2 { // duplicate (hash, offset) collapsed
		t.Fatalf("rows %+v", rows)
	}
	if _, err := m.GetSongByID(ctx, 99); err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	m.IngestSong(ctx, &SongMetadata{Title: "T2"}, nil)
	list, _ := m.ListSongs(ctx, 1, 0, "")
	if len(list) != 1 || list[0].Title != "T2" {
		t.Errorf("list %+v", list)
	}
	if list, _ := m.ListSongs(ctx, 10, 5, ""); len(list) != 0 {
		t.Errorf("offset past end: %+v", list)
	}
}
