# Bugs

## Offen 🐛

- **#001**: Windows Cross-Compilation benötigt EventToken.h Stub (WebView2 SDK Header fehlt in mingw-w64). Workaround: Stub-Datei wurde erstellt. Native Windows-Compilation ohne Stub empfohlen.

## Behoben ✅

- **#002** (Build 7): Vollbild-Modus zeigte OS-Fensterrahmen bei HTML5 Fullscreen API. Fix: `toggleFullscreen()` nutzt jetzt ausschließlich `nativeFullscreen()` (GTK auf Linux, WinAPI auf Windows). Der `fullscreenchange`-Event-Handler wurde entfernt, der neue State kommt direkt als Rückgabewert des Bindings zurück.
