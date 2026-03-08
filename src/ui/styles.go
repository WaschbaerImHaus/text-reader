// Package ui – CSS-Stile der MD-Reader-Oberfläche.
//
// Enthält alle CSS-Definitionen für Light-, Dark- und Retro-Mode,
// Toolbar, Suchleiste, TOC-Seitenleiste und Markdown-Rendering.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08
package ui

// htmlCSS ist das CSS-Template der Anwendung.
// Platzhalter (in Reihenfolge der fmt.Sprintf-Argumente):
//
//	%d  → Schriftgröße in px (CSS-Variable --font-size)
const htmlCSS = `<style>
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
`
