#!/usr/bin/env bash
# MD Reader – Docker Build-Skript
#
# Baut alle Targets via Docker-Container (Ubuntu 24.04, glibc 2.39).
# Erzeugte Binaries laufen auf Ubuntu 24.04+, Linux Mint 22+, Debian 12+.
#
# Verwendung:
#   ./docker/build.sh            # vollständiger Build (linux + windows + arm + installer)
#   ./docker/build.sh linux      # nur Linux x86_64
#   ./docker/build.sh windows    # nur Windows x86_64
#   ./docker/build.sh release    # Build + GitHub Release
#
# Autor: Kurt Ingwer
# Letzte Änderung: 2026-03-15

set -euo pipefail

# ── Konfiguration ─────────────────────────────────────────────────────────────
IMAGE_NAME="md-reader-builder"
IMAGE_TAG="ubuntu24"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
MAKE_TARGET="${1:-all}"

# ── Docker prüfen ─────────────────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
    echo "✗ Docker ist nicht installiert."
    echo "  Installation: https://docs.docker.com/engine/install/"
    exit 1
fi

echo "──────────────────────────────────────────────────────"
echo "  MD Reader Docker Build"
echo "  Basis-Image : Ubuntu 24.04 (glibc 2.39)"
echo "  Ziel        : Ubuntu 24.04+, Linux Mint 22+, Debian 12+"
echo "  Make-Target : ${MAKE_TARGET}"
echo "──────────────────────────────────────────────────────"

# ── Docker-Image bauen (nur wenn veraltet oder fehlend) ──────────────────────
DOCKERFILE="${SCRIPT_DIR}/Dockerfile"

# Prüfen ob Image existiert und neuer als Dockerfile ist
REBUILD=false
if ! docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" &>/dev/null; then
    echo "→ Image nicht gefunden – wird gebaut..."
    REBUILD=true
elif [ "${DOCKERFILE}" -nt "$(docker inspect --format='{{.Metadata.LastTagTime}}' "${IMAGE_NAME}:${IMAGE_TAG}" 2>/dev/null || echo 0)" ]; then
    echo "→ Dockerfile wurde geändert – Image wird neu gebaut..."
    REBUILD=true
else
    echo "→ Image bereits vorhanden: ${IMAGE_NAME}:${IMAGE_TAG}"
fi

if [ "${REBUILD}" = true ] || [ "${FORCE_REBUILD:-0}" = "1" ]; then
    echo "→ Baue Docker-Image..."
    docker build \
        --tag "${IMAGE_NAME}:${IMAGE_TAG}" \
        --file "${DOCKERFILE}" \
        "${SCRIPT_DIR}"
    echo "✓ Docker-Image gebaut: ${IMAGE_NAME}:${IMAGE_TAG}"
fi

# ── Go-Modul-Cache Volume anlegen (Wiederverwendung zwischen Builds) ──────────
docker volume create md-reader-gomod-cache &>/dev/null || true

# ── Build im Container ausführen ─────────────────────────────────────────────
echo "→ Starte Build-Container..."
docker run --rm \
    --volume "${PROJECT_DIR}:/project" \
    --volume "md-reader-gomod-cache:/go/pkg/mod" \
    --env GH_TOKEN="${GH_TOKEN:-}" \
    --workdir /project \
    "${IMAGE_NAME}:${IMAGE_TAG}" \
    make "${MAKE_TARGET}"

echo ""
echo "✓ Build abgeschlossen. Binaries in: ${PROJECT_DIR}/build/"
ls -lh "${PROJECT_DIR}/build/" 2>/dev/null || true
