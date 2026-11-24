package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dhwani/metadata"
)

func TestPublicCatalogAndRecognition(t *testing.T) {
	h := (&Server{Store: metadata.NewMemory(), Secret: []byte("test-secret-key-1234567890")}).Routes()

	for _, path := range []string{"/api/v1/songs", "/api/v1/genres"} {
		t.Run(path, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
			if res.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want %d", path, res.Code, http.StatusOK)
			}
		})
	}

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/recognize", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("anonymous POST /api/v1/recognize status = %d, want %d (public route should validate the missing upload)", res.Code, http.StatusBadRequest)
	}
}

func TestPersonalRoutesAndUploadsRequireAuthentication(t *testing.T) {
	h := (&Server{Store: metadata.NewMemory(), Secret: []byte("test-secret-key-1234567890")}).Routes()

	for _, test := range []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/me"},
		{http.MethodGet, "/api/v1/history"},
		{http.MethodPost, "/api/v1/songs"},
		{http.MethodGet, "/api/v1/collections"},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(test.method, test.path, nil))
			if res.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", test.method, test.path, res.Code, http.StatusUnauthorized)
			}
		})
	}
}
