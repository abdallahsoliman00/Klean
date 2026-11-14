#!/usr/bin/env sh
# Builds a single-file Klean for each platform into dist/.
# Each binary carries the UI inside it; copy it anywhere and double-click it.
set -e
cd "$(dirname "$0")/.."
# the UI is built once and embedded into every binary
(cd frontend && npm ci && npm run build)
rm -rf dist && mkdir dist
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  os=${target%/*}; arch=${target#*/}
  out="dist/klean-$os-$arch"; [ "$os" = windows ] && out="$out.exe"
  echo "building $out"
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "$out" ./backend
done
