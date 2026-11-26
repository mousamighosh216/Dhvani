// Package metadata stores songs, artists, fingerprints, users, history and collections.
package metadata

import (
	"context"
	"errors"
	"time"

	"dhwani/recognition/fingerprint"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicate      = errors.New("song already exists for this artist")
	ErrEmailTaken     = errors.New("an account with this email already exists")
	ErrNameTaken      = errors.New("you already have a collection with this name")
	ErrNotOwner       = errors.New("song not found or not uploaded by you")
	ErrPendingRemoval = errors.New("a removal request is already pending")
)

// FavouritesName is the collection every user gets on sign-up.
const FavouritesName = "Favourites"

type Song struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Artist        string `json:"artist"`
	Genre         string `json:"genre,omitempty"`
	CoverImageURL string `json:"cover_image_url,omitempty"` // empty → client shows its default
	Duration      int    `json:"duration_seconds"`
}

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	CreateDate   time.Time `json:"create_date"`
	IsAdmin      bool      `json:"is_admin"`
	PasswordHash string    `json:"-"`
}

type UserUpload struct {
	Song                 Song   `json:"song"`
	RemovalRequestStatus string `json:"removal_request_status,omitempty"`
}

type RemovalRequest struct {
	ID          int       `json:"id"`
	Song        Song      `json:"song"`
	RequestedBy string    `json:"requested_by"`
	RequestedAt time.Time `json:"requested_at"`
}

type HistoryItem struct {
	ID         int       `json:"id"`
	Song       Song      `json:"song"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

type Collection struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type SavedSong struct {
	Song       Song      `json:"song"`
	SavedAt    time.Time `json:"saved_at"`
	Confidence float64   `json:"confidence"` // latest recognition confidence, 0 if never identified
}

type GenreCount struct {
	Genre string `json:"genre"`
	Count int    `json:"count"`
}

type UserStats struct {
	Identifications int `json:"identifications"`
	UniqueSongs     int `json:"unique_songs"`
	ThisMonth       int `json:"this_month"`
	Saved           int `json:"saved"`
	Collections     int `json:"collections"`
}

type SongQuery struct {
	Q, Genre string
	Sort     string // "recent" (default), "artist", "popular"
	Limit    int
	Offset   int
}

// Entry is one stored fingerprint occurrence.
type Entry struct{ SongID, T int }

type Store interface {
	// catalog
	AddSong(ctx context.Context, userID int, s Song, fps []fingerprint.Fingerprint) (Song, int, error)
	Lookup(ctx context.Context, hashes []uint32) (map[uint32][]Entry, error)
	GetSong(ctx context.Context, id int) (Song, error)
	SearchSongs(ctx context.Context, q SongQuery) ([]Song, error)
	Genres(ctx context.Context) ([]GenreCount, error)

	// users
	CreateUser(ctx context.Context, name, email, passwordHash string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	UserByID(ctx context.Context, id int) (User, error)
	UpdateUserName(ctx context.Context, id int, name string) (User, error)
	UserStats(ctx context.Context, userID int) (UserStats, error)
	IsAdmin(ctx context.Context, userID int) (bool, error)

	// history
	AddHistory(ctx context.Context, userID, songID int, confidence float64) error
	ListHistory(ctx context.Context, userID, limit int) ([]HistoryItem, error)

	// collections; collectionID 0 means "all my collections"
	Collections(ctx context.Context, userID int) ([]Collection, error)
	CreateCollection(ctx context.Context, userID int, name string) (Collection, error)
	CollectionSongs(ctx context.Context, userID, collectionID int) ([]SavedSong, error)
	AddToCollection(ctx context.Context, userID, collectionID, songID int) error
	RemoveFromCollection(ctx context.Context, userID, collectionID, songID int) error

	// catalog ownership and moderation
	ListUserUploads(ctx context.Context, userID int) ([]UserUpload, error)
	RequestSongRemoval(ctx context.Context, userID, songID int) error
	ListRemovalRequests(ctx context.Context) ([]RemovalRequest, error)
	ReviewRemovalRequest(ctx context.Context, adminID, requestID int, approve bool) error
}
