#!/usr/bin/env bash
# Wächter für die glibc-Untergrenze der Linux-Programme von MD Reader.
#
# Aufruf: check-glibc.sh <max> <datei>...
#   <max>    höchste erlaubte glibc-Version, z. B. 2.39
#   <datei>  ELF-Dateien (x86_64, ARM64 oder ARMhf), die geprüft werden
#
# Liest mit "objdump -T" alle verlangten Symbolversionen GLIBC_x.y und
# vergleicht sie mit <max> (Versionsvergleich per "sort -V"). Ist ein Symbol
# neuer als <max>, werden die betroffenen Symbole ausgegeben und das Skript
# endet mit Exit 1. Sonst Exit 0.
#
# Grund: Die Build-Maschine hat eine neuere glibc als die Zielsysteme
# (Ubuntu 24.04 hat 2.39). Eine unbedachte Funktion könnte die Untergrenze
# still anheben.
#
# Umgebung:
#   OBJDUMP  überschreibt das Prüfprogramm (sonst je nach Architektur der Datei:
#            aarch64-linux-gnu-objdump bzw. arm-linux-gnueabihf-objdump, falls
#            vorhanden, sonst objdump)
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-10-28 (Aufgabe 8: glibc-Wächter)
set -euo pipefail

# Hilfetext bei falschem Aufruf
if [ "$#" -lt 2 ]; then
	echo "Aufruf: $0 <max-glibc-version> <datei>..." >&2
	exit 2
fi

max="${1}"
shift

# Nur Zahlen und Punkte erlaubt (die Version wird später in Mustern verwendet)
case "${max}" in
	''|*[!0-9.]*)
		echo "Ungültige Version: ${max}" >&2
		exit 2
		;;
esac

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

failed=0
for file in "$@"; do
	if [ ! -r "${file}" ]; then
		echo "FEHLER: Datei nicht lesbar: ${file}" >&2
		exit 2
	fi
	tool="$(pick_objdump "${file}")"
	if ! symbols="$("${tool}" -T -- "${file}")"; then
		echo "FEHLER: ${tool} -T scheiterte für ${file}" >&2
		exit 2
	fi
	# Alle verlangten glibc-Versionen (ohne GLIBC_PRIVATE), eindeutig
	versions="$(printf '%s\n' "${symbols}" | grep -o 'GLIBC_[0-9][0-9.]*' | sort -u || true)"
	too_new=""
	for version in ${versions}; do
		number="${version#GLIBC_}"
		# Neuer als max: größte der beiden Versionen ist nicht max selbst
		highest="$(printf '%s\n%s\n' "${number}" "${max}" | sort -V | tail -n 1)"
		if [ "${highest}" != "${max}" ]; then
			too_new="${too_new} ${version}"
		fi
	done
	if [ -n "${too_new}" ]; then
		echo "FEHLER: ${file} verlangt glibc neuer als ${max}:" >&2
		for version in ${too_new}; do
			printf '%s\n' "${symbols}" | grep -F "(${version})" | sed 's/^/\t/' >&2 || true
		done
		failed=1
	else
		top="$(printf '%s\n' "${versions}" | sed 's/^GLIBC_//' | sort -V | tail -n 1)"
		echo "OK: ${file} (höchste glibc-Version: ${top:-keine}, erlaubt bis ${max})"
	fi
done

exit "${failed}"
