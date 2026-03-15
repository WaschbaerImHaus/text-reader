# MD Reader – Docker Build-Container
#
# Basis: Ubuntu 24.04 (glibc 2.39)
# Erzeugte Binaries laufen auf: Ubuntu 24.04+, Linux Mint 22+, Debian 12+
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-03-15

FROM ubuntu:24.04

# Verhindert interaktive apt-Dialoge während des Build-Prozesses
ENV DEBIAN_FRONTEND=noninteractive

# ── Multiarch für ARM-Bibliotheken einrichten ─────────────────────────────────
RUN dpkg --add-architecture arm64 && \
    dpkg --add-architecture armhf

# ── ARM-Paketquellen hinzufügen (Ubuntu Ports für arm64 und armhf) ────────────
RUN echo "deb [arch=arm64,armhf] http://ports.ubuntu.com/ubuntu-ports noble main restricted universe multiverse" \
        >> /etc/apt/sources.list.d/arm-ports.list && \
    echo "deb [arch=arm64,armhf] http://ports.ubuntu.com/ubuntu-ports noble-updates main restricted universe multiverse" \
        >> /etc/apt/sources.list.d/arm-ports.list && \
    echo "deb [arch=arm64,armhf] http://ports.ubuntu.com/ubuntu-ports noble-security main restricted universe multiverse" \
        >> /etc/apt/sources.list.d/arm-ports.list && \
    # Vorhandene Quellen auf amd64/i386 beschränken, damit kein Konflikt entsteht
    sed -i 's/^deb \(http:\/\/archive\.ubuntu\.com\)/deb [arch=amd64,i386] \1/' /etc/apt/sources.list.d/*.list 2>/dev/null || true && \
    sed -i 's/^deb \(http:\/\/security\.ubuntu\.com\)/deb [arch=amd64,i386] \1/' /etc/apt/sources.list.d/*.list 2>/dev/null || true

# Ubuntu 24.04 verwendet /etc/apt/sources.list.d/ubuntu.sources (DEB822-Format)
# Dort die Architekturen einschränken
RUN if [ -f /etc/apt/sources.list.d/ubuntu.sources ]; then \
        sed -i 's/^Architectures:.*/Architectures: amd64 i386/' /etc/apt/sources.list.d/ubuntu.sources || \
        sed -i '/^Types: deb/a Architectures: amd64 i386' /etc/apt/sources.list.d/ubuntu.sources; \
    fi

# ── Paketlisten aktualisieren ─────────────────────────────────────────────────
RUN apt-get update

# ── Basis-Werkzeuge installieren ──────────────────────────────────────────────
RUN apt-get install -y \
    curl \
    git \
    build-essential \
    pkg-config \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# ── WebKitGTK 4.1 für Linux x86_64 ───────────────────────────────────────────
RUN apt-get update && apt-get install -y \
    libwebkit2gtk-4.1-dev \
    libgtk-3-dev \
    && rm -rf /var/lib/apt/lists/*

# ── ARM64-Bibliotheken für Cross-Compilation ──────────────────────────────────
RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    libwebkit2gtk-4.1-dev:arm64 \
    && rm -rf /var/lib/apt/lists/*

# ── ARMhf-Bibliotheken für Cross-Compilation ─────────────────────────────────
RUN apt-get update && apt-get install -y \
    gcc-arm-linux-gnueabihf \
    g++-arm-linux-gnueabihf \
    libwebkit2gtk-4.1-dev:armhf \
    && rm -rf /var/lib/apt/lists/*

# ── MinGW für Windows Cross-Compilation ──────────────────────────────────────
RUN apt-get update && apt-get install -y \
    gcc-mingw-w64-x86-64 \
    g++-mingw-w64-x86-64 \
    && rm -rf /var/lib/apt/lists/*

# ── NSIS für Windows Installer ───────────────────────────────────────────────
RUN apt-get update && apt-get install -y \
    nsis \
    && rm -rf /var/lib/apt/lists/*

# ── Go 1.26 installieren ──────────────────────────────────────────────────────
RUN curl -fsSL "https://go.dev/dl/go1.26.0.linux-amd64.tar.gz" \
    | tar -C /usr/local -xz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOMODCACHE="/go/pkg/mod"

# Go-Modul-Cache als Volume bereitstellen (spart Download-Zeit bei Wiederholung)
VOLUME ["/go/pkg/mod"]

# ── Arbeitsverzeichnis ────────────────────────────────────────────────────────
WORKDIR /project

# Standard-Befehl: vollständiger Build über das Makefile
CMD ["make", "all"]
