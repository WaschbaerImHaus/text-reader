#!/usr/bin/env bash
# Abnahme des Linux-Starters, der Selbst-Installation und des .deb (Build 47).
#
# Teil A (Docker, Image ubuntu:24.04 = Basis von KDE neon):
#   1. Starter ohne WebKit: Exit 1, Hinweis auf stderr (englisch).
#   2. Starter ohne WebKit mit LANG=de_DE.UTF-8: deutsche Meldung.
#   3. --install / --uninstall in HOME=/tmp/h, desktop-file-validate.
#   4. .deb installieren (zieht WebKit mit), Paket-GUI unter Xvfb,
#      WM_CLASS, Bildschirmfoto.
#   5. Starter mit installiertem WebKit: Fenster, keine Installationsfrage;
#      Gegenprobe ohne Paket-Desktop-Datei (dann muss die Frage kommen).
# Teil B (LXC unter eigenem Xvfb):
#   7. Kopie des Starters mit leerem HOME: WM_CLASS, zenity-Frage,
#      Bildschirmfoto, "Ja" per Return, danach Menüeintrag vorhanden;
#      zweiter Start ohne Frage.
# Teil C (nur Beobachtung, bewertet nicht):
#   8. Starter auf ubuntu:22.04 (glibc 2.35) ohne und mit WebKit.
#   9. arm64-Starter unter qemu-user (nur wenn qemu-aarch64 vorhanden).
# Zusätzlich (Teil A, 6.): Kopfdaten und Inhalt aller drei .deb-Pakete.
#
# Aufruf (aus beliebigem Verzeichnis):
#   debugging/linux/launcher-docker-test.sh [AUSGABEVERZEICHNIS]
# Umgebung:
#   SKIP_DOCKER=1   Teil A und C überspringen
#   SKIP_LXC=1      Teil B überspringen
#   LXC_DISPLAY=87  Nummer des eigenen Xvfb-Displays im LXC
#
# build/ wird nur lesend in die Container eingehängt; Protokolle und
# Bildschirmfotos landen im Ausgabeverzeichnis (Standard: mktemp -d).
# Gibt je Prüfung PASS/FAIL aus und endet mit Exit 1, wenn eine scheitert.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-28 (Abnahme Aufgabe 9)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$PROJECT_DIR/build"
OUT_DIR="${1:-$(mktemp -d)}"
mkdir -p "$OUT_DIR/shots"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"
IMAGE="ubuntu:24.04"
OLD_IMAGE="ubuntu:22.04"
BUILD_NUM="$(tr -d '[:space:]' <"$PROJECT_DIR/src/build.txt")"
LXC_DISPLAY="${LXC_DISPLAY:-87}"

# Anzahl gescheiterter Prüfungen.
FAILS=0
# PIDs eigener Hintergrundprozesse im LXC (werden beim Ende beendet).
OWN_PIDS=()

# pass/fail/info geben das Ergebnis bzw. eine Beobachtung aus.
pass() {
	echo "PASS $1"
}
fail() {
	echo "FAIL $1"
	FAILS=$((FAILS + 1))
}
info() {
	echo "INFO $*"
}

# check NAME BEFEHL… meldet PASS bei Exit 0 von BEFEHL, sonst FAIL.
check() {
	local name="$1"
	shift
	if "$@"; then
		pass "$name"
	else
		fail "$name"
	fi
}

# cleanup beendet nur die eigenen Prozesse (per PID, nie per Muster).
cleanup() {
	local pid
	for pid in "${OWN_PIDS[@]+"${OWN_PIDS[@]}"}"; do
		kill "$pid" 2>/dev/null || true
	done
}
trap cleanup EXIT

# run_stderr BEFEHL… führt BEFEHL aus, schreibt stderr nach $RUN_ERR und den
# Exit-Code nach $RUN_CODE (bricht wegen set -e nicht ab).
run_stderr() {
	RUN_ERR="$(mktemp -p "$OUT_DIR")"
	set +e
	"$@" >/dev/null 2>"$RUN_ERR"
	RUN_CODE=$?
	set -e
}

# docker_inner MODUS startet launcher-docker-inner.sh im Container und zählt
# dessen FAIL-Zeilen mit.
docker_inner() {
	local mode="$1" log="$OUT_DIR/inner-$1.log" code
	set +e
	docker run --rm \
		-v "$BUILD_DIR:/b:ro" \
		-v "$PROJECT_DIR/README.md:/doc/README.md:ro" \
		-v "$SCRIPT_DIR/launcher-docker-inner.sh:/inner.sh:ro" \
		-v "$OUT_DIR/$mode:/out" \
		"$IMAGE" bash /inner.sh "$mode" >"$log" 2>&1
	code=$?
	set -e
	cat "$log"
	local inner_fails
	inner_fails="$(grep -c '^FAIL ' "$log" || true)"
	FAILS=$((FAILS + inner_fails))
	if [ "$code" -ne 0 ] && [ "$inner_fails" -eq 0 ]; then
		fail "Container-Modus $mode endete mit Exit $code ohne FAIL-Zeile"
	fi
}

# ---------------------------------------------------------------------------
# Teil A: Docker
# ---------------------------------------------------------------------------
part_docker() {
	echo "== Teil A: Docker ($IMAGE) =="
	docker pull -q "$IMAGE" >/dev/null

	# 1. Ohne WebKit (kein Netz nötig), englisch.
	run_stderr docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$IMAGE" /b/md-reader
	cp "$RUN_ERR" "$OUT_DIR/1-nowebkit-en.err"
	info "1: Exit $RUN_CODE, stderr: $(tr '\n' '|' <"$RUN_ERR")"
	check "1.1 ohne WebKit: Exit 1" test "$RUN_CODE" -eq 1
	check "1.2 stderr enthält WebKitGTK" grep -q 'WebKitGTK' "$RUN_ERR"
	check "1.3 stderr enthält libwebkit2gtk-4.1.so.0" grep -q 'libwebkit2gtk-4.1.so.0' "$RUN_ERR"
	check "1.4 stderr enthält 'sudo apt install libwebkit2gtk-4.1-0'" \
		grep -q 'sudo apt install libwebkit2gtk-4.1-0' "$RUN_ERR"
	check "1.5 englische Meldung (Missing libraries:)" grep -q '^Missing libraries: ' "$RUN_ERR"
	# Bekannter Schönheitsfehler: Titel und Hinweis erscheinen zweimal
	# (writeTerminalHint + stderr-Asker). Nur Warnung, siehe Bericht.
	if [ "$(grep -c 'WebKitGTK is not installed' "$RUN_ERR")" -ne 1 ]; then
		info "WARN 1.6 Hinweis steht $(grep -c 'WebKitGTK is not installed' "$RUN_ERR")-mal auf stderr"
	fi

	# 2. Ohne WebKit, deutsch.
	run_stderr docker run --rm --network none -e LANG=de_DE.UTF-8 -v "$BUILD_DIR:/b:ro" "$IMAGE" /b/md-reader
	cp "$RUN_ERR" "$OUT_DIR/2-nowebkit-de.err"
	info "2: Exit $RUN_CODE, stderr: $(tr '\n' '|' <"$RUN_ERR")"
	check "2.1 deutsch: Exit 1" test "$RUN_CODE" -eq 1
	check "2.2 deutsch: 'WebKitGTK ist nicht installiert'" grep -q 'WebKitGTK ist nicht installiert' "$RUN_ERR"
	check "2.3 deutsch: 'Fehlende Bibliotheken:' mit libwebkit2gtk-4.1.so.0" \
		grep -q '^Fehlende Bibliotheken: .*libwebkit2gtk-4.1.so.0' "$RUN_ERR"
	check "2.4 deutsch: Befehl 'sudo apt install libwebkit2gtk-4.1-0'" \
		grep -q 'sudo apt install libwebkit2gtk-4.1-0' "$RUN_ERR"

	# Versionsabfrage als Rauchtest.
	check "2.5 --launcher-version = $BUILD_NUM" \
		test "$(docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$IMAGE" /b/md-reader --launcher-version)" = "$BUILD_NUM"

	# 3. --install / --uninstall (braucht Netz für desktop-file-utils).
	mkdir -p "$OUT_DIR/install"
	docker_inner install

	# 4. + 5. Paket und Starter mit WebKit.
	mkdir -p "$OUT_DIR/deb"
	docker_inner deb
	local xwd
	for xwd in deb launcher-deb launcher-nodeb; do
		if [ -s "$OUT_DIR/deb/$xwd.xwd" ]; then
			convert "xwd:$OUT_DIR/deb/$xwd.xwd" "$OUT_DIR/shots/$xwd.png"
			info "Bildschirmfoto: $OUT_DIR/shots/$xwd.png"
		fi
	done

	# 6. Kopfdaten und Inhalt aller .deb-Pakete (ohne Container).
	local arch deb
	for arch in amd64 arm64 armhf; do
		deb="$BUILD_DIR/md-reader_1.0.${BUILD_NUM}_${arch}.deb"
		check "6.$arch Kopf: Architecture/Version/Depends" bash -c "
			f=\$(dpkg-deb -f '$deb' Package Version Architecture Depends Recommends Maintainer)
			grep -qx 'Package: md-reader' <<<\"\$f\" &&
			grep -qx 'Version: 1.0.$BUILD_NUM' <<<\"\$f\" &&
			grep -qx 'Architecture: $arch' <<<\"\$f\" &&
			grep -qx 'Depends: libc6 (>= 2.38), libwebkit2gtk-4.1-0, libgtk-3-0' <<<\"\$f\" &&
			grep -qx 'Recommends: zenity, poppler-utils, ghostscript' <<<\"\$f\" &&
			grep -qx 'Maintainer: Kurt Ingwer <WaschbaerImHaus@users.noreply.github.com>' <<<\"\$f\""
		check "6.$arch Inhalt: Programm, Desktop-Datei, 5 Icons" bash -c "
			c=\$(dpkg-deb -c '$deb' | awk '{print \$NF}')
			grep -qx './usr/bin/md-reader' <<<\"\$c\" &&
			grep -qx './usr/share/applications/md-reader.desktop' <<<\"\$c\" &&
			test \$(grep -c '^./usr/share/icons/hicolor/.*/apps/md-reader\.\(png\|svg\)$' <<<\"\$c\") -eq 5"
	done
}

# ---------------------------------------------------------------------------
# Teil B: LXC unter eigenem Xvfb, zenity-Frage
# ---------------------------------------------------------------------------

# wait_window ARGS… wartet bis zu 20 s auf ein sichtbares Fenster
# (xdotool search ARGS) und gibt die erste ID aus.
wait_window() {
	local i id
	for i in $(seq 1 40); do
		id="$(xdotool search --onlyvisible "$@" 2>/dev/null | head -n 1 || true)"
		if [ -n "$id" ]; then
			echo "$id"
			return 0
		fi
		sleep 0.5
	done
	return 0
}

part_lxc() {
	echo "== Teil B: LXC unter Xvfb :$LXC_DISPLAY =="
	if ! command -v zenity >/dev/null; then
		fail "7.0 zenity fehlt (sudo apt-get install -y zenity)"
		return
	fi
	if [ -e "/tmp/.X11-unix/X$LXC_DISPLAY" ] || [ -e "/tmp/.X$LXC_DISPLAY-lock" ]; then
		fail "7.0 Display :$LXC_DISPLAY ist schon belegt (LXC_DISPLAY anders setzen)"
		return
	fi

	local home="$OUT_DIR/lxc-home" run="$OUT_DIR/lxc-run" exe="$OUT_DIR/md-reader-test"
	local win zwin pid wmclass app_pid i
	rm -rf "$home" "$run"
	mkdir -p "$home" "$run"
	chmod 700 "$run"
	cp "$BUILD_DIR/md-reader" "$exe"

	Xvfb ":$LXC_DISPLAY" -screen 0 1280x800x24 -nolisten tcp >"$OUT_DIR/lxc-xvfb.log" 2>&1 &
	OWN_PIDS+=("$!")
	sleep 2

	# Abgeschottete Umgebung: leeres HOME, kein Session-Bus (D-Bus-Weg aus),
	# deshalb fragt der Starter über zenity.
	local -a envargs=(
		env -u DBUS_SESSION_BUS_ADDRESS -u XDG_CONFIG_HOME -u XDG_DATA_HOME -u XDG_CACHE_HOME
		-u WAYLAND_DISPLAY -u LC_ALL -u LC_MESSAGES
		HOME="$home" XDG_RUNTIME_DIR="$run" DISPLAY=":$LXC_DISPLAY" LANG=de_DE.UTF-8
	)

	"${envargs[@]}" "$exe" "$PROJECT_DIR/README.md" >"$OUT_DIR/lxc-1.out" 2>"$OUT_DIR/lxc-1.err" &
	app_pid=$!
	OWN_PIDS+=("$app_pid")
	export DISPLAY=":$LXC_DISPLAY"

	win="$(wait_window --class md-reader)"
	check "7.1 Starter-Kopie öffnet Fenster" test -n "$win"
	if [ -n "$win" ]; then
		wmclass="$(xprop -id "$win" WM_CLASS)"
		info "LXC-Fenster: $wmclass"
		check "7.2 WM_CLASS = \"md-reader\", \"md-reader\"" \
			test "$wmclass" = 'WM_CLASS(STRING) = "md-reader", "md-reader"'
	else
		fail "7.2 WM_CLASS (kein Fenster)"
	fi

	zwin="$(wait_window --name 'MD Reader ins Menü aufnehmen')"
	sleep 1
	import -display ":$LXC_DISPLAY" -window root "$OUT_DIR/shots/lxc-offer.png"
	info "Bildschirmfoto: $OUT_DIR/shots/lxc-offer.png"
	check "7.3 zenity-Installationsfrage erscheint" test -n "$zwin"

	if [ -n "$zwin" ]; then
		xdotool windowfocus --sync "$zwin" 2>/dev/null || xdotool windowfocus "$zwin" || true
		xdotool key --clearmodifiers Return
	fi
	for i in $(seq 1 20); do
		[ -f "$home/.local/share/applications/md-reader.desktop" ] && break
		sleep 0.5
	done
	check "7.4 nach 'Ja': ~/.local/share/applications/md-reader.desktop existiert" \
		test -f "$home/.local/share/applications/md-reader.desktop"
	check "7.5 nach 'Ja': ~/.local/bin/md-reader existiert" test -x "$home/.local/bin/md-reader"
	check "7.6 nach 'Ja': kein launcher.json (Antwortspeicher unberührt)" \
		test ! -e "$home/.config/md-reader/launcher.json"

	# Bestätigung „installiert“ (zenity --info) fotografieren und schließen.
	zwin="$(wait_window --class zenity)"
	if [ -n "$zwin" ]; then
		sleep 1
		import -display ":$LXC_DISPLAY" -window root "$OUT_DIR/shots/lxc-installed.png"
		info "Bestätigung: $(xprop -id "$zwin" WM_NAME); Bildschirmfoto: $OUT_DIR/shots/lxc-installed.png"
		xdotool windowfocus --sync "$zwin" 2>/dev/null || xdotool windowfocus "$zwin" || true
		xdotool key --clearmodifiers Return
	else
		info "keine Bestätigung per zenity gesehen"
	fi
	sleep 1
	kill "$app_pid" 2>/dev/null || true
	wait "$app_pid" 2>/dev/null || true
	info "LXC-Start 1 stderr: $(tr '\n' '|' <"$OUT_DIR/lxc-1.err" | cut -c1-400)"

	# Zweiter Start: installiert, gleiche Version → keine Frage.
	"${envargs[@]}" "$exe" "$PROJECT_DIR/README.md" >"$OUT_DIR/lxc-2.out" 2>"$OUT_DIR/lxc-2.err" &
	app_pid=$!
	OWN_PIDS+=("$app_pid")
	win="$(wait_window --class md-reader)"
	sleep 5
	zwin="$(xdotool search --onlyvisible --class zenity 2>/dev/null | head -n 1 || true)"
	import -display ":$LXC_DISPLAY" -window root "$OUT_DIR/shots/lxc-second-start.png"
	info "Bildschirmfoto: $OUT_DIR/shots/lxc-second-start.png"
	check "7.7 zweiter Start: Fenster erscheint" test -n "$win"
	check "7.8 zweiter Start: keine Frage" test -z "$zwin"
	kill "$app_pid" 2>/dev/null || true
	wait "$app_pid" 2>/dev/null || true
}

# ---------------------------------------------------------------------------
# Teil C: Beobachtungen (nicht bewertet)
# ---------------------------------------------------------------------------
part_observe() {
	echo "== Teil C: Beobachtungen =="
	docker pull -q "$OLD_IMAGE" >/dev/null
	run_stderr docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$OLD_IMAGE" /b/md-reader
	cp "$RUN_ERR" "$OUT_DIR/8-2204-nowebkit.err"
	info "8.1 $OLD_IMAGE ohne WebKit: Exit $RUN_CODE, stderr: $(tr '\n' '|' <"$RUN_ERR")"

	set +e
	docker run --rm -v "$BUILD_DIR:/b:ro" "$OLD_IMAGE" bash -c '
		export DEBIAN_FRONTEND=noninteractive
		apt-get update >/dev/null 2>&1 && apt-get install -y libwebkit2gtk-4.1-0 >/dev/null 2>&1 || echo "apt fehlgeschlagen" >&2
		echo "--- ohne DISPLAY:" >&2
		/b/md-reader; echo "EXIT=$?" >&2
		echo "--- --install:" >&2
		HOME=/tmp/h /b/md-reader --install; echo "EXIT=$?" >&2
	' >"$OUT_DIR/8-2204-webkit.out" 2>"$OUT_DIR/8-2204-webkit.err"
	set -e
	info "8.2 $OLD_IMAGE mit WebKit: stdout: $(tr '\n' '|' <"$OUT_DIR/8-2204-webkit.out") stderr: $(tr '\n' '|' <"$OUT_DIR/8-2204-webkit.err")"

	if command -v qemu-aarch64 >/dev/null; then
		set +e
		info "9.1 arm64 --launcher-version: $(qemu-aarch64 "$BUILD_DIR/md-reader-linux-arm64" --launcher-version 2>&1)"
		local h9="$OUT_DIR/qemu-home"
		rm -rf "$h9"
		mkdir -p "$h9"
		info "9.2 arm64 --install: $(HOME="$h9" qemu-aarch64 "$BUILD_DIR/md-reader-linux-arm64" --install 2>&1 | tr '\n' '|'); Dateien: $(find "$h9" -type f | wc -l)"
		local out9 code9
		out9="$(env -u DISPLAY HOME="$h9" qemu-aarch64 "$BUILD_DIR/md-reader-linux-arm64" 2>&1)"
		code9=$?
		info "9.3 arm64 Start ohne arm64-WebKit: Exit $code9, $(tr '\n' '|' <<<"$out9" | cut -c1-400)"
		set -e
	else
		info "9 qemu-aarch64 nicht vorhanden, übersprungen"
	fi
}

if [ "${SKIP_DOCKER:-0}" != "1" ]; then
	part_docker
fi
if [ "${SKIP_LXC:-0}" != "1" ]; then
	part_lxc
fi
if [ "${SKIP_DOCKER:-0}" != "1" ]; then
	part_observe
fi

echo "== Ergebnis: $FAILS FAIL, Ausgabe in $OUT_DIR =="
if [ "$FAILS" -gt 0 ]; then
	exit 1
fi
exit 0
