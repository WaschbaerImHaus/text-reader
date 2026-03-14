# MD Reader

A simple, fast viewer for Markdown, EPUB, FB2, TXT and HTML with GitHub-like rendering.

## Features

- **Drag & Drop**: Drop files directly onto the window
- **Multi-Format**: Markdown (.md), EPUB (.epub), FictionBook (.fb2), Plaintext (.txt), HTML (.html, .htm), LaTeX (.tex)
- **GitHub Rendering**: Identical display to GitHub (tables, code highlighting, GFM)
- **Syntax Highlighting**: Code blocks with language detection (GitHub style)
- **LaTeX / Math**: Offline KaTeX rendering with theorem styles and custom macros
- **Zoom**: Double/halve font size (25% to 400%)
- **Layouts**: Portrait mode (750px width) or landscape mode (full width)
- **Fullscreen**: Native fullscreen mode (F11 or button)
- **Dark Mode**: GitHub Dark theme (☽ → ▓)
- **Retro Mode**: CRT phosphor green (▓ → ☀)
- **TOC Sidebar**: Automatic table of contents (☰), shown only when headings are present
- **Search**: Ctrl+F opens search bar with match navigation and highlighting
- **Persistent Settings**: Zoom, theme and layout are preserved across sessions
- **Last File**: Last opened file is automatically reloaded on startup
- **Scroll Position**: Scroll position is saved and restored on next launch
- **Local Images**: Relative image paths are embedded as Base64 Data URIs

## Controls

### Toolbar Buttons

| Button | Function |
|--------|----------|
| `☰`    | Toggle Table of Contents (TOC) |
| `▯` / `▭` | Switch portrait / landscape layout |
| `⊖`   | Halve font size (min. 25%) |
| `100%` | Current zoom level |
| `⊕`   | Double font size (max. 400%) |
| `☽` / `▓` / `☀` | Switch theme (Light → Dark → Retro) |
| `☐`   | Toggle fullscreen |
| `✕`   | Close window |

### Keyboard Shortcuts

| Key | Function |
|-----|----------|
| `F11`         | Toggle fullscreen |
| `Ctrl+F`      | Open/close search |
| `Ctrl++`      | Zoom in |
| `Ctrl+-`      | Zoom out |
| `Ctrl+0`      | Reset zoom |
| `Ctrl+Scroll` | Zoom with mouse wheel |
| `↑↓`          | Scroll |
| `PageUp/Down` | Page scroll |
| `Home/End`    | Jump to top/bottom |

## Installation

### Windows

Download `md-reader-setup.exe` from the [Releases](https://github.com/WaschbaerImHaus/text-reader/releases) page and run the installer.

- Start menu entry is created automatically
- Optional: desktop shortcut during installation
- Existing installations are automatically replaced

**Requirements:**
- Windows 10 or newer (WebView2 is pre-installed)
- Older systems: [Install WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)

### Linux

Download the appropriate binary for your architecture from the [Releases](https://github.com/WaschbaerImHaus/text-reader/releases) page:

| File | Architecture |
|------|-------------|
| `md-reader` | Linux x86_64 |
| `md-reader-linux-arm64` | Linux ARM64 |
| `md-reader-linux-armhf` | Linux ARMhf (32-bit) |

```bash
chmod +x md-reader
./md-reader [optional-file.md]
```

**Requirements:**
- WebKitGTK 4.1 (`libwebkit2gtk-4.1-0`)
- GTK 3 (`libgtk-3-0`)

## Usage

1. Start the program
2. Drag & drop a file onto the window
3. **or**: Start the program with a file path as argument

```bash
./md-reader /path/to/file.md
./md-reader /path/to/book.epub
./md-reader /path/to/novel.fb2
./md-reader /path/to/notes.txt
./md-reader /path/to/document.tex
```

The last opened file is automatically reopened on next start (including scroll position).

## Supported Formats

### Markdown (.md, .markdown)

Supported elements:
- Headings (H1–H6)
- Bold, italic, strikethrough
- Lists (ordered/unordered)
- Task lists `- [x]`
- Code (inline and blocks with syntax highlighting)
- Blockquotes
- Tables (GFM)
- Links and images
- Horizontal rules
- Autolinks

### EPUB (.epub)
EPUB 2 and 3 – chapters are read in spine order. Metadata title is used. Images are embedded.

### FictionBook 2 (.fb2)
XML-based format. Supports: chapters, paragraphs, italic/bold/strikethrough, poems, epigraphs, quotes.

### Plaintext (.txt)
Monospace display with preserved whitespace and line breaks.

### HTML (.html, .htm)
Body content is extracted. style/script tags are removed to avoid conflicts.

### LaTeX (.tex)
KaTeX offline rendering with support for theorem environments (theorem, lemma, proof, definition, corollary, remark, example) and common custom macros (`\R`, `\N`, `\Z`, `\Q`, `\C`, `\norm`, `\abs`, `\inner`, `\label`).

## Build from Source

```bash
# Build everything (Linux + Windows)
make all

# Linux only
make linux

# Windows only (requires mingw-w64)
make windows

# Run tests
make test
```

## Technology

- **Go 1.26** – Programming language
- **webview_go** – Cross-platform WebView integration
- **goldmark** – Markdown parser with GFM support
- **goldmark-highlighting** – Syntax highlighting (GitHub style)
- **KaTeX** – Offline LaTeX math rendering
- **WebKitGTK 4.1** (Linux) / **Edge WebView2** (Windows) – Browser engine
