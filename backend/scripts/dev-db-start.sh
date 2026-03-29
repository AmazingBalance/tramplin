#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PG_ROOT="${TRAMPLIN_PG_ROOT:-$ROOT_DIR/.local/postgres}"
PG_DATA="${TRAMPLIN_PGDATA:-$PG_ROOT/data}"
PG_LOG="${TRAMPLIN_PGLOG:-$PG_ROOT/postgres.log}"

if [ ! -f "$PG_DATA/PG_VERSION" ]; then
	echo "postgres data directory is not initialized: $PG_DATA" >&2
	echo "run make db-init first" >&2
	exit 1
fi

mkdir -p "$PG_ROOT"
if pg_ctl -D "$PG_DATA" status >/dev/null 2>&1; then
	echo "postgres is already running"
	exit 0
fi

pg_ctl -D "$PG_DATA" -l "$PG_LOG" -w start
