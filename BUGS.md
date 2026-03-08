# Bugs

## Offen 🐛

(Keine offenen Bugs)

## Behoben ✅

- **#001** (Build 11): Windows Cross-Compilation benötigt EventToken.h Stub (WebView2 SDK Header fehlt in mingw-w64). Workaround per EventToken.h-Stub im GOPATH bestätigt funktionstüchtig. Native Windows-Compilation ohne Stub bleibt empfohlen.

- **#002** (Build 7): Vollbild-Modus zeigte OS-Fensterrahmen bei HTML5 Fullscreen API. Fix: `toggleFullscreen()` nutzt jetzt ausschließlich `nativeFullscreen()` (GTK auf Linux, WinAPI auf Windows). Der `fullscreenchange`-Event-Handler wurde entfernt, der neue State kommt direkt als Rückgabewert des Bindings zurück.

- **#003** (Build 9): Per Drag & Drop geöffnete Dateien wurden beim nächsten Start nicht wiederhergestellt. Fix: Dateipfad wird aus dem `text/uri-list`-DataTransfer-Eintrag des Drop-Events extrahiert (Desktop-WebViews geben `file://`-URIs zurück) und per neuem `persistLastFile`-Binding in der Konfiguration gespeichert.

- **#004** (Build 9): Scroll-Position wurde beim Schließen nicht gespeichert und beim nächsten Start nicht wiederhergestellt. Fix: Scroll-Position wird debounced (500ms) bei jedem Scroll-Event und synchron beim Schließen gespeichert. Beim Start wird sie nach dem Rendern wiederhergestellt.
