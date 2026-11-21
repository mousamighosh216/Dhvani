// Package api is the REST layer.
package api

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"dhwani/auth"
	"dhwani/metadata"
	"dhwani/recognition"
	"dhwani/recognition/audio"
	"dhwani/recognition/matcher"
)

const (
	maxUpload          = 25 << 20
	minOtherConfidence = 0.05
)

type Server struct {
	Store  metadata.Store
	Secret []byte // HS256 key for JWTs
}

type authedHandler func(w http.ResponseWriter, r *http.Request, userID int)

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { ok(w, 200, "ok") })

	mux.HandleFunc("POST /api/v1/auth/register", s.register)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/me", s.authed(s.me))
	mux.HandleFunc("PATCH /api/v1/me", s.authed(s.updateMe))

	mux.HandleFunc("POST /api/v1/recognize", s.authed(s.recognize))
	mux.HandleFunc("GET /api/v1/history", s.authed(s.history))

	mux.HandleFunc("POST /api/v1/songs", s.authed(s.addSong))
	mux.HandleFunc("GET /api/v1/songs", s.authed(s.listSongs))
	mux.HandleFunc("GET /api/v1/songs/{id}", s.authed(s.getSong))
	mux.HandleFunc("GET /api/v1/genres", s.authed(s.genres))

	mux.HandleFunc("GET /api/v1/collections", s.authed(s.collections))
	mux.HandleFunc("POST /api/v1/collections", s.authed(s.createCollection))
	mux.HandleFunc("GET /api/v1/collections/{id}/songs", s.authed(s.collectionSongs)) // id "all" = every collection
	mux.HandleFunc("PUT /api/v1/collections/{id}/songs/{songId}", s.authed(s.saveSong))
	mux.HandleFunc("DELETE /api/v1/collections/{id}/songs/{songId}", s.authed(s.unsaveSong))
	return mux
}

func ok(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"status": "error", "message": msg})
}

func serverError(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	fail(w, 500, what)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, 400, "invalid JSON body")
		return false
	}
	return true
}

func intParam(r *http.Request, name string) (int, bool) {
	n, err := strconv.Atoi(r.PathValue(name))
	return n, err == nil && n > 0
}

func (s *Server) authed(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		uid, err := auth.Verify(s.Secret, tok)
		if !found || err != nil {
			fail(w, 401, "please sign in")
			return
		}
		h(w, r, uid)
	}
}

// readAudio parses the multipart "audio" field and decodes it.
func readAudio(w http.ResponseWriter, r *http.Request) ([]float32, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		fail(w, 400, "invalid upload (max 25 MB)")
		return nil, false
	}
	f, _, err := r.FormFile("audio")
	if err != nil {
		fail(w, 400, "missing 'audio' file")
		return nil, false
	}
	defer f.Close()
	x, err := audio.Decode(f)
	if err != nil {
		fail(w, 400, err.Error())
		return nil, false
	}
	if len(x) < audio.SampleRate {
		fail(w, 400, "audio must be at least 1 second")
		return nil, false
	}
	return x, true
}

type matchView struct {
	Song       metadata.Song `json:"song"`
	Confidence float64       `json:"confidence"`
}

func (s *Server) recognize(w http.ResponseWriter, r *http.Request, uid int) {
	x, good := readAudio(w, r)
	if !good {
		return
	}
	res, landmarks, err := recognition.Recognize(r.Context(), s.Store, x)
	if err != nil {
		serverError(w, "recognition failed", err)
		return
	}
	out := map[string]any{
		"matched": res.Matched, "confidence": res.Confidence, "votes": res.Votes,
		"landmarks": landmarks, "clip_seconds": float64(len(x)) / audio.SampleRate,
	}
	if res.Matched {
		song, err := s.Store.GetSong(r.Context(), res.SongID)
		if err != nil {
			serverError(w, "song lookup failed", err)
			return
		}
		out["song"] = song
		out["match_offset_seconds"] = float64(res.OffsetFrames*recognition.Hop) / audio.SampleRate
		if err := s.Store.AddHistory(r.Context(), uid, song.ID, res.Confidence); err != nil {
			log.Printf("add history: %v", err) // the match itself is still valid
		}
		others := []matchView{}
		for _, c := range res.Others {
			conf := matcher.Confidence(c.Votes, res.Total)
			if conf < minOtherConfidence { // chance alignments are noise, not "likely matches"
				continue
			}
			if o, err := s.Store.GetSong(r.Context(), c.SongID); err == nil {
				others = append(others, matchView{o, conf})
			}
		}
		out["others"] = others
	}
	ok(w, 200, out)
}

func (s *Server) addSong(w http.ResponseWriter, r *http.Request, uid int) {
	x, good := readAudio(w, r)
	if !good {
		return
	}
	song := metadata.Song{
		Title:         strings.TrimSpace(r.FormValue("title")),
		Artist:        strings.TrimSpace(r.FormValue("artist")),
		Genre:         strings.TrimSpace(r.FormValue("genre")),
		CoverImageURL: strings.TrimSpace(r.FormValue("cover_image_url")),
		Duration:      int(math.Ceil(float64(len(x)) / audio.SampleRate)),
	}
	if song.Title == "" || song.Artist == "" {
		fail(w, 400, "title and artist are required")
		return
	}
	saved, n, err := s.Store.AddSong(r.Context(), uid, song, recognition.Fingerprint(x))
	switch {
	case errors.Is(err, metadata.ErrDuplicate):
		fail(w, 409, err.Error())
	case err != nil:
		serverError(w, "could not save song", err)
	default:
		ok(w, 201, map[string]any{"song": saved, "fingerprints": n})
	}
}

func (s *Server) listSongs(w http.ResponseWriter, r *http.Request, _ int) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	songs, err := s.Store.SearchSongs(r.Context(), metadata.SongQuery{
		Q: strings.TrimSpace(q.Get("q")), Genre: strings.TrimSpace(q.Get("genre")),
		Sort: q.Get("sort"), Limit: limit, Offset: max(offset, 0),
	})
	if err != nil {
		serverError(w, "could not list songs", err)
		return
	}
	ok(w, 200, songs)
}

func (s *Server) getSong(w http.ResponseWriter, r *http.Request, _ int) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	song, err := s.Store.GetSong(r.Context(), id)
	if errors.Is(err, metadata.ErrNotFound) {
		fail(w, 404, "song not found")
		return
	}
	if err != nil {
		serverError(w, "could not load song", err)
		return
	}
	ok(w, 200, song)
}

func (s *Server) genres(w http.ResponseWriter, r *http.Request, _ int) {
	g, err := s.Store.Genres(r.Context())
	if err != nil {
		serverError(w, "could not load genres", err)
		return
	}
	ok(w, 200, g)
}
