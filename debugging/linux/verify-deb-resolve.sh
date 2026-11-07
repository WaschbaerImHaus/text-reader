#!/usr/bin/env bash
# Prüft, dass sich die drei md-reader-.deb-Pakete auf mehreren Distributionen
# auflösen lassen (apt-get install --simulate, es wird nichts installiert).
# Container: ubuntu:22.04, ubuntu:24.04, debian:trixie (Netz nötig, nur lesend).
#
# Aufruf (aus dem Projektroot, build/ enthält die .deb): debugging/linux/verify-deb-resolve.sh [build-nummer]
# Ausgabe je Kombination: "OK" oder "FEHLER" samt apt-Meldung; Exit 1 bei einem Fehler.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-11-07
set -euo pipefail

build_num="${1:-$(cat src/build.txt)}"
root="$(pwd)"
failed=0

# Skript, das IM Container läuft: Quellen je Architektur setzen, dann simulieren
read -r -d '' inner <<'INNER' || true
set -eu
arch="$1"; deb="$2"
. /etc/os-release
export DEBIAN_FRONTEND=noninteractive
rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*
if [ "${ID}" = "debian" ]; then
	cat > /etc/apt/sources.list <<SRC
deb http://deb.debian.org/debian ${VERSION_CODENAME} main
deb http://deb.debian.org/debian ${VERSION_CODENAME}-updates main
deb http://security.debian.org/debian-security ${VERSION_CODENAME}-security main
SRC
else
	cat > /etc/apt/sources.list <<SRC
deb [arch=amd64] http://archive.ubuntu.com/ubuntu ${VERSION_CODENAME} main universe
deb [arch=amd64] http://archive.ubuntu.com/ubuntu ${VERSION_CODENAME}-updates main universe
deb [arch=arm64,armhf] http://ports.ubuntu.com/ubuntu-ports ${VERSION_CODENAME} main universe
deb [arch=arm64,armhf] http://ports.ubuntu.com/ubuntu-ports ${VERSION_CODENAME}-updates main universe
SRC
fi
[ "${arch}" = "amd64" ] || dpkg --add-architecture "${arch}"
apt-get update -qq > /dev/null
apt-get install --simulate -y "${deb}"
INNER

for image in ubuntu:22.04 ubuntu:24.04 debian:trixie; do
	for arch in amd64 arm64 armhf; do
		deb="/b/md-reader_1.0.${build_num}_${arch}.deb"
		echo "== ${image} / ${arch}"
		if docker run --rm -v "${root}/build:/b:ro" "${image}" bash -c "${inner}" _ "${arch}" "${deb}" > /tmp/verify-deb.$$ 2>&1; then
			echo "OK"
		else
			echo "FEHLER"; tail -15 /tmp/verify-deb.$$
			failed=1
		fi
	done
done
rm -f /tmp/verify-deb.$$
exit "${failed}"
