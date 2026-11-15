# Dhwani 🎵 — Music Recognition (v1: Phases 1 & 2)

Dhwani identifies songs from short audio clips using classical DSP fingerprinting
(STFT → spectral peaks → anchor/target hashes → time-offset voting). The backend is a
Go monolith on PostgreSQL; the frontend is a minimal React chat UI.

**Scope of this version:** Phase 1 (recognition engine) and Phase 2 (REST API, Postgres,
React UI). Recommendations (Phase 3) and the formal benchmark suite (Phase 4) are not included.

## What it does

- Upload an audio file or record from the microphone, and get the song, artist, album, a match
  score, and where in the song your clip starts.
- Add songs to the catalog from the UI or the API. Each is fingerprinted and indexed.
- Formats: WAV decoded natively in Go, MP3 via `go-mp3`, everything else (browser WebM/Opus,
  OGG, M4A, FLAC) via **ffmpeg** on the server.

## Project layout

```text
dhwani/
├── docker-compose.yml          # Postgres 16, runs the migration on first start
├── Makefile                    # test / build / samples targets
├── database/migrations/001_initial_schema.sql
├── scripts/seed.sh             # ingest sample songs and try every clip
├── frontend/                   # React + Vite
│   └── src/{components,hooks,services}
└── backend/                    # Go
    ├── main.go
    ├── api/                    # router + handlers
    ├── metadata/               # models, PostgresStore, MemoryStore
    ├── recognition/
    │   ├── audio/              # WAV / MP3 / ffmpeg decoding, resampling
    │   ├── fft/                # STFT, Hann window, spectrogram PNG
    │   ├── peaks/              # 2D local-maximum peak picking
    │   ├── fingerprint/        # anchor-target pairing + 32-bit hash
    │   └── matcher/            # offset histogram voting + confidence
    ├── internal/synth/         # synthetic songs / noise for tests
    └── cmd/gensamples/         # writes sample songs and query clips
```

## How recognition works

| Step | Detail |
|---|---|
| Decode | Mono, scaled to [-1, 1], resampled (linear) to 44.1 kHz |
| STFT | Hann window N = 2048, hop 512 → **75% overlap**, 1024 bins (21.5 Hz each) |
| Peaks | Local maxima in a 25 × 31 frame/bin neighbourhood, within 40 dB of the loudest point, then balanced across 6 frequency bands |
| Hash | `f1 (10 bits) \| f2 (10 bits) \| Δt (12 bits)` = 32 bits. Each anchor pairs with up to 10 targets 1–96 frames ahead |
| Match | Look up hashes, compute `offset = t_db − t_query`, histogram per song (±1 frame tolerance), pick the tallest cluster |
| Accept | At least 8 aligned hashes **and** 1.5× the runner-up |

Confidence = (aligned hashes ÷ all hash hits) × a small damping term for tiny clusters. The UI
also shows the landmark count. Noisy clips score lower, and 20–50% can still be a clear match.

See the [audio recognition communication flow](docs/audio-recognition-flow.md) for the upload-to-result sequence.

## Run it

Prerequisites: Go ≥ 1.22, Node ≥ 18, PostgreSQL ≥ 15 (or Docker), ffmpeg.

```bash
# 1. Database
docker compose up -d            # or: createdb dhwani && psql -d dhwani -f database/migrations/001_initial_schema.sql
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/dhwani?sslmode=disable'

# 2. Backend (first run fetches pgx and go-mp3)
cd backend && go mod tidy && go run .

# 3. Frontend, in another terminal. The dev server proxies /api to :8080.
cd frontend && npm install && npm run dev      # http://localhost:5173
```

The microphone only works on `localhost` or HTTPS.

**Try it with generated audio** (no copyrighted music needed):

```bash
make samples                    # writes samples/songs and samples/clips
./scripts/seed.sh               # ingests 5 songs, then recognizes every clip
```

Or open the UI, click **Catalog**, add `samples/songs/song1.wav`, then upload
`samples/clips/song1_noisy10dB.wav` in the chat.

**No database handy?** `make run-nodeps` serves UI + API with an in-memory store
(data is lost on exit) and uses ffmpeg for MP3.

## API

All responses are `{"status":"success","data":…}` or `{"status":"error","message":…}`.

| Method & path | Purpose |
|---|---|
| `POST /api/v1/recognize` | multipart field `audio` (≤ 25 MB, ≥ 1 s) → `matched`, `confidence`, `votes`, `match_offset_seconds`, `song` |
| `POST /api/v1/songs` | multipart `audio`, `title`\*, `artist`\*, `album`, `release_date` (YYYY-MM-DD), `cover_art_url`, `spotify`, `youtube` → `201` with song + fingerprint count |
| `GET /api/v1/songs` | catalog, newest first by default (`?limit=&offset=`); use `?sort=artist` to sort alphabetically by artist, then song title |
| `GET /api/v1/songs/{id}` | one song with artist, album, links |
| `GET /healthz` | liveness |

Sample match response:

```json
{
  "status": "success",
  "data": {
    "matched": true,
    "confidence": 0.41,
    "votes": 32,
    "match_offset_seconds": 12,
    "song": {
      "id": 3, "title": "Synth Song 3", "duration_seconds": 60,
      "artist": { "id": 1, "name": "The Synths" },
      "album": { "id": 1, "title": "Test Tones", "release_date": "2024-01-03" },
      "external_links": { "spotify": "https://open.spotify.com/track/demo3" }
    }
  }
}
```

## Tests

```bash
make test                # backend (nodeps build) + frontend
make test-integration    # Postgres round trip; needs DHWANI_TEST_DB
```

- **Unit:** WAV parsing (8/16/24-bit, float, stereo, streamed headers), resampling, FFT vs. a naive
  DFT, Hann window, STFT shape and overlap, sine → correct bin, peak picking, hash bit packing,
  anchor/target pairing, offset voting, ambiguity rejection, confidence.
- **Engine (`recognition/engine_test.go`):** clean clips with correct offset, white noise at
  20/10/5 dB SNR, volume 0.1×–2.0× with clipping, 22.05 kHz queries, unknown song and
  noise → no match.
- **API:** ingest → get → list, validation errors, recognition match / no match, and a real
  **WebM/Opus upload through ffmpeg**, as the browser sends it.
- **Frontend (Vitest):** result card, API client, chat flow (match / no match / error / busy),
  catalog panel.

### Build tags

The development sandbox could not download Go modules. Code needing third-party modules is
therefore tagged `!nodeps` (`metadata/postgres.go`, `audio/mp3_gomp3.go`, `store_postgres.go`).
With `-tags nodeps`, an in-memory store and ffmpeg stand in so everything else could be tested.
On a normal machine, don't pass the tag. `go mod tidy` adds `pgx/v5` and `go-mp3`.
The SQL in `postgres.go` was run against a real PostgreSQL 16, but **the pgx code itself has not
been compiled yet**. Run `make test-integration` first.

## Known limits (deliberately left for the optimization phases)

- Parameters were tuned on synthetic tonal songs. Real music will need peak density and threshold
  retuning.
- Resampling is plain linear interpolation (no anti-alias filter).
- Whole files are decoded in memory. Full tracks are fine, hour-long mixes are not.
- Every query hash is looked up in one `ANY($1)` query. Latency will grow with catalog size.
- No auth, rate limiting, or audio storage. Only fingerprints and metadata are kept.
