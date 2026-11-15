package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"dhwani/internal/synth"
	"dhwani/metadata"
	"dhwani/recognition"
)

const sr = 44100

type env struct {
	srv   *httptest.Server
	songs map[string][]float64
}

func newEnv(t *testing.T) *env {
	t.Helper()
	store := metadata.NewMemoryStore()
	srv := httptest.NewServer(NewRouter(recognition.NewEngine(store), store, ""))
	t.Cleanup(srv.Close)
	return &env{srv: srv, songs: map[string][]float64{}}
}

func wavBytes(t *testing.T, x []float64) []byte {
	var b bytes.Buffer
	if err := synth.WriteWAV(&b, x, sr); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func multipartBody(t *testing.T, fields map[string]string, fileField, fileName string, file []byte) (*bytes.Buffer, string) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	if fileField != "" {
		fw, _ := mw.CreateFormFile(fileField, fileName)
		fw.Write(file)
	}
	mw.Close()
	return &body, mw.FormDataContentType()
}

type resp struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e *env) post(t *testing.T, path string, fields map[string]string, field, name string, file []byte) (int, resp) {
	body, ct := multipartBody(t, fields, field, name, file)
	res, err := http.Post(e.srv.URL+path, ct, body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var r resp
	json.NewDecoder(res.Body).Decode(&r)
	return res.StatusCode, r
}

func (e *env) get(t *testing.T, path string) (int, resp) {
	res, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var r resp
	json.NewDecoder(res.Body).Decode(&r)
	return res.StatusCode, r
}

func (e *env) ingest(t *testing.T, seed int64, title string) int {
	x := synth.Song(seed, 30, sr)
	e.songs[title] = x
	code, r := e.post(t, "/api/v1/songs", map[string]string{
		"title": title, "artist": "Synth Band", "album": "Test Album",
		"release_date": "2020-05-01", "spotify": "https://open.spotify.com/track/abc",
	}, "audio", "song.wav", wavBytes(t, x))
	if code != http.StatusCreated {
		t.Fatalf("ingest %s: %d %s %s", title, code, r.Status, r.Message)
	}
	var d struct {
		Song         metadata.SongMetadata `json:"song"`
		Fingerprints int                   `json:"fingerprints"`
	}
	json.Unmarshal(r.Data, &d)
	if d.Fingerprints == 0 || d.Song.ID == 0 {
		t.Fatalf("bad ingest response: %s", r.Data)
	}
	return d.Song.ID
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	res, err := http.Get(e.srv.URL + "/healthz")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("%v %v", res, err)
	}
}

func TestIngestThenGetAndList(t *testing.T) {
	e := newEnv(t)
	id := e.ingest(t, 1, "Alpha")
	e.ingest(t, 2, "Beta")

	code, r := e.get(t, "/api/v1/songs/"+itoa(id))
	if code != 200 {
		t.Fatalf("get: %d", code)
	}
	var s metadata.SongMetadata
	json.Unmarshal(r.Data, &s)
	if s.Title != "Alpha" || s.Artist.Name != "Synth Band" || s.Album == nil || s.Album.Title != "Test Album" ||
		s.Album.ReleaseDate != "2020-05-01" || s.ExternalLinks["spotify"] == "" || s.DurationSeconds != 30 {
		t.Errorf("unexpected song %+v", s)
	}

	code, r = e.get(t, "/api/v1/songs")
	var list struct{ Songs []metadata.SongMetadata }
	json.Unmarshal(r.Data, &list)
	if code != 200 || len(list.Songs) != 2 || list.Songs[0].Title != "Beta" {
		t.Errorf("list: %d %+v", code, list)
	}
}

func TestListSongsSortByArtist(t *testing.T) {
	store := metadata.NewMemoryStore()
	for _, song := range []metadata.SongMetadata{
		{Title: "Zebra", Artist: metadata.Artist{Name: "Beta"}},
		{Title: "Solo", Artist: metadata.Artist{Name: "Zeta"}},
		{Title: "Alpha", Artist: metadata.Artist{Name: "Beta"}},
		{Title: "Middle", Artist: metadata.Artist{Name: "Alpha"}},
	} {
		if _, err := store.IngestSong(context.Background(), &song, nil); err != nil {
			t.Fatal(err)
		}
	}
	srv := httptest.NewServer(NewRouter(recognition.NewEngine(store), store, ""))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/songs?sort=artist&limit=2&offset=1")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var response resp
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	var list struct{ Songs []metadata.SongMetadata }
	if err := json.Unmarshal(response.Data, &list); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || len(list.Songs) != 2 ||
		list.Songs[0].Artist.Name != "Beta" || list.Songs[0].Title != "Alpha" ||
		list.Songs[1].Artist.Name != "Beta" || list.Songs[1].Title != "Zebra" {
		t.Fatalf("unexpected sorted page: %d %+v", res.StatusCode, list.Songs)
	}

	if code, _ := (&env{srv: srv}).get(t, "/api/v1/songs?sort=unknown"); code != http.StatusBadRequest {
		t.Errorf("unsupported sort: %d", code)
	}
}

func TestGetSongErrors(t *testing.T) {
	e := newEnv(t)
	if code, _ := e.get(t, "/api/v1/songs/999"); code != 404 {
		t.Errorf("missing song: %d", code)
	}
	for _, id := range []string{"abc", "0", "-3"} {
		if code, _ := e.get(t, "/api/v1/songs/"+id); code != 400 {
			t.Errorf("id %q: %d", id, code)
		}
	}
}

func TestIngestValidation(t *testing.T) {
	e := newEnv(t)
	wav := wavBytes(t, synth.Song(1, 5, sr))
	cases := []struct {
		name   string
		fields map[string]string
		file   []byte
		want   int
	}{
		{"missing title", map[string]string{"artist": "A"}, wav, 400},
		{"missing artist", map[string]string{"title": "T"}, wav, 400},
		{"missing file", map[string]string{"title": "T", "artist": "A"}, nil, 400},
		{"bad date", map[string]string{"title": "T", "artist": "A", "release_date": "01/02/2020"}, wav, 400},
		{"corrupt audio", map[string]string{"title": "T", "artist": "A"}, []byte("RIFFxxxxWAVEnope"), 400},
		{"too short", map[string]string{"title": "T", "artist": "A"}, wavBytes(t, make([]float64, 500)), 400},
		{"silent", map[string]string{"title": "T", "artist": "A"}, wavBytes(t, make([]float64, 3*sr)), 422},
	}
	for _, c := range cases {
		field := "audio"
		if c.file == nil {
			field = ""
		}
		if code, r := e.post(t, "/api/v1/songs", c.fields, field, "x.wav", c.file); code != c.want || r.Status != "error" {
			t.Errorf("%s: got %d (%s), want %d", c.name, code, r.Message, c.want)
		}
	}
	if code, _ := e.get(t, "/api/v1/songs"); code != 200 {
		t.Error("list after failed ingests")
	}
}

func TestRecognizeMatch(t *testing.T) {
	e := newEnv(t)
	e.ingest(t, 1, "Alpha")
	idB := e.ingest(t, 2, "Beta")
	e.ingest(t, 3, "Gamma")

	clip := synth.AddNoise(synth.Clip(e.songs["Beta"], sr, 14, 6), 10, 5)
	code, r := e.post(t, "/api/v1/recognize", nil, "audio", "clip.wav", wavBytes(t, clip))
	if code != 200 || r.Status != "success" {
		t.Fatalf("%d %s %s", code, r.Status, r.Message)
	}
	var d struct {
		Matched    bool                  `json:"matched"`
		Confidence float64               `json:"confidence"`
		Offset     float64               `json:"match_offset_seconds"`
		Song       metadata.SongMetadata `json:"song"`
	}
	json.Unmarshal(r.Data, &d)
	if !d.Matched || d.Song.ID != idB || d.Song.Title != "Beta" {
		t.Fatalf("wrong result: %s", r.Data)
	}
	if d.Offset < 13.8 || d.Offset > 14.2 || d.Confidence <= 0 {
		t.Errorf("offset %.1f conf %.2f", d.Offset, d.Confidence)
	}
}

func TestRecognizeNoMatch(t *testing.T) {
	e := newEnv(t)
	e.ingest(t, 1, "Alpha")
	clip := synth.Clip(synth.Song(77, 20, sr), sr, 3, 6)
	code, r := e.post(t, "/api/v1/recognize", nil, "audio", "clip.wav", wavBytes(t, clip))
	var d struct{ Matched bool }
	json.Unmarshal(r.Data, &d)
	if code != 200 || d.Matched {
		t.Fatalf("expected matched=false, got %d %s", code, r.Data)
	}
}

func TestRecognizeErrors(t *testing.T) {
	e := newEnv(t)
	if code, _ := e.post(t, "/api/v1/recognize", nil, "", "", nil); code != 400 {
		t.Errorf("no file: %d", code)
	}
	if code, _ := e.post(t, "/api/v1/recognize", nil, "wrongname", "a.wav", wavBytes(t, make([]float64, sr))); code != 400 {
		t.Errorf("wrong field: %d", code)
	}
	if code, r := e.post(t, "/api/v1/recognize", nil, "audio", "a.wav", wavBytes(t, make([]float64, 2000))); code != 400 || !strings.Contains(r.Message, "short") {
		t.Errorf("short clip: %d %s", code, r.Message)
	}
	res, _ := http.Post(e.srv.URL+"/api/v1/recognize", "application/json", strings.NewReader("{}"))
	if res.StatusCode != 400 {
		t.Errorf("non-multipart: %d", res.StatusCode)
	}
}

// Browser recordings arrive as WebM/Opus and are converted by ffmpeg.
func TestRecognizeWebMFromBrowser(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	e := newEnv(t)
	id := e.ingest(t, 4, "Delta")
	clip := synth.Clip(e.songs["Delta"], sr, 9, 6)

	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "s16le", "-ar", "44100", "-ac", "1",
		"-i", "pipe:0", "-c:a", "libopus", "-f", "webm", "pipe:1")
	var raw bytes.Buffer
	for _, v := range clip {
		s := int16(v * 32767)
		raw.Write([]byte{byte(s), byte(s >> 8)})
	}
	cmd.Stdin = &raw
	webm, err := cmd.Output()
	if err != nil || len(webm) == 0 {
		t.Skipf("cannot encode webm: %v", err)
	}
	code, r := e.post(t, "/api/v1/recognize", nil, "audio", "rec.webm", webm)
	var d struct {
		Matched bool
		Song    metadata.SongMetadata
	}
	json.Unmarshal(r.Data, &d)
	if code != 200 || !d.Matched || d.Song.ID != id {
		t.Fatalf("webm: %d %s %s", code, r.Message, r.Data)
	}
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
