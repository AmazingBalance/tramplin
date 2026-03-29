#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PG_ROOT="${TRAMPLIN_PG_ROOT:-$ROOT_DIR/.local/postgres}"
PG_DATA="${TRAMPLIN_PGDATA:-$PG_ROOT/data}"

if [ ! -f "$PG_DATA/PG_VERSION" ]; then
	echo "postgres data directory is not initialized: $PG_DATA" >&2
	exit 1
fi

pg_ctl -D "$PG_DATA" status
