#!/usr/bin/env bash
# Build homestack for multiple OS/arch. Outputs go to application/ with names like:
#   homestack-darwin-arm64   (Apple Silicon Mac)
#   homestack-darwin-amd64   (Intel Mac)
#   homestack-linux-amd64
#   homestack-linux-arm64
#   homestack-windows-amd64.exe
set -e
cd "$(dirname "$0")/.."
mkdir -p application

build() {
  local goos=$1
  local goarch=$2
  local suffix=$3
  echo "Building $goos/$goarch -> application/homestack-$suffix"
  GOOS=$goos GOARCH=$goarch go build -o "application/homestack-$suffix" ./cmd/homestack
}

# macOS (Apple Silicon + Intel)
build darwin arm64 darwin-arm64
build darwin amd64 darwin-amd64

# Linux
build linux amd64 linux-amd64
build linux arm64 linux-arm64

# Windows
build windows amd64 windows-amd64.exe
build windows arm64 windows-arm64.exe

echo "Done. Binaries in application/:"
ls -la application/
