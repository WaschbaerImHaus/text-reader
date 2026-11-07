#!/usr/bin/env bash
# Hilfsskript für launcher-docker-test.sh: läuft IM Container (Ubuntu).
#
# Aufruf (nur über launcher-docker-test.sh):
#   launcher-docker-inner.sh install   # --install/--uninstall in HOME=/tmp/h
#   launcher-docker-inner.sh deb       # .deb installieren (zieht WebKit mit),
#                                      # danach Teil "deb-display" unter xvfb-run
#   launcher-docker-inner.sh f1        # .deb + Gegenprobe F1 (Kontextmenü,
#                                      # Navigation) unter xvfb-run
#   launcher-docker-inner.sh toold     # zu altes System: .deb wird abgelehnt,
#                                      # Starter zeigt „zu alt“ (Dialog) unter xvfb-run
# Die Modi *-display ruft das Skript selbst unter xvfb-run auf.
#
# Erwartete Einhängepunkte:
#   /b      build/ (nur lesen)
#   /doc    README.md des Projekts (nur lesen)
#   /out    Ausgabeverzeichnis (Protokolle, Bildschirmfotos als .xwd)
# Umgebung: TAG = Kurzname des Images (z. B. 24.04), steht vor jeder Prüfung.
#
# Jede Prüfung schreibt eine Zeile "PASS [TAG] <Name>" oder "FAIL [TAG] <Name>"
# auf stdout; das äußere Skript zählt die FAIL-Zeilen. Exit-Code = Anzahl FAIL
# (höchstens 99), damit auch ein Abbruch auffällt.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-11-01 (Endabnahme Build 47: xvfb-run, mehrere Images, F1, zu altes System)

set -euo pipefail

# Kurzname des Images für die Ausgabe.
TAG="${TAG:-?}"
# Anzahl der gescheiterten Prüfungen in diesem Prozess.
FAILS=0
# Text der Installationsfrage (englisch und deutsch), darf im Paketfall nie erscheinen.
OFFER_PATTERN='Add MD Reader to the menu|MD Reader ins Menü aufnehmen'
# Titel der Meldung „System zu alt“ (englisch und deutsch).
TOO_OLD_PATTERN='This Linux is too old|Dieses Linux ist zu alt'
# Bildschirmgröße und Optionen für xvfb-run (kein TCP).
XVFB_ARGS='-screen 0 1280x800x24 -nolisten tcp'
# Port des lokalen HTTP-Servers für F1 (nur 127.0.0.1, im Container-Netz).
F1_PORT=8765
# Ziel des Testlinks.
F1_URL="http://127.0.0.1:$F1_PORT/clicked"
# Bildschirmposition des Links (Überschrift H1 links oben im Fenster bei 0,0).
F1_LINK_X=90
F1_LINK_Y=116

# pass/fail geben das Ergebnis einer Prüfung aus.
pass() {
	echo "PASS [$TAG] $1"
}
fail() {
	echo "FAIL [$TAG] $1"
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
	echo "INFO [$TAG] $*"
}

# oneline gibt eine Datei einzeilig aus (Zeilenumbrüche als |, gekürzt).
oneline() {
	tr '\n' '|' <"$1" | cut -c1-600
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

# deb_path liefert den Pfad des amd64-Pakets in /b.
deb_path() {
	ls /b/md-reader_1.0.*_amd64.deb | head -n 1
}

# wait_window ARGS… wartet bis zu 20 s auf ein sichtbares Fenster
# (xdotool search --onlyvisible ARGS) und gibt die erste ID aus.
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

# count_viewable zählt die sichtbaren Fenster direkt unter der Wurzel.
# Kontextmenüs (GtkMenu-Popups) sind solche Fenster, auch ohne Fenstermanager.
count_viewable() {
	local n=0 id
	for id in $(xwininfo -root -children | awk '/^ +0x/{print $1}'); do
		if xwininfo -id "$id" 2>/dev/null | grep -q 'Map State: IsViewable'; then
			n=$((n + 1))
		fi
	done
	echo "$n"
}

# shot NAME schreibt ein Bildschirmfoto nach /out/NAME.xwd.
shot() {
	xwd -root -silent >"/out/$1.xwd"
}

# under_xvfb MODUS startet dieses Skript im Modus MODUS unter xvfb-run und
# übernimmt dessen Ausgabe; scheitert xvfb-run selbst, gilt das als FAIL.
under_xvfb() {
	local code
	set +e
	xvfb-run -a -n 91 -s "$XVFB_ARGS" bash "$0" "$1"
	code=$?
	set -e
	if [ "$code" -ne 0 ]; then
		# Die FAIL-Zeilen hat der Kindprozess schon ausgegeben; nur zählen.
		FAILS=$((FAILS + code))
	fi
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
	info "--install Ausgabe: $(oneline /out/install.out)"
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
	info "desktop-file-validate: Exit $code, Ausgabe: $(oneline /out/validate.out)"
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
	info "--uninstall Ausgabe: $(oneline /out/uninstall.out)"
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

# install_deb_and_tools installiert das .deb (Prüfung 4.1–4.4) und danach die
# Prüfwerkzeuge (4.5). Weitere Pakete können als Argumente folgen.
install_deb_and_tools() {
	check "4.1 apt-get install $(basename "$(deb_path)") gelingt" \
		apt_quiet apt-deb "$(deb_path)"
	check "4.2 WebKit mitgezogen (libwebkit2gtk-4.1-0 installiert)" \
		bash -c "dpkg-query -W -f='\${Status}' libwebkit2gtk-4.1-0 | grep -q 'install ok installed'"
	info "Pakete: md-reader $(dpkg-query -W -f='${Version}' md-reader 2>/dev/null), WebKit $(dpkg-query -W -f='${Version}' libwebkit2gtk-4.1-0 2>/dev/null), libc6 $(dpkg-query -W -f='${Version}' libc6)"
	check "4.3 /usr/share/applications/md-reader.desktop existiert" \
		test -f /usr/share/applications/md-reader.desktop
	check "4.4 /usr/bin/md-reader ist die GUI aus build/gui" \
		cmp -s /usr/bin/md-reader /b/gui/md-reader-amd64
	# Prüfwerkzeuge erst nach dem Paket, damit 4.1 nur das .deb misst.
	check "4.5 Prüfwerkzeuge installiert" \
		apt_quiet apt-tools xvfb xauth x11-utils xdotool x11-apps "$@"
}

# ---------------------------------------------------------------------------
# Modus "deb": Paketinstallation, dann GUI und Starter unter xvfb-run.
# ---------------------------------------------------------------------------
run_deb() {
	local code
	install_deb_and_tools

	# GUI ohne Display: klare Meldung statt Absturz.
	set +e
	env -u DISPLAY -u WAYLAND_DISPLAY /usr/bin/md-reader /doc/README.md >/out/nodisplay.out 2>&1
	code=$?
	set -e
	info "GUI ohne Display: Exit $code, $(oneline /out/nodisplay.out)"
	check "4.6 GUI ohne Display: Exit 1 mit Meldung" \
		bash -c "test '$code' -eq 1 && grep -q 'Keine grafische Oberfläche gefunden' /out/nodisplay.out"

	under_xvfb deb-display
}

run_deb_display() {
	local win pid comm wmclass gui_pid launcher_pid offer_win i
	info "xvfb-run: DISPLAY=$DISPLAY"

	# --- 4: Paket-GUI ---------------------------------------------------
	/usr/bin/md-reader /doc/README.md >/out/gui-deb.log 2>&1 &
	gui_pid=$!
	win="$(wait_window --class md-reader)"
	sleep 3
	check "4.7 xdotool findet sichtbares Fenster der Klasse md-reader" test -n "$win"
	if [ -n "$win" ]; then
		wmclass="$(xprop -id "$win" WM_CLASS)"
		info "Paket-GUI: $wmclass; $(xprop -id "$win" WM_NAME)"
		check "4.8 WM_CLASS = \"md-reader\", \"md-reader\"" \
			test "$wmclass" = 'WM_CLASS(STRING) = "md-reader", "md-reader"'
	else
		fail "4.8 WM_CLASS (kein Fenster)"
		info "Paket-GUI Protokoll: $(oneline /out/gui-deb.log)"
	fi
	comm="$(cat "/proc/$gui_pid/comm" 2>/dev/null || true)"
	check "4.9 Prozessname md-reader (ist: $comm)" test "$comm" = "md-reader"
	shot deb
	check "4.10 Bildschirmfoto deb.xwd geschrieben" test -s /out/deb.xwd
	kill "$gui_pid" 2>/dev/null || true
	wait "$gui_pid" 2>/dev/null || true
	sleep 1

	# --- 5: Starter mit installiertem WebKit, Paket installiert ----------
	mkdir -p /tmp/h5
	HOME=/tmp/h5 /b/md-reader /doc/README.md >/out/launcher-deb.out 2>/out/launcher-deb.err &
	launcher_pid=$!
	win="$(wait_window --class md-reader)"
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
	shot launcher-deb
	# Das .deb zieht zenity als Empfehlung mit: eine Frage käme als Fenster.
	offer_win="$(xdotool search --onlyvisible --name "$OFFER_PATTERN" 2>/dev/null | head -n 1 || true)"
	kill "$launcher_pid" 2>/dev/null || true
	wait "$launcher_pid" 2>/dev/null || true
	sleep 1
	info "Starter stderr: $(oneline /out/launcher-deb.err)"
	check "5.3 stderr ohne Installationsfrage" bash -c "! grep -Eq '$OFFER_PATTERN' /out/launcher-deb.err"
	check "5.3b kein Fragefenster (zenity) sichtbar" test -z "$offer_win"
	check "5.4 stderr ohne WebKit-Hinweis und ohne „zu alt“" \
		bash -c "! grep -Eq 'WebKitGTK|$TOO_OLD_PATTERN' /out/launcher-deb.err"
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
	win="$(wait_window --class md-reader)"
	offer_win="$(wait_window --name "$OFFER_PATTERN")"
	sleep 1
	shot launcher-nodeb
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
}

# ---------------------------------------------------------------------------
# Modus "f1": Kontextmenü gesperrt, Fenster navigiert nie weg (F1).
# ---------------------------------------------------------------------------
run_f1() {
	install_deb_and_tools python3
	under_xvfb f1-display
}

# server_requests zählt die Anfragen im Protokoll des HTTP-Servers.
server_requests() {
	grep -c '"GET ' /out/http.log 2>/dev/null || true
}

run_f1_display() {
	local http_pid gui_pid win name0 name base after after2 req0 zpid zwin i
	info "xvfb-run: DISPLAY=$DISPLAY"

	# Lokaler HTTP-Server, nur an 127.0.0.1 (Container-Netz, keine Ports veröffentlicht).
	mkdir -p /tmp/www
	echo "F1-ZIEL" >/tmp/www/clicked
	python3 -u -m http.server "$F1_PORT" --bind 127.0.0.1 --directory /tmp/www >/out/http.log 2>&1 &
	http_pid=$!
	sleep 1
	python3 -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:$F1_PORT/probe-server-alive', timeout=5)" 2>/dev/null || true
	check "F1.0 HTTP-Server protokolliert Anfragen (Probe)" grep -q 'probe-server-alive' /out/http.log
	req0="$(server_requests)"

	# Fake-xdg-open vor /usr/bin im PATH: protokolliert nur die Argumente.
	cat >/usr/local/bin/xdg-open <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >>/out/xdg-open.log
EOF
	chmod 755 /usr/local/bin/xdg-open
	: >/out/xdg-open.log

	# Dokument: nur eine Überschrift, die ein Link ist (bekannte Position).
	mkdir -p /tmp/f1
	printf '# [TESTLINK-F1](%s)\n\nAbsatz ohne Link.\n' "$F1_URL" >/tmp/f1/link.md

	/usr/bin/md-reader /tmp/f1/link.md >/out/gui-f1.log 2>&1 &
	gui_pid=$!
	win="$(wait_window --class md-reader)"
	sleep 3
	check "F1.1 GUI mit Linkdokument öffnet ein Fenster" test -n "$win"
	if [ -z "$win" ]; then
		info "GUI-Protokoll: $(oneline /out/gui-f1.log)"
		return
	fi
	name0="$(xprop -id "$win" WM_NAME)"
	shot f1-start

	# Rechtsklick auf den Link: kein Kontextmenü.
	base="$(count_viewable)"
	xdotool mousemove "$F1_LINK_X" "$F1_LINK_Y" click 3
	sleep 1.5
	after="$(count_viewable)"
	shot f1-rightclick
	info "sichtbare Fenster vor/nach Rechtsklick: $base/$after"
	check "F1.2 Rechtsklick auf Link: kein Kontextmenü" test "$after" -eq "$base"
	info "xdg-open nach Rechtsklick: $(oneline /out/xdg-open.log)"
	check "F1.3 Rechtsklick allein: kein xdg-open" test ! -s /out/xdg-open.log
	# Menü „bedienen“, falls doch eines da wäre: erster Eintrag, dann Escape.
	# (Ohne Menü kann Return den fokussierten Link auslösen – das ginge an
	# xdg-open und ist keine Navigation; bewertet wird nur der Server.)
	xdotool key --clearmodifiers Down Return
	sleep 2
	xdotool key --clearmodifiers Escape
	sleep 1
	info "xdg-open nach Down+Return: $(oneline /out/xdg-open.log)"

	# Tastatur-Kontextmenü (Shift+F10 und Menütaste) im Fenster.
	xdotool windowfocus "$win" 2>/dev/null || true
	xdotool key --clearmodifiers shift+F10
	sleep 1
	after="$(count_viewable)"
	xdotool key --clearmodifiers Menu
	sleep 1
	after2="$(count_viewable)"
	info "sichtbare Fenster nach Shift+F10/Menütaste: $after/$after2"
	check "F1.4 Shift+F10/Menütaste: kein Kontextmenü" \
		bash -c "test '$after' -eq '$base' && test '$after2' -eq '$base'"
	xdotool key --clearmodifiers Escape

	# Mittelklick: weder neues Fenster noch Navigation.
	xdotool mousemove "$F1_LINK_X" "$F1_LINK_Y" click 2
	sleep 2
	after="$(count_viewable)"
	info "xdg-open nach Mittelklick: $(oneline /out/xdg-open.log)"
	check "F1.5 Mittelklick auf Link: kein neues Fenster" test "$after" -eq "$base"
	check "F1.6 Rechts-/Mittelklick, Menütasten: keine Anfrage am Server" \
		test "$(server_requests)" -eq "$req0"

	# Linksklick: geht an den externen Browser (Fake-xdg-open), Fenster bleibt.
	: >/out/xdg-open.log
	xdotool mousemove "$F1_LINK_X" "$F1_LINK_Y" click 1
	sleep 3
	shot f1-leftclick
	info "xdg-open nach Linksklick: $(oneline /out/xdg-open.log)"
	check "F1.7 Linksklick: Link geht an xdg-open ($F1_URL)" grep -qx "$F1_URL" /out/xdg-open.log
	check "F1.8 Linksklick: keine Anfrage am Server (Fenster navigiert nicht)" \
		test "$(server_requests)" -eq "$req0"
	name="$(xprop -id "$win" WM_NAME 2>/dev/null || true)"
	check "F1.9 Fenster unverändert ($name0)" test "$name" = "$name0"
	check "F1.10 GUI läuft noch" kill -0 "$gui_pid"
	info "HTTP-Protokoll: $(oneline /out/http.log)"

	# Gegenprobe der Messmethode: Ein GTK-Eingabefeld (zenity --entry) zeigt
	# mit Shift+F10 sein Kontextmenü – count_viewable muss es sehen.
	zenity --entry --title "F1-Gegenprobe" --text "Gegenprobe" >/dev/null 2>&1 &
	zpid=$!
	zwin="$(wait_window --name 'F1-Gegenprobe')"
	sleep 1
	base="$(count_viewable)"
	if [ -n "$zwin" ]; then
		xdotool windowfocus "$zwin" 2>/dev/null || true
		sleep 0.5
		xdotool key --clearmodifiers shift+F10
		sleep 1.5
	fi
	after="$(count_viewable)"
	shot f1-countercheck
	info "Gegenprobe zenity: sichtbare Fenster vor/nach Shift+F10: $base/$after"
	check "F1.11 Gegenprobe: Messmethode erkennt ein GTK-Kontextmenü" test "$after" -gt "$base"
	kill "$zpid" 2>/dev/null || true
	kill "$gui_pid" 2>/dev/null || true
	kill "$http_pid" 2>/dev/null || true
	for i in 1 2 3; do
		wait 2>/dev/null || true
	done
}

# ---------------------------------------------------------------------------
# Modus "toold": System älter als die glibc-Grenze (z. B. ubuntu:20.04).
# ---------------------------------------------------------------------------
run_toold() {
	export DEBIAN_FRONTEND=noninteractive
	set +e
	apt-get update >/out/apt-deb.log 2>&1
	apt-get install -y "$(deb_path)" >>/out/apt-deb.log 2>&1
	local code=$?
	set -e
	info ".deb auf altem System: Exit $code, $(grep -E 'Depends: libc6|unmet' /out/apt-deb.log | tr '\n' '|')"
	check "T.1 apt lehnt das .deb ab (libc6 zu alt)" \
		bash -c "test '$code' -ne 0 && grep -q 'Depends: libc6 (>= ' /out/apt-deb.log"
	check "T.2 Prüfwerkzeuge und zenity installiert" \
		apt_quiet apt-tools xvfb xauth x11-utils xdotool x11-apps zenity
	under_xvfb toold-display
}

run_toold_display() {
	local launcher_pid win offer_win gui_win code i
	mkdir -p /tmp/ht
	HOME=/tmp/ht /b/md-reader /doc/README.md >/out/toold.out 2>/out/toold.err &
	launcher_pid=$!
	win="$(wait_window --name "$TOO_OLD_PATTERN")"
	sleep 1
	shot toold
	offer_win="$(xdotool search --onlyvisible --name "$OFFER_PATTERN" 2>/dev/null | head -n 1 || true)"
	gui_win="$(xdotool search --onlyvisible --class md-reader 2>/dev/null | head -n 1 || true)"
	check "T.3 Dialog „zu alt“ erscheint (Desktop-Start)" test -n "$win"
	if [ -n "$win" ]; then
		info "Dialog: $(xprop -id "$win" WM_NAME WM_CLASS | tr '\n' ' ')"
		xdotool windowfocus --sync "$win" 2>/dev/null || xdotool windowfocus "$win" || true
		xdotool key --clearmodifiers Return
	fi
	check "T.4 kein Installationsangebot" test -z "$offer_win"
	check "T.5 kein GUI-Fenster" test -z "$gui_win"
	code=""
	for i in $(seq 1 30); do
		if ! kill -0 "$launcher_pid" 2>/dev/null; then
			set +e
			wait "$launcher_pid"
			code=$?
			set -e
			break
		fi
		sleep 0.5
	done
	if [ -z "$code" ]; then
		kill "$launcher_pid" 2>/dev/null || true
		code="läuft noch"
	fi
	info "Starter auf altem System mit Display: Exit $code, stderr: $(oneline /out/toold.err)"
	check "T.6 Starter endet mit Exit 1" test "$code" = "1"
	check "T.7 stderr nennt „zu alt“ genau einmal" \
		test "$(grep -Ec "$TOO_OLD_PATTERN" /out/toold.err)" -eq 1
	check "T.8 keine Nutzerinstallation, kein launcher.json" bash -c "
		! test -e /tmp/ht/.local/share/applications/md-reader.desktop &&
		! test -e /tmp/ht/.config/md-reader/launcher.json"
}

case "${1:-}" in
	install) run_install ;;
	deb) run_deb ;;
	deb-display) run_deb_display ;;
	f1) run_f1 ;;
	f1-display) run_f1_display ;;
	toold) run_toold ;;
	toold-display) run_toold_display ;;
	count-viewable)
		count_viewable
		exit 0
		;;
	*)
		echo "Aufruf: $0 install|deb|f1|toold" >&2
		exit 2
		;;
esac

# Dateien für den Nutzer außerhalb lesbar machen (Container läuft als root).
chmod -R a+rX /out 2>/dev/null || true
if [ "$FAILS" -gt 99 ]; then
	FAILS=99
fi
exit "$FAILS"
