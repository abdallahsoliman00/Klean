#!/usr/bin/env sh
# Starts Switchboard at http://localhost:8787 (pass flags through, e.g. ./run.sh --port 9000)
set -e
cd "$(dirname "$0")/backend"
if ! command -v go >/dev/null 2>&1; then
  echo "Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/" >&2
  exit 1
fi
go build -o switchboard .
exec ./switchboard "$@"
