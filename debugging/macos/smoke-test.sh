#!/bin/bash
# Rauchtest für das macOS-App-Bundle (läuft im Workflow macos.yml).
#
# Startet "MD Reader.app" zweimal mit eigenem HOME (normale Ansicht und
# Text-Modus), prüft, dass der Prozess nach dem Laden noch lebt, und legt
# je ein Bildschirmfoto in build/ ab (als Workflow-Artefakt abrufbar).
# Im ersten Fall zusätzlich ⌘A und das Menü "Bearbeiten" (Menüleiste).
#
# Aufruf (aus dem Projektroot): debugging/macos/smoke-test.sh
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-24
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APP="$ROOT/build/MD Reader.app"
SAMPLE="$(mktemp -d)/sample.md"

cat > "$SAMPLE" <<'MD'
# MD Reader auf macOS

Ein Absatz mit **fett**, *kursiv* und `Code`. Umlaute: äöü ß ÄÖÜ.

| Spalte | Wert |
|--------|------|
| A      | 1    |
| B      | 2    |

Formel: $E = mc^2$

```go
fmt.Println("Hallo")
```
MD

# run_case startet die App wie aus dem Finder (open, damit sie aktiv wird und
# ihre Menüleiste zeigt) und macht ein Bildschirmfoto.
#   $1 Name des Falls (für den Dateinamen)
#   $2 Inhalt der state.json
#   $3 optional "keys": zusätzlich ⌘A und Menü "Bearbeiten" prüfen
run_case() {
	local name="$1" state="$2" extra="${3:-}"
	local home pid
	home="$(mktemp -d)"
	mkdir -p "$home/Library/Application Support/md-reader"
	printf '%s' "$state" > "$home/Library/Application Support/md-reader/state.json"

	open -n --env "HOME=$home" --stdout "$ROOT/build/macos-$name.log" \
		--stderr "$ROOT/build/macos-$name.log" -a "$APP" --args "$SAMPLE"
	sleep 10
	pid="$(pgrep -nf "$APP/Contents/MacOS/md-reader" || true)"
	if [ -z "$pid" ]; then
		echo "Fehler: App ($name) läuft nicht:" >&2
		cat "$ROOT/build/macos-$name.log" >&2 || true
		exit 1
	fi
	screencapture -x "$ROOT/build/macos-$name.png" || echo "Hinweis: screencapture fehlgeschlagen"

	if [ "$extra" = "keys" ]; then
		# Braucht Bedienungshilfen-Rechte für System Events; ohne sie nur Hinweis
		if osascript -e 'tell application "System Events" to keystroke "a" using command down'; then
			sleep 1
			screencapture -x "$ROOT/build/macos-$name-selectall.png" || true
			osascript -e 'tell application "System Events" to tell process "md-reader" to click menu bar item "Bearbeiten" of menu bar 1' || true
			sleep 1
			screencapture -x "$ROOT/build/macos-$name-menu.png" || true
			osascript -e 'tell application "System Events" to key code 53' || true
		else
			echo "Hinweis: System Events nicht erlaubt, Tastenprüfung übersprungen"
		fi
	fi

	kill "$pid" 2> /dev/null || true
	sleep 1
	echo "✓ Fall $name: App lief stabil"
}

run_case normal '{"theme":"light","fontSize":16,"layout":"landscape"}' keys
run_case textmode '{"theme":"light","fontSize":16,"layout":"landscape","textMode":true}'
