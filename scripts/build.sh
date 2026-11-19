#!/usr/bin/env bash
set -e

echo "=========================================="
echo " Building PC Monitor Server (Cross-Platform)"
echo "=========================================="

mkdir -p dist

# 1. Native Build
echo "--> Building native binary..."
go build -ldflags="-s -w" -o dist/pc_monitor_server ./cmd/server
cp dist/pc_monitor_server ./pc_monitor_server

# 2. Linux AMD64 & ARM64
echo "--> Cross-compiling Linux (amd64, arm64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/pc_monitor_linux_amd64 ./cmd/server
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/pc_monitor_linux_arm64 ./cmd/server

# 3. macOS Apple Silicon & Intel
echo "--> Cross-compiling macOS (arm64, amd64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/pc_monitor_darwin_arm64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/pc_monitor_darwin_amd64 ./cmd/server

# 4. Windows AMD64 & ARM64
echo "--> Cross-compiling Windows (amd64, arm64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/pc_monitor_windows_amd64.exe ./cmd/server
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/pc_monitor_windows_arm64.exe ./cmd/server

echo ""
echo "Build complete! All cross-platform binaries ready in ./dist/"
