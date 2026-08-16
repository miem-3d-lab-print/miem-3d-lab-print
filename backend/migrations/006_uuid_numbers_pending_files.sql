ALTER TABLE applications
    ALTER COLUMN number TYPE VARCHAR(36);

UPDATE applications
SET number = id::text
WHERE number <> id::text;

CREATE TABLE pending_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename     VARCHAR(255) NOT NULL,
    storage_path TEXT NOT NULL UNIQUE,
    size         INTEGER NOT NULL CHECK (size > 0),
    format       file_format NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX pending_files_user_id_idx ON pending_files (user_id);
CREATE INDEX pending_files_expires_at_idx ON pending_files (expires_at);
