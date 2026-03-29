#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
MIGRATIONS_DIR="$ROOT_DIR/db/migrations"
DB_URL="${TRAMPLIN_DATABASE_URL:-postgres://tramplin:tramplin@127.0.0.1:5432/tramplin?sslmode=disable}"

psql "$DB_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version varchar(255) PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);
SQL

for file in "$MIGRATIONS_DIR"/*.sql; do
	version=$(basename "$file")
	applied=$(psql "$DB_URL" -tAc "SELECT 1 FROM schema_migrations WHERE version = '$version'")
	if [ "$applied" = "1" ]; then
		continue
	fi

	echo "applying migration: $version"
	psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$file"
	psql "$DB_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version) VALUES ('$version')"
done
