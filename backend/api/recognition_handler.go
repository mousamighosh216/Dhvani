package api

import (
	"errors"
	"log"
	"math"
	"net/http"

	"dhwani/metadata"
	"dhwani/recognition"
	"dhwani/recognition/audio"
)

const (
	maxRecognizeBytes = 25 << 20 // 25 MB clip
	minClipSeconds    = 1.0
)

// RecognitionHandler identifies uploaded audio clips.
type RecognitionHandler struct {
	Engine      recognition.Engine
	MetaService metadata.Service
}

type recognizeData struct {
	Matched            bool                   `json:"matched"`
	Confidence         float64                `json:"confidence,omitempty"`
	Votes              int                    `json:"votes,omitempty"`
	MatchOffsetSeconds float64                `json:"match_offset_seconds,omitempty"`
	Song               *metadata.SongMetadata `json:"song,omitempty"`
}

// HandleRecognize accepts multipart/form-data with an "audio" file.
func (h *RecognitionHandler) HandleRecognize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRecognizeBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart/form-data with an 'audio' file (max 25 MB)")
		return
	}
	file, _, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'audio' file field")
		return
	}
	defer file.Close()

	clip, err := audio.DecodeAudio(file)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, audio.ErrFFmpegMissing) {
			status = http.StatusServiceUnavailable
		}
		log.Printf("recognize: decode: %v", err)
		writeError(w, status, "could not decode audio: "+shortErr(err))
		return
	}
	if clip.Duration() < minClipSeconds {
		writeError(w, http.StatusBadRequest, "audio clip is too short; record at least a few seconds")
		return
	}

	res, err := h.Engine.Recognize(r.Context(), clip)
	if err != nil {
		if errors.Is(err, recognition.ErrTooShort) {
			writeError(w, http.StatusBadRequest, "audio clip is too short")
			return
		}
		log.Printf("recognize: %v", err)
		writeError(w, http.StatusInternalServerError, "recognition failed")
		return
	}
	if res == nil {
		writeSuccess(w, http.StatusOK, recognizeData{Matched: false})
		return
	}
	song, err := h.MetaService.GetSongByID(r.Context(), res.SongID)
	if err != nil {
		log.Printf("recognize: metadata for song %d: %v", res.SongID, err)
		writeError(w, http.StatusInternalServerError, "matched song metadata is unavailable")
		return
	}
	writeSuccess(w, http.StatusOK, recognizeData{
		Matched:            true,
		Confidence:         math.Round(res.Confidence*100) / 100,
		Votes:              res.Votes,
		MatchOffsetSeconds: math.Round(res.OffsetSec*10) / 10,
		Song:               song,
	})
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
