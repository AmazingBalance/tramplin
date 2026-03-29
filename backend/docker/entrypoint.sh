#!/bin/sh
set -eu

MODE="${1:-api}"

if [ "$#" -gt 0 ]; then
	shift
fi

run_migrations() {
	if [ "${TRAMPLIN_RUN_MIGRATIONS:-true}" != "true" ]; then
		return 0
	fi
	/app/tramplin-migrate
}

wait_and_run_migrations() {
	attempts="${TRAMPLIN_STARTUP_RETRIES:-30}"
	delay="${TRAMPLIN_STARTUP_RETRY_DELAY:-2}"
	count=1

	while ! run_migrations; do
		if [ "$count" -ge "$attempts" ]; then
			echo "database is not ready after ${attempts} attempts" >&2
			return 1
		fi

		echo "database is not ready yet, retrying in ${delay}s..." >&2
		count=$((count + 1))
		sleep "$delay"
	done
}

case "$MODE" in
	api)
		wait_and_run_migrations
		exec /app/tramplin-api "$@"
		;;
	migrate)
		exec /app/tramplin-migrate "$@"
		;;
	seed)
		wait_and_run_migrations
		exec /app/tramplin-seed "$@"
		;;
	*)
		exec "$MODE" "$@"
		;;
esac
