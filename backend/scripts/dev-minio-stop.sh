#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
MINIO_ROOT="${TRAMPLIN_MINIO_ROOT:-$ROOT_DIR/.local/minio}"
MINIO_PID="${TRAMPLIN_MINIO_PID:-$MINIO_ROOT/minio.pid}"

if [ ! -f "$MINIO_PID" ]; then
	echo "minio is not running"
	exit 0
fi

PID=$(cat "$MINIO_PID" 2>/dev/null || true)
if [ -z "${PID:-}" ]; then
	rm -f "$MINIO_PID"
	echo "minio is not running"
	exit 0
fi

if ! kill -0 "$PID" >/dev/null 2>&1; then
	rm -f "$MINIO_PID"
	echo "minio is not running"
	exit 0
fi

kill "$PID"

ATTEMPT=0
while [ "$ATTEMPT" -lt 15 ]; do
	if ! kill -0 "$PID" >/dev/null 2>&1; then
		rm -f "$MINIO_PID"
		echo "minio stopped"
		exit 0
	fi
	ATTEMPT=$((ATTEMPT + 1))
	sleep 1
done

kill -9 "$PID" >/dev/null 2>&1 || true
rm -f "$MINIO_PID"
echo "minio stopped"
