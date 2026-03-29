#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
MINIO_ROOT="${TRAMPLIN_MINIO_ROOT:-$ROOT_DIR/.local/minio}"
MINIO_PID="${TRAMPLIN_MINIO_PID:-$MINIO_ROOT/minio.pid}"
MINIO_ADDRESS="${TRAMPLIN_OBJECT_STORAGE_ENDPOINT:-127.0.0.1:9000}"
MINIO_CONSOLE_ADDRESS="${TRAMPLIN_MINIO_CONSOLE_ADDRESS:-127.0.0.1:9001}"
MINIO_HEALTH_URL="http://${MINIO_ADDRESS}/minio/health/live"

if [ ! -f "$MINIO_PID" ]; then
	if curl -sS "$MINIO_HEALTH_URL" >/dev/null 2>&1; then
		echo "minio is running"
		echo "pid: unknown"
		echo "api: http://$MINIO_ADDRESS"
		echo "console: http://$MINIO_CONSOLE_ADDRESS"
		exit 0
	fi
	echo "minio is not running"
	exit 1
fi

PID=$(cat "$MINIO_PID" 2>/dev/null || true)
PROCESS_RUNNING=0
if [ -n "${PID:-}" ] && kill -0 "$PID" >/dev/null 2>&1; then
	PROCESS_RUNNING=1
fi

if curl -sS "$MINIO_HEALTH_URL" >/dev/null 2>&1; then
	echo "minio is running"
	if [ "$PROCESS_RUNNING" -eq 1 ]; then
		echo "pid: $PID"
	else
		echo "pid: unknown"
	fi
	echo "api: http://$MINIO_ADDRESS"
	echo "console: http://$MINIO_CONSOLE_ADDRESS"
	exit 0
fi

if [ "$PROCESS_RUNNING" -eq 1 ]; then
	echo "minio process exists but health check failed"
	echo "pid: $PID"
	exit 1
fi

echo "minio is not running"
exit 1
