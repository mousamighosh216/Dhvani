package metadata

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dhwani/recognition/fingerprint"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Postgres{pool}, nil
}

func pgCode(err error, code string) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && e.Code == code
}

const (
	songCols = `s.id, s.title, a.name, COALESCE(s.genre,''), COALESCE(s.cover_image_url,''), s.duration`
	songFrom = ` FROM songs s JOIN artists a ON a.id = s.artist_id`
)

func scanSong(row pgx.Row, extra ...any) (Song, error) {
	var s Song
	err := row.Scan(append(extra, &s.ID, &s.Title, &s.Artist, &s.Genre, &s.CoverImageURL, &s.Duration)...)
	return s, err
}

// ---- catalog ----

func (p *Postgres) AddSong(ctx context.Context, userID int, s Song, fps []fingerprint.Fingerprint) (Song, int, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return s, 0, err
	}
	defer tx.Rollback(ctx)

	var artistID int
	err = tx.QueryRow(ctx,
		`INSERT INTO artists (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`, s.Artist).Scan(&artistID)
	if err != nil {
		return s, 0, err
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO songs (title, artist_id, genre, cover_image_url, duration)
		 VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), $5)
		 ON CONFLICT (title, artist_id) DO NOTHING RETURNING id`,
		s.Title, artistID, s.Genre, s.CoverImageURL, s.Duration).Scan(&s.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx,
			`SELECT id FROM songs WHERE title = $1 AND artist_id = $2 FOR UPDATE`,
			s.Title, artistID).Scan(&s.ID)
		if err != nil {
			return s, 0, err
		}
		var hasFingerprints bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM fingerprints WHERE song_id = $1)`, s.ID).Scan(&hasFingerprints); err != nil {
			return s, 0, err
		}
		if hasFingerprints {
			return s, 0, ErrDuplicate
		}
		if _, err := tx.Exec(ctx,
			`UPDATE songs
			 SET genre = COALESCE(NULLIF($1,''), genre),
			     cover_image_url = COALESCE(NULLIF($2,''), cover_image_url),
			     duration = $3
			 WHERE id = $4`,
			s.Genre, s.CoverImageURL, s.Duration, s.ID); err != nil {
			return s, 0, err
		}
	} else if err != nil {
		return s, 0, err
	}
	if userID > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO updates (user_id, artist_id) VALUES ($1, $2) ON CONFLICT (artist_id, user_id) DO NOTHING`,
			userID, artistID); err != nil {
			return s, 0, err
		}
	}

	seen := make(map[[2]int64]struct{}, len(fps))
	rows := make([][]any, 0, len(fps))
	for _, f := range fps {
		k := [2]int64{int64(f.Hash), int64(f.T)}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		rows = append(rows, []any{s.ID, int64(f.Hash), f.T})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"fingerprints"},
		[]string{"song_id", "hash", "offset_frames"}, pgx.CopyFromRows(rows)); err != nil {
		return s, 0, err
	}
	return s, len(rows), tx.Commit(ctx)
}

func (p *Postgres) Lookup(ctx context.Context, hashes []uint32) (map[uint32][]Entry, error) {
	ids := make([]int64, len(hashes))
	for i, h := range hashes {
		ids[i] = int64(h)
	}
	rows, err := p.pool.Query(ctx,
		`SELECT hash, song_id, offset_frames FROM fingerprints WHERE hash = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uint32][]Entry{}
	for rows.Next() {
		var h int64
		var e Entry
		if err := rows.Scan(&h, &e.SongID, &e.T); err != nil {
			return nil, err
		}
		out[uint32(h)] = append(out[uint32(h)], e)
	}
	return out, rows.Err()
}

func (p *Postgres) GetSong(ctx context.Context, id int) (Song, error) {
	s, err := scanSong(p.pool.QueryRow(ctx, `SELECT `+songCols+songFrom+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (p *Postgres) SearchSongs(ctx context.Context, q SongQuery) ([]Song, error) {
	sql := `SELECT ` + songCols + songFrom
	var where []string
	var args []any
	if q.Q != "" {
		args = append(args, "%"+likeEscape(q.Q)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(s.title ILIKE $%d OR a.name ILIKE $%d OR s.genre ILIKE $%d)`, n, n, n))
	}
	if q.Genre != "" {
		args = append(args, q.Genre)
		where = append(where, fmt.Sprintf(`lower(s.genre) = lower($%d)`, len(args)))
	}
	if len(where) > 0 {
		sql += ` WHERE ` + strings.Join(where, ` AND `)
	}
	switch q.Sort {
	case "artist":
		sql += ` ORDER BY a.name, s.title`
	case "popular":
		sql += ` ORDER BY (SELECT count(*) FROM recognition_history h WHERE h.song_id = s.id) DESC, s.id DESC`
	default:
		sql += ` ORDER BY s.id DESC`
	}
	args = append(args, q.Limit, q.Offset)
	sql += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Song{}
	for rows.Next() {
		s, err := scanSong(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (p *Postgres) Genres(ctx context.Context) ([]GenreCount, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT genre, count(*) FROM songs WHERE genre IS NOT NULL GROUP BY genre ORDER BY count(*) DESC, genre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GenreCount{}
	for rows.Next() {
		var g GenreCount
		if err := rows.Scan(&g.Genre, &g.Count); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ---- users ----

const userCols = `id, name, email::text, COALESCE(create_date, NOW() AT TIME ZONE 'UTC'), password_hash`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreateDate, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (p *Postgres) CreateUser(ctx context.Context, name, email, hash string) (User, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING `+userCols, name, email, hash))
	if err != nil {
		if pgCode(err, "23505") {
			return u, ErrEmailTaken
		}
		return u, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO collections (user_id, name) VALUES ($1, $2)`, u.ID, FavouritesName); err != nil {
		return u, err
	}
	return u, tx.Commit(ctx)
}

func (p *Postgres) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(p.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email))
}

func (p *Postgres) UserByID(ctx context.Context, id int) (User, error) {
	return scanUser(p.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (p *Postgres) UpdateUserName(ctx context.Context, id int, name string) (User, error) {
	return scanUser(p.pool.QueryRow(ctx, `UPDATE users SET name = $2 WHERE id = $1 RETURNING `+userCols, id, name))
}

func (p *Postgres) UserStats(ctx context.Context, uid int) (UserStats, error) {
	var s UserStats
	err := p.pool.QueryRow(ctx, `SELECT
	  (SELECT count(*) FROM recognition_history WHERE user_id = $1),
	  (SELECT count(DISTINCT song_id) FROM recognition_history WHERE user_id = $1),
	  (SELECT count(*) FROM recognition_history WHERE user_id = $1 AND created_at >= date_trunc('month', NOW() AT TIME ZONE 'UTC')),
	  (SELECT count(DISTINCT cs.song_id) FROM collection_songs cs JOIN collections c ON c.id = cs.collection_id WHERE c.user_id = $1),
	  (SELECT count(*) FROM collections WHERE user_id = $1)`, uid).
		Scan(&s.Identifications, &s.UniqueSongs, &s.ThisMonth, &s.Saved, &s.Collections)
	return s, err
}

// ---- history ----

func (p *Postgres) AddHistory(ctx context.Context, userID, songID int, conf float64) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO recognition_history (user_id, song_id, confidence) VALUES ($1, $2, $3)`, userID, songID, conf)
	return err
}

func (p *Postgres) ListHistory(ctx context.Context, userID, limit int) ([]HistoryItem, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT h.id, h.confidence, h.created_at, `+songCols+
			` FROM recognition_history h JOIN songs s ON s.id = h.song_id JOIN artists a ON a.id = s.artist_id
			 WHERE h.user_id = $1 ORDER BY h.created_at DESC, h.id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryItem{}
	for rows.Next() {
		var h HistoryItem
		h.Song, err = scanSong(rows, &h.ID, &h.Confidence, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ---- collections ----

func (p *Postgres) Collections(ctx context.Context, userID int) ([]Collection, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT c.id, c.name, count(cs.song_id) FROM collections c
		 LEFT JOIN collection_songs cs ON cs.collection_id = c.id
		 WHERE c.user_id = $1 GROUP BY c.id ORDER BY c.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (p *Postgres) CreateCollection(ctx context.Context, userID int, name string) (Collection, error) {
	c := Collection{Name: name}
	err := p.pool.QueryRow(ctx,
		`INSERT INTO collections (user_id, name) VALUES ($1, $2) RETURNING id`, userID, name).Scan(&c.ID)
	if pgCode(err, "23505") {
		return c, ErrNameTaken
	}
	return c, err
}

func (p *Postgres) CollectionSongs(ctx context.Context, userID, collectionID int) ([]SavedSong, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT max(cs.added_at),
		   COALESCE((SELECT h.confidence FROM recognition_history h
		             WHERE h.user_id = $1 AND h.song_id = s.id ORDER BY h.created_at DESC, h.id DESC LIMIT 1), 0), `+songCols+
			` FROM collection_songs cs
			  JOIN collections c ON c.id = cs.collection_id
			  JOIN songs s ON s.id = cs.song_id
			  JOIN artists a ON a.id = s.artist_id
			 WHERE c.user_id = $1 AND ($2::int = 0 OR c.id = $2::int)
			 GROUP BY s.id, a.id
			 ORDER BY max(cs.added_at) DESC, s.id DESC`, userID, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedSong{}
	for rows.Next() {
		var v SavedSong
		v.Song, err = scanSong(rows, &v.SavedAt, &v.Confidence)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (p *Postgres) AddToCollection(ctx context.Context, userID, collectionID, songID int) error {
	var own bool
	if err := p.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM collections WHERE id = $1 AND user_id = $2)`, collectionID, userID).Scan(&own); err != nil {
		return err
	}
	if !own {
		return ErrNotFound
	}
	_, err := p.pool.Exec(ctx,
		`INSERT INTO collection_songs (collection_id, song_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, collectionID, songID)
	if pgCode(err, "23503") {
		return ErrNotFound
	}
	return err
}

func (p *Postgres) RemoveFromCollection(ctx context.Context, userID, collectionID, songID int) error {
	_, err := p.pool.Exec(ctx,
		`DELETE FROM collection_songs cs USING collections c
		 WHERE cs.collection_id = c.id AND c.user_id = $1 AND ($2::int = 0 OR c.id = $2::int) AND cs.song_id = $3`,
		userID, collectionID, songID)
	return err
}
