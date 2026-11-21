# Dhvani

Identifies songs from short audio clips (STFT → spectral peaks → anchor/target hashes → offset voting).
Go API + PostgreSQL + React/Vite frontend in the Dhvani mint/teal design (Manrope + Inter).

## Layout

```text
database/migrations/001_initial_schema.sql              your schema (artists, users, updates, songs, fingerprints)
database/migrations/002_accounts_history_collections.sql  password_hash, recognition_history, collections, collection_songs
docker-compose.yml                                      Postgres 16; applies both migrations on first start
backend/    main.go · auth/ · api/ · metadata/ · recognition/
frontend/   src/App.jsx · pages/ (Home, Explore, Library, History, Saved, Profile, AuthPage) · ui.jsx · styles.css
```

## Run

Needs Go ≥ 1.24, Node ≥ 18, ffmpeg and PostgreSQL (or Docker).

```bash
docker compose up -d
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/dhwani?sslmode=disable'
export JWT_SECRET='<16+ random characters>'     # if unset, a random key is used and sign-ins end on restart
cd backend && go mod tidy && go run .            # API on :8080  (no DATABASE_URL → in-memory store)
cd frontend && npm install && npm run dev        # UI on http://localhost:5173 (proxies /api to :8080)
```

Create an account in the UI, add songs under **Library → Grow the catalog**, then identify clips from the home page.
Microphone recording needs `localhost` or HTTPS.

## Screens

Splash · Sign in / Create account · Home (hero carousel, identify by microphone or upload, results with
other likely matches) · Search (`/` or ⌘K) · Explore (genres, search) · Library · History · Saved Matches
(collections) · My Account.

## API

JSON responses: `{"status":"success","data":…}` / `{"status":"error","message":…}`. Everything except
`/auth/*` and `/healthz` needs `Authorization: Bearer <token>`.

| Endpoint | Purpose |
|---|---|
| `POST /api/v1/auth/register`, `/auth/login` | `{name,email,password}` / `{email,password}` → `{token,user}` |
| `GET /api/v1/me`, `PATCH /api/v1/me` | profile + stats; rename |
| `POST /api/v1/recognize` | multipart `audio` → match, confidence, offset, others; records history |
| `GET /api/v1/history` | your identifications, newest first |
| `GET /api/v1/songs?q=&genre=&sort=recent\|artist\|popular` · `GET /songs/{id}` · `GET /genres` | catalog |
| `POST /api/v1/songs` | multipart `audio`, `title`, `artist`, `genre`, `cover_image_url` (records a row in `updates`) |
| `GET/POST /api/v1/collections` · `GET /collections/{id\|all}/songs` · `PUT/DELETE /collections/{id}/songs/{songId}` | saved matches |

## Decisions and deviations

- `user` → `users` (reserved word in PostgreSQL). `fingerprints` is `(song_id, hash, offset_frames)` with a non-unique, indexed `hash`.
- Passwords use PBKDF2-SHA256 (standard library) and tokens are HS256 JWTs, so there are no extra Go dependencies beyond pgx.
- The design's album, release year and "Open in music app" are not in the schema, so they are omitted. There is no audio preview, so no play buttons.
- "Free Plan 17/30" is display-only: the API does not limit identifications.
- Saved Matches = songs in any of your collections; the heart saves to **Favourites**. "Identified This Week" is computed from history.
- No rate limiting on login yet.

## Tests

`cd backend && go test ./...` — engine on synthetic songs: correct offset, 10 dB / 5 dB noise, unknown song, noise only.
