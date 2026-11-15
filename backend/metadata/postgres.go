//go:build !nodeps

package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"dhwani/recognition/fingerprint"
	"dhwani/recognition/matcher"
)

// PostgresStore implements Service and matcher.FingerprintStore on PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore connects to PostgreSQL using a connection URL.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// Close releases the connection pool.
func (s *PostgresStore) Close() { s.pool.Close() }

// IngestSong inserts artist, album, song and all fingerprints in one transaction.
func (s *PostgresStore) IngestSong(ctx context.Context, song *SongMetadata, fps []fingerprint.Fingerprint) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit

	id, err := insertSong(ctx, tx, song)
	if err != nil {
		return 0, err
	}
	if err := copyFingerprints(ctx, tx, id, fps); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

// SaveSong inserts artist, album and song rows and returns the song ID.
func (s *PostgresStore) SaveSong(ctx context.Context, song *SongMetadata) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	id, err := insertSong(ctx, tx, song)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

// SaveFingerprints bulk-inserts fingerprints with COPY. Each fingerprint's
// SongID must reference an existing song.
func (s *PostgresStore) SaveFingerprints(ctx context.Context, fps []fingerprint.Fingerprint) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	bySong := map[int][]fingerprint.Fingerprint{}
	for _, fp := range fps {
		bySong[fp.SongID] = append(bySong[fp.SongID], fp)
	}
	for id, group := range bySong {
		if err := copyFingerprints(ctx, tx, id, group); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func insertSong(ctx context.Context, tx pgx.Tx, song *SongMetadata) (int, error) {
	var artistID int
	err := tx.QueryRow(ctx,
		`INSERT INTO artists (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`, song.Artist.Name).Scan(&artistID)
	if err != nil {
		return 0, fmt.Errorf("upsert artist: %w", err)
	}

	var albumID *int
	if song.Album != nil && song.Album.Title != "" {
		var release, cover *string
		if song.Album.ReleaseDate != "" {
			release = &song.Album.ReleaseDate
		}
		if song.Album.CoverArtURL != "" {
			cover = &song.Album.CoverArtURL
		}
		var id int
		err := tx.QueryRow(ctx,
			`INSERT INTO albums (artist_id, title, release_date, cover_art_url)
			 VALUES ($1, $2, $3::date, $4)
			 ON CONFLICT (artist_id, title) DO UPDATE SET
			     release_date  = COALESCE(EXCLUDED.release_date, albums.release_date),
			     cover_art_url = COALESCE(EXCLUDED.cover_art_url, albums.cover_art_url)
			 RETURNING id`, artistID, song.Album.Title, release, cover).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("upsert album: %w", err)
		}
		albumID = &id
	}

	links := song.ExternalLinks
	if links == nil {
		links = map[string]string{}
	}
	linksJSON, err := json.Marshal(links)
	if err != nil {
		return 0, err
	}
	var songID int
	err = tx.QueryRow(ctx,
		`INSERT INTO songs (artist_id, album_id, title, duration_seconds, external_links)
		 VALUES ($1, $2, $3, $4, $5::jsonb)
		 RETURNING id`, artistID, albumID, song.Title, song.DurationSeconds, string(linksJSON)).Scan(&songID)
	if err != nil {
		return 0, fmt.Errorf("insert song: %w", err)
	}
	return songID, nil
}

func copyFingerprints(ctx context.Context, tx pgx.Tx, songID int, fps []fingerprint.Fingerprint) error {
	seen := make(map[[2]uint32]struct{}, len(fps))
	rows := make([][]any, 0, len(fps))
	for _, fp := range fps {
		k := [2]uint32{fp.Hash, fp.SongOffset}
		if _, dup := seen[k]; dup { // primary key is (hash, song_id, song_offset)
			continue
		}
		seen[k] = struct{}{}
		rows = append(rows, []any{int64(fp.Hash), int32(songID), int32(fp.SongOffset)})
	}
	_, err := tx.CopyFrom(ctx,
		pgx.Identifier{"fingerprints"},
		[]string{"hash", "song_id", "song_offset"},
		pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy fingerprints: %w", err)
	}
	return nil
}

// QueryHashes returns every stored fingerprint whose hash is in hashes,
// using the B-Tree primary key.
func (s *PostgresStore) QueryHashes(ctx context.Context, hashes []uint32) ([]matcher.DBMatch, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	arg := make([]int64, len(hashes))
	for i, h := range hashes {
		arg[i] = int64(h)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT hash, song_id, song_offset FROM fingerprints WHERE hash = ANY($1)`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []matcher.DBMatch
	for rows.Next() {
		var h int64
		var songID, offset int32
		if err := rows.Scan(&h, &songID, &offset); err != nil {
			return nil, err
		}
		out = append(out, matcher.DBMatch{Hash: uint32(h), SongID: int(songID), SongOffset: uint32(offset)})
	}
	return out, rows.Err()
}

const songSelect = `
	SELECT s.id, s.title, s.duration_seconds, s.external_links, s.created_at,
	       ar.id, ar.name,
	       al.id, al.title, al.cover_art_url, to_char(al.release_date, 'YYYY-MM-DD')
	FROM songs s
	JOIN artists ar ON ar.id = s.artist_id
	LEFT JOIN albums al ON al.id = s.album_id`

func scanSong(row pgx.Row) (*SongMetadata, error) {
	var (
		song                     SongMetadata
		linksJSON                []byte
		albumID                  *int32
		albumTitle, cover, relDt *string
		artistID                 int32
	)
	err := row.Scan(&song.ID, &song.Title, &song.DurationSeconds, &linksJSON, &song.CreatedAt,
		&artistID, &song.Artist.Name, &albumID, &albumTitle, &cover, &relDt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	song.Artist.ID = int(artistID)
	if err := json.Unmarshal(linksJSON, &song.ExternalLinks); err != nil {
		return nil, err
	}
	if albumID != nil {
		song.Album = &Album{ID: int(*albumID)}
		if albumTitle != nil {
			song.Album.Title = *albumTitle
		}
		if cover != nil {
			song.Album.CoverArtURL = *cover
		}
		if relDt != nil {
			song.Album.ReleaseDate = *relDt
		}
	}
	return &song, nil
}

// GetSongByID returns a song with artist, album and links.
func (s *PostgresStore) GetSongByID(ctx context.Context, id int) (*SongMetadata, error) {
	return scanSong(s.pool.QueryRow(ctx, songSelect+` WHERE s.id = $1`, id))
}

// ListSongs returns songs newest first.
func (s *PostgresStore) ListSongs(ctx context.Context, limit, offset int, sortBy string) ([]SongMetadata, error) {
	if limit <= 0 {
		limit = 50
	}
	orderBy := ` ORDER BY s.id DESC`
	if sortBy == "artist" {
		orderBy = ` ORDER BY lower(ar.name), lower(s.title), s.id`
	}
	rows, err := s.pool.Query(ctx, songSelect+orderBy+` LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SongMetadata
	for rows.Next() {
		song, err := scanSong(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *song)
	}
	return out, rows.Err()
}
