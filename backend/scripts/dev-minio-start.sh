#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
MINIO_ROOT="${TRAMPLIN_MINIO_ROOT:-$ROOT_DIR/.local/minio}"
MINIO_DATA="${TRAMPLIN_MINIO_DATA:-$MINIO_ROOT/data}"
MINIO_LOG="${TRAMPLIN_MINIO_LOG:-$MINIO_ROOT/minio.log}"
MINIO_PID="${TRAMPLIN_MINIO_PID:-$MINIO_ROOT/minio.pid}"
MINIO_ADDRESS="${TRAMPLIN_OBJECT_STORAGE_ENDPOINT:-127.0.0.1:9000}"
MINIO_CONSOLE_ADDRESS="${TRAMPLIN_MINIO_CONSOLE_ADDRESS:-127.0.0.1:9001}"
MINIO_ACCESS_KEY="${TRAMPLIN_OBJECT_STORAGE_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${TRAMPLIN_OBJECT_STORAGE_SECRET_KEY:-minioadmin}"
MINIO_HEALTH_URL="http://${MINIO_ADDRESS}/minio/health/live"

mkdir -p "$MINIO_ROOT" "$MINIO_DATA"

if [ -f "$MINIO_PID" ]; then
	PID=$(cat "$MINIO_PID" 2>/dev/null || true)
	if [ -n "${PID:-}" ] && kill -0 "$PID" >/dev/null 2>&1; then
		echo "minio is already running"
		exit 0
	fi
	rm -f "$MINIO_PID"
fi

if command -v setsid >/dev/null 2>&1; then
	MINIO_ROOT_USER="$MINIO_ACCESS_KEY" \
	MINIO_ROOT_PASSWORD="$MINIO_SECRET_KEY" \
	setsid minio server "$MINIO_DATA" \
		--address "$MINIO_ADDRESS" \
		--console-address "$MINIO_CONSOLE_ADDRESS" \
		>>"$MINIO_LOG" 2>&1 </dev/null &
else
	MINIO_ROOT_USER="$MINIO_ACCESS_KEY" \
	MINIO_ROOT_PASSWORD="$MINIO_SECRET_KEY" \
	nohup minio server "$MINIO_DATA" \
		--address "$MINIO_ADDRESS" \
		--console-address "$MINIO_CONSOLE_ADDRESS" \
		>>"$MINIO_LOG" 2>&1 </dev/null &
fi
PID=$!
echo "$PID" >"$MINIO_PID"

READY=0
ATTEMPT=0
while [ "$ATTEMPT" -lt 30 ]; do
	if ! kill -0 "$PID" >/dev/null 2>&1; then
		echo "minio failed to start; see $MINIO_LOG" >&2
		rm -f "$MINIO_PID"
		exit 1
	fi
	if curl -sS "$MINIO_HEALTH_URL" >/dev/null 2>&1; then
		READY=1
		break
	fi
	ATTEMPT=$((ATTEMPT + 1))
	sleep 1
done

if [ "$READY" -ne 1 ]; then
	echo "minio did not become ready; see $MINIO_LOG" >&2
	exit 1
fi

echo "minio started"
echo "data: $MINIO_DATA"
echo "api: http://$MINIO_ADDRESS"
echo "console: http://$MINIO_CONSOLE_ADDRESS"
echo "log: $MINIO_LOG"
