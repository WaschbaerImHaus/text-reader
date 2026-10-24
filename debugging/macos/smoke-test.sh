#!/bin/bash
# Rauchtest für das macOS-App-Bundle (läuft im Workflow macos.yml).
#
# Startet "MD Reader.app" zweimal mit eigenem HOME (normale Ansicht und
# Text-Modus), prüft, dass der Prozess nach dem Laden noch lebt, und legt
# je einen Bildschirmfoto in build/ ab (als Workflow-Artefakt abrufbar).
#
# Aufruf (aus dem Projektroot): debugging/macos/smoke-test.sh
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-24
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BIN="$ROOT/build/MD Reader.app/Contents/MacOS/md-reader"
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

# run_case startet die App mit einer Konfiguration und macht ein Bildschirmfoto.
#   $1 Name des Falls (für den Dateinamen)
#   $2 Inhalt der state.json
run_case() {
	local name="$1" state="$2"
	local home
	home="$(mktemp -d)"
	mkdir -p "$home/Library/Application Support/md-reader"
	printf '%s' "$state" > "$home/Library/Application Support/md-reader/state.json"

	HOME="$home" "$BIN" "$SAMPLE" > "$ROOT/build/macos-$name.log" 2>&1 &
	local pid=$!
	sleep 10
	if ! kill -0 "$pid" 2> /dev/null; then
		echo "Fehler: App ($name) ist abgestürzt:" >&2
		cat "$ROOT/build/macos-$name.log" >&2
		exit 1
	fi
	screencapture -x "$ROOT/build/macos-$name.png" || echo "Hinweis: screencapture fehlgeschlagen"
	kill "$pid" 2> /dev/null || true
	wait "$pid" 2> /dev/null || true
	echo "✓ Fall $name: App lief stabil"
}

run_case normal '{"theme":"light","fontSize":16,"layout":"landscape"}'
run_case textmode '{"theme":"light","fontSize":16,"layout":"landscape","textMode":true}'
