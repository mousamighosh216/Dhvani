package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// NewRouter wires all routes. If staticDir exists it is served at "/"
// (the built React app), so one process can serve UI and API.
func NewRouter(engine recognition.Engine, meta metadata.Service, staticDir string) http.Handler {
	rec := &RecognitionHandler{Engine: engine, MetaService: meta}
	
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/recognize", rec.HandleRecognize)

	if staticDir != "" {
		if st, err := os.Stat(staticDir); err == nil && st.IsDir() {
			mux.Handle("/", http.FileServer(http.Dir(staticDir)))
		}
	}
	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"status": "error", "message": msg})
}

func writeSuccess(w http.ResponseWriter, code int, data any) {
	writeJSON(w, code, map[string]any{"status": "success", "data": data})
}
