#!/usr/bin/env bash
# Wächter für die glibc-Untergrenze der Linux-Programme von MD Reader.
#
# Aufruf: check-glibc.sh <max> <datei>...
#         check-glibc.sh --max <datei>
#   <max>    höchste erlaubte glibc-Version, z. B. 2.39
#   <datei>  ELF-Dateien (x86_64, ARM64 oder ARMhf), die geprüft werden
#
# Prüfmodus: Liest mit "objdump -T" alle verlangten Symbolversionen GLIBC_x.y und
# vergleicht sie mit <max> (Versionsvergleich per "sort -V"). Ist ein Symbol
# neuer als <max>, werden die betroffenen Symbole ausgegeben und das Skript
# endet mit Exit 1. Sonst Exit 0.
#
# Modus --max: gibt nur die höchste verlangte glibc-Version der einen Datei
# aus (z. B. 2.38). build-deb.sh leitet daraus die Abhängigkeit libc6 ab.
# Enthält die Datei keine GLIBC-Symbole, endet das Skript mit Exit 1.
#
# Grund des Wächters: Die Build-Maschine hat eine neuere glibc als die
# Zielsysteme (Ubuntu 24.04 hat 2.39). Eine unbedachte Funktion könnte die
# Untergrenze still anheben.
#
# Exit-Codes: 0 = in Ordnung, 1 = zu neu / keine Symbole, 2 = Aufruf- oder Lesefehler.
#
# Umgebung:
#   OBJDUMP  überschreibt das Prüfprogramm (sonst je nach Architektur der Datei:
#            aarch64-linux-gnu-objdump bzw. arm-linux-gnueabihf-objdump, falls
#            vorhanden, sonst objdump)
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-28 (Fix-Runde 1: --max-Modus, Schleifen ohne Wortaufteilung)
set -euo pipefail

# Hilfetext bei falschem Aufruf
usage() {
	echo "Aufruf: $0 <max-glibc-version> <datei>..." >&2
	echo "        $0 --max <datei>" >&2
	exit 2
}

# Wählt das passende objdump anhand der Architektur der ELF-Datei.
# Byte 18 des ELF-Kopfs (e_machine): 0xb7 = AArch64, 0x28 = ARM.
pick_objdump() {
	local file="$1" machine candidate
	if [ -n "${OBJDUMP:-}" ]; then
		printf '%s\n' "${OBJDUMP}"
		return
	fi
	candidate="objdump"
	machine="$(od -An -tx1 -j18 -N1 -- "${file}" | tr -d ' \n')"
	case "${machine}" in
		b7) candidate="aarch64-linux-gnu-objdump" ;;
		28) candidate="arm-linux-gnueabihf-objdump" ;;
	esac
	# Fehlt das spezielle Programm, genügt das normale objdump (liest fremde ELF)
	if ! command -v "${candidate}" > /dev/null 2>&1; then
		candidate="objdump"
	fi
	printf '%s\n' "${candidate}"
}

# Liest die dynamischen Symbole einer Datei nach "symbols" (Text) und die
# eindeutigen glibc-Versionen (ohne GLIBC_PRIVATE) nach "versions" (eine je Zeile).
symbols=""
versions=""
read_symbols() {
	local file="$1" tool
	if [ ! -r "${file}" ]; then
		echo "FEHLER: Datei nicht lesbar: ${file}" >&2
		exit 2
	fi
	tool="$(pick_objdump "${file}")"
	if ! symbols="$("${tool}" -T -- "${file}")"; then
		echo "FEHLER: ${tool} -T scheiterte für ${file}" >&2
		exit 2
	fi
	versions="$(printf '%s\n' "${symbols}" | grep -o 'GLIBC_[0-9][0-9.]*' | sort -u || true)"
}

if [ "$#" -lt 2 ]; then
	usage
fi

# Modus --max: höchste verlangte Version ausgeben
if [ "$1" = "--max" ]; then
	if [ "$#" -ne 2 ]; then
		usage
	fi
	read_symbols "$2"
	if [ -z "${versions}" ]; then
		echo "FEHLER: $2 verlangt keine GLIBC-Symbole" >&2
		exit 1
	fi
	printf '%s\n' "${versions}" | sed 's/^GLIBC_//' | sort -V | tail -n 1
	exit 0
fi

max="$1"
shift

# Nur Zahlen und Punkte erlaubt (die Version wird später in Mustern verwendet)
case "${max}" in
	''|*[!0-9.]*)
		echo "Ungültige Version: ${max}" >&2
		exit 2
		;;
esac

failed=0
for file in "$@"; do
	read_symbols "${file}"
	# Versionen sammeln, die neuer als max sind (eine je Zeile)
	too_new=""
	while IFS= read -r version; do
		[ -n "${version}" ] || continue
		number="${version#GLIBC_}"
		# Neuer als max: größte der beiden Versionen ist nicht max selbst
		highest="$(printf '%s\n%s\n' "${number}" "${max}" | sort -V | tail -n 1)"
		if [ "${highest}" != "${max}" ]; then
			too_new="${too_new}${version}"$'\n'
		fi
	done <<< "${versions}"
	if [ -n "${too_new}" ]; then
		echo "FEHLER: ${file} verlangt glibc neuer als ${max}:" >&2
		while IFS= read -r version; do
			[ -n "${version}" ] || continue
			printf '%s\n' "${symbols}" | grep -F "(${version})" | sed 's/^/\t/' >&2 || true
		done <<< "${too_new}"
		failed=1
	else
		top="$(printf '%s\n' "${versions}" | sed 's/^GLIBC_//' | sort -V | tail -n 1)"
		echo "OK: ${file} (höchste glibc-Version: ${top:-keine}, erlaubt bis ${max})"
	fi
done

exit "${failed}"
