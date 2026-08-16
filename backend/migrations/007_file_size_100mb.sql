ALTER TABLE files
    DROP CONSTRAINT IF EXISTS files_size_check,
    ADD CONSTRAINT files_size_check CHECK (size > 0 AND size <= 104857600);

ALTER TABLE pending_files
    DROP CONSTRAINT IF EXISTS pending_files_size_check,
    ADD CONSTRAINT pending_files_size_check CHECK (size > 0 AND size <= 104857600);
