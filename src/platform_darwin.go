//go:build darwin

// Plattformcode für macOS (Cocoa, WKWebView).
//
// Enthält: Vollbild (NSWindow toggleFullScreen:), Dock-Icon,
// nativen Datei-Dialog (NSOpenPanel) und die Menüleiste.
// Die Menüleiste ist unter macOS Pflicht: Ohne sie erreichen ⌘C, ⌘V, ⌘A
// und ⌘Q den WKWebView nicht, weil Cocoa Tastenkürzel über das Menü verteilt.
// Drag & Drop braucht keinen nativen Handler: WKWebView liefert die Dateien
// im JS-Drop-Event (FileReader-Weg in scripts.js).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-28 (setupPlatformIdentity, ensureDisplayAvailable als No-Op)
package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#import <Cocoa/Cocoa.h>

// isWindowFullscreen liefert 1, wenn das Fenster im Vollbild ist.
static int isWindowFullscreen(void *window) {
	NSWindow *win = (NSWindow *)window;
	return ([win styleMask] & NSWindowStyleMaskFullScreen) != 0 ? 1 : 0;
}

// toggleWindowFullscreen schaltet das native macOS-Vollbild um.
static void toggleWindowFullscreen(void *window) {
	NSWindow *win = (NSWindow *)window;
	[win setCollectionBehavior:[win collectionBehavior] | NSWindowCollectionBehaviorFullScreenPrimary];
	[win toggleFullScreen:nil];
}

// setDockIconFromPNG setzt das Dock-Icon aus PNG-Rohdaten.
static void setDockIconFromPNG(const void *data, int length) {
	@autoreleasepool {
		NSData *pngData = [NSData dataWithBytes:data length:(NSUInteger)length];
		NSImage *image = [[NSImage alloc] initWithData:pngData];
		if (image != nil) {
			[NSApp setApplicationIconImage:image];
			[image release];
		}
	}
}

// runOpenPanel zeigt den Datei-Dialog (nur auf dem Hauptthread aufrufen).
// Rückgabe: mit strdup kopierter UTF-8-Pfad oder NULL bei Abbruch.
static char *runOpenPanel(void) {
	char *result = NULL;
	@autoreleasepool {
		NSOpenPanel *panel = [NSOpenPanel openPanel];
		[panel setTitle:@"Datei öffnen – MD Reader"];
		[panel setCanChooseFiles:YES];
		[panel setCanChooseDirectories:NO];
		[panel setAllowsMultipleSelection:NO];
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
		[panel setAllowedFileTypes:@[@"md", @"markdown", @"txt", @"epub", @"fb2",
			@"html", @"htm", @"tex", @"pdf", @"ps"]];
#pragma clang diagnostic pop
		[NSApp activateIgnoringOtherApps:YES];
		if ([panel runModal] == NSModalResponseOK) {
			NSURL *url = [[panel URLs] firstObject];
			const char *path = [[url path] UTF8String];
			if (path != NULL) {
				result = strdup(path);
			}
		}
	}
	return result;
}

// showOpenFileDialog führt den Dialog auf dem Hauptthread aus, auch wenn
// der Aufruf von einem anderen Thread kommt.
static char *showOpenFileDialog(void) {
	if ([NSThread isMainThread]) {
		return runOpenPanel();
	}
	__block char *result = NULL;
	dispatch_sync(dispatch_get_main_queue(), ^{
		result = runOpenPanel();
	});
	return result;
}

// addMenuItem hängt einen Menüeintrag mit Tastenkürzel an.
// Ziel ist nil → Cocoa sucht den Empfänger in der Responder-Kette
// (WKWebView für copy:/paste:, NSApp für terminate: usw.).
static void addMenuItem(NSMenu *menu, NSString *title, SEL action, NSString *key, NSEventModifierFlags modifiers) {
	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:action keyEquivalent:key];
	[item setKeyEquivalentModifierMask:modifiers];
	[menu addItem:item];
	[item release];
}

// addSubmenu legt ein Untermenü in der Menüleiste an und gibt es zurück.
static NSMenu *addSubmenu(NSMenu *bar, NSString *title) {
	NSMenuItem *barItem = [[NSMenuItem alloc] initWithTitle:title action:nil keyEquivalent:@""];
	NSMenu *menu = [[NSMenu alloc] initWithTitle:title];
	[barItem setSubmenu:menu];
	[bar addItem:barItem];
	[barItem release];
	[menu release];
	return menu;
}

// setupMainMenu baut die Standard-Menüleiste (App, Bearbeiten, Fenster).
static void setupMainMenu(void) {
	@autoreleasepool {
		NSEventModifierFlags cmd = NSEventModifierFlagCommand;
		NSMenu *bar = [[NSMenu alloc] initWithTitle:@""];

		// App-Menü (Titel setzt macOS selbst auf den App-Namen)
		NSMenu *appMenu = addSubmenu(bar, @"MD Reader");
		addMenuItem(appMenu, @"Über MD Reader", @selector(orderFrontStandardAboutPanel:), @"", 0);
		[appMenu addItem:[NSMenuItem separatorItem]];
		addMenuItem(appMenu, @"MD Reader ausblenden", @selector(hide:), @"h", cmd);
		addMenuItem(appMenu, @"Andere ausblenden", @selector(hideOtherApplications:), @"h", cmd | NSEventModifierFlagOption);
		addMenuItem(appMenu, @"Alle einblenden", @selector(unhideAllApplications:), @"", 0);
		[appMenu addItem:[NSMenuItem separatorItem]];
		addMenuItem(appMenu, @"MD Reader beenden", @selector(terminate:), @"q", cmd);

		// Bearbeiten: nötig für Kopieren/Einsetzen im WKWebView und im Suchfeld
		NSMenu *editMenu = addSubmenu(bar, @"Bearbeiten");
		addMenuItem(editMenu, @"Widerrufen", @selector(undo:), @"z", cmd);
		addMenuItem(editMenu, @"Wiederholen", @selector(redo:), @"z", cmd | NSEventModifierFlagShift);
		[editMenu addItem:[NSMenuItem separatorItem]];
		addMenuItem(editMenu, @"Ausschneiden", @selector(cut:), @"x", cmd);
		addMenuItem(editMenu, @"Kopieren", @selector(copy:), @"c", cmd);
		addMenuItem(editMenu, @"Einsetzen", @selector(paste:), @"v", cmd);
		addMenuItem(editMenu, @"Alles auswählen", @selector(selectAll:), @"a", cmd);

		// Fenster
		NSMenu *windowMenu = addSubmenu(bar, @"Fenster");
		addMenuItem(windowMenu, @"Im Dock ablegen", @selector(performMiniaturize:), @"m", cmd);
		addMenuItem(windowMenu, @"Zoomen", @selector(performZoom:), @"", 0);
		addMenuItem(windowMenu, @"Fenster schließen", @selector(performClose:), @"w", cmd);

		[NSApp setMainMenu:bar];
		[NSApp setWindowsMenu:windowMenu];
		[bar release];
	}
}
*/
import "C"

import (
	_ "embed"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// appIconPNGLarge ist das 256×256-Icon für das Dock (48 px wäre dort unscharf).
//
//go:embed ui/assets/favicon-256.png
var appIconPNGLarge []byte

// toggleNativeFullscreen schaltet das native macOS-Vollbild um.
//
// Der Zustand wird aus dem Fenster gelesen statt nur mitgezählt, weil der
// Nutzer das Vollbild auch über den grünen Fensterknopf wechseln kann.
// Muss auf dem Hauptthread laufen (Aufruf über w.Dispatch).
//
// @param w Die WebView-Instanz.
func toggleNativeFullscreen(w webview.WebView) {
	ptr := w.Window()
	if ptr == nil {
		return
	}
	wasFullscreen := C.isWindowFullscreen(ptr) != 0
	C.toggleWindowFullscreen(ptr)
	app.isFullscreen = !wasFullscreen
}

// setAppIcon setzt das Dock-Icon aus der eingebetteten PNG-Datei.
//
// Wichtig beim Start ohne App-Bundle (nacktes Binary im Terminal);
// im Bundle liefert Info.plist dasselbe Icon.
//
// @param w Die WebView-Instanz.
func setAppIcon(w webview.WebView) {
	data := appIconPNGLarge
	if len(data) == 0 {
		return
	}
	w.Dispatch(func() {
		C.setDockIconFromPNG(unsafe.Pointer(&data[0]), C.int(len(data)))
	})
}

// openFilePickerBlocking öffnet den macOS-Datei-Dialog und blockiert bis zur Auswahl.
//
// Bindings laufen unter macOS auf dem Hauptthread; der C-Teil weicht bei
// Bedarf selbst per dispatch_sync auf den Hauptthread aus.
//
// @param w Die WebView-Instanz (nicht benötigt, für gleiche Signatur).
// @return Gewählter Dateipfad oder leer bei Abbruch.
func openFilePickerBlocking(w webview.WebView) string {
	cpath := C.showOpenFileDialog()
	if cpath == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cpath))
	return C.GoString(cpath)
}

// setupNativeFileDrop ist unter macOS ein No-Op.
//
// WKWebView liefert die Dateien im JS-Drop-Event; scripts.js liest sie per
// FileReader und übergibt sie an processMarkdown bzw. processBinaryFile.
//
// @param w Die WebView-Instanz (nicht verwendet).
func setupNativeFileDrop(w webview.WebView) {}

// setupPlatformMenu baut die macOS-Menüleiste (App, Bearbeiten, Fenster).
//
// @param w Die WebView-Instanz (für den Hauptthread-Dispatch).
func setupPlatformMenu(w webview.WebView) {
	w.Dispatch(func() {
		C.setupMainMenu()
	})
}

// setupPlatformIdentity ist hier ein No-Op: macOS ordnet die App über das
// Bundle (Info.plist) zu, nicht über einen Programmnamen.
func setupPlatformIdentity() {}

// ensureDisplayAvailable ist hier ein No-Op: Eine .app läuft unter macOS
// immer in einer grafischen Sitzung.
func ensureDisplayAvailable() {}
