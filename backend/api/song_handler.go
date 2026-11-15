package api

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dhwani/metadata"
	"dhwani/recognition"
	"dhwani/recognition/audio"
)

const maxIngestBytes = 100 << 20 // 100 MB master track

// SongHandler manages the catalog.
type SongHandler struct {
	Engine      recognition.Engine
	MetaService metadata.Service
}

// HandleIngestSong accepts multipart/form-data:
//
//	audio (file, required), title (required), artist (required),
//	album, cover_art_url, release_date (YYYY-MM-DD), spotify, youtube.
func (h *SongHandler) HandleIngestSong(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxIngestBytes)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart/form-data (max 100 MB)")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	artist := strings.TrimSpace(r.FormValue("artist"))
	if title == "" || artist == "" {
		writeError(w, http.StatusBadRequest, "'title' and 'artist' are required")
		return
	}
	release := strings.TrimSpace(r.FormValue("release_date"))
	if release != "" {
		if _, err := time.Parse("2006-01-02", release); err != nil {
			writeError(w, http.StatusBadRequest, "'release_date' must be YYYY-MM-DD")
			return
		}
	}
	file, _, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'audio' file field")
		return
	}
	defer file.Close()

	track, err := audio.DecodeAudio(file)
	if err != nil {
		log.Printf("ingest: decode: %v", err)
		writeError(w, http.StatusBadRequest, "could not decode audio: "+shortErr(err))
		return
	}
	fps, err := h.Engine.FingerprintSong(0, track)
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio is too short to fingerprint")
		return
	}
	if len(fps) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no fingerprints could be extracted (silent audio?)")
		return
	}

	song := &metadata.SongMetadata{
		Title:           title,
		DurationSeconds: int(math.Round(track.Duration())),
		Artist:          metadata.Artist{Name: artist},
		ExternalLinks:   map[string]string{},
	}
	if album := strings.TrimSpace(r.FormValue("album")); album != "" {
		song.Album = &metadata.Album{
			Title:       album,
			CoverArtURL: strings.TrimSpace(r.FormValue("cover_art_url")),
			ReleaseDate: release,
		}
	}
	for _, k := range []string{"spotify", "youtube"} {
		if v := strings.TrimSpace(r.FormValue(k)); v != "" {
			song.ExternalLinks[k] = v
		}
	}

	id, err := h.MetaService.IngestSong(r.Context(), song, fps)
	if err != nil {
		log.Printf("ingest: store: %v", err)
		writeError(w, http.StatusInternalServerError, "could not save song")
		return
	}
	saved, err := h.MetaService.GetSongByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "song saved but could not be read back")
		return
	}
	log.Printf("ingested song %d %q: %d fingerprints", id, title, len(fps))
	writeSuccess(w, http.StatusCreated, map[string]any{"song": saved, "fingerprints": len(fps)})
}

// HandleGetSongByID returns one song with artist, album and links.
func (h *SongHandler) HandleGetSongByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid song id")
		return
	}
	song, err := h.MetaService.GetSongByID(r.Context(), id)
	if errors.Is(err, metadata.ErrNotFound) {
		writeError(w, http.StatusNotFound, "song not found")
		return
	}
	if err != nil {
		log.Printf("get song: %v", err)
		writeError(w, http.StatusInternalServerError, "could not load song")
		return
	}
	writeSuccess(w, http.StatusOK, song)
}

// HandleListSongs returns the catalog, newest first by default (?limit=&offset=&sort=artist).
func (h *SongHandler) HandleListSongs(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("sort")
	if sortBy != "" && sortBy != "artist" {
		writeError(w, http.StatusBadRequest, "unsupported sort (use 'artist')")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	songs, err := h.MetaService.ListSongs(r.Context(), limit, offset, sortBy)
	if err != nil {
		log.Printf("list songs: %v", err)
		writeError(w, http.StatusInternalServerError, "could not list songs")
		return
	}
	if songs == nil {
		songs = []metadata.SongMetadata{}
	}
	writeSuccess(w, http.StatusOK, map[string]any{"songs": songs})
}
