BEGIN;

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE songs ADD COLUMN IF NOT EXISTS uploaded_by_user_id INT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_songs_uploader ON songs (uploaded_by_user_id);

CREATE TABLE IF NOT EXISTS song_removal_requests (
    id SERIAL PRIMARY KEY,
    song_id INT REFERENCES songs(id) ON DELETE SET NULL,
    requested_by_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    reviewed_by_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_pending_removal_per_song
    ON song_removal_requests (song_id) WHERE status = 'pending';

COMMIT;
