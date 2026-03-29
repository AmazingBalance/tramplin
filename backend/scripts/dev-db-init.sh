#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PG_ROOT="${TRAMPLIN_PG_ROOT:-$ROOT_DIR/.local/postgres}"
PG_DATA="${TRAMPLIN_PGDATA:-$PG_ROOT/data}"
PG_RUN="${TRAMPLIN_PGSOCK:-$PG_ROOT/run}"
PG_LOG="${TRAMPLIN_PGLOG:-$PG_ROOT/postgres.log}"
PG_PORT="${TRAMPLIN_PGPORT:-5432}"
APP_DB_USER="${TRAMPLIN_APP_DB_USER:-tramplin}"
APP_DB_PASSWORD="${TRAMPLIN_APP_DB_PASSWORD:-tramplin}"
APP_DB_NAME="${TRAMPLIN_APP_DB_NAME:-tramplin}"

mkdir -p "$PG_ROOT" "$PG_RUN"

if [ ! -f "$PG_DATA/PG_VERSION" ]; then
	initdb -D "$PG_DATA" -U postgres --encoding=UTF8 --auth-local=trust --auth-host=scram-sha-256

	{
		echo ""
		echo "# Tramplin local development overrides"
		echo "listen_addresses = '127.0.0.1'"
		echo "port = $PG_PORT"
		echo "unix_socket_directories = '$PG_RUN'"
	} >> "$PG_DATA/postgresql.conf"
else
	echo "postgres data directory already initialized at $PG_DATA"
fi

if ! pg_ctl -D "$PG_DATA" status >/dev/null 2>&1; then
	pg_ctl -D "$PG_DATA" -l "$PG_LOG" -w start
fi

psql -h "$PG_RUN" -U postgres -d postgres -v ON_ERROR_STOP=1 <<SQL
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$APP_DB_USER') THEN
        EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', '$APP_DB_USER', '$APP_DB_PASSWORD');
    ELSE
        EXECUTE format('ALTER ROLE %I WITH LOGIN PASSWORD %L', '$APP_DB_USER', '$APP_DB_PASSWORD');
    END IF;
END
\$\$;
SQL

if ! psql -h "$PG_RUN" -U postgres -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$APP_DB_NAME'" | grep -q 1; then
	createdb -h "$PG_RUN" -U postgres -O "$APP_DB_USER" "$APP_DB_NAME"
fi

echo "postgres initialized"
echo "data: $PG_DATA"
echo "socket: $PG_RUN"
echo "database: $APP_DB_NAME"
echo "role: $APP_DB_USER"

DB_URL="${TRAMPLIN_DATABASE_URL:-postgres://$APP_DB_USER:$APP_DB_PASSWORD@127.0.0.1:$PG_PORT/$APP_DB_NAME?sslmode=disable}"
TRAMPLIN_DATABASE_URL="$DB_URL" "$ROOT_DIR/scripts/dev-db-migrate.sh"
