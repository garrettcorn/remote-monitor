# Windows PowerShell Build Script for PC Monitor Server
param(
    [switch]$AllPlatforms = $false
)

$ErrorActionPreference = "Stop"
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Building PC Monitor Server" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

New-Item -ItemType Directory -Force -Path "dist" | Out-Null

# 1. Native Windows Build
Write-Host "--> Building windows/amd64..." -ForegroundColor Green
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/pc_monitor_windows_amd64.exe ./cmd/server
Copy-Item dist/pc_monitor_windows_amd64.exe pc_monitor_server.exe -Force

if ($AllPlatforms) {
    # 2. Windows ARM64 (Snapdragon X Elite, Surface Pro)
    Write-Host "--> Cross-compiling windows/arm64..." -ForegroundColor Green
    $env:GOOS = "windows"
    $env:GOARCH = "arm64"
    go build -ldflags="-s -w" -o dist/pc_monitor_windows_arm64.exe ./cmd/server

    # 3. Linux AMD64
    Write-Host "--> Cross-compiling linux/amd64..." -ForegroundColor Green
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -ldflags="-s -w" -o dist/pc_monitor_linux_amd64 ./cmd/server

    # 4. Linux ARM64 (Raspberry Pi, Linux Cloud, Android Termux)
    Write-Host "--> Cross-compiling linux/arm64..." -ForegroundColor Green
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    go build -ldflags="-s -w" -o dist/pc_monitor_linux_arm64 ./cmd/server

    # 5. macOS Apple Silicon (M1/M2/M3/M4)
    Write-Host "--> Cross-compiling darwin/arm64..." -ForegroundColor Green
    $env:GOOS = "darwin"
    $env:GOARCH = "arm64"
    go build -ldflags="-s -w" -o dist/pc_monitor_darwin_arm64 ./cmd/server

    # 6. macOS Intel
    Write-Host "--> Cross-compiling darwin/amd64..." -ForegroundColor Green
    $env:GOOS = "darwin"
    $env:GOARCH = "amd64"
    go build -ldflags="-s -w" -o dist/pc_monitor_darwin_amd64 ./cmd/server
}

Write-Host "`nBuild complete! Binaries located in ./dist/" -ForegroundColor Cyan
