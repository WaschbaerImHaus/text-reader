// Package ui stellt die HTML/CSS/JS-Oberfläche der MD-Reader-App bereit.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-07
package ui

import "fmt"

// UIConfig enthält alle Werte die das initiale HTML beeinflussen.
type UIConfig struct {
	// FontSize ist die Schriftgröße in Pixeln.
	FontSize int
	// Theme ist das Farbschema: "light", "dark" oder "retro".
	Theme string
	// IsPortrait gibt an ob der Hochformat-Modus aktiv ist.
	IsPortrait bool
}

// BuildInitialHTML erstellt das vollständige HTML-Dokument für den WebView.
//
// Enthält alle CSS-Stile, JavaScript-Logik und die initiale Benutzeroberfläche.
// Die gespeicherte Konfiguration (Theme, Layout, Schriftgröße) wird direkt
// in das HTML eingebettet und beim Start automatisch wiederhergestellt.
//
// @param cfg UIConfig mit den initialen Einstellungswerten.
// @return Vollständiges HTML-Dokument als String.
func BuildInitialHTML(cfg UIConfig) string {
	// Theme-Klasse für das <body>-Tag bestimmen
	themeClass := ""
	if cfg.Theme == "dark" || cfg.Theme == "retro" {
		themeClass = cfg.Theme
	}
	// isPortrait als JS-Boolean-String
	portraitStr := "false"
	if cfg.IsPortrait {
		portraitStr = "true"
	}

	// Format-Argumente (in Reihenfolge der %d/%s Platzhalter im Template):
	// 1: cfg.FontSize  → CSS --font-size
	// 2: themeClass    → <body class="...">
	// 3: cfg.FontSize  → JS var fontSize
	// 4: cfg.FontSize  → JS var defaultFontSize
	// 5: portraitStr   → JS var isPortrait
	return fmt.Sprintf(htmlTemplate,
		cfg.FontSize, // 1
		themeClass,   // 2
		cfg.FontSize, // 3
		cfg.FontSize, // 4
		portraitStr,  // 5
	)
}

// htmlTemplate ist das vollständige HTML-Template der Anwendung.
// Platzhalter: %%  = literal %, %d = int, %s = string
const htmlTemplate = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>MD Reader</title>
<style>
/* ========================================================
   Basis-Reset und globale CSS-Variablen
   ======================================================== */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --font-size: %dpx;
  --bg-color: #ffffff;
  --text-color: #24292f;
  --border-color: #d0d7de;
  --toolbar-bg: #f6f8fa;
  --code-bg: #f6f8fa;
  --blockquote-border: #d0d7de;
  --blockquote-color: #57606a;
  --link-color: #0969da;
  --heading-color: #24292f;
  --table-header-bg: #f6f8fa;
  --table-alt-row: #f6f8fa;
  --max-content-width: 100%%;
  --content-padding: 32px;
  --toolbar-height: 44px;
  --toc-width: 280px;
  --search-bar-height: 44px;
}

html, body { height: 100%%; width: 100%%; overflow: hidden; background-color: var(--bg-color); color: var(--text-color); }

/* ========================================================
   Toolbar (Werkzeugleiste)
   ======================================================== */
#toolbar {
  position: fixed; top: 0; left: 0; right: 0;
  height: var(--toolbar-height);
  background-color: var(--toolbar-bg);
  border-bottom: 1px solid var(--border-color);
  display: flex; align-items: center; justify-content: center;
  z-index: 1000; user-select: none; -webkit-user-select: none;
  -webkit-app-region: drag;
}
#toolbar button { -webkit-app-region: no-drag; }
#toolbar-center { display: flex; align-items: center; gap: 2px; }
#btn-close { position: absolute; right: 8px; top: 50%%; transform: translateY(-50%%); }

.toolbar-btn {
  background: transparent; border: 1px solid transparent; border-radius: 6px;
  color: var(--text-color); cursor: pointer; font-size: 16px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  height: 32px; min-width: 32px; padding: 0 8px;
  display: flex; align-items: center; justify-content: center;
  transition: background-color 0.1s, border-color 0.1s; line-height: 1;
}
.toolbar-btn:hover { background-color: rgba(175,184,193,0.2); border-color: var(--border-color); }
.toolbar-btn:active { background-color: rgba(175,184,193,0.4); }
#btn-close:hover { background-color: #ff5f5740; border-color: #ff5f57; color: #cc0000; }
.toolbar-separator { width: 1px; height: 20px; background-color: var(--border-color); margin: 0 4px; }
#zoom-label { font-size: 12px; color: var(--blockquote-color); min-width: 40px; text-align: center; padding: 0 4px; }

/* ========================================================
   Suchleiste (unter Toolbar, standardmäßig ausgeblendet)
   ======================================================== */
#search-bar {
  position: fixed; top: var(--toolbar-height); left: 0; right: 0;
  height: var(--search-bar-height);
  background-color: var(--toolbar-bg);
  border-bottom: 1px solid var(--border-color);
  display: none; align-items: center; gap: 6px; padding: 0 12px;
  z-index: 999;
}
#search-bar.visible { display: flex; }
#search-input {
  flex: 1; height: 28px; padding: 0 8px; border-radius: 6px;
  border: 1px solid var(--border-color); background: var(--bg-color);
  color: var(--text-color); font-size: 14px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  outline: none;
}
#search-input:focus { border-color: var(--link-color); box-shadow: 0 0 0 2px rgba(9,105,218,0.15); }
#search-count { font-size: 12px; color: var(--blockquote-color); white-space: nowrap; min-width: 50px; }
mark.search-match { background-color: #fff3b0; color: inherit; border-radius: 2px; }
mark.search-match.current { background-color: #ff9500; color: #fff; }

/* ========================================================
   Inhaltsverzeichnis-Seitenleiste (TOC)
   ======================================================== */
#toc-sidebar {
  position: fixed; top: var(--toolbar-height); left: calc(-1 * var(--toc-width));
  width: var(--toc-width); bottom: 0;
  background-color: var(--toolbar-bg);
  border-right: 1px solid var(--border-color);
  overflow-y: auto; transition: left 0.2s ease; z-index: 998;
  display: flex; flex-direction: column;
}
#toc-sidebar.open { left: 0; }
#toc-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; border-bottom: 1px solid var(--border-color);
  font-size: 13px; font-weight: 600;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: var(--blockquote-color); text-transform: uppercase; letter-spacing: 0.05em;
  position: sticky; top: 0; background-color: var(--toolbar-bg); z-index: 1;
}
#toc-list {
  list-style: none; padding: 8px 0; flex: 1;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
#toc-list li a {
  display: block; padding: 5px 12px; color: var(--text-color);
  text-decoration: none; font-size: 13px; line-height: 1.4;
  border-radius: 4px; margin: 1px 4px;
  transition: background-color 0.1s;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
#toc-list li a:hover { background-color: rgba(175,184,193,0.2); }
#toc-list li a.toc-active { background-color: rgba(9,105,218,0.1); color: var(--link-color); }

/* ========================================================
   Haupt-Inhaltsbereich
   ======================================================== */
#main {
  position: fixed; top: var(--toolbar-height); left: 0; right: 0; bottom: 0;
  overflow-y: auto; overflow-x: hidden; scroll-behavior: smooth;
  transition: left 0.2s ease, top 0.1s ease;
}

/* ========================================================
   Drop-Zone (initiale Ansicht)
   ======================================================== */
#drop-zone {
  display: flex; flex-direction: column; align-items: center;
  justify-content: center; height: 100%%; gap: 16px; padding: 32px;
}
#drop-zone.dragover { background-color: #ddf4ff; border: 3px dashed var(--link-color); border-radius: 12px; }
.drop-icon { font-size: 64px; line-height: 1; opacity: 0.4; }
.drop-title { font-size: 20px; font-weight: 600; color: var(--text-color); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
.drop-subtitle { font-size: 14px; color: var(--blockquote-color); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
.drop-error { font-size: 14px; color: #cf222e; background-color: #fff0f0; border: 1px solid #ff818266; border-radius: 6px; padding: 8px 16px; display: none; }

/* ========================================================
   Inhaltsbereich (Dokument)
   ======================================================== */
#content-wrapper { display: none; width: 100%%; padding: var(--content-padding); }
#content {
  max-width: var(--max-content-width); margin: 0 auto;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif;
  font-size: var(--font-size); line-height: 1.6; color: var(--text-color); word-wrap: break-word;
}

/* ========================================================
   Markdown-Stile (GitHub-kompatibel)
   ======================================================== */
#content h1, #content h2, #content h3,
#content h4, #content h5, #content h6 { margin-top: 24px; margin-bottom: 16px; font-weight: 600; line-height: 1.25; color: var(--heading-color); }
#content h1 { font-size: 2em; padding-bottom: 0.3em; border-bottom: 1px solid var(--border-color); }
#content h2 { font-size: 1.5em; padding-bottom: 0.3em; border-bottom: 1px solid var(--border-color); }
#content h3 { font-size: 1.25em; }
#content h4 { font-size: 1em; }
#content h5 { font-size: 0.875em; }
#content h6 { font-size: 0.85em; color: var(--blockquote-color); }
#content h1 + *, #content h2 + *, #content h3 + *, #content h4 + * { margin-top: 0; }
#content p { margin-top: 0; margin-bottom: 16px; }
#content a { color: var(--link-color); text-decoration: none; }
#content a:hover { text-decoration: underline; }
#content strong { font-weight: 600; }
#content em { font-style: italic; }
#content code { padding: 0.2em 0.4em; font-size: 85%%; white-space: break-spaces; background-color: rgba(175,184,193,0.2); border-radius: 6px; font-family: ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, Liberation Mono, monospace; }
#content pre { padding: 16px; overflow: auto; font-size: 85%%; line-height: 1.45; background-color: var(--code-bg); border-radius: 6px; margin-bottom: 16px; border: 1px solid var(--border-color); }
#content pre code { display: block; padding: 0; margin: 0; overflow: visible; line-height: inherit; word-wrap: normal; background-color: transparent; border: 0; font-size: 100%%; white-space: pre; border-radius: 0; }
#content .chroma { background-color: var(--code-bg) !important; }
#content blockquote { margin: 0 0 16px 0; padding: 0 1em; color: var(--blockquote-color); border-left: 4px solid var(--blockquote-border); }
#content blockquote > :first-child { margin-top: 0; }
#content blockquote > :last-child { margin-bottom: 0; }
#content ul, #content ol { margin-top: 0; margin-bottom: 16px; padding-left: 2em; }
#content ul ul, #content ul ol, #content ol ul, #content ol ol { margin-top: 0; margin-bottom: 0; }
#content li { margin-top: 0.25em; }
#content li > p { margin-top: 16px; }
#content li + li { margin-top: 0.25em; }
#content .task-list-item { list-style-type: none; padding-left: 0; }
#content .task-list-item input { margin: 0 0.2em 0.25em -1.6em; vertical-align: middle; }
#content table { display: block; width: 100%%; width: max-content; max-width: 100%%; overflow: auto; border-spacing: 0; border-collapse: collapse; margin-bottom: 16px; }
#content table th { font-weight: 600; padding: 6px 13px; border: 1px solid var(--border-color); background-color: var(--table-header-bg); }
#content table td { padding: 6px 13px; border: 1px solid var(--border-color); }
#content table tr { background-color: var(--bg-color); border-top: 1px solid var(--border-color); }
#content table tr:nth-child(2n) { background-color: var(--table-alt-row); }
#content hr { height: 4px; padding: 0; margin: 24px 0; background-color: var(--border-color); border: 0; border-radius: 2px; }
#content img { max-width: 100%%; box-sizing: content-box; background-color: var(--bg-color); border-radius: 6px; }
#content del { text-decoration: line-through; }
#content del code { text-decoration: inherit; }
#content kbd { display: inline-block; padding: 3px 5px; font: 11px ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, Liberation Mono, monospace; line-height: 10px; color: var(--text-color); vertical-align: middle; background-color: #f6f8fa; border: solid 1px rgba(175,184,193,0.2); border-radius: 6px; box-shadow: inset 0 -1px 0 rgba(175,184,193,0.2); }
#content .footnotes { font-size: 85%%; }

/* ========================================================
   Ebook-spezifische Stile
   ======================================================== */
#content .txt-content { font-family: ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, Liberation Mono, monospace; font-size: 0.95em; line-height: 1.5; white-space: pre-wrap; word-wrap: break-word; background: transparent; border: none; padding: 0; margin: 0; }
hr.epub-chapter-separator, hr.fb2-chapter-separator { height: 2px; margin: 40px 0; background: linear-gradient(to right, transparent, var(--border-color), transparent); border: none; }
.fb2-section { margin-bottom: 16px; }
.fb2-epigraph { margin: 16px 0; padding: 8px 16px; color: var(--blockquote-color); border-left: 4px solid var(--blockquote-border); font-style: italic; }
.fb2-poem { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; font-size: inherit; white-space: pre-wrap; background: transparent; border: none; padding: 8px 0 8px 2em; margin: 16px 0; line-height: 1.8; color: var(--blockquote-color); border-left: 3px solid var(--blockquote-border); }
.fb2-cite { margin: 16px 0; padding: 8px 16px; border-left: 4px solid var(--blockquote-border); color: var(--blockquote-color); }

/* ========================================================
   Scrollbar-Styling (WebKit)
   ======================================================== */
#main::-webkit-scrollbar { width: 8px; }
#main::-webkit-scrollbar-track { background: transparent; }
#main::-webkit-scrollbar-thumb { background-color: rgba(0,0,0,0.2); border-radius: 4px; }
#main::-webkit-scrollbar-thumb:hover { background-color: rgba(0,0,0,0.4); }
#toc-sidebar::-webkit-scrollbar { width: 4px; }
#toc-sidebar::-webkit-scrollbar-thumb { background-color: rgba(0,0,0,0.15); border-radius: 4px; }

/* ========================================================
   Dark Mode (GitHub Dark)
   ======================================================== */
body.dark {
  --bg-color: #0d1117; --text-color: #e6edf3; --border-color: #30363d;
  --toolbar-bg: #161b22; --code-bg: #161b22; --blockquote-border: #3d444d;
  --blockquote-color: #9198a1; --link-color: #4493f8;
  --heading-color: #e6edf3; --table-header-bg: #161b22; --table-alt-row: #161b22;
}
body.dark .toolbar-btn:hover { background-color: rgba(255,255,255,0.1); }
body.dark .toolbar-btn:active { background-color: rgba(255,255,255,0.18); }
body.dark #main::-webkit-scrollbar-thumb { background-color: rgba(255,255,255,0.15); }
body.dark #main::-webkit-scrollbar-thumb:hover { background-color: rgba(255,255,255,0.3); }
body.dark .drop-error { background-color: #2a1010; border-color: #6d2828; color: #ff8080; }
body.dark mark.search-match { background-color: #5a4500; color: #e6edf3; }
body.dark mark.search-match.current { background-color: #b45000; color: #fff; }
body.dark #search-input { background: #0d1117; }

/* ========================================================
   Retro Dark Mode (CRT-Phosphor Grün)
   ======================================================== */
body.retro {
  --bg-color: #030a03; --text-color: #39ff14; --border-color: #1a4d1a;
  --toolbar-bg: #040c04; --code-bg: #050f05; --blockquote-border: #1f6b1f;
  --blockquote-color: #22cc22; --link-color: #80ff60; --heading-color: #66ff44;
  --table-header-bg: #071207; --table-alt-row: #040d04;
}
body.retro #content { font-family: "Courier New", Courier, monospace !important; }
body.retro #content, body.retro #content h1, body.retro #content h2, body.retro #content h3 { text-shadow: 0 0 8px rgba(57,255,20,0.5); }
body.retro #content code { background-color: #071207; color: #80ff60; }
body.retro #content pre { border-color: #1a4d1a; box-shadow: 0 0 10px rgba(57,255,20,0.15) inset; }
body.retro #main::after { content: ''; position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.08) 2px, rgba(0,0,0,0.08) 4px); pointer-events: none; z-index: 9999; }
body.retro .toolbar-btn:hover { background-color: rgba(57,255,20,0.15); border-color: #39ff14; }
body.retro .toolbar-btn:active { background-color: rgba(57,255,20,0.3); }
body.retro #main::-webkit-scrollbar-thumb { background-color: rgba(57,255,20,0.3); }
body.retro #main::-webkit-scrollbar-thumb:hover { background-color: rgba(57,255,20,0.6); }
body.retro .drop-error { background-color: #0a0500; border-color: #4d2200; color: #ff8c00; }
body.retro #content h1, body.retro #content h2 { border-bottom-color: #1a4d1a; }
body.retro #content a { text-decoration: underline; }
body.retro #content th, body.retro #content td { border-color: #1a4d1a; }
body.retro #content tr { border-top-color: #1a4d1a; }
body.retro mark.search-match { background-color: #1a3d00; color: #39ff14; }
body.retro mark.search-match.current { background-color: #3d7a00; color: #fff; }
body.retro #search-input { background: #030a03; color: #39ff14; }
</style>
</head>
<body class="%s">

<!-- ============================================================
     Toolbar
     ============================================================ -->
<div id="toolbar">
  <div id="toolbar-center">

    <!-- TOC-Button (nur sichtbar wenn Überschriften vorhanden) -->
    <button class="toolbar-btn" id="btn-toc" onclick="toggleTOC()"
            title="Inhaltsverzeichnis" style="display:none">&#9776;</button>

    <div class="toolbar-separator" id="sep-toc" style="display:none"></div>

    <!-- Hochformat / Querformat -->
    <button class="toolbar-btn" id="btn-layout" onclick="toggleLayout()"
            title="Hochformat / Querformat umschalten">&#9647;</button>

    <div class="toolbar-separator"></div>

    <!-- Zoom -->
    <button class="toolbar-btn" id="btn-zoom-out" onclick="zoomOut()"
            title="Schrift verkleinern (Strg+-)">&#8854;</button>
    <span id="zoom-label">100%%</span>
    <button class="toolbar-btn" id="btn-zoom-in" onclick="zoomIn()"
            title="Schrift vergrößern (Strg++)">&#8853;</button>

    <div class="toolbar-separator"></div>

    <!-- Suche -->
    <button class="toolbar-btn" id="btn-search" onclick="openSearch()"
            title="Suchen (Strg+F)">&#128269;</button>

    <div class="toolbar-separator"></div>

    <!-- Theme-Wechsel -->
    <button class="toolbar-btn" id="btn-theme" onclick="cycleTheme()"
            title="Dunkelmodus einschalten">&#9790;</button>

    <div class="toolbar-separator"></div>

    <!-- Vollbild -->
    <button class="toolbar-btn" id="btn-fullscreen" onclick="toggleFullscreen()"
            title="Vollbild (F11)">&#9634;</button>

  </div>
  <!-- Schließen-Button rechts -->
  <button class="toolbar-btn" id="btn-close" onclick="closeApp()"
          title="Schließen">&#10005;</button>
</div>

<!-- ============================================================
     Suchleiste (unter Toolbar)
     ============================================================ -->
<div id="search-bar">
  <input type="text" id="search-input" placeholder="Suchen..."
         autocomplete="off" oninput="onSearchInput(this.value)"
         onkeydown="onSearchKeydown(event)">
  <span id="search-count"></span>
  <button class="toolbar-btn" onclick="searchPrev()" title="Vorheriges (Umschalt+Enter)">&#8593;</button>
  <button class="toolbar-btn" onclick="searchNext()" title="N&#228;chstes (Enter)">&#8595;</button>
  <button class="toolbar-btn" onclick="closeSearch()" title="Suche schlie&#223;en (Esc)">&#10005;</button>
</div>

<!-- ============================================================
     TOC-Seitenleiste
     ============================================================ -->
<div id="toc-sidebar">
  <div id="toc-header">
    <span>Inhalt</span>
    <button class="toolbar-btn" onclick="closeTOC()" title="Schlie&#223;en"
            style="min-width:24px;height:24px;font-size:13px;">&#10005;</button>
  </div>
  <ul id="toc-list"></ul>
</div>

<!-- ============================================================
     Haupt-Inhaltsbereich
     ============================================================ -->
<div id="main">

  <!-- Drop-Zone -->
  <div id="drop-zone">
    <div class="drop-icon">&#128218;</div>
    <div class="drop-title">Datei hier ablegen</div>
    <div class="drop-subtitle">.md &middot; .epub &middot; .txt &middot; .fb2 und weitere</div>
    <div class="drop-error" id="drop-error"></div>
  </div>

  <!-- Inhaltsbereich (nach dem Laden sichtbar) -->
  <div id="content-wrapper">
    <div id="content"></div>
  </div>

</div>

<script>
// ============================================================
// Globale Zustandsvariablen
// ============================================================

/** Aktuelle Schriftgröße in Pixeln (aus gespeicherter Konfiguration) */
var fontSize = %d;

/** Standard-Schriftgröße (für Zoom-Prozentberechnung) */
var defaultFontSize = %d;

/** Maximale Schriftgröße (400%% von Standard) */
var maxFontSize = defaultFontSize * 4;

/** Minimale Schriftgröße (25%% von Standard) */
var minFontSize = defaultFontSize * 0.25;

/** Hochformat-Modus aktiv? (aus gespeicherter Konfiguration) */
var isPortrait = %s;

/** Vollbild aktiv? */
var isFullscreen = false;

/** Aktuelles Theme: 'light', 'dark' oder 'retro' */
var currentTheme = 'light';

/** TOC-Seitenleiste geöffnet? */
var tocOpen = false;

/** Suchleiste sichtbar? */
var searchBarVisible = false;

/** Letzte Suchanfrage */
var lastSearchTerm = '';

/** Gefundene Treffer (mark-Elemente) */
var searchMatches = [];

/** Index des aktuell hervorgehobenen Treffers */
var searchCurrentIdx = -1;

// DOM-Referenzen
var mainEl = document.getElementById('main');
var dropZone = document.getElementById('drop-zone');

// ============================================================
// Initialisierung (gespeicherte Einstellungen wiederherstellen)
// ============================================================

/** Schriftgröße anwenden */
applyFontSize();

/** Layout wiederherstellen */
(function initLayout() {
  if (isPortrait) {
    document.documentElement.style.setProperty('--max-content-width', '750px');
    var btn = document.getElementById('btn-layout');
    btn.textContent = '\u25AD'; // ▭
    btn.title = 'Querformat anzeigen';
  }
})();

/** Theme wiederherstellen */
(function initTheme() {
  // currentTheme aus gespeicherter Konfiguration (wird als JS-Variable vom Go-Template gesetzt)
  // Wir lesen es aus der Body-Klasse die Go bereits gesetzt hat
  if (document.body.classList.contains('dark')) {
    currentTheme = 'dark';
  } else if (document.body.classList.contains('retro')) {
    currentTheme = 'retro';
  }
  updateThemeButton();
})();

// ============================================================
// Drag & Drop
// ============================================================

/** Unterstützte Dateiendungen (muss mit IsSupportedFile in renderer/markdown.go übereinstimmen) */
var supportedExtensions = ['.md', '.markdown', '.txt', '.fb2', '.epub'];

/** Prüft ob eine Datei ein unterstütztes Format hat */
function isSupportedFile(filename) {
  var lower = filename.toLowerCase();
  return supportedExtensions.some(function(ext) { return lower.endsWith(ext); });
}

/** Konvertiert ArrayBuffer sicher zu Base64 (in Chunks um Stack-Überlauf zu vermeiden) */
function arrayBufferToBase64(buffer) {
  var binary = '';
  var bytes = new Uint8Array(buffer);
  var chunkSize = 8192;
  for (var i = 0; i < bytes.length; i += chunkSize) {
    var chunk = bytes.subarray(i, Math.min(i + chunkSize, bytes.length));
    binary += String.fromCharCode.apply(null, chunk);
  }
  return btoa(binary);
}

document.addEventListener('dragover', function(e) {
  e.preventDefault(); e.stopPropagation();
  dropZone.classList.add('dragover');
});

document.addEventListener('dragleave', function(e) {
  if (e.clientX === 0 || e.clientY === 0 ||
      e.clientX >= window.innerWidth || e.clientY >= window.innerHeight) {
    dropZone.classList.remove('dragover');
  }
});

/** Verarbeitet das Ablegen einer Datei: EPUB binär, alle anderen als UTF-8-Text */
document.addEventListener('drop', function(e) {
  e.preventDefault(); e.stopPropagation();
  dropZone.classList.remove('dragover');

  var files = e.dataTransfer.files;
  if (!files || files.length === 0) return;
  var file = files[0];

  if (!isSupportedFile(file.name)) {
    showDropError('Nicht unterst\u00FCtztes Format: ' + file.name +
      '\n\nUnterst\u00FCtzt: ' + supportedExtensions.join(', '));
    return;
  }
  hideDropError();

  var reader = new FileReader();
  if (file.name.toLowerCase().endsWith('.epub')) {
    // EPUB: binär lesen, als Base64 an Go senden
    reader.onload = function(ev) {
      window.processEpub(arrayBufferToBase64(ev.target.result), file.name)
        .then(handleRenderResult).catch(function(e) { showDropError('Fehler: ' + e); });
    };
    reader.onerror = function() { showDropError('EPUB konnte nicht gelesen werden.'); };
    reader.readAsArrayBuffer(file);
  } else {
    // Text-Formate: als UTF-8 lesen
    reader.onload = function(ev) {
      window.processMarkdown(ev.target.result, file.name)
        .then(handleRenderResult).catch(function(e) { showDropError('Fehler: ' + e); });
    };
    reader.onerror = function() { showDropError('Datei konnte nicht gelesen werden.'); };
    reader.readAsText(file, 'utf-8');
  }
});

/** Verarbeitet das Render-Ergebnis von Go */
function handleRenderResult(result) {
  if (result && result.error) {
    showDropError('Fehler: ' + result.error);
  } else if (result) {
    showContent(result.html, result.title);
  }
}

// ============================================================
// Inhalt anzeigen / verstecken
// ============================================================

/** Zeigt gerenderten Inhalt an und baut das TOC auf */
function showContent(html, title) {
  closeSearch(); // Suche zurücksetzen
  document.getElementById('drop-zone').style.display = 'none';
  var wrapper = document.getElementById('content-wrapper');
  wrapper.style.display = 'block';
  document.getElementById('content').innerHTML = html;
  document.title = title ? title + ' - MD Reader' : 'MD Reader';
  mainEl.scrollTop = 0;
  buildTOC(); // TOC nach dem Rendern aufbauen
}

function showDropError(msg) {
  var el = document.getElementById('drop-error');
  el.textContent = msg;
  el.style.display = 'block';
  setTimeout(function() { el.style.display = 'none'; }, 6000);
}

function hideDropError() {
  document.getElementById('drop-error').style.display = 'none';
}

// ============================================================
// Zoom-Funktionen
// ============================================================

function zoomIn() {
  if (fontSize >= maxFontSize) return;
  fontSize = Math.min(fontSize * 2, maxFontSize);
  applyFontSize();
  saveState();
}

function zoomOut() {
  if (fontSize <= minFontSize) return;
  fontSize = Math.max(fontSize / 2, minFontSize);
  applyFontSize();
  saveState();
}

/** Setzt Schriftgröße auf Standard zurück (Strg+0) */
function resetZoom() {
  fontSize = defaultFontSize;
  applyFontSize();
  saveState();
}

function applyFontSize() {
  document.documentElement.style.setProperty('--font-size', fontSize + 'px');
  updateZoomLabel();
}

function updateZoomLabel() {
  var pct = Math.round((fontSize / defaultFontSize) * 100);
  document.getElementById('zoom-label').textContent = pct + '%%';
}

// ============================================================
// Theme-Wechsel (zyklisch: hell → dunkel → retro → hell)
// ============================================================

function cycleTheme() {
  var body = document.body;
  if (currentTheme === 'light') {
    body.classList.remove('retro'); body.classList.add('dark');
    currentTheme = 'dark';
  } else if (currentTheme === 'dark') {
    body.classList.remove('dark'); body.classList.add('retro');
    currentTheme = 'retro';
  } else {
    body.classList.remove('dark', 'retro');
    currentTheme = 'light';
  }
  updateThemeButton();
  saveState();
}

function updateThemeButton() {
  var btn = document.getElementById('btn-theme');
  if (currentTheme === 'dark') {
    btn.textContent = '\u2593'; btn.title = 'Retro-Modus einschalten';
  } else if (currentTheme === 'retro') {
    btn.textContent = '\u2600'; btn.title = 'Hellen Modus einschalten';
  } else {
    btn.textContent = '\u263E'; btn.title = 'Dunkelmodus einschalten';
  }
}

// ============================================================
// Layout: Hochformat / Querformat
// ============================================================

function toggleLayout() {
  isPortrait = !isPortrait;
  var btn = document.getElementById('btn-layout');
  if (isPortrait) {
    document.documentElement.style.setProperty('--max-content-width', '750px');
    btn.textContent = '\u25AD'; btn.title = 'Querformat anzeigen';
  } else {
    document.documentElement.style.setProperty('--max-content-width', '100%%');
    btn.textContent = '\u25AF'; btn.title = 'Hochformat anzeigen';
  }
  saveState();
}

// ============================================================
// Vollbild (Bug #002 Fix: immer native API, kein HTML5 Fullscreen)
// ============================================================

function toggleFullscreen() {
  if (typeof window.nativeFullscreen === 'function') {
    window.nativeFullscreen().then(function(isNowFullscreen) {
      isFullscreen = isNowFullscreen;
      updateFullscreenButton();
    });
  }
}

function updateFullscreenButton() {
  var btn = document.getElementById('btn-fullscreen');
  if (isFullscreen) {
    btn.textContent = '\u2715'; btn.title = 'Vollbild verlassen';
  } else {
    btn.textContent = '\u2610'; btn.title = 'Vollbild (F11)';
  }
}

// ============================================================
// TOC (Inhaltsverzeichnis-Seitenleiste)
// ============================================================

/** Baut das TOC aus den Überschriften im Inhaltsbereich */
function buildTOC() {
  var content = document.getElementById('content');
  var headings = content.querySelectorAll('h1, h2, h3, h4, h5, h6');
  var tocBtn = document.getElementById('btn-toc');
  var tocSep = document.getElementById('sep-toc');
  var list = document.getElementById('toc-list');
  list.innerHTML = '';

  if (headings.length === 0) {
    // Kein TOC für dieses Format (z.B. TXT)
    tocBtn.style.display = 'none';
    tocSep.style.display = 'none';
    if (tocOpen) closeTOC();
    return;
  }

  tocBtn.style.display = '';
  tocSep.style.display = '';

  headings.forEach(function(h, idx) {
    // Eindeutige ID sicherstellen (für Scrollen)
    if (!h.id) { h.id = 'toc-h-' + idx; }

    var li = document.createElement('li');
    var a = document.createElement('a');
    a.href = '#' + h.id;
    a.textContent = h.textContent.trim();
    // Einrückung je nach Überschriften-Ebene
    var level = parseInt(h.tagName.substring(1));
    li.style.paddingLeft = ((level - 1) * 14) + 'px';
    a.addEventListener('click', function(e) {
      e.preventDefault();
      h.scrollIntoView({ behavior: 'smooth', block: 'start' });
      // Auf kleinen Fenstern TOC nach Klick schließen
      if (window.innerWidth < 900) closeTOC();
    });
    li.appendChild(a);
    list.appendChild(li);
  });
}

function toggleTOC() {
  if (tocOpen) closeTOC(); else openTOC();
}

function openTOC() {
  tocOpen = true;
  document.getElementById('toc-sidebar').classList.add('open');
  mainEl.style.left = 'var(--toc-width)';
}

function closeTOC() {
  tocOpen = false;
  document.getElementById('toc-sidebar').classList.remove('open');
  mainEl.style.left = '0';
}

// ============================================================
// Suchfunktion
// ============================================================

function openSearch() {
  searchBarVisible = true;
  var bar = document.getElementById('search-bar');
  bar.classList.add('visible');
  // #main nach unten schieben
  var topPx = 44 + 44; // toolbar + search bar
  mainEl.style.top = topPx + 'px';
  document.getElementById('search-input').focus();
  document.getElementById('search-input').select();
  // Aktuelle Suche wiederholen falls vorhanden
  if (lastSearchTerm) {
    document.getElementById('search-input').value = lastSearchTerm;
    performSearch(lastSearchTerm);
  }
}

function closeSearch() {
  searchBarVisible = false;
  document.getElementById('search-bar').classList.remove('visible');
  mainEl.style.top = '44px'; // nur Toolbar
  clearHighlights();
  document.getElementById('search-count').textContent = '';
  lastSearchTerm = '';
}

function onSearchInput(term) {
  lastSearchTerm = term;
  performSearch(term);
}

function onSearchKeydown(e) {
  if (e.key === 'Enter') {
    e.preventDefault();
    if (e.shiftKey) searchPrev(); else searchNext();
  } else if (e.key === 'Escape') {
    closeSearch();
  }
}

function searchNext() {
  if (searchMatches.length === 0) return;
  goToMatch(searchCurrentIdx + 1);
}

function searchPrev() {
  if (searchMatches.length === 0) return;
  goToMatch(searchCurrentIdx - 1);
}

/**
 * Sucht Text im Inhaltsbereich per TreeWalker und hebt Treffer mit <mark> hervor.
 * Verarbeitet Textknoten in umgekehrter Reihenfolge um DOM-Positionen stabil zu halten.
 */
function performSearch(term) {
  clearHighlights();
  if (!term || term.length === 0) {
    document.getElementById('search-count').textContent = '';
    return;
  }

  var content = document.getElementById('content');
  if (!content) return;

  var regex;
  try { regex = new RegExp(escapeRegexChars(term), 'gi'); }
  catch(e) { return; }

  // Alle passenden Textknoten sammeln
  var walker = document.createTreeWalker(content, NodeFilter.SHOW_TEXT, null, false);
  var textNodes = [];
  var node;
  while ((node = walker.nextNode())) {
    var parent = node.parentElement;
    if (parent) {
      var tag = parent.tagName.toLowerCase();
      if (tag === 'script' || tag === 'style' || tag === 'mark') continue;
    }
    regex.lastIndex = 0;
    if (regex.test(node.nodeValue)) textNodes.push(node);
  }

  // In umgekehrter Reihenfolge verarbeiten (DOM-Stabilität)
  var allMarks = [];
  for (var i = textNodes.length - 1; i >= 0; i--) {
    var tn = textNodes[i];
    var text = tn.nodeValue;
    var frag = document.createDocumentFragment();
    var last = 0;
    var m;
    var nodeMarks = [];
    regex.lastIndex = 0;
    while ((m = regex.exec(text)) !== null) {
      if (m.index > last) frag.appendChild(document.createTextNode(text.substring(last, m.index)));
      var mark = document.createElement('mark');
      mark.className = 'search-match';
      mark.textContent = m[0];
      frag.appendChild(mark);
      nodeMarks.push(mark);
      last = regex.lastIndex;
    }
    if (last < text.length) frag.appendChild(document.createTextNode(text.substring(last)));
    tn.parentNode.replaceChild(frag, tn);
    // Knoten in Vorwärts-Reihenfolge an den Anfang des Gesamtarrays stellen
    for (var j = nodeMarks.length - 1; j >= 0; j--) allMarks.unshift(nodeMarks[j]);
  }

  searchMatches = allMarks;
  searchCurrentIdx = -1;
  if (searchMatches.length > 0) goToMatch(0);
  else document.getElementById('search-count').textContent = 'Kein Ergebnis';
}

/** Springt zum Treffer mit dem angegebenen Index */
function goToMatch(idx) {
  if (searchMatches.length === 0) return;
  if (searchCurrentIdx >= 0 && searchCurrentIdx < searchMatches.length) {
    searchMatches[searchCurrentIdx].classList.remove('current');
  }
  searchCurrentIdx = ((idx %% searchMatches.length) + searchMatches.length) %% searchMatches.length;
  var current = searchMatches[searchCurrentIdx];
  current.classList.add('current');
  current.scrollIntoView({ behavior: 'smooth', block: 'center' });
  document.getElementById('search-count').textContent =
    (searchCurrentIdx + 1) + ' / ' + searchMatches.length;
}

/** Entfernt alle Suchmarkierungen aus dem Inhalt */
function clearHighlights() {
  var marks = document.querySelectorAll('mark.search-match');
  for (var i = 0; i < marks.length; i++) {
    var mark = marks[i];
    mark.parentNode.replaceChild(document.createTextNode(mark.textContent), mark);
  }
  var content = document.getElementById('content');
  if (content) content.normalize(); // Benachbarte Textknoten zusammenführen
  searchMatches = [];
  searchCurrentIdx = -1;
}

/** Escaped Sonderzeichen für sichere RegExp-Nutzung */
function escapeRegexChars(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// ============================================================
// Zustand speichern (Einstellungen persistieren)
// ============================================================

/** Sendet aktuelle Einstellungen an Go zum Speichern */
function saveState() {
  if (typeof window.persistState === 'function') {
    window.persistState(
      Math.round(fontSize),
      currentTheme,
      isPortrait ? 'portrait' : 'landscape'
    );
  }
}

// ============================================================
// Fenster schließen
// ============================================================

function closeApp() {
  if (typeof window.closeApp === 'function') window.closeApp();
}

// ============================================================
// Tastatur-Shortcuts
// ============================================================

document.addEventListener('keydown', function(e) {
  var tag = document.activeElement ? document.activeElement.tagName.toLowerCase() : '';
  var inInput = (tag === 'input' || tag === 'textarea');

  // Vollbild: F11
  if (e.key === 'F11') { e.preventDefault(); toggleFullscreen(); return; }

  // Suche öffnen: Strg+F
  if (e.ctrlKey && e.key === 'f') { e.preventDefault(); openSearch(); return; }

  // Zoom-Shortcuts (nur außerhalb von Eingabefeldern)
  if (e.ctrlKey && !inInput) {
    if (e.key === '+' || e.key === '=') { e.preventDefault(); zoomIn(); return; }
    if (e.key === '-')                  { e.preventDefault(); zoomOut(); return; }
    if (e.key === '0')                  { e.preventDefault(); resetZoom(); return; }
  }

  // Escape: Suche oder Vollbild schließen
  if (e.key === 'Escape') {
    if (searchBarVisible) { closeSearch(); return; }
    if (isFullscreen) { toggleFullscreen(); return; }
  }

  // Scrollen mit Pfeiltasten (nur außerhalb von Eingabefeldern)
  if (!inInput) {
    var step = 60; var stepH = 80;
    switch (e.key) {
      case 'ArrowDown':  e.preventDefault(); mainEl.scrollBy(0, step); break;
      case 'ArrowUp':    e.preventDefault(); mainEl.scrollBy(0, -step); break;
      case 'ArrowRight': e.preventDefault(); mainEl.scrollBy(stepH, 0); break;
      case 'ArrowLeft':  e.preventDefault(); mainEl.scrollBy(-stepH, 0); break;
      case 'PageDown':   e.preventDefault(); mainEl.scrollBy(0, mainEl.clientHeight * 0.9); break;
      case 'PageUp':     e.preventDefault(); mainEl.scrollBy(0, -mainEl.clientHeight * 0.9); break;
      case 'Home':       e.preventDefault(); mainEl.scrollTo(0, 0); break;
      case 'End':        e.preventDefault(); mainEl.scrollTo(0, mainEl.scrollHeight); break;
    }
  }
});

// ============================================================
// Strg+Mausrad → Zoom
// ============================================================

document.addEventListener('wheel', function(e) {
  if (e.ctrlKey) {
    e.preventDefault();
    if (e.deltaY < 0) zoomIn(); else zoomOut();
  }
}, { passive: false });

</script>
</body>
</html>`
