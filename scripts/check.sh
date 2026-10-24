#!/bin/bash
# 检查 mb2-go 是否在跑，没跑就启动。
# crontab 每分钟检查一次：
#   * * * * * /opt/mb2-go/scripts/check.sh >/dev/null 2>&1

set -eu

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$APP_DIR/mb2-go"
CFG="$APP_DIR/configs/config.yaml"
PIDFILE="$APP_DIR/data/mb2-go.pid"
WATCH_LOG="$APP_DIR/logs/watchdog.log"

mkdir -p "$APP_DIR/data" "$APP_DIR/logs"

log() {
	echo "$(date '+%F %T') $*" >>"$WATCH_LOG"
}

is_alive() {
	local pid="$1"
	[[ -n "$pid" ]] || return 1
	kill -0 "$pid" 2>/dev/null || return 1
	if [[ -r "/proc/$pid/cmdline" ]]; then
		tr '\0' ' ' <"/proc/$pid/cmdline" | grep -q 'mb2-go' || return 1
	fi
	return 0
}

current_pid() {
	if [[ -f "$PIDFILE" ]]; then
		local pid
		pid="$(cat "$PIDFILE" 2>/dev/null || true)"
		if is_alive "$pid"; then
			echo "$pid"
			return 0
		fi
	fi
	pgrep -f "$BIN -config $CFG" | head -n1 || true
}

if [[ ! -x "$BIN" ]]; then
	log "ERROR binary not found or not executable: $BIN"
	exit 1
fi

if [[ ! -f "$CFG" ]]; then
	log "ERROR config not found: $CFG"
	exit 1
fi

pid="$(current_pid || true)"
if [[ -n "${pid:-}" ]]; then
	echo "$pid" >"$PIDFILE"
	exit 0
fi

cd "$APP_DIR"
nohup "$BIN" -config "$CFG" >>"$APP_DIR/logs/nohup.out" 2>&1 &
echo $! >"$PIDFILE"
log "process not running, started pid=$!"
