# Dhvani

Identifies songs from short audio clips (STFT → spectral peaks → anchor/target hashes → offset voting).
Go API + PostgreSQL + React/Vite frontend in the Dhvani mint/teal design (Manrope + Inter).

## Layout

```text
database/migrations/001_initial_schema.sql              your schema (artists, users, updates, songs, fingerprints)
database/migrations/002_accounts_history_collections.sql  password_hash, recognition_history, collections, collection_songs
database/migrations/004_catalog_ownership_and_moderation.sql  song upload ownership and removal requests
docker-compose.yml                                      Postgres 16; applies migrations on first start
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

For an existing database, apply unapplied migrations manually; Docker only runs init scripts when its data volume is first created:

```bash
docker compose exec -T db psql -U postgres -d dhwani -f /docker-entrypoint-initdb.d/004_catalog_ownership_and_moderation.sql
# Or, with psql installed locally:
psql "$DATABASE_URL" -f database/migrations/004_catalog_ownership_and_moderation.sql
```

To grant admin review access, first create the admin account, then set its role directly in PostgreSQL:

```sql
UPDATE users SET is_admin = TRUE WHERE email = 'admin@example.com';
```

After the splash screen, identify clips and browse the shared catalog without signing in. Sign in to save songs or view personal history; create an account or sign in to upload a song with its details. Original audio files are discarded after fingerprinting; the catalog stores metadata and fingerprints, not playable/downloadable audio. Uploaders can request removal, and an admin must approve before a song is removed globally.
Microphone recording needs `localhost` or HTTPS.

Existing catalog songs have no uploader attribution because the previous schema only tracked artist-level updates. They remain public, but only uploads made after migration 004 appear in an account's upload list or can be submitted for uploader removal review.

## Screens

Splash · Home (identify by microphone or upload, results with
other likely matches) · Search (`/` or ⌘K) · Explore (genres, search) · Library · History · Saved Matches
(collections) · My Account · Sign in / Create account · Upload a Song.

## API

JSON responses: `{"status":"success","data":…}` / `{"status":"error","message":…}`. Personal data,
song uploads, and removal requests need `Authorization: Bearer <token>`. Recognition and read-only
catalog endpoints are public. Favorites and collections remain private to each user.

| Endpoint | Purpose |
|---|---|
| `POST /api/v1/auth/register`, `/auth/login` | `{name,email,password}` / `{email,password}` → `{token,user}` |
| `GET /api/v1/me`, `PATCH /api/v1/me` | profile + stats; rename |
| `POST /api/v1/recognize` | Public multipart `audio` → match, confidence, offset, others; records history only when signed in |
| `GET /api/v1/history` | your identifications, newest first |
| `GET /api/v1/songs?q=&genre=&sort=recent\|artist\|popular` · `GET /songs/{id}` · `GET /genres` | Public catalog |
| `POST /api/v1/songs` | Authenticated multipart `audio`, `title`, `artist`, `genre`, `cover_image_url`; stores fingerprints and attributes the song to the uploader |
| `GET /api/v1/me/uploads` | your uploaded catalog songs and removal-request status |
| `POST /api/v1/songs/{id}/removal-requests` | uploader requests admin review; does not immediately remove the shared song |
| `GET /api/v1/admin/removal-requests` · `PATCH /api/v1/admin/removal-requests/{id}` | admins list requests and approve (`{"decision":"approve"}`) or reject (`{"decision":"reject"}`); approval removes the song and its fingerprints, history, and saved references for everyone |
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
