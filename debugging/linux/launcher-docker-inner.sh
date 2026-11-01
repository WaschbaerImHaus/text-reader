#!/usr/bin/env bash
# Hilfsskript für launcher-docker-test.sh: läuft IM Container (ubuntu:24.04).
#
# Aufruf (nur über launcher-docker-test.sh):
#   launcher-docker-inner.sh install   # --install/--uninstall in HOME=/tmp/h
#   launcher-docker-inner.sh deb       # .deb installieren, GUI unter Xvfb,
#                                      # Starter mit installiertem WebKit
#
# Erwartete Einhängepunkte:
#   /b      build/ (nur lesen)
#   /doc    README.md des Projekts (nur lesen)
#   /out    Ausgabeverzeichnis (Protokolle, Bildschirmfotos als .xwd)
#
# Jede Prüfung schreibt eine Zeile "PASS <Name>" oder "FAIL <Name>" auf
# stdout; das äußere Skript zählt die FAIL-Zeilen. Exit-Code = Anzahl FAIL
# (höchstens 99), damit auch ein Abbruch auffällt.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-28 (Abnahme Aufgabe 9)

set -euo pipefail

# Anzahl der gescheiterten Prüfungen in diesem Container.
FAILS=0
# Nummer des Xvfb-Displays im Container (eigener Namensraum, kein Konflikt mit dem LXC).
CONTAINER_DISPLAY=":91"
# Text der Installationsfrage (englisch und deutsch), darf im Paketfall nie erscheinen.
OFFER_PATTERN='Add MD Reader to the menu|MD Reader ins Menü aufnehmen'

# pass/fail geben das Ergebnis einer Prüfung aus.
pass() {
	echo "PASS $1"
}
fail() {
	echo "FAIL $1"
	FAILS=$((FAILS + 1))
}

# check NAME BEFEHL… führt BEFEHL aus und meldet PASS bei Exit 0, sonst FAIL.
check() {
	local name="$1"
	shift
	if "$@"; then
		pass "$name"
	else
		fail "$name"
	fi
}

# info gibt eine Beobachtung aus, die nicht bewertet wird.
info() {
	echo "INFO $*"
}

# apt_quiet installiert Pakete ohne Rückfragen, Protokoll nach /out/<Name>.log.
#
# @param $1 Protokollname
# @param $@ Pakete bzw. .deb-Pfade
apt_quiet() {
	local log="/out/$1.log"
	shift
	export DEBIAN_FRONTEND=noninteractive
	apt-get update >"$log" 2>&1 && apt-get install -y "$@" >>"$log" 2>&1
}

# wait_visible_window wartet bis zu 20 s auf ein sichtbares Fenster der
# Klasse md-reader und gibt dessen ID aus (leer bei Zeitablauf).
wait_visible_window() {
	local i id
	for i in $(seq 1 40); do
		id="$(xdotool search --onlyvisible --class md-reader 2>/dev/null | head -n 1 || true)"
		if [ -n "$id" ]; then
			echo "$id"
			return 0
		fi
		sleep 0.5
	done
	return 0
}

# ---------------------------------------------------------------------------
# Modus "install": Selbst-Installation und Entfernen in einem leeren HOME.
# ---------------------------------------------------------------------------
run_install() {
	local home="/tmp/h"
	local data="$home/.local/share"
	local files=(
		"$home/.local/bin/md-reader"
		"$data/applications/md-reader.desktop"
		"$data/icons/hicolor/16x16/apps/md-reader.png"
		"$data/icons/hicolor/32x32/apps/md-reader.png"
		"$data/icons/hicolor/48x48/apps/md-reader.png"
		"$data/icons/hicolor/256x256/apps/md-reader.png"
		"$data/icons/hicolor/scalable/apps/md-reader.svg"
	)
	local f code count

	check "3.0 desktop-file-utils installiert" apt_quiet apt-install desktop-file-utils

	mkdir -p "$home"
	export HOME="$home"
	unset XDG_CONFIG_HOME XDG_DATA_HOME XDG_CACHE_HOME

	set +e
	/b/md-reader --install >/out/install.out 2>&1
	code=$?
	set -e
	info "--install Ausgabe: $(tr '\n' '|' </out/install.out)"
	check "3.1 --install Exit 0" test "$code" -eq 0

	count=0
	for f in "${files[@]}"; do
		if [ -f "$f" ]; then
			count=$((count + 1))
		else
			info "fehlt: $f"
		fi
	done
	check "3.2 alle 7 Dateien angelegt ($count/7)" test "$count" -eq 7
	check "3.3 Programm ausführbar und identisch mit /b/md-reader" \
		bash -c "test -x '${files[0]}' && cmp -s '${files[0]}' /b/md-reader"

	set +e
	desktop-file-validate "${files[1]}" >/out/validate.out 2>&1
	code=$?
	set -e
	info "desktop-file-validate: Exit $code, Ausgabe: $(tr '\n' '|' </out/validate.out)"
	check "3.4 desktop-file-validate ohne Fehler" \
		bash -c "test '$code' -eq 0 && ! grep -q 'error:' /out/validate.out"
	check "3.5 Exec zeigt auf ~/.local/bin/md-reader" \
		grep -qx "Exec=\"$home/.local/bin/md-reader\" %f" "${files[1]}"
	check "3.6 Categories=Office;Viewer; und StartupWMClass=md-reader" \
		bash -c "grep -qx 'Categories=Office;Viewer;' '${files[1]}' && grep -qx 'StartupWMClass=md-reader' '${files[1]}'"
	check "3.7 MimeType enthält text/markdown" grep -q '^MimeType=text/markdown;' "${files[1]}"

	# Nutzerdaten anlegen, die --uninstall behalten muss.
	mkdir -p "$home/.config/md-reader" "$home/.cache/md-reader" "$data/md-reader"
	echo '{"installPrompt":"never"}' >"$home/.config/md-reader/launcher.json"

	set +e
	/b/md-reader --uninstall >/out/uninstall.out 2>&1
	code=$?
	set -e
	info "--uninstall Ausgabe: $(tr '\n' '|' </out/uninstall.out)"
	check "3.8 --uninstall Exit 0" test "$code" -eq 0

	count=0
	for f in "${files[@]}"; do
		if [ -e "$f" ]; then
			count=$((count + 1))
			info "noch vorhanden: $f"
		fi
	done
	check "3.9 alle 7 Dateien entfernt" test "$count" -eq 0
	check "3.10 Ausgabe nennt die behaltenen Pfade" bash -c "
		grep -qx '$home/.config/md-reader' /out/uninstall.out &&
		grep -qx '$home/.cache/md-reader' /out/uninstall.out &&
		grep -qx '$data/md-reader' /out/uninstall.out"
	check "3.11 Nutzerdaten noch vorhanden" bash -c "
		test -f '$home/.config/md-reader/launcher.json' &&
		test -d '$home/.cache/md-reader' && test -d '$data/md-reader'"
}

# ---------------------------------------------------------------------------
# Modus "deb": Paketinstallation, GUI unter Xvfb, Starter mit WebKit.
# ---------------------------------------------------------------------------
run_deb() {
	local code win pid comm wmclass xvfb_pid gui_pid launcher_pid offer_win i

	check "4.1 apt-get install /b/md-reader_1.0.47_amd64.deb gelingt" \
		apt_quiet apt-deb /b/md-reader_1.0.47_amd64.deb
	check "4.2 WebKit mitgezogen (libwebkit2gtk-4.1-0 installiert)" \
		bash -c "dpkg-query -W -f='\${Status}' libwebkit2gtk-4.1-0 | grep -q 'install ok installed'"
	info "Paketfelder: $(dpkg-query -W -f='${Version} | Depends: ${Depends} | Recommends: ${Recommends}' md-reader)"
	check "4.3 /usr/share/applications/md-reader.desktop existiert" \
		test -f /usr/share/applications/md-reader.desktop
	check "4.4 /usr/bin/md-reader ist die GUI aus build/gui" \
		cmp -s /usr/bin/md-reader /b/gui/md-reader-amd64

	# Prüfwerkzeuge erst nach dem Paket, damit 4.1 nur das .deb misst.
	check "4.5 Prüfwerkzeuge (xvfb, x11-utils, xdotool, x11-apps) installiert" \
		apt_quiet apt-tools xvfb x11-utils xdotool x11-apps

	# GUI ohne Display: klare Meldung statt Absturz.
	set +e
	env -u DISPLAY -u WAYLAND_DISPLAY /usr/bin/md-reader /doc/README.md >/out/nodisplay.out 2>&1
	code=$?
	set -e
	info "GUI ohne Display: Exit $code, $(tr '\n' '|' </out/nodisplay.out)"
	check "4.6 GUI ohne Display: Exit 1 mit Meldung" \
		bash -c "test '$code' -eq 1 && grep -q 'Keine grafische Oberfläche gefunden' /out/nodisplay.out"

	Xvfb "$CONTAINER_DISPLAY" -screen 0 1280x800x24 -nolisten tcp >/out/xvfb.log 2>&1 &
	xvfb_pid=$!
	export DISPLAY="$CONTAINER_DISPLAY"
	sleep 2

	# --- 4: Paket-GUI ---------------------------------------------------
	/usr/bin/md-reader /doc/README.md >/out/gui-deb.log 2>&1 &
	gui_pid=$!
	win="$(wait_visible_window)"
	sleep 3
	check "4.7 xdotool findet sichtbares Fenster der Klasse md-reader" test -n "$win"
	if [ -n "$win" ]; then
		wmclass="$(xprop -id "$win" WM_CLASS)"
		info "Paket-GUI: $wmclass; $(xprop -id "$win" WM_NAME)"
		check "4.8 WM_CLASS = \"md-reader\", \"md-reader\"" \
			test "$wmclass" = 'WM_CLASS(STRING) = "md-reader", "md-reader"'
	else
		fail "4.8 WM_CLASS (kein Fenster)"
	fi
	comm="$(cat "/proc/$gui_pid/comm" 2>/dev/null || true)"
	check "4.9 Prozessname md-reader (ist: $comm)" test "$comm" = "md-reader"
	xwd -root -silent >/out/deb.xwd
	check "4.10 Bildschirmfoto deb.xwd geschrieben" test -s /out/deb.xwd
	kill "$gui_pid" 2>/dev/null || true
	wait "$gui_pid" 2>/dev/null || true
	sleep 1

	# --- 5: Starter mit installiertem WebKit, Paket installiert ----------
	mkdir -p /tmp/h5
	HOME=/tmp/h5 /b/md-reader /doc/README.md >/out/launcher-deb.out 2>/out/launcher-deb.err &
	launcher_pid=$!
	win="$(wait_visible_window)"
	sleep 5
	check "5.1 Starter /b/md-reader öffnet ein Fenster" test -n "$win"
	if [ -n "$win" ]; then
		info "Starter-GUI: $(xprop -id "$win" WM_CLASS)"
		pid="$(xdotool getwindowpid "$win" 2>/dev/null || true)"
		comm="$(cat "/proc/$pid/comm" 2>/dev/null || true)"
		info "Starter-GUI: PID $pid (Starter-PID $launcher_pid), comm=$comm"
		check "5.2 GUI-Prozess heißt md-reader (exec aus memfd)" test "$comm" = "md-reader"
	else
		fail "5.2 GUI-Prozess (kein Fenster)"
	fi
	xwd -root -silent >/out/launcher-deb.xwd
	# Das .deb zieht zenity als Empfehlung mit: eine Frage käme als Fenster.
	offer_win="$(xdotool search --onlyvisible --name "$OFFER_PATTERN" 2>/dev/null | head -n 1 || true)"
	kill "$launcher_pid" 2>/dev/null || true
	wait "$launcher_pid" 2>/dev/null || true
	sleep 1
	info "Starter stderr: $(tr '\n' '|' </out/launcher-deb.err | cut -c1-400)"
	check "5.3 stderr ohne Installationsfrage" bash -c "! grep -Eq '$OFFER_PATTERN' /out/launcher-deb.err"
	check "5.3b kein Fragefenster (zenity) sichtbar" test -z "$offer_win"
	check "5.4 stderr ohne WebKit-Hinweis" bash -c "! grep -q 'WebKitGTK' /out/launcher-deb.err"
	check "5.5 kein launcher.json und keine Nutzerinstallation geschrieben" bash -c "
		! test -e /tmp/h5/.config/md-reader/launcher.json &&
		! test -e /tmp/h5/.local/share/applications/md-reader.desktop"

	# --- 5b: Gegenprobe ohne Paket-Desktop-Datei -------------------------
	# Ohne /usr/share/applications/md-reader.desktop MUSS die Frage kommen,
	# sonst wären 5.3/5.3b wertlos. zenity ist als Empfehlung des .deb
	# installiert, also erscheint die Frage als Fenster (stderr erbt der
	# Angebotsprozess nur bei einem Terminal). Escape = letzter Knopf "Später".
	mv /usr/share/applications/md-reader.desktop /tmp/md-reader.desktop.bak
	mkdir -p /tmp/h5b
	HOME=/tmp/h5b /b/md-reader /doc/README.md >/out/launcher-nodeb.out 2>/out/launcher-nodeb.err &
	launcher_pid=$!
	win="$(wait_visible_window)"
	offer_win=""
	for i in $(seq 1 30); do
		offer_win="$(xdotool search --onlyvisible --name "$OFFER_PATTERN" 2>/dev/null | head -n 1 || true)"
		[ -n "$offer_win" ] && break
		sleep 0.5
	done
	sleep 1
	xwd -root -silent >/out/launcher-nodeb.xwd
	check "5.6 Gegenprobe: Fenster erscheint" test -n "$win"
	check "5.7 Gegenprobe: ohne Paket erscheint die Frage (zenity)" test -n "$offer_win"
	if [ -n "$offer_win" ]; then
		info "Gegenprobe Frage: $(xprop -id "$offer_win" WM_NAME WM_CLASS | tr '\n' ' ')"
		xdotool windowfocus --sync "$offer_win" 2>/dev/null || xdotool windowfocus "$offer_win" || true
		xdotool key --clearmodifiers Escape
	fi
	for i in $(seq 1 20); do
		[ -f /tmp/h5b/.config/md-reader/launcher.json ] && break
		sleep 0.5
	done
	info "Gegenprobe launcher.json: $(cat /tmp/h5b/.config/md-reader/launcher.json 2>&1)"
	check "5.8 Gegenprobe: Escape speichert 'later'" \
		grep -q '"later"' /tmp/h5b/.config/md-reader/launcher.json
	check "5.9 Gegenprobe: keine Nutzerinstallation nach Escape" \
		test ! -e /tmp/h5b/.local/share/applications/md-reader.desktop
	kill "$launcher_pid" 2>/dev/null || true
	wait "$launcher_pid" 2>/dev/null || true
	mv /tmp/md-reader.desktop.bak /usr/share/applications/md-reader.desktop
	info "Gegenprobe stderr: $(tr '\n' '|' </out/launcher-nodeb.err | cut -c1-400)"

	kill "$xvfb_pid" 2>/dev/null || true
	wait "$xvfb_pid" 2>/dev/null || true
}

case "${1:-}" in
	install) run_install ;;
	deb) run_deb ;;
	*)
		echo "Aufruf: $0 install|deb" >&2
		exit 2
		;;
esac

# Dateien für den Nutzer außerhalb lesbar machen (Container läuft als root).
chmod -R a+rX /out 2>/dev/null || true
if [ "$FAILS" -gt 99 ]; then
	FAILS=99
fi
exit "$FAILS"
