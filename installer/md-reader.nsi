; MD Reader – Windows NSIS Installer
;
; Erstellt einen minimalen Windows-Installer mit:
;   - Startmenü-Eintrag (Pflicht)
;   - Optionales Desktop-Symbol (Komponente wählbar)
;   - Deinstallations-Eintrag in "Programme und Features"
;
; Autor: Kurt Ingwer
; Letzte Änderung: 2026-03-08
;
; Build-Befehl (aus Projektroot):
;   makensis installer/md-reader.nsi
;
; Voraussetzungen:
;   - build/md-reader.exe muss existieren (Windows x86_64 Build)
;   - src/ui/assets/favicon.ico muss existieren (App-Icon)

; ============================================================
; Allgemeine Einstellungen
; ============================================================

; Benötigte NSIS-Includes (müssen vor Verwendung stehen)
!include "MUI2.nsh"
!include "FileFunc.nsh"

; Compressor für kleinere Installer-Dateigröße
SetCompressor /SOLID lzma

; Unicode-Unterstützung für Pfade mit Sonderzeichen
Unicode True

; Installer-Metadaten
!define APP_NAME        "MD Reader"
!define APP_VERSION     "1.0.19"
!define APP_PUBLISHER   "Kurt Ingwer"
!define APP_DESCRIPTION "Markdown-Betrachter mit GitHub-ähnlichem Rendering"
!define APP_URL         "https://bitbucket.org/von-null/md-reader"

; Installations- und Deinstallationsschlüssel für die Registry
!define REG_UNINSTALL   "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

; Pfad zur Installationsdatei (relativ zum installer/-Verzeichnis)
!define EXE_SRC         "..\build\md-reader.exe"
!define ICO_SRC         "..\src\ui\assets\favicon.ico"

; ============================================================
; Installer-Konfiguration
; ============================================================

Name                "${APP_NAME} ${APP_VERSION}"
OutFile             "..\build\md-reader-setup.exe"
InstallDir          "$PROGRAMFILES64\${APP_NAME}"
InstallDirRegKey    HKLM "Software\${APP_NAME}" "InstallDir"

; Moderne UI für ansprechende Optik

; Installer-Icon
!define MUI_ICON    "${ICO_SRC}"
!define MUI_UNICON  "${ICO_SRC}"

; Hintergrundbild links (optional, auskommentiert falls nicht vorhanden)
; !define MUI_WELCOMEFINISHPAGE_BITMAP "installer\welcome.bmp"

; Startmenü-Variable (wird für Deinstallation gespeichert)
Var StartMenuFolder

; ============================================================
; Seiten des Installers
; ============================================================

; Willkommensseite
!insertmacro MUI_PAGE_WELCOME

; Lizenzseite (auskommentiert – kein separates Lizenzdokument vorhanden)
; !insertmacro MUI_PAGE_LICENSE "LICENSE.txt"

; Installations-Verzeichnis auswählen
!insertmacro MUI_PAGE_DIRECTORY

; Komponenten auswählen (Desktop-Symbol optional)
!insertmacro MUI_PAGE_COMPONENTS

; Startmenü-Ordner auswählen
!define MUI_STARTMENUPAGE_REGISTRY_ROOT      "HKLM"
!define MUI_STARTMENUPAGE_REGISTRY_KEY       "Software\${APP_NAME}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "StartMenuFolder"
!define MUI_STARTMENUPAGE_DEFAULTFOLDER      "${APP_NAME}"
!insertmacro MUI_PAGE_STARTMENU Application $StartMenuFolder

; Installationsfortschritt
!insertmacro MUI_PAGE_INSTFILES

; Abschlussseite mit Option "Jetzt starten"
!define MUI_FINISHPAGE_RUN          "$INSTDIR\md-reader.exe"
!define MUI_FINISHPAGE_RUN_TEXT     "MD Reader jetzt starten"
!insertmacro MUI_PAGE_FINISH

; ============================================================
; Deinstallations-Seiten
; ============================================================

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; ============================================================
; Spracheinstellungen
; ============================================================

; Deutsch als primäre Sprache, Englisch als Fallback
!insertmacro MUI_LANGUAGE "German"
!insertmacro MUI_LANGUAGE "English"

; ============================================================
; Komponenten (wählbare Teile der Installation)
; ============================================================

; Pflicht-Komponente: Programmdateien + Startmenü
Section "MD Reader (erforderlich)" SecMain
    SectionIn RO  ; Kann nicht abgewählt werden

    ; Installationsverzeichnis erstellen und Dateien kopieren
    SetOutPath "$INSTDIR"
    File "${EXE_SRC}"
    File "${ICO_SRC}"

    ; Registry: Installationspfad speichern
    WriteRegStr HKLM "Software\${APP_NAME}" "InstallDir" "$INSTDIR"

    ; Startmenü-Eintrag anlegen
    !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
        CreateDirectory "$SMPROGRAMS\$StartMenuFolder"
        ; Verknüpfung zur Anwendung
        CreateShortcut  "$SMPROGRAMS\$StartMenuFolder\${APP_NAME}.lnk" \
                        "$INSTDIR\md-reader.exe" "" \
                        "$INSTDIR\favicon.ico" 0
        ; Verknüpfung zum Deinstaller
        CreateShortcut  "$SMPROGRAMS\$StartMenuFolder\Deinstallieren.lnk" \
                        "$INSTDIR\Uninstall.exe"
    !insertmacro MUI_STARTMENU_WRITE_END

    ; Deinstaller erzeugen
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Windows "Programme und Features" Eintrag
    WriteRegStr   HKLM "${REG_UNINSTALL}" "DisplayName"          "${APP_NAME}"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "DisplayVersion"       "${APP_VERSION}"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "Publisher"            "${APP_PUBLISHER}"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "Comments"             "${APP_DESCRIPTION}"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "URLInfoAbout"         "${APP_URL}"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "InstallLocation"      "$INSTDIR"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "UninstallString"      "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr   HKLM "${REG_UNINSTALL}" "QuietUninstallString" "$\"$INSTDIR\Uninstall.exe$\" /S"
    WriteRegStr   HKLM "${REG_UNINSTALL}" "DisplayIcon"          "$INSTDIR\favicon.ico"
    WriteRegDWORD HKLM "${REG_UNINSTALL}" "NoModify"             1
    WriteRegDWORD HKLM "${REG_UNINSTALL}" "NoRepair"             1

    ; Geschätzte Installationsgröße in KB (für Programme und Features)
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "${REG_UNINSTALL}" "EstimatedSize" "$0"

SectionEnd

; Optionale Komponente: Desktop-Symbol
Section "Desktop-Symbol" SecDesktop
    CreateShortcut "$DESKTOP\${APP_NAME}.lnk" \
                   "$INSTDIR\md-reader.exe" "" \
                   "$INSTDIR\favicon.ico" 0
SectionEnd

; ============================================================
; Komponenten-Beschreibungen (Tooltip in der Auswahl)
; ============================================================

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecMain}    \
        "Installiert MD Reader in das Programmverzeichnis und legt einen Startmenü-Eintrag an."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} \
        "Legt ein Symbol auf dem Desktop an."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ============================================================
; Deinstallations-Sektion
; ============================================================

Section "Uninstall"
    ; Anwendungsdateien entfernen
    Delete "$INSTDIR\md-reader.exe"
    Delete "$INSTDIR\favicon.ico"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir  "$INSTDIR"

    ; Startmenü-Einträge entfernen
    !insertmacro MUI_STARTMENU_GETFOLDER Application $StartMenuFolder
    Delete "$SMPROGRAMS\$StartMenuFolder\${APP_NAME}.lnk"
    Delete "$SMPROGRAMS\$StartMenuFolder\Deinstallieren.lnk"
    RMDir  "$SMPROGRAMS\$StartMenuFolder"

    ; Desktop-Symbol entfernen (falls vorhanden)
    Delete "$DESKTOP\${APP_NAME}.lnk"

    ; Registry-Einträge entfernen
    DeleteRegKey HKLM "${REG_UNINSTALL}"
    DeleteRegKey HKLM "Software\${APP_NAME}"

SectionEnd

