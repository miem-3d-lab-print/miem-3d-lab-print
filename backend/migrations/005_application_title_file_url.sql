ALTER TABLE applications
    ADD COLUMN title VARCHAR(255) NOT NULL DEFAULT 'Заявка на 3D-печать',
    ADD COLUMN file_url TEXT;

ALTER TABLE applications
    ADD CONSTRAINT applications_title_not_blank CHECK (length(btrim(title)) > 0),
    ADD CONSTRAINT applications_file_url_length CHECK (file_url IS NULL OR length(file_url) <= 2048);
