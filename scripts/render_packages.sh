#!/usr/bin/env bash
# Render Homebrew, Scoop and winget manifests for a release from its
# checksums.txt.
#
#   scripts/render_packages.sh <version> <dir-with-checksums.txt> <outdir>
#
# Output: <outdir>/homebrew/pc-monitor.rb, <outdir>/scoop/pc-monitor.json,
#         <outdir>/winget/manifests/g/garrettcorn/PCMonitor/<version>/*.yaml
set -euo pipefail

VERSION="$1"; DIST="$2"; OUT="$3"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SUMS="$DIST/checksums.txt"

sha() {
  local h
  h="$(awk -v f="$1" '{n=$2; sub(/^\*/, "", n)} n==f {print $1}' "$SUMS")"
  if [ -z "$h" ]; then echo "missing $1 in checksums.txt" >&2; exit 1; fi
  echo "$h"
}
upper() { echo "$1" | tr '[:lower:]' '[:upper:]'; }

V="$VERSION"
DA=$(sha "pc_monitor_${V}_darwin_arm64.tar.gz")
DI=$(sha "pc_monitor_${V}_darwin_amd64.tar.gz")
LA=$(sha "pc_monitor_${V}_linux_arm64.tar.gz")
LI=$(sha "pc_monitor_${V}_linux_amd64.tar.gz")
WI=$(sha "pc_monitor_${V}_windows_amd64.zip")
WA=$(sha "pc_monitor_${V}_windows_arm64.zip")
SI=$(sha "pc_monitor_${V}_windows_amd64_setup.exe")
SA=$(sha "pc_monitor_${V}_windows_arm64_setup.exe")
DATE=$(date -u +%Y-%m-%d)

render() { # template output
  mkdir -p "$(dirname "$2")"
  sed -e "s/{{VERSION}}/$V/g" -e "s/{{DATE}}/$DATE/g" \
      -e "s/{{SHA_DARWIN_ARM64}}/$DA/g" -e "s/{{SHA_DARWIN_AMD64}}/$DI/g" \
      -e "s/{{SHA_LINUX_ARM64}}/$LA/g" -e "s/{{SHA_LINUX_AMD64}}/$LI/g" \
      -e "s/{{SHA_WINDOWS_AMD64}}/$WI/g" -e "s/{{SHA_WINDOWS_ARM64}}/$WA/g" \
      -e "s/{{SHA_SETUP_AMD64_UPPER}}/$(upper "$SI")/g" -e "s/{{SHA_SETUP_ARM64_UPPER}}/$(upper "$SA")/g" \
      "$1" > "$2"
  if grep -q '{{' "$2"; then echo "unrendered placeholder in $2" >&2; exit 1; fi
}

render "$ROOT/packaging/homebrew/pc-monitor.rb.tmpl" "$OUT/homebrew/pc-monitor.rb"
render "$ROOT/packaging/scoop/pc-monitor.json.tmpl" "$OUT/scoop/pc-monitor.json"
WG="$OUT/winget/manifests/g/garrettcorn/PCMonitor/$V"
for f in garrettcorn.PCMonitor.yaml garrettcorn.PCMonitor.installer.yaml garrettcorn.PCMonitor.locale.en-US.yaml; do
  render "$ROOT/packaging/winget/$f.tmpl" "$WG/$f"
done
find "$OUT" -type f | sort
