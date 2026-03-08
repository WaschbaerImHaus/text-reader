<script>
// ============================================================
// Globale Zustandsvariablen
// ============================================================

/** Aktuelle Schriftgröße in Pixeln (aus gespeicherter Konfiguration) */
var fontSize = {{FONT_SIZE}};

/** Standard-Schriftgröße (für Zoom-Prozentberechnung) */
var defaultFontSize = {{DEFAULT_FONT_SIZE}};

/** Maximale Schriftgröße (400% von Standard) */
var maxFontSize = defaultFontSize * 4;

/** Minimale Schriftgröße (25% von Standard) */
var minFontSize = defaultFontSize * 0.25;

/** Hochformat-Modus aktiv? (aus gespeicherter Konfiguration) */
var isPortrait = {{IS_PORTRAIT}};

/** Vollbild aktiv? */
var isFullscreen = false;

/** Aktuelles Theme: 'light', 'dark' oder 'retro' */
var currentTheme = 'light';

/** Hash des aktuell angezeigten Dateiinhalts (FNV-64a, von Go berechnet) */
var currentFileHash = '';

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
    updatePortraitWidth();
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

/**
 * Extrahiert den vollständigen Dateipfad aus einem Drop-Event.
 *
 * Desktop-WebViews (WebKitGTK, WebView2) stellen den Dateipfad über die
 * text/uri-list DataTransfer-Eigenschaft bereit (z.B. "file:///home/user/doc.md").
 * Diese Information ist in Browser-Umgebungen aus Sicherheitsgründen nicht verfügbar,
 * aber in nativen Desktop-Apps ist sie zugänglich.
 *
 * @param {DragEvent} e - Das Drop-Event.
 * @returns {string} Vollständiger Dateipfad oder leerer String wenn nicht verfügbar.
 */
function extractFilePathFromDrop(e) {
  var uriList = '';
  try { uriList = e.dataTransfer.getData('text/uri-list') || ''; } catch(ex) {}
  if (!uriList) return '';
  // Erste gültige file://-URI aus der Liste nehmen (ignoriert Kommentarzeilen)
  var lines = uriList.split('\n');
  for (var i = 0; i < lines.length; i++) {
    var line = lines[i].replace(/\r/g, '').trim();
    if (line && !line.startsWith('#') && line.startsWith('file://')) {
      // file:///home/user/doc.md → /home/user/doc.md (Linux)
      // file:///C:/Users/user/doc.md → C:/Users/user/doc.md (Windows)
      var path = decodeURIComponent(line.replace(/^file:\/\//, ''));
      // Windows-Fix (#005): file:///C:/... ergibt /C:/... → führenden Slash vor Laufwerksbuchstabe entfernen
      path = path.replace(/^\/([A-Za-z]:)/, '$1');
      return path;
    }
  }
  return '';
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

  // Vollständigen Dateipfad aus der URI-Liste des Drop-Events extrahieren.
  // Wird nach erfolgreichem Rendern an Go übergeben damit die Datei beim
  // nächsten Start automatisch wieder geöffnet werden kann.
  var droppedFilePath = extractFilePathFromDrop(e);

  var reader = new FileReader();
  if (file.name.toLowerCase().endsWith('.epub')) {
    // EPUB: binär lesen, als Base64 an Go senden
    reader.onload = function(ev) {
      window.processEpub(arrayBufferToBase64(ev.target.result), file.name)
        .then(function(result) {
          handleRenderResult(result);
          // Pfad nach erfolgreichem Rendern speichern
          if (!result || !result.error) saveLastFile(droppedFilePath);
        }).catch(function(e) { showDropError('Fehler: ' + e); });
    };
    reader.onerror = function() { showDropError('EPUB konnte nicht gelesen werden.'); };
    reader.readAsArrayBuffer(file);
  } else {
    // Text-Formate: als UTF-8 lesen
    reader.onload = function(ev) {
      window.processMarkdown(ev.target.result, file.name)
        .then(function(result) {
          handleRenderResult(result);
          // Pfad nach erfolgreichem Rendern speichern
          if (!result || !result.error) saveLastFile(droppedFilePath);
        }).catch(function(e) { showDropError('Fehler: ' + e); });
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
    showContent(result.html, result.title, result.scrollPos || 0, result.fileHash || '');
  }
}

// ============================================================
// Inhalt anzeigen / verstecken
// ============================================================

/**
 * Zeigt gerenderten Inhalt an, baut das TOC auf und stellt die Scroll-Position wieder her.
 *
 * @param {string} html      Gerendertes HTML.
 * @param {string} title     Dokumenttitel.
 * @param {number} scrollPos Wiederherzustellende Scroll-Position in Pixeln (0 = Anfang).
 * @param {string} fileHash  FNV-64a-Hash des Dateiinhalts (für Scroll-History).
 */
function showContent(html, title, scrollPos, fileHash) {
  closeSearch(); // Suche zurücksetzen
  currentFileHash = fileHash || '';
  document.getElementById('drop-zone').style.display = 'none';
  var wrapper = document.getElementById('content-wrapper');
  wrapper.style.display = 'block';
  document.getElementById('content').innerHTML = html;
  document.title = title ? title + ' - MD Reader' : 'MD Reader';
  mainEl.scrollTop = 0;
  buildTOC(); // TOC nach dem Rendern aufbauen
  initMermaid(); // Mermaid-Diagramme rendern
  // Gespeicherte Scroll-Position nach kurzem Delay wiederherstellen
  // (Delay nötig damit das DOM vollständig gerendert ist)
  if (scrollPos && scrollPos > 0) {
    setTimeout(function() { mainEl.scrollTop = scrollPos; }, 80);
  }
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

/**
 * Berechnet und setzt die maximale Inhaltsbreite im Hochformat-Modus.
 *
 * Die Breite skaliert linear mit dem Zoom: 750px × Zoom-Faktor,
 * begrenzt auf Fensterbreite − 32 px. Im Querformat wird nichts geändert.
 */
function updatePortraitWidth() {
  if (!isPortrait) return;
  var zoomFactor = fontSize / defaultFontSize;
  // Mindestbreite: 750px (auch bei < 100% Zoom); wächst linear ab 100%
  var targetWidth = Math.max(750, Math.round(750 * zoomFactor));
  var maxAvail = window.innerWidth - 32;
  document.documentElement.style.setProperty(
    '--max-content-width',
    Math.min(targetWidth, maxAvail) + 'px'
  );
}

function applyFontSize() {
  document.documentElement.style.setProperty('--font-size', fontSize + 'px');
  updateZoomLabel();
  updatePortraitWidth();
}

function updateZoomLabel() {
  var pct = Math.round((fontSize / defaultFontSize) * 100);
  document.getElementById('zoom-label').textContent = pct + '%';
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
    updatePortraitWidth(); // Breite zoom-abhängig setzen
    btn.textContent = '\u25AD'; btn.title = 'Querformat anzeigen';
  } else {
    document.documentElement.style.setProperty('--max-content-width', '100%');
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
  searchCurrentIdx = ((idx % searchMatches.length) + searchMatches.length) % searchMatches.length;
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

/**
 * Sendet den Dateipfad an Go zum Speichern als zuletzt geöffnete Datei.
 *
 * Wird nach Drag & Drop aufgerufen. Die Scroll-Position wird separat
 * per saveScrollPos() gespeichert (Hash-basiert).
 *
 * @param {string} path Vollständiger Dateipfad.
 */
function saveLastFile(path) {
  if (typeof window.persistLastFile === 'function') {
    window.persistLastFile(path || '');
  }
}


/** Timer-Handle für debounced Scroll-Speicherung */
var scrollSaveTimer = null;

/**
 * Speichert die Scroll-Position verzögert (debounced, 500ms nach letztem Scroll).
 *
 * Verhindert zu viele Schreibzugriffe auf die Konfigurationsdatei während
 * des Scrollens. Nutzt den Dateiinhalt-Hash als Schlüssel (Hash-basierte History).
 */
function onScrollDebounced() {
  if (scrollSaveTimer) clearTimeout(scrollSaveTimer);
  scrollSaveTimer = setTimeout(function() {
    var wrapper = document.getElementById('content-wrapper');
    if (wrapper && wrapper.style.display !== 'none' && currentFileHash) {
      if (typeof window.saveScrollPos === 'function') {
        window.saveScrollPos(currentFileHash, mainEl ? mainEl.scrollTop : 0);
      }
    }
  }, 500);
}

// ============================================================
// Datei öffnen (nativer Dialog – Lösung für #005)
// ============================================================

/**
 * Öffnet den nativen Datei-Öffnen-Dialog via Go-Binding.
 *
 * Hintergrund: WebView2 (Windows) exponiert aus Sicherheitsgründen keine
 * Dateipfade im DataTransfer. Der native Dialog umgeht dies und liefert
 * den vollständigen Pfad, der dann in lastFile gespeichert werden kann.
 */
function openFile() {
  if (typeof window.openFilePicker === 'function') {
    window.openFilePicker();
  }
}

// ============================================================
// Fenster schließen
// ============================================================

function closeApp() {
  // Scroll-Position vor dem Beenden synchron speichern (Hash-basiert)
  if (currentFileHash && typeof window.saveScrollPos === 'function') {
    window.saveScrollPos(currentFileHash, mainEl ? mainEl.scrollTop : 0);
  }
  // Kurzen Moment warten damit saveScrollPos (async Go-Binding) abgeschlossen wird
  setTimeout(function() {
    if (typeof window._closeAppNative === 'function') window._closeAppNative();
  }, 80);
}

// ============================================================
// Tastatur-Shortcuts
// ============================================================

document.addEventListener('keydown', function(e) {
  var tag = document.activeElement ? document.activeElement.tagName.toLowerCase() : '';
  var inInput = (tag === 'input' || tag === 'textarea');

  // Vollbild: F11
  if (e.key === 'F11') { e.preventDefault(); toggleFullscreen(); return; }

  // Datei öffnen: Strg+O
  if (e.ctrlKey && e.key === 'o') { e.preventDefault(); openFile(); return; }

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

// ============================================================
// Scroll-Position debounced speichern
// ============================================================

// mainEl scroll → Scroll-Position nach 500ms Inaktivität in Konfiguration schreiben
if (mainEl) {
  mainEl.addEventListener('scroll', onScrollDebounced, { passive: true });
}

// Fenstergrößenänderung → Portrait-Breite neu berechnen
window.addEventListener('resize', function() {
  updatePortraitWidth();
}, { passive: true });

// ============================================================
// Mermaid.js – Diagramm-Rendering (Flowcharts, Sequenzdiagramme usw.)
// ============================================================

/**
 * Sucht Mermaid-Code-Blöcke im gerenderten Inhalt und lädt Mermaid.js
 * vom CDN, um sie als SVG-Diagramme darzustellen.
 *
 * Mermaid-Syntax in Markdown:
 *   ```mermaid
 *   graph TD
 *   A --> B
 *   ```
 *
 * goldmark rendert dies als <pre><code class="language-mermaid">...</code></pre>.
 * Diese Funktion wandelt sie in <div class="mermaid">...</div> um.
 *
 * Erfordert Internetzugang zum CDN. Ohne Verbindung bleibt der Code-Block sichtbar.
 */
function initMermaid() {
  // Mermaid-Code-Blöcke finden (goldmark + chroma erzeugen verschiedene Strukturen)
  var blocks = document.querySelectorAll(
    'pre code.language-mermaid, code.language-mermaid, pre > code[class*="mermaid"]'
  );
  if (blocks.length === 0) return;

  // Code-Blöcke in Mermaid-Divs umwandeln
  blocks.forEach(function(block) {
    var pre = block.parentNode;
    var div = document.createElement('div');
    div.className = 'mermaid';
    div.textContent = block.textContent;
    if (pre && pre.tagName === 'PRE') {
      pre.parentNode.replaceChild(div, pre);
    } else {
      block.parentNode.replaceChild(div, block);
    }
  });

  // Mermaid.js dynamisch vom CDN laden
  if (typeof mermaid !== 'undefined') {
    // Bereits geladen → direkt rendern
    renderMermaid();
    return;
  }
  var script = document.createElement('script');
  script.src = 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js';
  script.onload = renderMermaid;
  script.onerror = function() {
    // Kein Internet: Mermaid-Divs als Code-Block wieder anzeigen
    document.querySelectorAll('div.mermaid').forEach(function(d) {
      var pre = document.createElement('pre');
      var code = document.createElement('code');
      code.textContent = d.textContent;
      pre.appendChild(code);
      d.parentNode.replaceChild(pre, d);
    });
  };
  document.head.appendChild(script);
}

/** Initialisiert und rendert alle Mermaid-Diagramme */
function renderMermaid() {
  var theme = currentTheme === 'dark' ? 'dark' : currentTheme === 'retro' ? 'base' : 'default';
  mermaid.initialize({ startOnLoad: false, theme: theme, securityLevel: 'strict' });
  mermaid.run({ querySelector: '.mermaid' });
}
</script>
</body>
</html>
