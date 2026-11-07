#!/usr/bin/env bash
# Abnahme des Linux-Starters, der Selbst-Installation und des .deb (Build 47).
#
# Teil A (Docker, je Image; Standard ubuntu:22.04, ubuntu:24.04, ubuntu:26.04):
#   1. Starter ohne WebKit: Exit 1, Hinweis genau einmal auf stderr (englisch).
#   2. Starter ohne WebKit mit LANG=de_DE.UTF-8: deutsche Meldung.
#   3. --install / --uninstall in HOME=/tmp/h, desktop-file-validate.
#   4. .deb per apt installieren (zieht WebKit mit), Paket-GUI unter xvfb-run,
#      WM_CLASS, Prozessname, Bildschirmfoto.
#   5. Starter mit installiertem Paket: Fenster, keine Installationsfrage;
#      Gegenprobe ohne Paket-Desktop-Datei (dann muss die Frage kommen).
# Teil F (Docker, ein Image, Standard ubuntu:24.04): Gegenprobe F1 –
#   Rechtsklick/Tastatur öffnen kein Kontextmenü, das Fenster navigiert nie
#   weg (lokaler HTTP-Server an 127.0.0.1 sieht keine Anfrage), Linksklick geht
#   an den externen Browser (Fake-xdg-open protokolliert).
# Teil T (Docker, zu altes System, Standard ubuntu:20.04 mit glibc 2.31):
#   Meldung „zu alt“ (Terminal und Dialog), Exit 1, --install verweigert,
#   apt lehnt das .deb ab.
# Teil D: Kopfdaten und Inhalt aller drei .deb-Pakete (ohne Container).
# Teil B (LXC unter eigenem Xvfb): Kopie des Starters mit leerem HOME:
#   WM_CLASS, zenity-Frage, „Ja“ per Return, Menüeintrag vorhanden; zweiter
#   Start ohne Frage.
# Teil C (nur Beobachtung, bewertet nicht): arm64-Starter unter qemu-user.
#
# Aufruf (aus beliebigem Verzeichnis):
#   debugging/linux/launcher-docker-test.sh [-i IMAGE]… [-o ALT-IMAGE] [-f F1-IMAGE] [AUSGABEVERZEICHNIS]
#     -i IMAGE   Image für Teil A (mehrfach möglich; ersetzt die Standardliste)
#     -o IMAGE   zu altes Image für Teil T (Standard ubuntu:20.04)
#     -f IMAGE   Image für Teil F (Standard ubuntu:24.04)
# Umgebung:
#   SKIP_DOCKER=1   Teile A, F, T, C überspringen
#   SKIP_IMAGES=1   nur Teil A überspringen
#   SKIP_F1=1       nur Teil F überspringen
#   SKIP_TOO_OLD=1  nur Teil T überspringen
#   SKIP_LXC=1      Teil B überspringen
#   LXC_DISPLAY=87  Nummer des eigenen Xvfb-Displays im LXC
#
# build/ wird nur lesend in die Container eingehängt; Protokolle und
# Bildschirmfotos landen im Ausgabeverzeichnis (Standard: mktemp -d).
# Container laufen mit --rm und ohne veröffentlichte Ports; Server im
# Container binden nur an 127.0.0.1. Gibt je Prüfung PASS/FAIL aus und endet
# mit Exit 1, wenn eine scheitert.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-11-01 (Endabnahme Build 47: mehrere Images, F1, zu altes System)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$PROJECT_DIR/build"
BUILD_NUM="$(tr -d '[:space:]' <"$PROJECT_DIR/src/build.txt")"
LXC_DISPLAY="${LXC_DISPLAY:-87}"

# Images für Teil A, F und T (per Option änderbar).
IMAGES=()
OLD_IMAGE="ubuntu:20.04"
F1_IMAGE="ubuntu:24.04"
while getopts "i:o:f:" opt; do
	case "$opt" in
		i) IMAGES+=("$OPTARG") ;;
		o) OLD_IMAGE="$OPTARG" ;;
		f) F1_IMAGE="$OPTARG" ;;
		*)
			echo "Aufruf: $0 [-i IMAGE]… [-o ALT-IMAGE] [-f F1-IMAGE] [AUSGABEVERZEICHNIS]" >&2
			exit 2
			;;
	esac
done
shift $((OPTIND - 1))
if [ "${#IMAGES[@]}" -eq 0 ]; then
	IMAGES=("ubuntu:22.04" "ubuntu:24.04" "ubuntu:26.04")
fi

OUT_DIR="${1:-$(mktemp -d)}"
mkdir -p "$OUT_DIR/shots"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"

# Text der Meldung „zu alt“ (englisch und deutsch).
TOO_OLD_EN="MD Reader: This Linux is too old"
TOO_OLD_DE="MD Reader: Dieses Linux ist zu alt"

# Anzahl gescheiterter Prüfungen.
FAILS=0
# Kurzname des gerade geprüften Images (steht vor jeder Prüfung).
CUR_TAG="-"
# PIDs eigener Hintergrundprozesse im LXC (werden beim Ende beendet).
OWN_PIDS=()
# Ergebnis je Teil für die Zusammenfassung ("Teil: n FAIL").
SUMMARY=()

# pass/fail/info geben das Ergebnis bzw. eine Beobachtung aus.
pass() {
	echo "PASS [$CUR_TAG] $1"
}
fail() {
	echo "FAIL [$CUR_TAG] $1"
	FAILS=$((FAILS + 1))
}
info() {
	echo "INFO [$CUR_TAG] $*"
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

# oneline gibt eine Datei einzeilig aus (Zeilenumbrüche als |).
oneline() {
	tr '\n' '|' <"$1"
}

# cleanup beendet nur die eigenen Prozesse (per PID, nie per Muster).
cleanup() {
	local pid
	for pid in "${OWN_PIDS[@]+"${OWN_PIDS[@]}"}"; do
		kill "$pid" 2>/dev/null || true
	done
}
trap cleanup EXIT

# tag_of IMAGE liefert den Kurznamen (ubuntu:24.04 → 24.04).
tag_of() {
	local tag="${1##*:}"
	echo "${tag//\//_}"
}

# run_stderr DATEI BEFEHL… führt BEFEHL aus, schreibt stderr nach DATEI und
# den Exit-Code nach $RUN_CODE (bricht wegen set -e nicht ab).
run_stderr() {
	local file="$1"
	shift
	set +e
	"$@" >/dev/null 2>"$file"
	RUN_CODE=$?
	set -e
}

# docker_inner IMAGE MODUS AUSGABE startet launcher-docker-inner.sh im
# Container und zählt dessen FAIL-Zeilen mit.
docker_inner() {
	local image="$1" mode="$2" out="$3" log code inner_fails
	log="$out/inner-$mode.log"
	mkdir -p "$out/$mode"
	set +e
	docker run --rm \
		-e TAG="$CUR_TAG" \
		-v "$BUILD_DIR:/b:ro" \
		-v "$PROJECT_DIR/README.md:/doc/README.md:ro" \
		-v "$SCRIPT_DIR/launcher-docker-inner.sh:/inner.sh:ro" \
		-v "$out/$mode:/out" \
		"$image" bash /inner.sh "$mode" >"$log" 2>&1
	code=$?
	set -e
	cat "$log"
	inner_fails="$(grep -c '^FAIL ' "$log" || true)"
	FAILS=$((FAILS + inner_fails))
	if [ "$code" -ne 0 ] && [ "$inner_fails" -eq 0 ]; then
		fail "Container-Modus $mode endete mit Exit $code ohne FAIL-Zeile"
	fi
}

# convert_shots VERZEICHNIS PRÄFIX wandelt alle .xwd in VERZEICHNIS nach
# $OUT_DIR/shots/PRÄFIX-<name>.png.
convert_shots() {
	local dir="$1" prefix="$2" xwd name
	for xwd in "$dir"/*.xwd; do
		[ -s "$xwd" ] || continue
		name="$(basename "$xwd" .xwd)"
		convert "xwd:$xwd" "$OUT_DIR/shots/$prefix-$name.png"
		info "Bildschirmfoto: $OUT_DIR/shots/$prefix-$name.png"
	done
}

# summarize TEIL FAILS_VORHER trägt das Ergebnis eines Teils ein.
summarize() {
	SUMMARY+=("$1: $((FAILS - $2)) FAIL")
}

# ---------------------------------------------------------------------------
# Teil A: ein Image
# ---------------------------------------------------------------------------
part_image() {
	local image="$1" out err before="$FAILS"
	CUR_TAG="$(tag_of "$image")"
	out="$OUT_DIR/$CUR_TAG"
	mkdir -p "$out"
	echo "== Teil A: $image =="
	docker pull -q "$image" >/dev/null
	info "$(docker run --rm --network none "$image" bash -c 'grep PRETTY_NAME /etc/os-release; ldd --version | head -n 1' | tr '\n' ' ')"

	# 1. Ohne WebKit (kein Netz nötig), englisch.
	err="$out/1-nowebkit-en.err"
	run_stderr "$err" docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$image" /b/md-reader
	info "1: Exit $RUN_CODE, stderr: $(oneline "$err")"
	check "1.1 ohne WebKit: Exit 1" test "$RUN_CODE" -eq 1
	check "1.2 stderr enthält WebKitGTK" grep -q 'WebKitGTK' "$err"
	check "1.3 stderr enthält libwebkit2gtk-4.1.so.0" grep -q 'libwebkit2gtk-4.1.so.0' "$err"
	check "1.4 stderr enthält 'sudo apt install libwebkit2gtk-4.1-0'" \
		grep -q 'sudo apt install libwebkit2gtk-4.1-0' "$err"
	check "1.5 englische Meldung (Missing libraries:)" grep -q '^Missing libraries: ' "$err"
	check "1.6 Hinweis genau einmal" bash -c "
		test \"\$(grep -c 'WebKitGTK is not installed' '$err')\" -eq 1 &&
		test \"\$(grep -c 'sudo apt install libwebkit2gtk-4.1-0' '$err')\" -eq 1"
	check "1.7 keine Meldung „zu alt“" bash -c "! grep -q 'too old' '$err'"

	# 2. Ohne WebKit, deutsch.
	err="$out/2-nowebkit-de.err"
	run_stderr "$err" docker run --rm --network none -e LANG=de_DE.UTF-8 -v "$BUILD_DIR:/b:ro" "$image" /b/md-reader
	info "2: Exit $RUN_CODE, stderr: $(oneline "$err")"
	check "2.1 deutsch: Exit 1" test "$RUN_CODE" -eq 1
	check "2.2 deutsch: 'WebKitGTK ist nicht installiert' genau einmal" \
		test "$(grep -c 'WebKitGTK ist nicht installiert' "$err")" -eq 1
	check "2.3 deutsch: 'Fehlende Bibliotheken:' mit libwebkit2gtk-4.1.so.0" \
		grep -q '^Fehlende Bibliotheken: .*libwebkit2gtk-4.1.so.0' "$err"
	check "2.4 deutsch: Befehl 'sudo apt install libwebkit2gtk-4.1-0'" \
		grep -q 'sudo apt install libwebkit2gtk-4.1-0' "$err"

	# Versionsabfrage als Rauchtest.
	check "2.5 --launcher-version = $BUILD_NUM" \
		test "$(docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$image" /b/md-reader --launcher-version)" = "$BUILD_NUM"

	# 3. --install / --uninstall (braucht Netz für desktop-file-utils).
	docker_inner "$image" install "$out"
	# 4. + 5. Paket, GUI und Starter mit installiertem Paket.
	docker_inner "$image" deb "$out"
	convert_shots "$out/deb" "$CUR_TAG"
	summarize "$image" "$before"
}

# ---------------------------------------------------------------------------
# Teil F: Gegenprobe F1 (Kontextmenü, Navigation)
# ---------------------------------------------------------------------------
part_f1() {
	local before="$FAILS"
	CUR_TAG="F1-$(tag_of "$F1_IMAGE")"
	echo "== Teil F: F1-Gegenprobe in $F1_IMAGE =="
	docker pull -q "$F1_IMAGE" >/dev/null
	mkdir -p "$OUT_DIR/f1"
	docker_inner "$F1_IMAGE" f1 "$OUT_DIR/f1"
	convert_shots "$OUT_DIR/f1/f1" "$(tag_of "$F1_IMAGE")"
	summarize "F1 ($F1_IMAGE)" "$before"
}

# ---------------------------------------------------------------------------
# Teil T: zu altes System
# ---------------------------------------------------------------------------
part_too_old() {
	local before="$FAILS" out err home
	CUR_TAG="alt-$(tag_of "$OLD_IMAGE")"
	out="$OUT_DIR/too-old"
	mkdir -p "$out"
	echo "== Teil T: zu altes System $OLD_IMAGE =="
	docker pull -q "$OLD_IMAGE" >/dev/null
	info "$(docker run --rm --network none "$OLD_IMAGE" bash -c 'grep PRETTY_NAME /etc/os-release; ldd --version | head -n 1' | tr '\n' ' ')"

	err="$out/start-en.err"
	run_stderr "$err" docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$OLD_IMAGE" /b/md-reader
	info "Start: Exit $RUN_CODE, stderr: $(oneline "$err")"
	check "T.0a Exit 1" test "$RUN_CODE" -eq 1
	check "T.0b Meldung „$TOO_OLD_EN“ genau einmal" test "$(grep -c "$TOO_OLD_EN" "$err")" -eq 1
	check "T.0c nennt die nötige glibc-Version" grep -q 'Required: glibc 2\.[0-9]* or newer' "$err"
	check "T.0d kein WebKit-Hinweis (hülfe nicht)" bash -c "! grep -q 'libwebkit2gtk' '$err'"

	err="$out/start-de.err"
	run_stderr "$err" docker run --rm --network none -e LANG=de_DE.UTF-8 -v "$BUILD_DIR:/b:ro" "$OLD_IMAGE" /b/md-reader
	info "Start deutsch: Exit $RUN_CODE, stderr: $(oneline "$err")"
	check "T.0e deutsch: Exit 1 und „$TOO_OLD_DE“" \
		bash -c "test '$RUN_CODE' -eq 1 && grep -q '$TOO_OLD_DE' '$err'"

	err="$out/install.err"
	set +e
	docker run --rm --network none -v "$BUILD_DIR:/b:ro" "$OLD_IMAGE" bash -c '
		mkdir -p /tmp/h; HOME=/tmp/h /b/md-reader --install; code=$?
		echo "FILES=$(find /tmp/h -type f | wc -l)"; exit $code' >"$out/install.out" 2>"$err"
	RUN_CODE=$?
	set -e
	info "--install: Exit $RUN_CODE, stdout: $(oneline "$out/install.out") stderr: $(oneline "$err")"
	check "T.0f --install verweigert (Exit ≠ 0, „zu alt“, keine Dateien)" bash -c "
		test '$RUN_CODE' -ne 0 && grep -q '$TOO_OLD_EN' '$err' && grep -qx 'FILES=0' '$out/install.out'"

	docker_inner "$OLD_IMAGE" toold "$out"
	convert_shots "$out/toold" "too-old"
	summarize "zu alt ($OLD_IMAGE)" "$before"
}

# ---------------------------------------------------------------------------
# Teil D: Kopfdaten und Inhalt aller .deb-Pakete (ohne Container)
# ---------------------------------------------------------------------------
part_debs() {
	local before="$FAILS" arch deb libc
	CUR_TAG="deb"
	echo "== Teil D: .deb-Pakete =="
	for arch in amd64 arm64 armhf; do
		deb="$BUILD_DIR/md-reader_1.0.${BUILD_NUM}_${arch}.deb"
		libc="$(dpkg-deb -f "$deb" Depends | sed -n 's/.*libc6 (>= \([0-9.]*\)).*/\1/p')"
		info "$arch: Depends libc6 >= $libc"
		check "D.$arch Kopf: Package/Version/Architecture/Depends/Recommends/Maintainer" bash -c "
			f=\$(dpkg-deb -f '$deb' Package Version Architecture Depends Recommends Maintainer)
			grep -qx 'Package: md-reader' <<<\"\$f\" &&
			grep -qx 'Version: 1.0.$BUILD_NUM' <<<\"\$f\" &&
			grep -qx 'Architecture: $arch' <<<\"\$f\" &&
			grep -Eqx 'Depends: libc6 \(>= 2\.[0-9]+\), libwebkit2gtk-4.1-0, libgtk-3-0' <<<\"\$f\" &&
			grep -qx 'Recommends: zenity, poppler-utils, ghostscript' <<<\"\$f\" &&
			grep -qx 'Maintainer: Kurt Ingwer <WaschbaerImHaus@users.noreply.github.com>' <<<\"\$f\""
		check "D.$arch libc6-Grenze ≤ 2.35 (läuft auf Ubuntu 22.04)" \
			bash -c "test -n '$libc' && test \"\$(printf '%s\n' '$libc' 2.35 | sort -V | tail -n 1)\" = 2.35"
		check "D.$arch Inhalt: Programm, Desktop-Datei, 5 Icons" bash -c "
			c=\$(dpkg-deb -c '$deb' | awk '{print \$NF}')
			grep -qx './usr/bin/md-reader' <<<\"\$c\" &&
			grep -qx './usr/share/applications/md-reader.desktop' <<<\"\$c\" &&
			test \$(grep -c '^./usr/share/icons/hicolor/.*/apps/md-reader\.\(png\|svg\)$' <<<\"\$c\") -eq 5"
	done
	summarize ".deb-Kopfdaten" "$before"
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
	local before="$FAILS"
	CUR_TAG="lxc"
	echo "== Teil B: LXC unter Xvfb :$LXC_DISPLAY =="
	if ! command -v zenity >/dev/null; then
		fail "7.0 zenity fehlt (sudo apt-get install -y zenity)"
		summarize "LXC" "$before"
		return
	fi
	if [ -e "/tmp/.X11-unix/X$LXC_DISPLAY" ] || [ -e "/tmp/.X$LXC_DISPLAY-lock" ]; then
		fail "7.0 Display :$LXC_DISPLAY ist schon belegt (LXC_DISPLAY anders setzen)"
		summarize "LXC" "$before"
		return
	fi

	local home="$OUT_DIR/lxc-home" run="$OUT_DIR/lxc-run" exe="$OUT_DIR/md-reader-test"
	local win zwin wmclass app_pid i
	# Nur Unterordner des eigenen Ausgabeverzeichnisses werden geleert.
	rm -rf "${OUT_DIR:?}/lxc-home" "${OUT_DIR:?}/lxc-run"
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
	info "LXC-Start 1 stderr: $(oneline "$OUT_DIR/lxc-1.err" | cut -c1-400)"

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
	summarize "LXC" "$before"
}

# ---------------------------------------------------------------------------
# Teil C: Beobachtungen (nicht bewertet)
# ---------------------------------------------------------------------------
part_observe() {
	CUR_TAG="qemu"
	echo "== Teil C: Beobachtungen =="
	if command -v qemu-aarch64 >/dev/null; then
		local h9="$OUT_DIR/qemu-home" out9 code9
		set +e
		info "9.1 arm64 --launcher-version: $(qemu-aarch64 "$BUILD_DIR/md-reader-linux-arm64" --launcher-version 2>&1)"
		rm -rf "${OUT_DIR:?}/qemu-home"
		mkdir -p "$h9"
		info "9.2 arm64 --install: $(HOME="$h9" qemu-aarch64 "$BUILD_DIR/md-reader-linux-arm64" --install 2>&1 | tr '\n' '|'); Dateien: $(find "$h9" -type f | wc -l)"
		out9="$(env -u DISPLAY HOME="$h9" qemu-aarch64 "$BUILD_DIR/md-reader-linux-arm64" 2>&1)"
		code9=$?
		info "9.3 arm64 Start ohne binfmt: Exit $code9, $(tr '\n' '|' <<<"$out9" | cut -c1-400)"
		set -e
	else
		info "9 qemu-aarch64 nicht vorhanden, übersprungen"
	fi
}

if [ "${SKIP_DOCKER:-0}" != "1" ]; then
	if [ "${SKIP_IMAGES:-0}" != "1" ]; then
		for image in "${IMAGES[@]}"; do
			part_image "$image"
		done
	fi
	if [ "${SKIP_F1:-0}" != "1" ]; then
		part_f1
	fi
	if [ "${SKIP_TOO_OLD:-0}" != "1" ]; then
		part_too_old
	fi
fi
part_debs
if [ "${SKIP_LXC:-0}" != "1" ]; then
	part_lxc
fi
if [ "${SKIP_DOCKER:-0}" != "1" ]; then
	part_observe
fi

echo "== Zusammenfassung =="
for line in "${SUMMARY[@]+"${SUMMARY[@]}"}"; do
	echo "  $line"
done
echo "== Ergebnis: $FAILS FAIL, Ausgabe in $OUT_DIR =="
if [ "$FAILS" -gt 0 ]; then
	exit 1
fi
exit 0
