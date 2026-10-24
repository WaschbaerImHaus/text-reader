#!/bin/bash
# Baut das macOS-App-Bundle "MD Reader.app" (Universal: Apple Silicon + Intel)
# und packt es als build/md-reader-macos.zip.
#
# Läuft NUR auf einem Mac (Xcode Command Line Tools + Go nötig), weil
# webview_go Cocoa/WebKit über CGo einbindet und das macOS-SDK dafür auf
# Linux nicht (lizenzkonform) verfügbar ist. Im Projekt baut der Workflow
# .github/workflows/macos.yml damit auf einem GitHub-Mac-Runner.
#
# Signatur: ad hoc (codesign -s -). Apple Silicon startet nur signierte
# Programme; eine Notarisierung (Apple-Entwicklerkonto) gibt es nicht,
# daher fragt Gatekeeper beim ersten Start nach (siehe README).
#
# Aufruf (aus dem Projektroot): installer/macos/build-app.sh
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-24
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BUILD_DIR="$ROOT/build"
APP_NAME="MD Reader"
APP="$BUILD_DIR/$APP_NAME.app"
BUILD_NUM="$(cat "$ROOT/src/build.txt")"
VERSION="1.0.$BUILD_NUM"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if [ "$(uname -s)" != "Darwin" ]; then
	echo "Fehler: Dieses Skript läuft nur unter macOS." >&2
	exit 1
fi

mkdir -p "$BUILD_DIR"

# 1. Binaries für beide Architekturen bauen und zu einem Universal-Binary vereinen
cd "$ROOT/src"
for arch in arm64 amd64; do
	echo "→ Kompiliere für macOS $arch..."
	CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" MACOSX_DEPLOYMENT_TARGET=12.0 \
		go build -ldflags="-s -w" -o "$WORK/md-reader-$arch" .
done
lipo -create -output "$WORK/md-reader" "$WORK/md-reader-arm64" "$WORK/md-reader-amd64"
lipo -info "$WORK/md-reader"

# 2. Icon (.icns) aus dem 256-px-PNG erzeugen
ICONSET="$WORK/AppIcon.iconset"
SRC_ICON="$ROOT/src/ui/assets/favicon-256.png"
mkdir -p "$ICONSET"
for size in 16 32 128 256; do
	sips -z "$size" "$size" "$SRC_ICON" --out "$ICONSET/icon_${size}x${size}.png" > /dev/null
done
sips -z 64 64 "$SRC_ICON" --out "$ICONSET/icon_32x32@2x.png" > /dev/null
sips -z 32 32 "$SRC_ICON" --out "$ICONSET/icon_16x16@2x.png" > /dev/null
cp "$SRC_ICON" "$ICONSET/icon_128x128@2x.png"
iconutil -c icns "$ICONSET" -o "$WORK/AppIcon.icns"

# 3. Bundle-Struktur anlegen
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$WORK/md-reader" "$APP/Contents/MacOS/md-reader"
cp "$WORK/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"
cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>$APP_NAME</string>
	<key>CFBundleDisplayName</key><string>$APP_NAME</string>
	<key>CFBundleIdentifier</key><string>io.github.waschbaerimhaus.md-reader</string>
	<key>CFBundleExecutable</key><string>md-reader</string>
	<key>CFBundleIconFile</key><string>AppIcon</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>$VERSION</string>
	<key>CFBundleVersion</key><string>$BUILD_NUM</string>
	<key>CFBundleDevelopmentRegion</key><string>de</string>
	<key>LSMinimumSystemVersion</key><string>12.0</string>
	<key>LSApplicationCategoryType</key><string>public.app-category.productivity</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSHumanReadableCopyright</key><string>Kurt Ingwer</string>
</dict>
</plist>
PLIST
plutil -lint "$APP/Contents/Info.plist"

# 4. Ad-hoc signieren (Pflicht für Apple Silicon) und prüfen
codesign --force --deep --sign - "$APP"
codesign --verify --verbose=2 "$APP"

# 5. Als ZIP packen (ditto erhält Signatur und Rechte)
rm -f "$BUILD_DIR/md-reader-macos.zip"
ditto -c -k --keepParent "$APP" "$BUILD_DIR/md-reader-macos.zip"
echo "✓ $BUILD_DIR/md-reader-macos.zip ($VERSION)"
