-- ARTISTS
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ALBUMS 
CREATE TABLE albums (
    id SERIAL PRIMARY KEY,
    artist_id SERIAL REFERENCES artists(id) NOT NULL,
    title VARCHAR(100) NOT NULL,
    release_year int,
    cover_art_url text,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- SONGS
CREATE TABLE songs (
    id SERIAL PRIMARY KEY,
    album_id SERIAL REFERENCES albums(id) NOT NULL,
    title VARCHAR(100) NOT NULL,
    duration_seconds  DOUBLE PRECISION,
    audio_url text,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FINGERPRINTS 
CREATE TABLE fingerprints (
    hash BIGSERIAL,
    song_id SERIAL  NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    song_offset BIGSERIAL
);

-- INDEX
CREATE INDEX idx_fingerprints_hash ON fingerprints USING btree (hash);