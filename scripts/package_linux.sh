#!/usr/bin/env bash
# Build Linux packages from a prebuilt binary.
#
#   scripts/package_linux.sh <version> <amd64|arm64> <binary> <outdir> deb|rpm|appimage...
#
# deb: needs dpkg-deb. rpm: needs rpmbuild (apt install rpm).
# appimage: downloads appimagetool (continuous) for the host architecture;
#           needs zsync (apt install zsync) for the .zsync update file.
set -euo pipefail

VERSION="$1"; ARCH="$2"; BIN="$3"; OUT="$4"; shift 4
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

case "$ARCH" in
  amd64) RPMARCH=x86_64;  AIARCH=x86_64 ;;
  arm64) RPMARCH=aarch64; AIARCH=aarch64 ;;
  *) echo "unknown arch $ARCH" >&2; exit 1 ;;
esac

ICON="$WORK/pc-monitor.png"
(cd "$ROOT" && go run ./cmd/pkgtool icon-png 256 "$ICON")
DESKTOP="$ROOT/packaging/linux/pc-monitor.desktop"

for kind in "$@"; do
  case "$kind" in
  deb)
    PKG="$WORK/deb/pc-monitor_${VERSION}_${ARCH}"
    mkdir -p "$PKG/DEBIAN" "$PKG/usr/bin" "$PKG/usr/share/applications" \
             "$PKG/usr/share/icons/hicolor/256x256/apps" "$PKG/usr/share/doc/pc-monitor"
    install -m 0755 "$BIN" "$PKG/usr/bin/pc_monitor"
    install -m 0644 "$DESKTOP" "$PKG/usr/share/applications/pc-monitor.desktop"
    install -m 0644 "$ICON" "$PKG/usr/share/icons/hicolor/256x256/apps/pc-monitor.png"
    install -m 0644 "$ROOT/README.md" "$ROOT/CHANGELOG.md" "$PKG/usr/share/doc/pc-monitor/"
    cat > "$PKG/DEBIAN/control" <<EOF
Package: pc-monitor
Version: ${VERSION}
Architecture: ${ARCH}
Maintainer: garrettcorn <https://github.com/garrettcorn/remote-monitor>
Section: utils
Priority: optional
Homepage: https://github.com/garrettcorn/remote-monitor
Description: Monitor and control this computer from the PC Monitor dashboard
 Lightweight server that streams system telemetry to the PC Monitor web
 dashboard and, after PIN pairing, allows remote control.
EOF
    dpkg-deb --build --root-owner-group "$PKG" "$OUT/pc-monitor_${VERSION}_${ARCH}.deb"
    ;;
  rpm)
    SRC="$WORK/rpmsrc"
    mkdir -p "$SRC" "$WORK/rpmbuild"
    install -m 0755 "$BIN" "$SRC/pc_monitor"
    cp "$DESKTOP" "$ICON" "$SRC/"
    rpmbuild -bb --target "$RPMARCH" \
      --define "_topdir $WORK/rpmbuild" --define "pkgversion $VERSION" --define "srcdir $SRC" \
      "$ROOT/packaging/linux/pc-monitor.spec"
    find "$WORK/rpmbuild/RPMS" -name '*.rpm' -exec cp {} "$OUT/" \;
    ;;
  appimage)
    APPDIR="$WORK/PC_Monitor.AppDir"
    mkdir -p "$APPDIR/usr/bin"
    install -m 0755 "$BIN" "$APPDIR/usr/bin/pc_monitor"
    install -m 0755 "$ROOT/packaging/appimage/AppRun" "$APPDIR/AppRun"
    cp "$DESKTOP" "$APPDIR/pc-monitor.desktop"
    cp "$ICON" "$APPDIR/pc-monitor.png"
    ln -sf pc-monitor.png "$APPDIR/.DirIcon"
    TOOL="$WORK/appimagetool"
    HOSTARCH="$(uname -m)"
    curl -fsSL -o "$TOOL" "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-${HOSTARCH}.AppImage"
    chmod +x "$TOOL"
    NAME="pc_monitor-${VERSION}-${AIARCH}.AppImage"
    UPDATE="gh-releases-zsync|garrettcorn|remote-monitor|latest|pc_monitor-*-${AIARCH}.AppImage.zsync"
    (cd "$OUT" && ARCH="$AIARCH" "$TOOL" --appimage-extract-and-run -u "$UPDATE" "$APPDIR" "$NAME")
    # appimagetool writes <name>.zsync when update info is given; make sure it exists.
    if [ ! -f "$OUT/$NAME.zsync" ]; then
      (cd "$OUT" && zsyncmake -u "$NAME" -o "$NAME.zsync" "$NAME")
    fi
    ;;
  *) echo "unknown package kind $kind" >&2; exit 1 ;;
  esac
done
ls -la "$OUT"
