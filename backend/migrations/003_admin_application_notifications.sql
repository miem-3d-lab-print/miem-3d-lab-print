ALTER TABLE users
    ADD COLUMN application_notifications BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX users_application_notifications_idx ON users (id)
    WHERE role = 'admin' AND application_notifications = true;
