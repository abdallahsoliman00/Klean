#!/usr/bin/env sh
# Builds Klean and starts it at http://localhost:8787, opening the browser.
# Flags pass through, e.g. ./run.sh --port 9000 --no-browser
set -e
cd "$(dirname "$0")"
if ! command -v go >/dev/null 2>&1; then
  echo "Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/" >&2
  exit 1
fi
if ! command -v npm >/dev/null 2>&1; then
  echo "Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/" >&2
  exit 1
fi
(cd frontend && npm ci && npm run build)
go build -o klean ./backend
exec ./klean "$@"
