#!/usr/bin/env bash
# Builds the release CLI binaries into dist/ and the wasm module plus its
# matching Go loader into npm/.
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p dist
for os in darwin linux windows; do
  for arch in amd64 arm64; do
    ext=
    if [[ $os == windows ]]; then ext=.exe; fi
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags='-s -w' \
      -o "dist/ddi-xlsx-$os-$arch$ext" ./cmd/ddi-xlsx
  done
done

GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o npm/ddi-xlsx.wasm ./wasm
# wasm_exec.js must come from the toolchain that built the wasm: its import
# object changes between Go releases.
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" npm/wasm_exec.js
