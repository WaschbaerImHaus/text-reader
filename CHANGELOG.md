# Changelog

All notable changes to MD Reader are documented here.

---

## Build 34 – 2026-04-15

### Added
- PDF support: native rendering via WebKit (Linux) and Edge WebView2 (Windows)
- PostScript (.ps) support: converts to PDF via Ghostscript if available, falls back to plain text display
- Windows installer: optional Ghostscript component for PostScript rendering
- File picker and drag & drop now accept .pdf and .ps files

---

## Build 27 – Docker Build-Container (2026-03-15)

- Added `docker/Dockerfile`: Ubuntu 24.04 base with Go 1.26, WebKitGTK 4.1, MinGW, ARM cross-compilers, NSIS
- Added `docker/build.sh`: build script with automatic image caching
- New Makefile targets: `docker-build`, `docker-build-release`, `docker-rebuild`
- Binaries now target Ubuntu 24.04+, Linux Mint 22+, Debian 12+ (glibc ≥ 2.39)

---

## Build 26 – Fix CXXABI_1.3.15 on Linux Mint (2026-03-15)

- Static libstdc++ linking to resolve `CXXABI_1.3.15` symbol errors on older distributions
- Ensures compatibility with Linux Mint 21 and similar glibc 2.35 environments

---

## Build 25 – English README & GitHub Release Setup (2026-03-15)

- README.md rewritten in English as user-facing manual
- GitHub release workflow configured
- Updated installer and start menu entries

---

## Build 24 – Fix `\label{eq:…}` displayed as plain text in KaTeX (2026-03-11)

- `\label{...}` macros inside math environments are now stripped before KaTeX rendering
- Prevents literal `\label{eq:foo}` from appearing in rendered equations

---

## Build 23 – LaTeX Rendering Improvements (2026-03-11)

- CSS added for all LaTeX elements (`.latex-document`, `.latex-title-block`, theorem environments)
- `\newtheorem` environments rendered with display names (Theorem, Lemma, Proof …)
- `proof` environment: "Proof. ... ∎" format
- `\newcommand` and `\DeclareMathOperator` extracted from preamble and passed to KaTeX as `window.__latexMacros`
- New tests: theorem/proof/macro parsing

---

## Build 22 – LaTeX .tex Support & EPUB Image Embedding (2026-03-11)

- New format: `.tex` files rendered via new `renderer/latex.go`
  - Sections, formatting, lists, verbatim, math protection, footnotes, links
  - Theorem environments, proof, abstract, figure, tabular
- EPUB images now embedded as Base64 Data-URIs (previously broken due to ZIP path resolution)
- Platform file dialogs updated to include `*.tex` filter
- 17 new LaTeX tests, 3 new EPUB image tests

---

## Build 20 – Windows NSIS Installer (2026-03-08)

- `installer/md-reader.nsi`: LZMA-compressed installer with MUI2 wizard
- Start menu entry (required) and optional desktop icon
- Uninstaller removes all files, shortcuts and registry entries
- Output: `build/md-reader-setup.exe`

---

## Build 19 – KaTeX Offline Math Rendering (2026-03-08)

- KaTeX v0.16.37 fully embedded (no CDN dependency)
- All fonts converted to Base64 Data-URIs at runtime
- Supports: `$...$`, `$$...$$`, `\(...\)`, `\[...\]`, equation/align environments
- New `src/ui/katex.go` with `go:embed` for all KaTeX assets

---

## Build 18 – Bug Fixes (2026-03-08)

- **Bug #006**: Zoom display after restart showed wrong default font size
- **Bug #007**: Console window appeared on Windows startup (missing `-H windowsgui`)
- **Bug #008**: App icon missing in Windows Explorer (ICO regenerated with 16/32/48/256px)

---

## Build 14 – Scroll History & Portrait Zoom Scaling (2026-03-08)

- **Scroll History**: Scroll position saved per file using FNV-64a hash (max 200 entries)
- **Portrait mode**: Width now scales with zoom level (`750px × factor`, capped at window width − 32px)

---

## Build 11 – Security, Mermaid.js, ARM Builds (2026-03-08)

- CSP Meta-Tag added (`script-src`, `style-src`, `font-src`, `img-src`)
- Path validation in `persistLastFile` binding (RISK-004 fixed)
- Mermaid.js integration for flowcharts/sequence diagrams (CDN with code-block fallback)
- Linux ARM64 and ARMhf cross-compilation added
- `bindings.go` extracted from `main.go`; `go:embed` for all CSS/JS/HTML assets
- Binaries stripped (`-ldflags="-s -w"`)

---

## Build 9 – Drag & Drop & Scroll Position Fixes (2026-03-08)

- **Bug #003**: Last file not saved on drag & drop (fixed via `text/uri-list` from DataTransfer)
- **Bug #004**: Scroll position not persisted/restored (debounced save + restore on startup)

---

## Build 8 – Refactoring: ui/template.go Split (2026-03-08)

- `ui/template.go` (992 lines) split into 4 semantic files:
  - `ui/template.go` – Go code only
  - `ui/styles.go` – CSS constants
  - `ui/html_body.go` – HTML structure
  - `ui/scripts.go` – JavaScript logic
- 15 new UI tests added

---

## Build 7 – Multi-Format Support, TOC, Search (2026-03-07)

- **New formats**: EPUB 2/3, FictionBook 2 (.fb2), Plaintext (.txt)
- TOC sidebar: auto-built from headings, toggle with ☰
- Search: Ctrl+F, TreeWalker-based with `<mark>` highlighting and match navigation
- Zoom via Ctrl+Scroll and Ctrl+/−/0
- Last opened file restored on startup
- Settings persistent (`state.json`): zoom, theme, layout

---

## Build 5 – Dark Mode & Retro Mode (2026-03-05)

- GitHub Dark theme: `#0d1117` background, `#e6edf3` text
- Retro CRT mode: phosphor green `#39ff14`, scanlines effect
- Cyclic button ☾ → ▓ → ☀

---

## Build 4 – Windows: No Console Window (2026-03-05)

- Added `-ldflags="-H windowsgui"` to Windows build to suppress CMD window

---

## Build 3 – Keyboard Scroll (2026-03-05)

- Arrow keys: 60px vertical / 80px horizontal
- Page Up/Down: 90% of viewport height
- Home/End: top/bottom of document

---

## Build 1 – Initial Release (2026-03-05)

- Markdown viewer with GitHub-like rendering (goldmark + GFM)
- Syntax highlighting (chroma, GitHub style)
- Drag & Drop for `.md` files
- Portrait / Landscape layout toggle
- Zoom (25%–400%) with display in toolbar
- Native fullscreen (F11 / button)
- Close button (✕)
- Linux (WebKitGTK 4.1) and Windows (Edge WebView2) support
- Cross-compilation: Linux x86_64 and Windows x86_64
