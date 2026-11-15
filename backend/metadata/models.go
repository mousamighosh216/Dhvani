// Package metadata holds song models and persistence (Postgres + in-memory).
package metadata

import (
	"context"
	"errors"
	"time"

	"dhwani/recognition/fingerprint"
)

// ErrNotFound is returned when a song does not exist.
var ErrNotFound = errors.New("metadata: not found")

// Artist of a song.
type Artist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Album is optional on a song.
type Album struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	CoverArtURL string `json:"cover_art_url,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"` // YYYY-MM-DD
}

// SongMetadata is the domain model returned by the API.
type SongMetadata struct {
	ID              int               `json:"id"`
	Title           string            `json:"title"`
	DurationSeconds int               `json:"duration_seconds"`
	Artist          Artist            `json:"artist"`
	Album           *Album            `json:"album"`
	ExternalLinks   map[string]string `json:"external_links"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Service is the application-facing persistence contract.
type Service interface {
	// IngestSong stores the song and its fingerprints atomically and returns the new song ID.
	IngestSong(ctx context.Context, song *SongMetadata, fps []fingerprint.Fingerprint) (int, error)
	GetSongByID(ctx context.Context, id int) (*SongMetadata, error)
	ListSongs(ctx context.Context, limit, offset int, sortBy string) ([]SongMetadata, error)
}
