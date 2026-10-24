#!/bin/bash
set -eu
APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PIDFILE="$APP_DIR/data/mb2-go.pid"
BIN="$APP_DIR/mb2-go"
CFG="$APP_DIR/configs/config.yaml"

stop_pid() {
	local pid="$1"
	[[ -n "$pid" ]] || return 0
	kill "$pid" 2>/dev/null || true
	for _ in 1 2 3 4 5; do
		kill -0 "$pid" 2>/dev/null || return 0
		sleep 1
	done
	kill -9 "$pid" 2>/dev/null || true
}

if [[ -f "$PIDFILE" ]]; then
	stop_pid "$(cat "$PIDFILE")"
	rm -f "$PIDFILE"
fi

pgrep -f "$BIN -config $CFG" | while read -r pid; do
	stop_pid "$pid"
done

echo "mb2-go stopped"
