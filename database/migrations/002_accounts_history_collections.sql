BEGIN;

-- Email + password sign-in. Existing rows get '' (cannot log in until a password is set).
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';

-- One row per successful recognition.
CREATE TABLE IF NOT EXISTS recognition_history (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id INT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX IF NOT EXISTS idx_history_user_created ON recognition_history (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_song ON recognition_history (song_id);

-- Named lists of saved songs. "Favourites" is created for every new user.
CREATE TABLE IF NOT EXISTS collections (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 60),
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    UNIQUE (user_id, name)
);

CREATE TABLE IF NOT EXISTS collection_songs (
    collection_id INT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    song_id INT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    added_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    PRIMARY KEY (collection_id, song_id)
);

CREATE INDEX IF NOT EXISTS idx_collection_songs_song ON collection_songs (song_id);

COMMIT;
