#!/usr/bin/env bash
# Baut das Debian-Paket (.deb) von MD Reader.
#
# Aufruf: build-deb.sh <arch> <gui-datei> <build-nummer>
#   <arch>          amd64 | arm64 | armhf
#   <gui-datei>     reine GUI-Datei (ohne Starter), z. B. build/gui/md-reader-amd64
#   <build-nummer>  Build-Nummer, ergibt die Version 1.0.<build-nummer>
#
# Ergebnis: build/md-reader_1.0.<build-nummer>_<arch>.deb (im Projektwurzelverzeichnis)
#
# Inhalt (7 Dateien):
#   usr/bin/md-reader                                  die GUI (0755)
#   usr/share/applications/md-reader.desktop           vom Starter erzeugt
#   usr/share/icons/hicolor/{16x16,32x32,48x48,256x256}/apps/md-reader.png
#   usr/share/icons/hicolor/scalable/apps/md-reader.svg
#
# Die Abhängigkeiten werden NICHT mit dpkg-shlibdeps ermittelt: Auf der
# Build-Maschine (Ubuntu 25.10) würde es die neuen Versionen von libwebkit2gtk
# eintragen, und das Paket wäre auf Ubuntu 24.04 nicht installierbar. Fest
# stehen libwebkit2gtk-4.1-0 und libgtk-3-0 im Skript. Nur die Untergrenze von
# libc6 wird aus der höchsten GLIBC-Symbolversion der GUI-Datei abgeleitet
# (check-glibc.sh --max), heute 2.38.
#
# Die Desktop-Datei kommt aus dem Starter (--print-desktop-entry), damit es nur
# eine einzige Vorlage gibt. Die Build-Nummer wird dabei mit -ldflags gesetzt,
# sonst stünde X-MD-Reader-Build=0 in der Datei.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-28 (Fix-Runde 1: GUI-Datei prüfen, libc6-Grenze aus der GUI)
set -euo pipefail

# Aufruf prüfen
if [ "$#" -ne 3 ]; then
	echo "Aufruf: $0 <amd64|arm64|armhf> <gui-datei> <build-nummer>" >&2
	exit 2
fi

arch="$1"
gui_file="$2"
build_num="$3"

case "${arch}" in
	amd64|arm64|armhf) ;;
	*)
		echo "Unbekannte Architektur: ${arch} (erlaubt: amd64, arm64, armhf)" >&2
		exit 2
		;;
esac

# Build-Nummer wird in Dateinamen, Version und ldflags verwendet: nur Ziffern
case "${build_num}" in
	''|*[!0-9]*)
		echo "Ungültige Build-Nummer: ${build_num}" >&2
		exit 2
		;;
esac

if [ ! -f "${gui_file}" ]; then
	echo "GUI-Datei fehlt: ${gui_file}" >&2
	exit 2
fi

# GUI-Datei prüfen: ELF-Datei, passende Architektur, dynamisch gelinkt.
# Schützt davor, den statischen Starter oder eine Datei der falschen
# Architektur in ein Paket zu packen.
magic="$(od -An -tx1 -N4 -- "${gui_file}" | tr -d ' \n')"
if [ "${magic}" != "7f454c46" ]; then
	echo "FEHLER: ${gui_file} ist keine ELF-Datei" >&2
	exit 2
fi
# Byte 18 des ELF-Kopfs (e_machine): 0x3e = x86-64, 0xb7 = AArch64, 0x28 = ARM
case "${arch}" in
	amd64) want_machine="3e" ;;
	arm64) want_machine="b7" ;;
	armhf) want_machine="28" ;;
esac
machine="$(od -An -tx1 -j18 -N1 -- "${gui_file}" | tr -d ' \n')"
if [ "${machine}" != "${want_machine}" ]; then
	echo "FEHLER: ${gui_file} passt nicht zur Architektur ${arch} (e_machine 0x${machine}, erwartet 0x${want_machine})" >&2
	exit 2
fi
# Die GUI braucht einen Interpreter (dynamisch gelinkt); der Starter ist statisch
if ! readelf -lW -- "${gui_file}" | grep -q 'INTERP'; then
	echo "FEHLER: ${gui_file} ist statisch gelinkt – erwartet wird die dynamische GUI, nicht der Starter" >&2
	exit 2
fi

# Projektwurzel = zwei Ebenen über diesem Skript (installer/linux/)
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
version="1.0.${build_num}"
output="${root}/build/md-reader_${version}_${arch}.deb"
icon_dir="${root}/src/ui/assets"

# Untergrenze für libc6 = höchste GLIBC-Version, die die GUI wirklich braucht
libc_min="$("${root}/installer/linux/check-glibc.sh" --max "${gui_file}")"
if [ -z "${libc_min}" ]; then
	echo "FEHLER: keine GLIBC-Version in ${gui_file} gefunden" >&2
	exit 1
fi

# Gleiche Dateirechte unabhängig von der umask des Aufrufers
umask 022

# Staging-Verzeichnis im Scratch; wird am Ende immer gelöscht
staging="$(mktemp -d)"
trap 'rm -rf -- "${staging}"' EXIT
# mktemp legt 0700 an; das Wurzelverzeichnis des Pakets muss 0755 sein
chmod 0755 -- "${staging}"

# 1. Programm: die reine GUI
install -D -m 0755 -- "${gui_file}" "${staging}/usr/bin/md-reader"

# 2. Desktop-Datei aus dem Starter (CGO aus, läuft auf der Build-Architektur)
mkdir -p -- "${staging}/usr/share/applications"
(cd "${root}/src" && CGO_ENABLED=0 go run -ldflags "-X main.buildNumber=${build_num}" \
	./launcher --print-desktop-entry /usr/bin/md-reader) \
	> "${staging}/usr/share/applications/md-reader.desktop"
chmod 0644 -- "${staging}/usr/share/applications/md-reader.desktop"

# 3. Icons
for size in 16 32 48 256; do
	install -D -m 0644 -- "${icon_dir}/favicon-${size}.png" \
		"${staging}/usr/share/icons/hicolor/${size}x${size}/apps/md-reader.png"
done
install -D -m 0644 -- "${icon_dir}/favicon.svg" \
	"${staging}/usr/share/icons/hicolor/scalable/apps/md-reader.svg"

# 4. Steuerdatei. Installed-Size in KiB (nur Nutzdaten, ohne DEBIAN)
installed_size="$(du -sk -- "${staging}" | cut -f1)"
mkdir -p -- "${staging}/DEBIAN"
chmod 0755 -- "${staging}/DEBIAN"
cat > "${staging}/DEBIAN/control" <<CONTROL
Package: md-reader
Version: ${version}
Architecture: ${arch}
Maintainer: Kurt Ingwer <WaschbaerImHaus@users.noreply.github.com>
Installed-Size: ${installed_size}
Depends: libc6 (>= ${libc_min}), libwebkit2gtk-4.1-0, libgtk-3-0
Recommends: zenity, poppler-utils, ghostscript
Section: text
Priority: optional
Homepage: https://github.com/WaschbaerImHaus/text-reader
Description: Document viewer for Markdown, EPUB, FB2, LaTeX, PDF and PostScript
 MD Reader shows documents with a GitHub-like look. It opens Markdown,
 plain text, FB2, EPUB, LaTeX, HTML, PDF and PostScript files, with themes,
 zoom, search, a table of contents and a full screen mode.
 .
 PDF files need poppler-utils, PostScript files need ghostscript, and the
 file dialog uses zenity or kdialog.
CONTROL
chmod 0644 -- "${staging}/DEBIAN/control"

# 5. Paket bauen: Eigentümer root:root ohne echte Root-Rechte (fakeroot)
mkdir -p -- "${root}/build"
fakeroot dpkg-deb --build --root-owner-group "${staging}" "${output}"
echo "OK: ${output}"
