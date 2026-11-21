package api

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dhwani/auth"
	"dhwani/metadata"
)

// same pattern as check_email_format in 001_initial_schema.sql
var emailRe = regexp.MustCompile(`(?i)^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$`)

const tokenTTL = 30 * 24 * time.Hour

func (s *Server) session(w http.ResponseWriter, code int, u metadata.User) {
	ok(w, code, map[string]any{"token": auth.Sign(s.Secret, u.ID, tokenTTL), "user": u})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Name, in.Email = strings.TrimSpace(in.Name), strings.TrimSpace(in.Email)
	switch {
	case in.Name == "" || len(in.Name) > 80:
		fail(w, 400, "enter your name (up to 80 characters)")
		return
	case !emailRe.MatchString(in.Email):
		fail(w, 400, "enter a valid email address")
		return
	case len(in.Password) < 8 || len(in.Password) > 128:
		fail(w, 400, "password must be 8–128 characters")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		serverError(w, "could not create account", err)
		return
	}
	u, err := s.Store.CreateUser(r.Context(), in.Name, in.Email, hash)
	switch {
	case errors.Is(err, metadata.ErrEmailTaken):
		fail(w, 409, err.Error())
	case err != nil:
		serverError(w, "could not create account", err)
	default:
		s.session(w, 201, u)
	}
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	u, err := s.Store.UserByEmail(r.Context(), strings.TrimSpace(in.Email))
	if err != nil && !errors.Is(err, metadata.ErrNotFound) {
		serverError(w, "could not sign in", err)
		return
	}
	if err != nil || !auth.CheckPassword(in.Password, u.PasswordHash) {
		fail(w, 401, "invalid email or password")
		return
	}
	s.session(w, 200, u)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, uid int) {
	u, err := s.Store.UserByID(r.Context(), uid)
	if errors.Is(err, metadata.ErrNotFound) {
		fail(w, 401, "please sign in")
		return
	}
	if err != nil {
		serverError(w, "could not load profile", err)
		return
	}
	st, err := s.Store.UserStats(r.Context(), uid)
	if err != nil {
		serverError(w, "could not load profile", err)
		return
	}
	ok(w, 200, map[string]any{"user": u, "stats": st})
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request, uid int) {
	var in struct{ Name string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 80 {
		fail(w, 400, "enter your name (up to 80 characters)")
		return
	}
	u, err := s.Store.UpdateUserName(r.Context(), uid, in.Name)
	if err != nil {
		serverError(w, "could not update profile", err)
		return
	}
	ok(w, 200, u)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request, uid int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	h, err := s.Store.ListHistory(r.Context(), uid, limit)
	if err != nil {
		serverError(w, "could not load history", err)
		return
	}
	ok(w, 200, h)
}

func (s *Server) collections(w http.ResponseWriter, r *http.Request, uid int) {
	c, err := s.Store.Collections(r.Context(), uid)
	if err != nil {
		serverError(w, "could not load collections", err)
		return
	}
	ok(w, 200, c)
}

func (s *Server) createCollection(w http.ResponseWriter, r *http.Request, uid int) {
	var in struct{ Name string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 60 {
		fail(w, 400, "collection name must be 1–60 characters")
		return
	}
	c, err := s.Store.CreateCollection(r.Context(), uid, in.Name)
	switch {
	case errors.Is(err, metadata.ErrNameTaken):
		fail(w, 409, err.Error())
	case err != nil:
		serverError(w, "could not create collection", err)
	default:
		ok(w, 201, c)
	}
}

// collectionID reads {id}: a number, or "all" for every collection (stored as 0).
func collectionID(r *http.Request) (int, bool) {
	if r.PathValue("id") == "all" {
		return 0, true
	}
	return intParam(r, "id")
}

func (s *Server) collectionSongs(w http.ResponseWriter, r *http.Request, uid int) {
	id, good := collectionID(r)
	if !good {
		fail(w, 400, "invalid collection")
		return
	}
	v, err := s.Store.CollectionSongs(r.Context(), uid, id)
	if err != nil {
		serverError(w, "could not load songs", err)
		return
	}
	ok(w, 200, v)
}

func (s *Server) saveSong(w http.ResponseWriter, r *http.Request, uid int) {
	cid, good1 := intParam(r, "id")
	sid, good2 := intParam(r, "songId")
	if !good1 || !good2 {
		fail(w, 400, "invalid ids")
		return
	}
	err := s.Store.AddToCollection(r.Context(), uid, cid, sid)
	switch {
	case errors.Is(err, metadata.ErrNotFound):
		fail(w, 404, "collection or song not found")
	case err != nil:
		serverError(w, "could not save song", err)
	default:
		ok(w, 200, "saved")
	}
}

func (s *Server) unsaveSong(w http.ResponseWriter, r *http.Request, uid int) {
	cid, good1 := collectionID(r)
	sid, good2 := intParam(r, "songId")
	if !good1 || !good2 {
		fail(w, 400, "invalid ids")
		return
	}
	if err := s.Store.RemoveFromCollection(r.Context(), uid, cid, sid); err != nil {
		serverError(w, "could not remove song", err)
		return
	}
	ok(w, 200, "removed")
}
