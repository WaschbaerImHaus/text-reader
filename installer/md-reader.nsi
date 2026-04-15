; MD Reader – Windows NSIS Installer
;
; Creates a Windows installer with:
;   - Mandatory start menu entry
;   - Optional desktop shortcut (selectable component)
;   - Uninstall entry in "Programs and Features"
;   - Silent removal of existing installation before new install
;
; Author: Kurt Ingwer
; Last modified: 2026-03-14
;
; Build command (from project root):
;   makensis installer/md-reader.nsi
;
; Requirements:
;   - build/md-reader.exe must exist (Windows x86_64 build)
;   - src/ui/assets/favicon.ico must exist (app icon)

; ============================================================
; General Settings
; ============================================================

; Required NSIS includes (must be declared before use)
!include "MUI2.nsh"
!include "FileFunc.nsh"

; Compressor for smaller installer file size
SetCompressor /SOLID lzma

; Unicode support for paths with special characters
Unicode True

; Installer metadata
!define APP_NAME        "MD Reader"
!define APP_VERSION     "1.0.37"
!define APP_PUBLISHER   "Kurt Ingwer"
!define APP_DESCRIPTION "Markdown viewer with GitHub-like rendering"
!define APP_URL         "https://github.com/WaschbaerImHaus/text-reader"

; Registry keys for install/uninstall
!define REG_UNINSTALL   "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

; Path to the binary (relative to installer/ directory)
!define EXE_SRC         "..\build\md-reader.exe"
!define ICO_SRC         "..\src\ui\assets\favicon.ico"

; Ghostscript – für PostScript-Unterstützung (optional)
!define GS_VERSION      "10.05.0"
!define GS_VERSION_FLAT "10050"
!define GS_INSTALLER    "gs${GS_VERSION_FLAT}w64.exe"
!define GS_URL          "https://github.com/ArtifexSoftware/ghostpdl-downloads/releases/download/gs${GS_VERSION_FLAT}/${GS_INSTALLER}"

; ============================================================
; Installer Configuration
; ============================================================

Name                "${APP_NAME} ${APP_VERSION}"
OutFile             "..\build\md-reader-setup.exe"
InstallDir          "$PROGRAMFILES64\${APP_NAME}"
InstallDirRegKey    HKLM "Software\${APP_NAME}" "InstallDir"

; Request elevated privileges so we can write to Program Files
RequestExecutionLevel admin

; Modern UI for a clean look
!define MUI_ICON    "${ICO_SRC}"
!define MUI_UNICON  "${ICO_SRC}"

; Start menu folder variable (saved for uninstall)
Var StartMenuFolder

; ============================================================
; Installer Pages
; ============================================================

; Welcome page
!insertmacro MUI_PAGE_WELCOME

; License page (commented out – no separate license file)
; !insertmacro MUI_PAGE_LICENSE "LICENSE.txt"

; Installation directory selection
!insertmacro MUI_PAGE_DIRECTORY

; Component selection (desktop shortcut is optional)
!insertmacro MUI_PAGE_COMPONENTS

; Start menu folder selection
!define MUI_STARTMENUPAGE_REGISTRY_ROOT      "HKLM"
!define MUI_STARTMENUPAGE_REGISTRY_KEY       "Software\${APP_NAME}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "StartMenuFolder"
!define MUI_STARTMENUPAGE_DEFAULTFOLDER      "${APP_NAME}"
!insertmacro MUI_PAGE_STARTMENU Application $StartMenuFolder

; Installation progress
!insertmacro MUI_PAGE_INSTFILES

; Finish page with option to launch now
!define MUI_FINISHPAGE_RUN          "$INSTDIR\md-reader.exe"
!define MUI_FINISHPAGE_RUN_TEXT     "Launch MD Reader now"
!insertmacro MUI_PAGE_FINISH

; ============================================================
; Uninstaller Pages
; ============================================================

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; ============================================================
; Language Settings
; ============================================================

; English as primary language, German as fallback
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "German"

; ============================================================
; Helper: silent removal of any previous installation
; ============================================================

; Called automatically before files are written.
; Reads the uninstaller path from the registry and runs it silently.
Function .onInit
    ; Check whether a previous installation exists
    ReadRegStr $0 HKLM "${REG_UNINSTALL}" "UninstallString"
    StrCmp $0 "" done_uninstall

    ; Run the old uninstaller silently (/S = silent mode)
    ; We use ExecWait so the uninstaller finishes before we copy new files
    ExecWait '$0 /S'

done_uninstall:
FunctionEnd

; ============================================================
; Components (selectable parts of the installation)
; ============================================================

; Mandatory component: program files + start menu
Section "MD Reader (required)" SecMain
    SectionIn RO  ; Cannot be deselected

    ; Create installation directory and copy files
    SetOutPath "$INSTDIR"
    File "${EXE_SRC}"
    File "${ICO_SRC}"

    ; Registry: save installation path
    WriteRegStr HKLM "Software\${APP_NAME}" "InstallDir" "$INSTDIR"

    ; Create start menu entry
    !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
        CreateDirectory "$SMPROGRAMS\$StartMenuFolder"
        ; Shortcut to the application
        CreateShortcut  "$SMPROGRAMS\$StartMenuFolder\${APP_NAME}.lnk" \
                        "$INSTDIR\md-reader.exe" "" \
                        "$INSTDIR\favicon.ico" 0
        ; Shortcut to the uninstaller
        CreateShortcut  "$SMPROGRAMS\$StartMenuFolder\Uninstall.lnk" \
                        "$INSTDIR\Uninstall.exe"
    !insertmacro MUI_STARTMENU_WRITE_END

    ; Generate uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Windows "Programs and Features" entry
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

    ; Estimated installation size in KB (for Programs and Features)
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "${REG_UNINSTALL}" "EstimatedSize" "$0"

SectionEnd

; Optional component: desktop shortcut
Section "Desktop Shortcut" SecDesktop
    CreateShortcut "$DESKTOP\${APP_NAME}.lnk" \
                   "$INSTDIR\md-reader.exe" "" \
                   "$INSTDIR\favicon.ico" 0
SectionEnd

; Optionale Komponente: Ghostscript für PostScript-Unterstützung
Section "Ghostscript (PostScript support)" SecGhostscript

    ; Prüfen ob Ghostscript bereits installiert ist.
    ; Der offizielle Windows-Installer legt gswin64c.exe an (nicht gs.exe),
    ; daher beide Namen prüfen.
    nsExec::ExecToStack 'cmd /C where gswin64c.exe'
    Pop $0
    StrCmp $0 "0" gs_already_installed
    nsExec::ExecToStack 'cmd /C where gs.exe'
    Pop $0
    StrCmp $0 "0" gs_already_installed

    ; Ghostscript via PowerShell herunterladen
    DetailPrint "Downloading Ghostscript ${GS_VERSION}..."
    nsExec::ExecToLog 'powershell -NoProfile -Command "Invoke-WebRequest -Uri \"${GS_URL}\" -OutFile \"$TEMP\${GS_INSTALLER}\" -UseBasicParsing"'
    Pop $0
    StrCmp $0 "0" gs_download_ok

    ; Download fehlgeschlagen
    MessageBox MB_OK|MB_ICONINFORMATION \
        "Ghostscript download failed.$\n$\nPostScript files will be displayed as plain text.$\nYou can install Ghostscript manually from ghostscript.com."
    Goto gs_done

gs_download_ok:
    ; Ghostscript-Installer silent ausführen
    DetailPrint "Installing Ghostscript ${GS_VERSION}..."
    ExecWait '"$TEMP\${GS_INSTALLER}" /S' $0
    Delete "$TEMP\${GS_INSTALLER}"
    StrCmp $0 "0" gs_done

    ; Installation fehlgeschlagen
    MessageBox MB_OK|MB_ICONINFORMATION \
        "Ghostscript installation failed.$\n$\nPostScript files will be displayed as plain text."
    Goto gs_done

gs_already_installed:
    DetailPrint "Ghostscript already installed – skipping."

gs_done:
SectionEnd

; ============================================================
; Component descriptions (tooltip in component selection)
; ============================================================

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecMain}        \
        "Installs MD Reader to the program directory and creates a start menu entry."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop}     \
        "Creates a shortcut on the desktop."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecGhostscript} \
        "Downloads and installs GPL Ghostscript for PostScript (.ps) rendering. Optional – .ps files will display as plain text without it."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ============================================================
; Uninstall section
; ============================================================

Section "Uninstall"
    ; Remove application files
    Delete "$INSTDIR\md-reader.exe"
    Delete "$INSTDIR\favicon.ico"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir  "$INSTDIR"

    ; Remove start menu entries
    !insertmacro MUI_STARTMENU_GETFOLDER Application $StartMenuFolder
    Delete "$SMPROGRAMS\$StartMenuFolder\${APP_NAME}.lnk"
    Delete "$SMPROGRAMS\$StartMenuFolder\Uninstall.lnk"
    RMDir  "$SMPROGRAMS\$StartMenuFolder"

    ; Remove desktop shortcut (if present)
    Delete "$DESKTOP\${APP_NAME}.lnk"

    ; Remove registry entries
    DeleteRegKey HKLM "${REG_UNINSTALL}"
    DeleteRegKey HKLM "Software\${APP_NAME}"

SectionEnd
