//go:build !nodeps && integration

package metadata

import (
	"context"
	"os"
	"testing"

	"dhwani/recognition/fingerprint"
)

// Run with a migrated database:
//
//	DHWANI_TEST_DB=postgres://postgres:postgres@localhost:5432/dhwani_test?sslmode=disable \
//	  go test -tags integration ./metadata/
func TestPostgresRoundTrip(t *testing.T) {
	dsn := os.Getenv("DHWANI_TEST_DB")
	if dsn == "" {
		t.Skip("DHWANI_TEST_DB not set")
	}
	ctx := context.Background()
	st, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	song := &SongMetadata{
		Title: "Round Trip", DurationSeconds: 42,
		Artist:        Artist{Name: "Test Artist"},
		Album:         &Album{Title: "Test Album", ReleaseDate: "2011-10-18", CoverArtURL: "https://example.com/c.jpg"},
		ExternalLinks: map[string]string{"spotify": "https://open.spotify.com/track/x"},
	}
	fps := []fingerprint.Fingerprint{
		{Hash: 0xFFFFFFFF, SongOffset: 1}, {Hash: 0xFFFFFFFF, SongOffset: 1}, // duplicate is de-duplicated
		{Hash: 5, SongOffset: 9},
	}
	id, err := st.IngestSong(ctx, song, fps)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.GetSongByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Round Trip" || got.Artist.Name != "Test Artist" || got.Album == nil ||
		got.Album.ReleaseDate != "2011-10-18" || got.ExternalLinks["spotify"] == "" {
		t.Fatalf("unexpected song: %+v", got)
	}
	rows, err := st.QueryHashes(ctx, []uint32{0xFFFFFFFF, 5, 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %+v", rows)
	}
	for _, r := range rows {
		if r.SongID != id {
			t.Errorf("row %+v has wrong song", r)
		}
	}
	if _, err := st.GetSongByID(ctx, -1); err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
