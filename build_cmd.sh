#!/usr/bin/env bash
set -euo pipefail

# One-click build for all cmd targets.
mkdir -p dist

export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

echo "Building cmd/crawler..."
go build -ldflags="-s -w" -o dist/crawler_linux_amd64 ./cmd/crawler

echo "Building cmd/fills..."
go build -ldflags="-s -w" -o dist/fills_linux_amd64 ./cmd/fills

echo "Building cmd/snapshot..."
go build -ldflags="-s -w" -o dist/snapshot_linux_amd64 ./cmd/snapshot

echo "compilation succeeded, generated binaries in dist/."
