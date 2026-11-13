#!/usr/bin/env sh
# Runs Klean for development: the Go backend plus the Vite dev server with hot reload.
#
#   ./scripts/dev.sh                     backend on :8787, UI on http://localhost:5173
#   ./scripts/dev.sh --mock              UI only, backed by the in-memory mock (no Go server)
#   ./scripts/dev.sh -- --host 0.0.0.0   extra arguments go to the Vite dev server
#
# Ctrl+C stops both. The dev database and binary live in .dev/ (gitignored).
# Needs Go 1.26+ (https://go.dev/dl/) and Node.js 24+ (https://nodejs.org/).
set -eu
cd "$(dirname "$0")/.."

mock=0
if [ "${1:-}" = "--mock" ]; then
  mock=1
  shift
fi
[ "${1:-}" = "--" ] && shift

die() { printf '\n%s\n' "$1" >&2; exit 1; }

command -v npm >/dev/null 2>&1 || die "Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/"
if [ "$mock" = 0 ]; then
  command -v go >/dev/null 2>&1 || die "Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/"
fi

if [ ! -d frontend/node_modules ]; then
  echo "Installing frontend dependencies..."
  (cd frontend && npm ci)
fi

if [ "$mock" = 1 ]; then
  echo "UI:      http://localhost:5173  (mock backend, no Go server)"
  (cd frontend && exec npm run dev:mock -- "$@")
fi

# The Go binary embeds frontend/dist and won't compile without it.
if [ ! -f frontend/dist/index.html ]; then
  echo "Building the frontend once, so the backend can embed it..."
  (cd frontend && npm run build)
fi

mkdir -p .dev
go build -o .dev/klean ./backend
.dev/klean --headless --db .dev/klean.db &
backend_pid=$!
trap 'kill "$backend_pid" 2>/dev/null || true' EXIT INT TERM

echo
echo "backend: http://localhost:8787"
echo "UI:      http://localhost:5173"
echo

(cd frontend && npm run dev -- "$@")
