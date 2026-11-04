BEGIN;

CREATE TABLE IF NOT EXISTS artists (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
);

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS user (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email CITEXT NOT NULL UNIQUE,
    create_date TIMESTAMP DEFAULT NOW(),

    CONSTRAINT check_email_format CHECK (email ~* '^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$')
);

-- this table stores data about which user made last update on which song
CREATE TABLE IF NOT EXISTS updates (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    artist_id INT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    UNIQUE (artist_id, user_id)
);

CREATE TABLE IF NOT EXISTS songs (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    artist_id INT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    genre TEXT,
    -- default link if the column is omitted during INSERT 
    -- implemented in the client layer
    cover_image_url TEXT,
    duration INT NOT NULL CHECK (duration > 0),
    UNIQUE (title, artist_id)
);

CREATE TABLE IF NOT EXISTS fingerprints  (
    id SERIAL PRIMARY KEY,
    song_id INT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    hash BIGINT NULL UNIQUE,
    UNIQUE (song_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_fingerprints_song_id ON fingerprints (song_id);

COMMIT;
