package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dhwani/auth"
	"dhwani/metadata"
)

func TestPublicCatalogAndRecognition(t *testing.T) {
	store := metadata.NewMemory()
	if _, _, err := store.AddSong(context.Background(), 1,
		metadata.Song{Title: "Track", Artist: "Artist", Genre: "Jazz", Duration: 30}, nil); err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: store, Secret: []byte("a-test-secret-long-enough")}
	for _, path := range []string{
		"/api/v1/songs",
		"/api/v1/songs/1",
		"/api/v1/genres",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recognize", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unsigned recognition status = %d, want upload validation status %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUploadAndPersonalEndpointsRequireAuthentication(t *testing.T) {
	server := &Server{Store: metadata.NewMemory(), Secret: []byte("a-test-secret-long-enough")}
	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/api/v1/songs"},
		{http.MethodGet, "/api/v1/me/uploads"},
		{http.MethodPost, "/api/v1/songs/1/removal-requests"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", tc.method, tc.path, rec.Code, http.StatusUnauthorized)
		}
	}
}

func TestRemovalQueueRequiresAdministrator(t *testing.T) {
	store := metadata.NewMemory()
	user, err := store.CreateUser(context.Background(), "Reviewer", "reviewer@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("a-test-secret-long-enough")
	server := &Server{Store: store, Secret: secret}
	token := auth.Sign(secret, user.ID, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/removal-requests", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	store.SetAdmin(user.ID, true)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/removal-requests", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want %d", rec.Code, http.StatusOK)
	}
}
