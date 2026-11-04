// it holds the models for the metadata service
package metadata

import (
	"context"
	"errors"
)

// if song doesnt exists
var ErrSongNotFound = errors.New("metadata: song not found")

type Artist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// song data storage model
type Song struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	ArtistID      int    `json:"artist_id"`
	Genre         string `json:"genre"`
	CoverImageURL string `json:"cover_image_url"`
	Duration      int    `json:"duration"` // in seconds
}

// this table stores data about which user made last update on which song
type Update struct {
	ID       int `json:"id"`
	UserID   int `json:"user_id"`
	ArtistID int `json:"artist_id"`
}

type Service interface {
	IngestSong(ctx context.Context, song *Song, fps []fingerprint.Fingerprint) (int, error)
}
