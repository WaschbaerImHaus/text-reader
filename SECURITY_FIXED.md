# Behobene Sicherheitsprobleme

## Behoben ✅

### RISK-013 (Fund F1): Kontextmenü navigierte das App-Fenster zu fremden Seiten (Build 47 – 2026-11-01)
**Problem** (Audit Build 47, unter Linux nachgewiesen): Rechtsklick auf einen Link → WebKitGTK-Menüpunkt „Verweis öffnen“ lud die fremde Seite ins **App-Fenster** – am JS-Klick-Handler (RISK-008) vorbei. Die Dokument-CSP gilt dort nicht, webview_go stellt die Go-Bindings jeder Seite bereit (Beacon mit `openExternal/loadNativeFile/persistState/openFilePicker: function`). Folgen: Phishing-Oberfläche im MD-Reader-Fenster, `openExternal`/`loadNativeFile`/`openFilePicker` aus fremdem Code.
**Fix (zwei Schichten)**:
1. **Nativer Navigationsschutz (Linux)**: `decide-policy`-Handler am WebKitWebView (`platform_linux.go`, vor dem ersten `SetHtml` verbunden). `NAVIGATION_ACTION` nur für die eigene Seite (`about:blank` – so lädt webview_go `SetHtml` unter WebKitGTK, gemessen) und genau die eigene Temp-Ansichtsdatei (`ui.TempViewFileURL`), jeweils auch mit Anker; alles andere wird abgelehnt (Erlaubnisliste `ui.AllowNavigation`, fail closed). `NEW_WINDOW_ACTION` wird immer abgelehnt. Abgelehnte http/https/mailto-Ziele öffnet `handleNavigationRequest` bei Nutzeraktion (Geste oder `LINK_CLICKED`, so meldet WebKitGTK „Verweis öffnen“) über `extlink` im Standardprogramm. Zusätzlich unterdrückt das GTK-Signal `context-menu` (Rückgabe TRUE) das Menü nativ – wirkt auch, bevor das Seitenskript geladen ist.
2. **JS (alle Plattformen)**: Nach Nutzerentscheidung (2026-11-01) wird das Kontextmenü im **gesamten** App-Fenster unterdrückt, nicht nur auf Links: `suppressContextMenu` (bedingungslos `preventDefault`) als `contextmenu`-Listener auf `document` in der Capture-Phase, verdrahtet in `initUIListeners` (keine Inline-Handler, CSP). Die App bietet nur das System-Menü an, es wird nicht gebraucht. Kopieren per Strg+C/⌘C bleibt unberührt (unter Xvfb geprüft).
Windows/macOS haben nur Schicht 2 – Restrisiko siehe SECURITY_RISKS.md (RISK-013).

**Dateien**: `src/platform_linux.go`, `src/platform_linux_nav.go` (neu), `src/platform_windows.go`, `src/platform_darwin.go` (No-Op), `src/main.go`, `src/bindings.go`, `src/ui/viewfile.go`, `src/ui/assets/scripts.js`
**Tests**: `TestAllowNavigation`, `TestAllowNavigation_NoTempViewURL`, `TestTempViewFileURL`, `TestHandleNavigationRequest`, `TestScripts_ContextMenuSuppressed`; GUI-Gegenprobe unter Xvfb (Server auf 127.0.0.1, Rechtsklick + erster Menüpunkt): vorher GET + Beacon, nachher kein Request – je Schicht einzeln nachgewiesen.

### RISK-014 (Fund F2): Gefährliche Link-Schemata in Markdown (Build 47 – 2026-11-01)
**Problem**: CVE-2026-5160 – goldmark 1.7.8 machte aus `[a](javascript&colon;…)`, `[b](&#106;avascript:…)`, `![i](javascript&colon;…)` und dem Autolink `<javascript:…>` echte `javascript:`-Ziele. Ausführung blockierten Link-Handler und CSP, aber die Invariante „Links nur http/https/mailto/#/relativ“ galt für Markdown nicht. Auch goldmark 1.8.7 filtert nur per Sperrliste (`smb:` – NTLM-Leck unter Windows –, `ms-settings:`, `ftp:`, `tel:` kommen durch).
**Fix**: goldmark 1.7.8 → 1.8.7, golang.org/x/net 0.61.0 → 0.62.0 (Scanner #93/#94, go-Direktive bleibt 1.26.0). Neuer AST-Transformer `linkAllowListTransformer` (`renderer/markdown.go`): Link- und Autolink-Ziele müssen `isSafeLinkURL` bestehen (dieselbe Erlaubnisliste wie `SanitizeHTML`), Bildziele `isSafeMarkdownImageURL` (relativ, absoluter lokaler Pfad, `C:\…`, `data:image/`). Geprüft wird roh und nach Entity-Entschlüsselung. Abgelehnte Links/Bilder werden ausgepackt: Text bzw. Alternativtext bleibt, der Verweis entfällt. Folge: externe Bilder (`![x](https://…)`) erscheinen als Alternativtext statt als (von der CSP ohnehin blockiertes) kaputtes Bild.

**Dateien**: `src/renderer/markdown.go`, `src/go.mod`, `src/go.sum`
**Tests**: `TestParseMarkdown_DangerousLinkSchemesRemoved` (auch mit goldmark 1.7.8 grün = Transformer wirkt allein), `TestParseMarkdown_SafeLinksKept`, `TestIsSafeMarkdownImageURL`

### RISK-015 (Fund F3): Installation kopierte die Datei unter dem Pfad statt der laufenden Datei (Build 47 – 2026-11-01)
**Problem** (TOCTOU): Der Angebotsprozess fragt bis zu 120 s und kopierte danach die Datei unter dem Pfad aus `os.Executable`. Wurde sie in dieser Zeit ersetzt (fremd beschreibbarer Ordner), landete die fremde Datei als `~/.local/bin/md-reader` samt Startmenüeintrag.
**Fix**: Quelle ist immer die laufende Datei selbst: `/proc/self/exe` wird beim Start des Angebots (vor der Frage) bzw. bei `--install` geöffnet (`launcher.openSelf`) und als `*os.File` an `linuxsetup.Install(l, self, build, icons)` / `HandleInstallOffer(e, c, self, icons)` übergeben; gelesen wird per `ReadAt` ab Position 0, „schon installiert“ per `fstat` + `os.SameFile`. Der Pfad dient nur noch `Decide`.

**Dateien**: `src/linuxsetup/install.go`, `src/linuxsetup/flow.go`, `src/launcher/main.go`
**Tests**: `TestOfferInstall_CopiesRunningBinaryNotPath`, `TestRun_InstallOpenSelfFails` (bestehende Install-/Flow-Tests auf die neue Signatur umgestellt)

### Härtungen H1–H5 (Build 47 – 2026-11-01, kein Angreiferpfad)
- **H1** Steuerzeichen in Installationspfaden: `linuxsetup.ValidatePath` lehnt leere Pfade, ungültiges UTF-8 und alle Steuerzeichen (< 0x20, 0x7f, C1) ab; `Install` prüft Programm-, Desktop- und Icon-Pfad, bevor etwas angelegt wird; `--print-desktop-entry` mit solchem Pfad → Exit 2. Verhindert eingeschleuste Schlüssel (`Exec=`) in der Desktop-Datei. Tests: `TestInstall_RejectsControlCharsInBinPath`, `TestValidatePath`, `TestRun_PrintDesktopEntryRejectsControlChars`.
- **H2** Desktop-Datei und Icons werden wie das Programm atomar geschrieben (`writeFileAtomic`: Altlast `.tmp` entfernen, `.tmp` mit `O_CREATE|O_EXCL`, Modus setzen, `Rename`). Symlinks an Ziel oder `.tmp` werden ersetzt, nie verfolgt; kein halber Eintrag nach Abbruch. `O_NOFOLLOW` gibt es im Paket `syscall` unter Windows nicht (linuxsetup muss dort übersetzbar bleiben); `O_EXCL` folgt laut POSIX nie einem Symlink und deckt das ab. Test: `TestInstall_DesktopAndIconsReplaceSymlinks`.
- **H3** `pkexec`/`apt-get` nur noch als `/usr/bin/pkexec` und `/usr/bin/apt-get` (`PkexecPath`, `AptGetPath`), keine PATH-Suche; fehlen sie dort, gibt es nur den Hinweis. Tests: `TestAptCommandsAreFixed`, `TestHandleMissing_PkexecOnlyInPATH`.
- **H4** Makefile: `GLIBC_MAX := 2.39`; eine Umgebungsvariable `GLIBC_MAX` (auch mit `make -e`) bricht mit `$(error)` ab, `make GLIBC_MAX=2.30 …` auf der Befehlszeile (Wächtertest) funktioniert weiter. Geprüft per `make -n`.
- **H5** Starter: `--` beendet die Optionen (`md-reader -- --uninstall` startet die Oberfläche mit dem Argument `--uninstall`), im Dateikopf von `launcher/main.go` beschrieben. Tests: `TestParseCommand` (4 neue Fälle), `TestRun_DoubleDashEndsOptions`.

### RISK-011: Umgehung der Gerätenamen-Sperre bei Bildpfaden (Build 44 – 2026-10-22)
**Problem** (Security-Review zu Build 43): Windows entfernt Leerzeichen/Punkte am Segmentende – `COM1 .png` zeigt trotzdem auf die serielle Schnittstelle; `CONIN$`, `CONOUT$`, `CLOCK$` und hochgestellte Ziffern (`COM¹`) fehlten; nur der Dateiname, nicht die Verzeichnisse wurden geprüft. Folge: Lesen eines Geräts kann die App blockieren (DoS).
**Fix**: `isSafePathSegment` prüft jedes Pfadsegment: Segmente mit Leerzeichen/Punkt am Ende werden abgelehnt (außer `.`/`..`), Gerätename = Teil vor dem ersten Punkt ohne Leerzeichen am Ende; Sperrliste um `conin$`, `conout$`, `clock$`, `com¹²³`, `lpt¹²³` ergänzt.

**Datei**: `src/renderer/images.go`
**Tests**: `TestIsReadableImagePath_DeviceNameEvasionsRejected`

### RISK-010: Umgehung der Netzwerkpfad-Erkennung bei Bildpfaden (Build 43 – 2026-10-14)
**Problem** (Security-Review zu Build 42): `isNetworkPath` arbeitete als Sperrliste (`//`, `\\` …). Unter Windows führen weitere Schreibweisen ins Netzwerk, z. B. `\??\UNC\server\share\x.png` (Go hält ihn für absolut, Windows reicht ihn an SMB weiter) → NTLM-Hash-Leck. Für Markdown praktisch nicht ausnutzbar (goldmark kodiert Backslashes als `%5C`), aber nicht robust.
**Fix**: `isReadableImagePath` als plattformunabhängige Erlaubnisliste (fail closed): nur relative Pfade, Laufwerksbuchstabe mit Trenner (`C:\…`) oder einfacher Wurzel-Trenner; abgelehnt werden `\\…`, `//…`, `\??\…` (inkl. `\\?\`, `\\.\`), laufwerksrelative Pfade (`C:bild.png`), NTFS-Datenströme (`bild.png:x`) und reservierte Gerätenamen (`COM1.png`, `NUL.png` …).

**Datei**: `src/renderer/images.go`
**Tests**: `images_internal_test.go` (3 Tests, inkl. Windows-Fälle unter Linux)

### RISK-008: Externe Links öffnen fremde Websites im App-Fenster (Build 42 – 2026-10-14)
**Problem**: Ein Klick auf einen http(s)-Link navigierte das App-Fenster zur fremden Website; webview_go injiziert die Go-Bindings in jede Seite.
**Fix**: Neues Paket `extlink` (Typ `Opener`): strenge URL-Prüfung (nur http/https mit Host und mailto, keine Steuer-/Leer-/Anführungszeichen, max. 2048 Zeichen) und Start des Standardprogramms ohne Shell (`xdg-open` bzw. `rundll32 url.dll,FileProtocolHandler`). Neues Binding `openExternal`; der Link-Handler in `scripts.js` verhindert jede Navigation des App-Fensters (auch `data:`, `file:`, Mittelklick) und übergibt nur http/https/mailto an Go. Verifiziert in WebKitGTK mit protokollierendem Fake-`xdg-open`.

**Dateien**: `src/extlink/extlink.go`, `src/bindings.go`, `src/ui/assets/scripts.js`
**Tests**: `extlink_test.go` (8), `TestScripts_ExternalLinksOpenedNatively`

### RISK-009: NTLM-Hash-Leck über file:-Ressourcen (Build 42 – 2026-10-14)
**Problem** (Security-Review zu Build 41): `style="background:url(file://server/…)"` aus EPUBs und `img-src file:` in der CSP; Markdown-Bilder mit UNC-Pfad. Unter Windows → SMB-Zugriff auf fremden Server → NTLM-Hash kann abfließen.
**Fix**: (1) Bereiniger verwirft `style`-Werte mit `url`, `image`, `import`, `expression`, `behavior`, `binding`, `position`, Backslash (CSS-Escapes), `@` oder `/*`; einfache Stile bleiben. (2) CSP `img-src data:` (ohne `file:`/`blob:`). (3) `ResolveImagePaths` bettet jetzt auch absolute lokale Bildpfade ein (funktionieren dadurch überall), liest UNC-/Netzwerkpfade nie und bettet nur echte Bilddateien ein (feste Endungstabelle; vorher wurde jede Datei als „Bild“ eingebettet).

**Dateien**: `src/renderer/sanitize.go`, `src/renderer/images.go`, `src/ui/template.go`
**Tests**: `TestSanitizeHTML_StyleWithResourceLoadsRemoved`, `TestSanitizeHTML_SimpleStylesKept`, `TestResolveImagePaths{AbsoluteLocalFileEmbedded,UNCNeverRead,OnlyImageTypes}`, `TestBuildInitialHTML_CSPImagesOnlyData`

### RISK-006/007: Skript-Ausführung über EPUB- und LaTeX-Dateien (Build 41 – 2026-10-14)
**Problem**: Präparierte EPUB-Dateien (Event-Attribute, `javascript:`-Links, `<iframe srcdoc>`, `<meta refresh>`) und LaTeX-Dateien (ungefilterter Fließtext, `\href{javascript:…}`) konnten JavaScript im Viewer ausführen und die Go-Bindings aufrufen. Nachgewiesen in Chrome-Headless (Build 40: `onerror` lief).
**Fix (zweistufig)**:
1. **Bereinigung**: Neuer Whitelist-Bereiniger `renderer.SanitizeHTML` (Tokenizer `golang.org/x/net/html`): nur bekannte Elemente/Attribute, Links nur http/https/mailto/#/relativ (Prüfung nach Entfernen von Steuerzeichen, Entities werden vorher entschlüsselt), Bilder nur `data:image/…`, gefährliche Container (script, style, iframe, object, template, foreignObject …) samt Inhalt entfernt, Kommentare verworfen, Text und Attribute neu escaped. Angewendet auf das gesamte EPUB-HTML und die LaTeX-Ausgabe; LaTeX-Formeln werden zusätzlich HTML-escaped, Makros als nicht ausführbarer JSON-Datenblock übergeben.
2. **Strikte CSP**: `script-src` ohne `'unsafe-inline'`, stattdessen Zufalls-Nonce (128 Bit) je Seite für die eigenen Skripte und exakt `https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js`; zusätzlich `base-uri 'none'`, `form-action 'none'`, `frame-src 'none'`. Alle Inline-Handler (`onclick=` …) im Template durch `addEventListener` ersetzt. Eingeschleuste Handler werden damit auch dann blockiert, wenn die Bereinigung etwas übersehen sollte (in Chrome nachgewiesen).

**Dateien**: `src/renderer/sanitize.go`, `src/renderer/epub.go`, `src/renderer/latex.go`, `src/ui/template.go`, `src/ui/assets/body.html`, `src/ui/assets/scripts.js`
**Tests**: `sanitize_test.go` (17), `TestParseEpubMalicious*`, `TestParseLaTeX{RawHTMLSanitized,JavascriptLinksRemoved,MathEscaped,MacrosJSONEscaped}`, `TestBuildInitialHTML_{CSPScriptSrcStrict,CSPExtraDirectives,AllScriptsHaveNonce,NonceRandomPerPage}`, `TestBodyTemplate_NoInlineEventHandlers`, `TestScripts_ButtonsWiredByListener`

### RISK-005: Skript-Injection über den Seitentitel (Build 40 – 2026-10-14)
**Problem**: Der Titel (erste `# `-Zeile der Datei) wurde unescaped in `<title>` eingesetzt. Eine Datei mit `# </title><script>…</script>` konnte Skript ausführen (CSP erlaubt `'unsafe-inline'`) und damit die Go-Bindings aufrufen.
**Fix**: `html.EscapeString()` auf den Titel in `BuildInitialHTML`; Tests `TestBuildInitialHTML_TitleEscaped` / `_TitleUmlautsUnchanged`.

**Datei**: `src/ui/template.go`

### RISK-001: HTML-Injection durch Markdown (Build 11 – 2026-03-08)
**Fix**: `html.WithUnsafe()` aus goldmark-Konfiguration entfernt. Eingebettetes HTML in Markdown wird jetzt escaped statt gerendert. `<script>`, `<iframe>` usw. können nicht mehr über MD-Dateien injiziert werden.

**Datei**: `src/renderer/markdown.go`

### RISK-002: WebView ohne Content-Security-Policy (Build 11 – 2026-03-08)
**Fix**: CSP-Meta-Tag in `htmlDocHead` eingefügt: `default-src 'none'; script-src 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; img-src data: blob:; connect-src 'none';`

**Datei**: `src/ui/template.go`

### RISK-004: Path-Traversal via persistLastFile-Binding (Build 11 – 2026-03-08)
**Fix**: Pfad-Validierung im `persistLastFile`-Binding: Pfad wird nur gespeichert wenn `renderer.IsSupportedFile()` true zurückgibt und `os.Stat()` eine existierende Nicht-Verzeichnis-Datei bestätigt.

**Datei**: `src/bindings.go`
