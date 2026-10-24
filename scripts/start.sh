#!/bin/bash
set -eu
APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
"$APP_DIR/scripts/check.sh"
echo "mb2-go started, pid=$(cat "$APP_DIR/data/mb2-go.pid")"
