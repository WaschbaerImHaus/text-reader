# Bugs

## Offen 🐛

(Keine offenen Bugs)

## Behoben ✅

- **#001** (Build 11): Windows Cross-Compilation benötigt EventToken.h Stub (WebView2 SDK Header fehlt in mingw-w64). Workaround per EventToken.h-Stub im GOPATH bestätigt funktionstüchtig. Native Windows-Compilation ohne Stub bleibt empfohlen.

- **#002** (Build 7): Vollbild-Modus zeigte OS-Fensterrahmen bei HTML5 Fullscreen API. Fix: `toggleFullscreen()` nutzt jetzt ausschließlich `nativeFullscreen()` (GTK auf Linux, WinAPI auf Windows). Der `fullscreenchange`-Event-Handler wurde entfernt, der neue State kommt direkt als Rückgabewert des Bindings zurück.

- **#003** (Build 9): Per Drag & Drop geöffnete Dateien wurden beim nächsten Start nicht wiederhergestellt. Fix: Dateipfad wird aus dem `text/uri-list`-DataTransfer-Eintrag des Drop-Events extrahiert (Desktop-WebViews geben `file://`-URIs zurück) und per neuem `persistLastFile`-Binding in der Konfiguration gespeichert.

- **#004** (Build 9): Scroll-Position wurde beim Schließen nicht gespeichert und beim nächsten Start nicht wiederhergestellt. Fix: Scroll-Position wird debounced (500ms) bei jedem Scroll-Event und synchron beim Schließen gespeichert. Beim Start wird sie nach dem Rendern wiederhergestellt.

- **#005** (Build 13): Auf Windows wurde `lastFile` nach Drag & Drop nicht gespeichert. Ursache: WebView2 exponiert aus Sicherheitsgründen **keine Dateipfade** im DataTransfer – `text/uri-list` bleibt in WebView2 bewusst leer. Fix: Nativer Datei-Öffnen-Dialog via `GetOpenFileNameW` (Windows) / GTK FileChooser (Linux), aufrufbar per 📂-Button oder Strg+O. Der Drag & Drop zeigt Dateien weiterhin an, aber nur der native Dialog speichert `lastFile` zuverlässig.

- **#006** (Build 18): Zoom-Anzeige zeigte beim Start immer 100%, obwohl der Inhalt bei z.B. 200% blieb. Ursache: `{{DEFAULT_FONT_SIZE}}` wurde in `ui/template.go` fälschlicherweise mit dem gespeicherten `FontSize` (z.B. 32) ersetzt statt mit der Konstante 16. Dadurch war `defaultFontSize = 32` und `Math.round(32/32*100) = 100%`. Fix: `UIConfig` um Feld `DefaultFontSize` erweitert; `main.go` übergibt die Konstante `defaultFontSize = 16` getrennt vom gespeicherten `FontSize`.

- **#007** (Build 18): Beim Windows-Start erschien ein Konsolenfenster. Ursache: Fehlender Linker-Flag `-H windowsgui` im Windows-Build. Fix: `go build -ldflags="-H windowsgui"` für Windows-Ziel.

- **#008** (Build 18): App-Icon war im Windows Explorer nicht sichtbar. Ursache: Die `.ico`-Datei enthielt nur 16×16 und 32×32 – Windows Explorer benötigt 48×48 und 256×256. Fix: `.ico` mit ImageMagick auf 4 Größen (256, 48, 32, 16) regeneriert; `.syso` neu kompiliert.
