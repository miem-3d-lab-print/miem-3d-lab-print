-- materials
CREATE TABLE materials (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX materials_name_key ON materials (lower(name));

-- colors
CREATE TABLE colors (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id UUID         NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    name        VARCHAR(100) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX colors_material_name_key ON colors (material_id, lower(name));

-- applications
CREATE TABLE applications (
    id                    UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    number                VARCHAR(12)          NOT NULL,
    user_id               UUID                 NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    snapshot_full_name    VARCHAR(255)         NOT NULL,
    snapshot_email        VARCHAR(255)         NOT NULL,
    position              application_position NOT NULL,
    purpose               TEXT                 NOT NULL,
    material_id           UUID                 NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    snapshot_material_name VARCHAR(100)        NOT NULL,
    color_matters         BOOLEAN              NOT NULL,
    color_id              UUID                 REFERENCES colors(id) ON DELETE RESTRICT,
    snapshot_color_name   VARCHAR(100),
    desired_date          DATE                 NOT NULL,
    comment               TEXT,
    status                application_status   NOT NULL DEFAULT 'new',
    rejection_reason      TEXT,
    files_delete_after    TIMESTAMPTZ,
    deadline_notified     BOOLEAN              NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ          NOT NULL DEFAULT now(),
    CONSTRAINT color_consistency CHECK (
        (color_matters = false AND color_id IS NULL AND snapshot_color_name IS NULL)
        OR
        (color_matters = true AND color_id IS NOT NULL AND snapshot_color_name IS NOT NULL)
    ),
    CONSTRAINT rejection_reason_check CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL)
);
CREATE UNIQUE INDEX applications_number_key ON applications (number);
CREATE INDEX applications_user_idx ON applications (user_id);
CREATE INDEX applications_created_idx ON applications (created_at DESC);
CREATE INDEX applications_status_idx ON applications (status);
CREATE INDEX applications_active_per_user_idx ON applications (user_id)
    WHERE status IN ('new', 'in_review', 'printing');
CREATE INDEX applications_deadline_idx ON applications (desired_date)
    WHERE status = 'new';
CREATE INDEX applications_files_ttl_idx ON applications (files_delete_after)
    WHERE files_delete_after IS NOT NULL;

-- files
CREATE TABLE files (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    filename       VARCHAR(255) NOT NULL,
    storage_path   VARCHAR(512) NOT NULL,
    size           INTEGER      NOT NULL CHECK (size > 0 AND size <= 20971520),
    format         file_format  NOT NULL,
    deleted_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX files_application_idx ON files (application_id);
CREATE UNIQUE INDEX files_storage_path_key ON files (storage_path);

-- status_history
CREATE TABLE status_history (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID               NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    status         application_status NOT NULL,
    comment        TEXT,
    changed_by     UUID               NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now()
);
CREATE INDEX status_history_app_idx ON status_history (application_id, created_at);
CREATE INDEX status_history_status_created_idx ON status_history (status, created_at);

-- seed materials
INSERT INTO materials (name, description, is_active) VALUES
    ('PLA', 'Жёсткий и экологичный пластик. Подходит для большинства учебных проектов.', true),
    ('ABS', 'Прочный и ударостойкий материал. Рекомендуется для функциональных деталей.', true),
    ('TPU', 'Гибкий и эластичный пластик. Подходит для изделий, требующих упругости.', true);
