-- ENUM types
CREATE TYPE user_role AS ENUM ('user', 'admin');
CREATE TYPE rate_limit_type AS ENUM ('email', 'ip');
CREATE TYPE application_status AS ENUM (
    'new', 'in_review', 'printing', 'ready', 'issued', 'rejected', 'cancelled'
);
CREATE TYPE application_position AS ENUM (
    'bachelor', 'master', 'postgraduate', 'employee'
);
CREATE TYPE file_format AS ENUM ('STL', 'STEP', '3MF');

-- users
CREATE TABLE users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255),
    telegram        VARCHAR(255),
    max             VARCHAR(255),
    telegram_id     BIGINT,
    role            user_role   NOT NULL DEFAULT 'user',
    consent_given   BOOLEAN     NOT NULL DEFAULT false,
    consent_given_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_consent_check CHECK (NOT consent_given OR consent_given_at IS NOT NULL)
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));
CREATE UNIQUE INDEX users_telegram_id_key ON users (telegram_id) WHERE telegram_id IS NOT NULL;
CREATE INDEX users_role_admin_idx ON users (id) WHERE role = 'admin';

-- otp_codes
CREATE TABLE otp_codes (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    code_hash     VARCHAR(60)  NOT NULL,
    attempts      SMALLINT     NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    expires_at    TIMESTAMPTZ  NOT NULL,
    blocked_until TIMESTAMPTZ,
    is_used       BOOLEAN      NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX otp_codes_email_idx ON otp_codes (email) WHERE is_used = false;
CREATE INDEX otp_codes_cleanup_idx ON otp_codes (expires_at);

-- refresh_tokens
CREATE TABLE refresh_tokens (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash           VARCHAR(64) NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL,
    revoked_at           TIMESTAMPTZ,
    replaced_by_token_id UUID        REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX refresh_tokens_hash_key ON refresh_tokens (token_hash);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_cleanup_idx ON refresh_tokens (expires_at);
