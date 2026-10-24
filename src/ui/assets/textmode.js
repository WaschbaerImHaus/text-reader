// ============================================================
// Text-Modus (C64-ähnliche Monospace-Ansicht)
// ============================================================
//
// Zeigt Textdokumente (auch EPUB) nur in der eingebetteten Pixelschrift und
// in zwei frei wählbaren Farben an:
//   - Schriftgröße aus der kürzeren Seite des Inhaltsbereichs (minus Rand):
//     50 Spalten, 32 Zeilen mit 1,5-fachem Zeilenabstand.
//   - Bilder werden 1-Bit-gerastert (geordnetes Dithering, Bayer 8x8); ein
//     Bildpunkt ist so groß wie ein Pixel der Schrift (Schriftgröße / 8).
//   - Formeln, Diagramme und SVG-Bilder schneiden SVG-Filter hart auf zwei
//     Farben (Filter in body.html, Farbwerte setzt updateTextModeFilters).
//
// Die Datei läuft als eigener Skriptblock VOR dem Hauptskript (scripts.js).
// Beim Laden definiert sie nur Konstanten und Funktionen und fasst das DOM
// nicht an – so lassen sich die reinen Funktionen mit Node testen
// (testdata/textmode_test.js). Die Zustandsvariablen textModeEnabled,
// textModeAvailable, textModeForeground, textModeBackground und mainEl
// stehen in scripts.js.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24

/** Zeichen je Zeile im Text-Modus */
var TEXT_MODE_COLUMNS = 50;

/** Sichtbare Zeilen je Bildschirm im Text-Modus */
var TEXT_MODE_ROWS = 32;

/** Zeilenabstand (Vielfaches der Schriftgröße) */
var TEXT_MODE_LINE_HEIGHT = 1.5;

/** Rand je Seite in Pixeln (muss zum Padding in textmode.css passen) */
var TEXT_MODE_MARGIN = 16;

/** Die Schrift ist in einem 8x8-Raster gezeichnet: 1 Schrift-Pixel = Größe / 8 */
var TEXT_MODE_GLYPH_PIXELS = 8;

/** Kleinste Schriftgröße in Pixeln (winzige Fenster) */
var TEXT_MODE_MIN_FONT_SIZE = 4;

/**
 * Bayer-Matrix 8x8 für geordnetes Dithering (Werte 0..63).
 *
 * Rekursiver Aufbau: M(2n) = [[4M, 4M+2], [4M+3, 4M+1]]. Geordnetes Dithering
 * ergibt das typische Kreuzraster der 80er-Jahre-Spiele.
 */
var BAYER_8X8 = (function buildBayerMatrix() {
	var matrix = [[0]];
	while (matrix.length < 8) {
		var size = matrix.length;
		var next = [];
		for (var y = 0; y < size * 2; y++) {
			next.push(new Array(size * 2));
		}
		for (var yy = 0; yy < size; yy++) {
			for (var xx = 0; xx < size; xx++) {
				var base = matrix[yy][xx] * 4;
				next[yy][xx] = base;
				next[yy][xx + size] = base + 2;
				next[yy + size][xx] = base + 3;
				next[yy + size][xx + size] = base + 1;
			}
		}
		matrix = next;
	}
	return matrix;
})();

// ============================================================
// Reine Funktionen (ohne DOM, mit Node getestet)
// ============================================================

/**
 * Berechnet die Schriftgröße aus der kürzeren Seite des Inhaltsbereichs.
 *
 * Maßgeblich ist nur die kürzere Seite: 1920x1080 und 1080x1090 ergeben
 * dieselbe Größe. Die Zeichen sind quadratisch (1 Zeichen = 1em breit), daher
 * passen bei Größe = Seite / 50 genau 50 Spalten in die Breite; 32 Zeilen mit
 * 1,5-fachem Abstand brauchen nur das 48-fache und passen ebenfalls.
 *
 * @param {number} width  Breite des Inhaltsbereichs in px.
 * @param {number} height Höhe des Inhaltsbereichs in px.
 * @param {number} margin Rand je Seite in px.
 * @returns {number} Ganzzahlige Schriftgröße in px (mindestens 4).
 * @lastModified 2026-10-24
 */
function computeTextModeFontSize(width, height, margin) {
	var side = Math.min(width, height) - 2 * margin;
	if (!isFinite(side) || side <= 0) return TEXT_MODE_MIN_FONT_SIZE;
	var byColumns = side / TEXT_MODE_COLUMNS;
	var byRows = side / (TEXT_MODE_ROWS * TEXT_MODE_LINE_HEIGHT);
	// Abrunden auf ganze Pixel: so ragt die 50. Spalte nie über den Rand
	var size = Math.floor(Math.min(byColumns, byRows));
	return Math.max(TEXT_MODE_MIN_FONT_SIZE, size);
}

/**
 * Wandelt "#rrggbb" in RGB-Werte um.
 *
 * @param {string} hex Farbe als "#rrggbb".
 * @returns {?{r:number,g:number,b:number}} RGB (0..255) oder null bei ungültiger Farbe.
 * @lastModified 2026-10-24
 */
function hexToRgb(hex) {
	var match = /^#([0-9a-fA-F]{6})$/.exec(hex || '');
	if (!match) return null;
	var value = parseInt(match[1], 16);
	return { r: (value >> 16) & 255, g: (value >> 8) & 255, b: value & 255 };
}

/**
 * Wahrgenommene Helligkeit einer Farbe (Gewichtung nach ITU-R BT.601).
 *
 * @param {{r:number,g:number,b:number}} rgb Farbe.
 * @returns {number} Helligkeit 0..255.
 * @lastModified 2026-10-24
 */
function colorLuminance(rgb) {
	return 0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b;
}

/**
 * Ordnet Vorder- und Hintergrundfarbe nach Helligkeit.
 *
 * Bilder behalten so ihren Charakter: Helles im Bild wird zur helleren der
 * beiden Farben, Dunkles zur dunkleren – egal welche davon die Schrift ist.
 *
 * @param {string} foregroundHex Schriftfarbe "#rrggbb".
 * @param {string} backgroundHex Hintergrundfarbe "#rrggbb".
 * @returns {{bright:Object, dark:Object}} Hellere und dunklere Farbe als RGB.
 * @lastModified 2026-10-24
 */
function splitBrightDark(foregroundHex, backgroundHex) {
	var fg = hexToRgb(foregroundHex) || { r: 255, g: 255, b: 255 };
	var bg = hexToRgb(backgroundHex) || { r: 0, g: 0, b: 0 };
	if (colorLuminance(fg) >= colorLuminance(bg)) {
		return { bright: fg, dark: bg };
	}
	return { bright: bg, dark: fg };
}

/**
 * Baut die tableValues für einen diskreten SVG-Farbfilter.
 *
 * feFuncX type="discrete" mit zwei Werten: Kanal unter 50 % → erster Wert,
 * ab 50 % → zweiter Wert. Werte im Bereich 0..1.
 *
 * @param {{r:number,g:number,b:number}} below Farbe für dunkle Stellen.
 * @param {{r:number,g:number,b:number}} above Farbe für helle Stellen.
 * @returns {{r:string,g:string,b:string}} tableValues je Kanal.
 * @lastModified 2026-10-24
 */
function filterTableValues(below, above) {
	function unit(value) {
		return String(Number((value / 255).toFixed(4)));
	}
	return {
		r: unit(below.r) + ' ' + unit(above.r),
		g: unit(below.g) + ' ' + unit(above.g),
		b: unit(below.b) + ' ' + unit(above.b)
	};
}

/**
 * Berechnet das Raster (Bildpunkte) für ein Bild im Text-Modus.
 *
 * Das Bild wird höchstens so breit wie der Inhalt angezeigt; jeder Bildpunkt
 * ist ein Schrift-Pixel groß. Bei 50 Spalten à 8 Pixel sind das höchstens
 * 400 Bildpunkte in der Breite – die Rasterung bleibt dadurch schnell.
 *
 * @param {number} naturalWidth  Ursprüngliche Bildbreite in px.
 * @param {number} naturalHeight Ursprüngliche Bildhöhe in px.
 * @param {number} maxWidth      Verfügbare Breite in px (0 = unbegrenzt).
 * @param {number} pixelSize     Größe eines Bildpunkts in px.
 * @returns {{cols:number, rows:number}} Rastergröße (0x0 bei ungültigen Werten).
 * @lastModified 2026-10-24
 */
function ditherGridSize(naturalWidth, naturalHeight, maxWidth, pixelSize) {
	if (!(naturalWidth > 0 && naturalHeight > 0 && pixelSize > 0)) {
		return { cols: 0, rows: 0 };
	}
	var displayWidth = maxWidth > 0 ? Math.min(naturalWidth, maxWidth) : naturalWidth;
	var cols = Math.max(1, Math.round(displayWidth / pixelSize));
	var rows = Math.max(1, Math.round(cols * naturalHeight / naturalWidth));
	return { cols: cols, rows: rows };
}

/**
 * Rastert RGBA-Pixel auf 1 Bit (geordnetes Dithering mit Bayer 8x8).
 *
 * @param {Uint8ClampedArray} rgba Pixeldaten (4 Byte je Pixel).
 * @param {number} width  Breite in Pixeln.
 * @param {number} height Höhe in Pixeln.
 * @returns {Uint8Array} Je Pixel 1 = hell, 0 = dunkel, 255 = transparent.
 * @lastModified 2026-10-24
 */
function ditherToMask(rgba, width, height) {
	var mask = new Uint8Array(width * height);
	for (var y = 0; y < height; y++) {
		var row = BAYER_8X8[y & 7];
		for (var x = 0; x < width; x++) {
			var i = y * width + x;
			var p = i * 4;
			// Überwiegend durchsichtige Pixel bleiben durchsichtig (Hintergrund scheint durch)
			if (rgba[p + 3] < 128) {
				mask[i] = 255;
				continue;
			}
			var luminance = 0.299 * rgba[p] + 0.587 * rgba[p + 1] + 0.114 * rgba[p + 2];
			// Schwelle aus der Bayer-Matrix, gleichmäßig über 0..255 verteilt
			var threshold = (row[x & 7] + 0.5) * 255 / 64;
			mask[i] = luminance > threshold ? 1 : 0;
		}
	}
	return mask;
}

/**
 * Färbt eine 1-Bit-Maske in zwei Farben ein.
 *
 * @param {Uint8Array} mask Maske aus ditherToMask.
 * @param {number} width  Breite in Pixeln.
 * @param {number} height Höhe in Pixeln.
 * @param {{r:number,g:number,b:number}} bright Farbe für helle Pixel.
 * @param {{r:number,g:number,b:number}} dark   Farbe für dunkle Pixel.
 * @returns {Uint8ClampedArray} RGBA-Pixeldaten.
 * @lastModified 2026-10-24
 */
function paintMask(mask, width, height, bright, dark) {
	var out = new Uint8ClampedArray(width * height * 4);
	for (var i = 0; i < width * height; i++) {
		var p = i * 4;
		if (mask[i] === 255) continue; // bleibt durchsichtig (alles 0)
		var color = mask[i] === 1 ? bright : dark;
		out[p] = color.r;
		out[p + 1] = color.g;
		out[p + 2] = color.b;
		out[p + 3] = 255;
	}
	return out;
}

// ============================================================
// DOM-Teil (nur im WebView)
// ============================================================

/** Zuletzt gesetzte Schriftgröße (erkennt, ob Bilder neu gerastert werden müssen) */
var textModeFontSize = 0;

/** Timer für verzögerte Neuberechnung beim Ändern der Fenstergröße */
var textModeResizeTimer = null;

/** Timer für verzögertes Umfärben der Bilder beim Ziehen im Farbwähler */
var textModeRepaintTimer = null;

/**
 * Ist der Text-Modus gerade wirksam?
 *
 * @returns {boolean} true wenn eingeschaltet UND das Dokument ein Textformat ist.
 * @lastModified 2026-10-24
 */
function isTextModeActive() {
	return textModeEnabled && textModeAvailable;
}

/**
 * Initialisiert den Text-Modus beim Seitenaufbau (vom Hauptskript aufgerufen).
 *
 * Blendet Knopf, Trenner und Farbfelder nur bei Textformaten ein, verdrahtet die
 * Farbfelder (keine Inline-Handler wegen CSP) und wendet einen gespeicherten
 * eingeschalteten Modus an.
 *
 * @lastModified 2026-10-24
 */
function initTextMode() {
	if (!textModeAvailable) return;
	var btn = document.getElementById('btn-textmode');
	var sep = document.getElementById('sep-textmode');
	var colors = document.getElementById('tm-colors');
	if (btn) btn.style.display = '';
	if (sep) sep.style.display = '';
	// Farbfelder belegen immer ihren Platz (sichtbar nur im Text-Modus, CSS)
	if (colors) colors.style.display = 'flex';

	var fgInput = document.getElementById('tm-fg');
	var bgInput = document.getElementById('tm-bg');
	if (fgInput && bgInput) {
		// 'input' feuert laufend beim Ziehen → sofort anzeigen;
		// 'change' feuert beim Schließen des Dialogs → dann speichern
		var onInput = function() { setTextModeColors(fgInput.value, bgInput.value); };
		var onChange = function() { onInput(); persistTextModeState(); };
		fgInput.addEventListener('input', onInput);
		bgInput.addEventListener('input', onInput);
		fgInput.addEventListener('change', onChange);
		bgInput.addEventListener('change', onChange);
	}

	window.addEventListener('resize', onTextModeResize, { passive: true });
	applyTextMode();
}

/**
 * Schaltet den Text-Modus um (Toolbar-Knopf) und hält die Leseposition.
 *
 * @lastModified 2026-10-24
 */
function toggleTextMode() {
	if (!textModeAvailable) return;
	// Position als Anteil merken: die Seitenhöhe ändert sich mit der Schrift
	var ratio = mainEl.scrollHeight > 0 ? mainEl.scrollTop / mainEl.scrollHeight : 0;
	textModeEnabled = !textModeEnabled;
	applyTextMode();
	mainEl.scrollTo({ top: Math.round(ratio * mainEl.scrollHeight), behavior: 'instant' });
	persistTextModeState();
}

/**
 * Wendet den aktuellen Zustand an: Klasse, Größe, Farben und Bilder.
 *
 * @lastModified 2026-10-24
 */
function applyTextMode() {
	var active = isTextModeActive();
	document.body.classList.toggle('textmode', active);
	updateTextModeButton();
	if (active) {
		applyTextModeSize();
		updateTextModeColors();
		ditherAllImages();
	} else {
		restoreAllImages();
	}
}

/**
 * Aktualisiert Symbol-Titel und Zustand des Toolbar-Knopfs.
 *
 * @lastModified 2026-10-24
 */
function updateTextModeButton() {
	var btn = document.getElementById('btn-textmode');
	if (!btn) return;
	btn.title = textModeEnabled ? 'Text-Modus ausschalten' : 'Text-Modus einschalten';
	btn.setAttribute('aria-pressed', textModeEnabled ? 'true' : 'false');
}

/**
 * Setzt die Schriftgröße aus der Größe des Inhaltsbereichs.
 *
 * @returns {boolean} true wenn sich die Größe geändert hat.
 * @lastModified 2026-10-24
 */
function applyTextModeSize() {
	var size = computeTextModeFontSize(mainEl.clientWidth, mainEl.clientHeight, TEXT_MODE_MARGIN);
	document.documentElement.style.setProperty('--tm-font-size', size + 'px');
	var changed = size !== textModeFontSize;
	textModeFontSize = size;
	return changed;
}

/**
 * Fenstergröße geändert → Schrift neu berechnen, Bilder bei Bedarf neu rastern.
 * Verzögert, damit beim Ziehen am Fensterrand nicht jedes Ereignis rechnet.
 *
 * @lastModified 2026-10-24
 */
function onTextModeResize() {
	if (textModeResizeTimer) clearTimeout(textModeResizeTimer);
	textModeResizeTimer = setTimeout(function() {
		if (!isTextModeActive()) return;
		if (applyTextModeSize()) ditherAllImages();
	}, 150);
}

/**
 * Übernimmt neue Farben aus den Farbfeldern.
 *
 * @param {string} foreground Schriftfarbe "#rrggbb".
 * @param {string} background Hintergrundfarbe "#rrggbb".
 * @lastModified 2026-10-24
 */
function setTextModeColors(foreground, background) {
	// Nur exakt "#rrggbb" übernehmen (Go prüft beim Speichern erneut)
	if (hexToRgb(foreground)) textModeForeground = foreground.toLowerCase();
	if (hexToRgb(background)) textModeBackground = background.toLowerCase();
	updateTextModeColors();
	// Bilder verzögert umfärben (Maske bleibt, nur die Farben ändern sich)
	if (textModeRepaintTimer) clearTimeout(textModeRepaintTimer);
	textModeRepaintTimer = setTimeout(function() {
		if (isTextModeActive()) ditherAllImages();
	}, 100);
}

/**
 * Setzt CSS-Variablen, Farbfelder und SVG-Filter auf die aktuellen Farben.
 *
 * @lastModified 2026-10-24
 */
function updateTextModeColors() {
	var root = document.documentElement.style;
	root.setProperty('--tm-fg', textModeForeground);
	root.setProperty('--tm-bg', textModeBackground);
	var fgInput = document.getElementById('tm-fg');
	var bgInput = document.getElementById('tm-bg');
	if (fgInput && fgInput.value !== textModeForeground) fgInput.value = textModeForeground;
	if (bgInput && bgInput.value !== textModeBackground) bgInput.value = textModeBackground;
	updateTextModeFilters();
}

/**
 * Trägt die Farben in die SVG-Filter ein (body.html, #tm-filters).
 *
 *   - ink-dark:  dunkle Striche (helle Diagramme) → Schrift, Rest → Hintergrund
 *   - ink-light: helle Striche (dunkle Diagramme) → Schrift, Rest → Hintergrund
 *   - photo:     Bilder: hell → hellere Farbe, dunkel → dunklere Farbe
 *
 * @lastModified 2026-10-24
 */
function updateTextModeFilters() {
	var fg = hexToRgb(textModeForeground);
	var bg = hexToRgb(textModeBackground);
	if (!fg || !bg) return;
	var pair = splitBrightDark(textModeForeground, textModeBackground);
	setFilterTables('tm-filter-ink-dark', filterTableValues(fg, bg));
	setFilterTables('tm-filter-ink-light', filterTableValues(bg, fg));
	setFilterTables('tm-filter-photo', filterTableValues(pair.dark, pair.bright));
}

/**
 * Setzt die tableValues der Farbkanäle eines SVG-Filters.
 *
 * @param {string} filterId ID des <filter>-Elements.
 * @param {{r:string,g:string,b:string}} values tableValues je Kanal.
 * @lastModified 2026-10-24
 */
function setFilterTables(filterId, values) {
	var filter = document.getElementById(filterId);
	if (!filter) return;
	// Namensraum-Suche: SVG-Tagnamen sind gemischt geschrieben (feFuncR)
	var ns = 'http://www.w3.org/2000/svg';
	var channels = [['feFuncR', values.r], ['feFuncG', values.g], ['feFuncB', values.b]];
	channels.forEach(function(entry) {
		var funcs = filter.getElementsByTagNameNS(ns, entry[0]);
		for (var i = 0; i < funcs.length; i++) funcs[i].setAttribute('tableValues', entry[1]);
	});
}

/**
 * Sendet den Zustand an Go zum Speichern (Binding persistTextMode).
 *
 * @lastModified 2026-10-24
 */
function persistTextModeState() {
	if (typeof window.persistTextMode === 'function') {
		window.persistTextMode(textModeEnabled, textModeForeground, textModeBackground);
	}
}

/** Namensraum für xlink:href in SVG-Bildern (ältere EPUBs/Calibre-Cover) */
var XLINK_NS = 'http://www.w3.org/1999/xlink';

/**
 * Liefert alle Bilder im Inhalt: <img> und <image> in SVGs (EPUB-Cover).
 *
 * @returns {Element[]} Bildelemente.
 * @lastModified 2026-10-24
 */
function textModeImages() {
	return Array.prototype.slice.call(document.querySelectorAll('#content img, #content svg image'));
}

/**
 * Rastert alle Bilder im Inhalt nacheinander (mit Pausen, damit die
 * Oberfläche bei Büchern mit vielen Bildern bedienbar bleibt).
 *
 * @lastModified 2026-10-24
 */
function ditherAllImages() {
	var images = textModeImages();
	var index = 0;
	(function next() {
		if (!isTextModeActive()) return;
		var end = Math.min(index + 8, images.length);
		for (; index < end; index++) ditherImageElement(images[index]);
		if (index < images.length) setTimeout(next, 0);
	})();
}

/**
 * Liest die Bildquelle eines <img> oder SVG-<image>.
 *
 * @param {Element} el Bildelement.
 * @returns {string} Quelle (data:-URI) oder ''.
 * @lastModified 2026-10-24
 */
function imageSourceOf(el) {
	if (el.tagName.toLowerCase() === 'img') return el.getAttribute('src') || '';
	return el.getAttribute('href') || el.getAttributeNS(XLINK_NS, 'href') || '';
}

/**
 * Rastert ein Bild 1-Bit in den aktuellen Farben und ersetzt seine Anzeige.
 *
 * Das Original bleibt in el.__tmSource erhalten (für das Ausschalten und
 * für neue Größen). Die 1-Bit-Maske wird zwischengespeichert: Beim Farbwechsel
 * wird nur neu eingefärbt. Scheitert das Auslesen der Pixel, bleibt der
 * CSS-Filter (harter Zwei-Farben-Schnitt) als Ersatz aktiv.
 *
 *   - <img>: höchstens Inhaltsbreite, Breite = Bildpunkte × Schrift-Pixel.
 *   - SVG-<image>: Größe bestimmt das SVG; gerastert wird in der
 *     angezeigten Breite, das SVG skaliert das Ergebnis selbst.
 *
 * @param {Element} el Bildelement (<img> oder SVG-<image>).
 * @lastModified 2026-10-24
 */
function ditherImageElement(el) {
	var isImg = el.tagName.toLowerCase() === 'img';
	var source = el.__tmSource;
	if (!source) {
		source = el.__tmSource = {
			src: imageSourceOf(el),
			hadHref: el.hasAttribute('href'),
			styleWidth: el.style.width,
			image: new Image(),
			mask: null,
			maskKey: '',
			failed: false
		};
		source.image.src = source.src;
	}
	if (source.failed || !source.src) return;
	// Original noch nicht dekodiert → nach dem Laden erneut versuchen
	if (!source.image.complete || !source.image.naturalWidth) {
		if (!source.waiting) {
			source.waiting = true;
			source.image.addEventListener('load', function() {
				source.waiting = false;
				if (isTextModeActive()) ditherImageElement(el);
			}, { once: true });
		}
		return;
	}

	var pixelSize = textModeFontSize / TEXT_MODE_GLYPH_PIXELS;
	var naturalWidth = source.image.naturalWidth;
	var naturalHeight = source.image.naturalHeight;
	var maxWidth;
	if (isImg) {
		var content = document.getElementById('content');
		maxWidth = content ? content.clientWidth : 0;
	} else {
		// SVG: in der tatsächlich angezeigten Breite rastern (auch vergrößert)
		var shownWidth = el.getBoundingClientRect().width;
		if (!(shownWidth > 0)) return;
		naturalHeight = shownWidth * naturalHeight / naturalWidth;
		naturalWidth = shownWidth;
		maxWidth = 0;
	}
	var key = pixelSize + ':' + maxWidth + ':' + naturalWidth;

	var canvas = document.createElement('canvas');
	if (!source.mask || source.maskKey !== key) {
		var grid = ditherGridSize(naturalWidth, naturalHeight, maxWidth, pixelSize);
		if (!grid.cols) return;
		canvas.width = grid.cols;
		canvas.height = grid.rows;
		var ctx = canvas.getContext('2d');
		ctx.imageSmoothingEnabled = true;
		ctx.imageSmoothingQuality = 'high';
		ctx.drawImage(source.image, 0, 0, grid.cols, grid.rows);
		var pixels;
		try {
			pixels = ctx.getImageData(0, 0, grid.cols, grid.rows);
		} catch (e) {
			source.failed = true; // z. B. "tainted canvas" → CSS-Filter bleibt
			return;
		}
		source.mask = ditherToMask(pixels.data, grid.cols, grid.rows);
		source.cols = grid.cols;
		source.rows = grid.rows;
		source.maskKey = key;
	} else {
		canvas.width = source.cols;
		canvas.height = source.rows;
	}

	var pair = splitBrightDark(textModeForeground, textModeBackground);
	var paintCtx = canvas.getContext('2d');
	var out = paintCtx.createImageData(source.cols, source.rows);
	out.data.set(paintMask(source.mask, source.cols, source.rows, pair.bright, pair.dark));
	paintCtx.putImageData(out, 0, 0);
	var dithered = canvas.toDataURL('image/png');
	if (isImg) {
		el.src = dithered;
		// Jeder Bildpunkt = ein Schrift-Pixel; Glättung aus (image-rendering: pixelated)
		el.style.width = (source.cols * pixelSize) + 'px';
	} else {
		el.setAttribute('href', dithered); // href hat Vorrang vor xlink:href
		// Gerastertes Bild hat schon zwei Farben → Zwei-Farben-Filter am SVG aus
		if (el.ownerSVGElement) el.ownerSVGElement.classList.add('tm-dithered-svg');
	}
	el.classList.add('tm-dithered');
}

/**
 * Stellt beim Ausschalten alle Originalbilder wieder her.
 *
 * @lastModified 2026-10-24
 */
function restoreAllImages() {
	textModeImages().forEach(function(el) {
		var source = el.__tmSource;
		if (!source || !el.classList.contains('tm-dithered')) return;
		if (el.tagName.toLowerCase() === 'img') {
			el.src = source.src;
			el.style.width = source.styleWidth;
		} else if (source.hadHref) {
			el.setAttribute('href', source.src);
		} else {
			el.removeAttribute('href'); // xlink:href gilt wieder
		}
		if (el.ownerSVGElement) el.ownerSVGElement.classList.remove('tm-dithered-svg');
		el.classList.remove('tm-dithered');
	});
}
