# Build-Image für die Linux-GUI von MD Reader (cgo + webview_go + WebKitGTK 4.1).
#
# Gebaut wird auf Ubuntu 22.04 (glibc 2.35), damit die Programme auf Ubuntu 22.04+,
# Linux Mint 21+, Debian 12+, Raspberry Pi OS Bookworm und KDE neon laufen.
# Je Zielarchitektur gibt es ein eigenes Image (md-reader-build:22.04-<arch>):
#   docker build --build-arg TARGET_ARCH=amd64|arm64|armhf -t md-reader-build:22.04-<arch> installer/linux/docker
# Das Image selbst ist immer amd64; für arm64/armhf kommen Cross-Compiler und die
# WebKit-Entwicklungspakete der Zielarchitektur (Multiarch) hinzu.
#
# Go wird aus dem offiziellen Tarball installiert, die SHA-256-Summe ist festgeschrieben
# (Quelle: https://go.dev/dl/?mode=json). Bei einem Go-Update beide ARGs ändern.
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-11-01
FROM ubuntu:22.04

ARG TARGET_ARCH=amd64
ARG GO_VERSION=1.27.2
ARG GO_SHA256=ecbadb99091a3f46e31f5f934b068b1864eafa7995211b39eaddf76996045fe5

ENV DEBIAN_FRONTEND=noninteractive

# Paketquellen neu schreiben: amd64 vom Hauptspiegel, ARM-Architekturen von
# ports.ubuntu.com. Jede Zeile trägt [arch=…], sonst fragt apt den falschen
# Spiegel nach der jeweils anderen Architektur und bekommt 404.
RUN set -eux; \
	printf '%s\n' \
		'deb [arch=amd64] http://archive.ubuntu.com/ubuntu jammy main universe' \
		'deb [arch=amd64] http://archive.ubuntu.com/ubuntu jammy-updates main universe' \
		'deb [arch=amd64] http://security.ubuntu.com/ubuntu jammy-security main universe' \
		> /etc/apt/sources.list; \
	if [ "${TARGET_ARCH}" != "amd64" ]; then \
		printf '%s\n' \
			"deb [arch=${TARGET_ARCH}] http://ports.ubuntu.com/ubuntu-ports jammy main universe" \
			"deb [arch=${TARGET_ARCH}] http://ports.ubuntu.com/ubuntu-ports jammy-updates main universe" \
			"deb [arch=${TARGET_ARCH}] http://ports.ubuntu.com/ubuntu-ports jammy-security main universe" \
			>> /etc/apt/sources.list; \
		dpkg --add-architecture "${TARGET_ARCH}"; \
	fi

# Werkzeuge, Cross-Compiler (nur ARM) und WebKit-Entwicklungspakete der Zielarchitektur
RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends build-essential pkg-config ca-certificates curl git; \
	case "${TARGET_ARCH}" in \
		amd64) apt-get install -y --no-install-recommends libwebkit2gtk-4.1-dev ;; \
		arm64) apt-get install -y --no-install-recommends g++-aarch64-linux-gnu libwebkit2gtk-4.1-dev:arm64 ;; \
		armhf) apt-get install -y --no-install-recommends g++-arm-linux-gnueabihf libwebkit2gtk-4.1-dev:armhf ;; \
		*) echo "Unbekannte TARGET_ARCH: ${TARGET_ARCH}" >&2; exit 1 ;; \
	esac; \
	rm -rf /var/lib/apt/lists/*

# Go aus dem offiziellen Tarball, Prüfsumme festgeschrieben
RUN set -eux; \
	curl -fsSL -o /tmp/go.tgz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"; \
	echo "${GO_SHA256}  /tmp/go.tgz" | sha256sum -c -; \
	tar -C /usr/local -xzf /tmp/go.tgz; \
	rm /tmp/go.tgz

ENV PATH=/usr/local/go/bin:${PATH} \
	GOPATH=/go \
	GOTOOLCHAIN=local \
	TARGET_ARCH=${TARGET_ARCH}
