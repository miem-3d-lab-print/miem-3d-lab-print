#!/bin/sh

set -eu

: "${POSTGRES_HOST:=db}"
: "${POSTGRES_PORT:=5432}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"

export PGPASSWORD="$POSTGRES_PASSWORD"

psql_args="-h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB"

psql $psql_args -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version varchar(255) PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);
SQL

for migration in /migrations/*.sql; do
    version=$(basename "$migration")
    applied=$(psql $psql_args -Atqc "SELECT 1 FROM schema_migrations WHERE version = '$version'")

    if [ "$applied" = "1" ]; then
        continue
    fi

    echo "Applying migration $version"
    psql $psql_args -1 -v ON_ERROR_STOP=1 <<SQL
SELECT pg_advisory_xact_lock(31032026);
\i $migration
INSERT INTO schema_migrations (version) VALUES ('$version');
SQL
done
