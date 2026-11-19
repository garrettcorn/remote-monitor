# Release Helper Script for PC Monitor (Windows PowerShell)
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Version,

    [switch]$SkipTests = $false,
    [switch]$Push = $false
)

$ErrorActionPreference = "Stop"
$cleanVer = $Version.TrimStart("v")
$tag = "v$cleanVer"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " PC Monitor Release Workflow: $tag" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Validate Common Changelog
Write-Host "--> Validating CHANGELOG.md (common-changelog.org)..." -ForegroundColor Yellow
go run ./cmd/changelog validate

# 2. Verify version exists in CHANGELOG.md
Write-Host "--> Verifying release notes exist for [$cleanVer]..." -ForegroundColor Yellow
$notes = go run ./cmd/changelog extract $cleanVer
if (-not $notes) {
    Write-Host "Error: No entry for [$cleanVer] found in CHANGELOG.md." -ForegroundColor Red
    Write-Host "Run: go run ./cmd/changelog bump $cleanVer" -ForegroundColor Red
    exit 1
}

Write-Host "`nRelease Notes Preview:" -ForegroundColor Green
Write-Host $notes

# 3. Run Tests
if (-not $SkipTests) {
    Write-Host "--> Running test suite..." -ForegroundColor Yellow
    go test -v ./...
}

# 4. Git Tagging
Write-Host "--> Creating Git tag: $tag..." -ForegroundColor Yellow
git tag -a $tag -m "Release $tag"

Write-Host "`nTag $tag created successfully!" -ForegroundColor Green

if ($Push) {
    Write-Host "--> Pushing commit and tag to origin..." -ForegroundColor Yellow
    git push origin main
    git push origin $tag
    Write-Host "Pushed to GitHub! Release pipeline triggered." -ForegroundColor Green
} else {
    Write-Host "To trigger the GitHub Actions release workflow, run:" -ForegroundColor Cyan
    Write-Host "  git push origin main" -ForegroundColor White
    Write-Host "  git push origin $tag`n" -ForegroundColor White
}
