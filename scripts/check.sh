#!/usr/bin/env sh
# Checks the whole repo the way CI does: formatting, lint, types, build and tests.
#
#   ./scripts/check.sh          report problems and exit non-zero
#   ./scripts/check.sh --fix    reformat and auto-fix what can be fixed, then check
#
# Run it before committing, and AI agents should run it before saying they're done.
# Needs Go 1.26+ (https://go.dev/dl/) and Node.js 24+ (https://nodejs.org/).
set -eu
cd "$(dirname "$0")/.."

fix=0
case "${1:-}" in
  --fix) fix=1 ;;
  "") ;;
  *) echo "usage: $0 [--fix]" >&2; exit 2 ;;
esac

step() { printf '\n==> %s\n' "$1"; }
die() { printf '\n%s\n' "$1" >&2; exit 1; }

command -v go >/dev/null 2>&1 || die "Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/"
command -v npm >/dev/null 2>&1 || die "Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/"

# The directory of every Go package, so gofmt only sees Go code (not node_modules or dist).
go_dirs() { go list -f '{{.Dir}}' ./...; }

step "Go formatting (gofmt)"
if [ "$fix" = 1 ]; then
  go_dirs | xargs gofmt -w
else
  unformatted=$(go_dirs | xargs gofmt -l)
  [ -z "$unformatted" ] || die "Not gofmt'd; run ./scripts/check.sh --fix:
$unformatted"
fi

if [ ! -d frontend/node_modules ]; then
  step "Installing frontend dependencies (npm ci)"
  (cd frontend && npm ci)
fi

step "Frontend lint (oxlint)"
if [ "$fix" = 1 ]; then
  (cd frontend && npm run lint -- --fix)
else
  (cd frontend && npm run lint)
fi

step "Frontend type check (tsgo)"
(cd frontend && npm run typecheck)

# go vet and go test compile the embedded UI, so frontend/dist must exist first.
step "Frontend build (vite)"
(cd frontend && npm run build)

step "Go vet"
go vet ./...

step "Go tests"
go test ./...

printf '\nAll checks passed.\n'
