#!/usr/bin/env bash
set -euo pipefail
# Simple helper to run backend and flutter app.
# Usage:
#   ./dev.sh backend   # start Go backend
#   ./dev.sh app       # run Flutter app (choose first iOS simulator if available)
#   ./dev.sh all       # start backend (background) then app

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"

run_backend() {
  echo "[dev] Starting backend (STRIPE_AUTO_SUBSCRIBE=1)"
  (cd "$BACKEND_DIR" && STRIPE_AUTO_SUBSCRIBE=1 go run .)
}

run_app() {
  echo "[dev] Detecting devices..."
  local dev_id
  dev_id=$(flutter devices --machine 2>/dev/null | grep -E '"id":' | head -1 | sed -E 's/.*"id":"([^"]+)".*/\1/') || true
  if [ -z "$dev_id" ]; then
    echo "[dev] No device found. Open an iOS simulator or connect a device and retry." >&2
    exit 1
  fi
  echo "[dev] Using device $dev_id"
  flutter run -d "$dev_id"
}

case "${1:-all}" in
  backend) run_backend ;;
  app) run_app ;;
  all)
    run_backend &
    sleep 2
    run_app
    ;;
  *) echo "Usage: $0 {backend|app|all}"; exit 1 ;;
 esac
