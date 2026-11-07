# MD Reader

A simple, fast viewer for Markdown, EPUB, FB2, TXT and HTML with GitHub-like rendering – for Windows, Linux and macOS.

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
- **Text-Only Mode** (⌨): C64-style view for text documents (Markdown, TXT, FB2, EPUB, LaTeX, HTML) – everything in one pixel monospace font, same size, headings bold; 50 columns and 32 lines with 1.5 line spacing, sized from the shorter window side; images 1-bit dithered, tables drawn with lines, formulas as 1-bit graphics; foreground and background color freely selectable
- **TOC Sidebar**: Automatic table of contents (☰), shown only when headings are present
- **Search**: Ctrl+F opens search bar with match navigation and highlighting
- **Persistent Settings**: Zoom, theme and layout are preserved across sessions
- **Last File**: Last opened file is automatically reloaded on startup
- **Scroll Position**: Scroll position is saved and restored on next launch
- **Local Images**: Relative and absolute local image paths (png, jpg, gif, webp, svg, bmp, ico, avif) are embedded as Base64 Data URIs; network (UNC) paths are not loaded
- **External Links**: Clicking an http/https/mailto link opens it in your default browser or mail client – the viewer itself never navigates away

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
| `⌨`   | Toggle text-only mode (text formats only) |
| two color boxes | Text and background color (visible in text-only mode) |
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
| `Ctrl+O`      | Open file |
| `↑↓`          | Scroll |
| `PageUp/Down` | Page scroll |
| `Home/End`    | Jump to top/bottom |

On macOS use `Cmd` instead of `Ctrl`; `Cmd+C`, `Cmd+A`, `Cmd+Q` work via the menu bar.

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

Download from the [Releases](https://github.com/WaschbaerImHaus/text-reader/releases) page – either a single file or a `.deb` package for your architecture:

| Single file | `.deb` package | Architecture |
|-------------|----------------|--------------|
| `md-reader` | `md-reader_1.0.<build>_amd64.deb` | Linux x86_64 |
| `md-reader-linux-arm64` | `md-reader_1.0.<build>_arm64.deb` | Linux ARM64 |
| `md-reader-linux-armhf` | `md-reader_1.0.<build>_armhf.deb` | Linux ARMhf (32-bit) |

**Single file** – nothing to install, just start it:

```bash
chmod +x md-reader
./md-reader [optional-file.md]
```

The file is a small launcher with the program built in. It checks whether the required
libraries are present before it starts.

- **WebKitGTK missing** (typical on KDE neon): a desktop notification says so. On systems
  with `apt` it offers **Install now**, which runs
  `pkexec apt-get install libwebkit2gtk-4.1-0` after a graphical password prompt;
  MD Reader then starts by itself. On other distributions the message shows the matching
  command. By hand: `sudo apt install libwebkit2gtk-4.1-0` (Debian/Ubuntu/Mint),
  `sudo dnf install webkit2gtk4.1` (Fedora), `sudo pacman -S webkit2gtk-4.1` (Arch),
  `sudo zypper install libwebkit2gtk-4_1-0` (openSUSE).
- **Application menu**: on the first start MD Reader asks once whether it should be added
  to the application menu (**Yes** / **Later** / **Never**). You can do it at any time:

  ```bash
  ./md-reader --install            # add to the menu (use --force to overwrite)
  ./md-reader --uninstall          # remove it again; your settings are kept
  ```

  This works per user and needs no root: program in `~/.local/bin/md-reader`, menu entry
  (category Office) in `~/.local/share/applications/md-reader.desktop`, icons under
  `~/.local/share/icons/hicolor`. MD Reader also appears in "Open with" for md, markdown,
  txt, epub, fb2, tex, html, htm, pdf and ps files – it does not become the default app.
  Starting a newer downloaded file offers to replace the installed one.
  `--` ends the options (`./md-reader -- --uninstall` opens a file named `--uninstall`).

**`.deb` package** – installs system-wide; WebKitGTK and GTK come in as dependencies:

```bash
sudo apt install ./md-reader_1.0.<build>_amd64.deb
```

If both the `.deb` package and the per-user installation exist, the per-user menu entry
(`~/.local/share/applications/md-reader.desktop`) hides the package entry, because both use the
same desktop ID. So run `md-reader --uninstall` (from the downloaded file) before switching to the `.deb`.

**Supported systems:** Ubuntu 22.04 or newer, Linux Mint 21 or newer, Debian 12 or newer,
Raspberry Pi OS Bookworm (Raspberry Pi 2 or newer; armhf = ARMv7), KDE neon (glibc 2.34 or newer). Older systems get a clear
message instead of a loader error.

**Requirements:**
- WebKitGTK 4.1 (`libwebkit2gtk-4.1-0`) and GTK 3 (`libgtk-3-0`) – the `.deb` installs them
- Optional: `zenity` (file dialog), `poppler-utils` (PDF), `ghostscript` (PostScript)

**Good to know:** the right mouse button (context menu) is disabled in the whole window.
External images in Markdown (`![text](https://…)`) are not loaded; their alt text is shown instead.

### macOS

Download `md-reader-macos.zip` from the [Releases](https://github.com/WaschbaerImHaus/text-reader/releases) page (it appears a few minutes after the other files), unzip it and move `MD Reader.app` to *Applications*.

The app is not notarized by Apple. On first start macOS refuses to open it:
open *System Settings → Privacy & Security*, scroll down and click **Open Anyway**.
Alternatively in Terminal:

```bash
xattr -dr com.apple.quarantine "/Applications/MD Reader.app"
```

**Requirements:**
- macOS 13 (Ventura) or newer, Apple Silicon or Intel
- Optional for PDF/PostScript: `brew install poppler ghostscript`

The Linux files `md-reader-linux-arm64` / `-armhf` do **not** run on a Mac – they are Linux programs.

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

# macOS app bundle (only on a Mac with Xcode Command Line Tools)
make macos
```

## Technology

- **Go 1.26** – Programming language
- **webview_go** – Cross-platform WebView integration
- **goldmark** – Markdown parser with GFM support
- **goldmark-highlighting** – Syntax highlighting (GitHub style)
- **KaTeX** – Offline LaTeX math rendering
- **WebKitGTK 4.1** (Linux) / **Edge WebView2** (Windows) / **WKWebView** (macOS) – Browser engine

## Fonts

The text-only mode uses the pixel font **Press Start 2P** by CodeMan38 (Cody "CodeMan38" Boisclair), licensed under the SIL Open Font License 1.1 (`src/ui/assets/fonts/OFL.txt`).
