// Unit-Tests für die reinen Funktionen aus ui/assets/textmode.js (ohne DOM).
//
// Aufruf: node testdata/textmode_test.js assets/textmode.js
// Wird von TestTextModeJS_Node (textmode_test.go) gestartet.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-24
'use strict';

const fs = require('fs');
const vm = require('vm');

// textmode.js in eine eigene Umgebung ohne document/window laden:
// Die Datei darf beim Laden nichts am DOM tun, nur Funktionen definieren.
const source = fs.readFileSync(process.argv[2], 'utf8');
const sandbox = { Math, Uint8Array, Uint8ClampedArray, parseInt, isFinite, Number };
vm.createContext(sandbox);
vm.runInContext(source, sandbox);

let failures = 0;
let passed = 0;

/**
 * Vergleicht zwei Werte und zählt Fehler.
 * @param {string} name Beschreibung des Prüfpunkts.
 * @param {*} actual Tatsächlicher Wert.
 * @param {*} expected Erwarteter Wert.
 */
function check(name, actual, expected) {
	const a = JSON.stringify(actual);
	const e = JSON.stringify(expected);
	if (a !== e) {
		failures++;
		console.log('FEHLER ' + name + ': ' + a + ' statt ' + e);
	} else {
		passed++;
	}
}

// ---------- Schriftgröße: kürzere Seite / 50, mit Rand ----------
const size = sandbox.computeTextModeFontSize;
// 1920x1080 und 1080x1090 ergeben dieselbe Größe (kürzere Seite 1080)
check('1920x1080', size(1920, 1080, 16), 20);          // (1080-32)/50 = 20,96 → 20
check('1080x1090', size(1080, 1090, 16), 20);
check('gleich bei Querformat/Hochformat', size(1920, 1080, 16), size(1080, 1920, 16));
// 50 Spalten passen in die Breite, 32 Zeilen × 1,5 in die Höhe
[[1920, 1080], [1080, 1920], [800, 600], [1366, 768], [3840, 2160]].forEach(function (wh) {
	const fs = size(wh[0], wh[1], 16);
	check('50 Spalten passen ' + wh, fs * 50 <= wh[0] - 32, true);
	check('32 Zeilen passen ' + wh, fs * 32 * 1.5 <= wh[1] - 32, true);
});
check('ganzzahlige Pixel', Number.isInteger(size(1001, 999, 16)), true);
// Randfälle: winziges Fenster, 0, negative oder ungültige Werte → Mindestgröße
check('winzig', size(100, 100, 16), 4);
check('null', size(0, 0, 16), 4);
check('negativ', size(-50, 500, 16), 4);
check('NaN', size(NaN, 500, 16), 4);
check('ohne Rand', size(1000, 1000, 0), 20);

// ---------- Farben ----------
check('hexToRgb', sandbox.hexToRgb('#40318d'), { r: 64, g: 49, b: 141 });
check('hexToRgb groß', sandbox.hexToRgb('#FFAA00'), { r: 255, g: 170, b: 0 });
check('hexToRgb ungültig', sandbox.hexToRgb('red'), null);
check('hexToRgb kurz', sandbox.hexToRgb('#fff'), null);
check('Luminanz weiß > schwarz',
	sandbox.colorLuminance({ r: 255, g: 255, b: 255 }) > sandbox.colorLuminance({ r: 0, g: 0, b: 0 }), true);

// Hell/dunkel-Zuordnung: Bild-Helligkeit landet auf der helleren Farbe
let pair = sandbox.splitBrightDark('#7869c4', '#40318d');
check('C64: hell = Vordergrund', pair.bright, { r: 120, g: 105, b: 196 });
check('C64: dunkel = Hintergrund', pair.dark, { r: 64, g: 49, b: 141 });
pair = sandbox.splitBrightDark('#000000', '#ffffff');
check('schwarz auf weiß: hell = Hintergrund', pair.bright, { r: 255, g: 255, b: 255 });
check('schwarz auf weiß: dunkel = Vordergrund', pair.dark, { r: 0, g: 0, b: 0 });
pair = sandbox.splitBrightDark('#123456', '#123456');
check('gleiche Farben: kein Absturz', pair.bright, pair.dark);

// SVG-Filterwerte: 0..1 mit 4 Nachkommastellen, erst Wert für "unter 50 %"
check('filterTableValues', sandbox.filterTableValues({ r: 0, g: 255, b: 51 }, { r: 255, g: 0, b: 102 }),
	{ r: '0 1', g: '1 0', b: '0.2 0.4' });

// ---------- Rastergröße ----------
const grid = sandbox.ditherGridSize;
// 800x600-Bild, Inhalt 1000 px breit, Bildpunkt 2,5 px → natürliche Breite 800 / 2,5 = 320
check('Raster natürliche Größe', grid(800, 600, 1000, 2.5), { cols: 320, rows: 240 });
// Bild breiter als Inhalt → auf Inhaltsbreite begrenzt
check('Raster begrenzt', grid(4000, 2000, 1000, 2.5), { cols: 400, rows: 200 });
// Winzige Bilder bleiben mindestens 1x1
check('Raster winzig', grid(1, 1, 1000, 2.5), { cols: 1, rows: 1 });
check('Raster ungültig', grid(0, 0, 1000, 2.5), { cols: 0, rows: 0 });
check('Raster Bildpunkt 0', grid(100, 100, 1000, 0), { cols: 0, rows: 0 });

// ---------- 1-Bit-Rasterung (Bayer 8x8) ----------
/**
 * Erzeugt ein RGBA-Feld mit einer einheitlichen Farbe.
 */
function solid(w, h, r, g, b, a) {
	const d = new Uint8ClampedArray(w * h * 4);
	for (let i = 0; i < w * h; i++) {
		d[i * 4] = r; d[i * 4 + 1] = g; d[i * 4 + 2] = b; d[i * 4 + 3] = a;
	}
	return d;
}
/** Zählt die Werte einer Maske. */
function countMask(mask, value) {
	let n = 0;
	for (let i = 0; i < mask.length; i++) if (mask[i] === value) n++;
	return n;
}
const dither = sandbox.ditherToMask;
check('weiß → alles hell', countMask(dither(solid(8, 8, 255, 255, 255, 255), 8, 8), 1), 64);
check('schwarz → alles dunkel', countMask(dither(solid(8, 8, 0, 0, 0, 255), 8, 8), 0), 64);
// 50 % Grau → genau die Hälfte der Bayer-Schwellen liegt darunter
check('grau 50 % → halb hell', countMask(dither(solid(8, 8, 128, 128, 128, 255), 8, 8), 1), 32);
// 25 % Grau → ein Viertel hell
check('grau 25 % → viertel hell', countMask(dither(solid(8, 8, 64, 64, 64, 255), 8, 8), 1), 16);
// Transparenz bleibt transparent (Hintergrund scheint durch)
check('transparent', countMask(dither(solid(4, 4, 255, 255, 255, 0), 4, 4), 255), 16);
check('halbtransparent = deckend', countMask(dither(solid(4, 4, 255, 255, 255, 200), 4, 4), 1), 16);
check('nur zwei Zustände + transparent', (function () {
	const m = dither(solid(16, 16, 90, 140, 200, 255), 16, 16);
	for (let i = 0; i < m.length; i++) if (m[i] !== 0 && m[i] !== 1) return false;
	return true;
})(), true);
// Das Muster ist geordnet (Bayer): gleiches Grau ergibt in jeder 8x8-Kachel dasselbe Muster
check('Bayer-Kacheln gleich', (function () {
	const m = dither(solid(16, 8, 100, 100, 100, 255), 16, 8);
	for (let y = 0; y < 8; y++) for (let x = 0; x < 8; x++) {
		if (m[y * 16 + x] !== m[y * 16 + x + 8]) return false;
	}
	return true;
})(), true);

// Einfärben: hell → helle Farbe, dunkel → dunkle Farbe, transparent → Alpha 0
const painted = sandbox.paintMask(new Uint8Array([1, 0, 255]), 3, 1,
	{ r: 10, g: 20, b: 30 }, { r: 40, g: 50, b: 60 });
check('paintMask', Array.from(painted), [10, 20, 30, 255, 40, 50, 60, 255, 0, 0, 0, 0]);

console.log(passed + ' JS-Prüfungen bestanden, ' + failures + ' fehlgeschlagen');
process.exit(failures === 0 ? 0 : 1);
