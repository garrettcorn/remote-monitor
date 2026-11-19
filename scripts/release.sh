#!/usr/bin/env bash
set -e

if [ -z "$1" ]; then
  echo "Usage: ./scripts/release.sh <version> [--push]"
  echo "Example: ./scripts/release.sh 1.0.0 --push"
  exit 1
fi

VERSION="${1#v}"
TAG="v${VERSION}"
PUSH=false

if [ "$2" == "--push" ]; then
  PUSH=true
fi

echo "=========================================="
echo " PC Monitor Release Workflow: ${TAG}"
echo "=========================================="

# 1. Validate Common Changelog
echo "--> Validating CHANGELOG.md (common-changelog.org)..."
go run ./cmd/changelog validate

# 2. Verify version exists in CHANGELOG.md
echo "--> Verifying release notes exist for [${VERSION}]..."
go run ./cmd/changelog extract "${VERSION}" > /dev/null

echo "--> Preview of Release Notes:"
go run ./cmd/changelog extract "${VERSION}"
echo ""

# 3. Run Tests
echo "--> Running test suite..."
go test -v ./...

# 4. Create Git Tag
echo "--> Tagging ${TAG}..."
git tag -a "${TAG}" -m "Release ${TAG}"
echo "✓ Tag ${TAG} created!"

if [ "$PUSH" = true ]; then
  echo "--> Pushing commits and tag to origin..."
  git push origin main
  git push origin "${TAG}"
  echo "✓ Pushed! GitHub Actions release pipeline has been triggered."
else
  echo ""
  echo "To trigger the automated GitHub Actions release, run:"
  echo "  git push origin main"
  echo "  git push origin ${TAG}"
fi
