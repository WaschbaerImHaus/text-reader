// Plattformspezifische Implementierung für Linux (GTK + CGo).
//
// Enthält nativen Vollbild-Toggle und nativen Datei-Öffnen-Dialog
// via GTK-FileChooserDialog. Der Datei-Dialog muss auf dem GTK-Hauptthread
// ausgeführt werden (GTK ist nicht thread-sicher).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-08

//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0

#include <gtk/gtk.h>

// toggleWindowFullscreen wechselt den Vollbild-Modus des GTK-Fensters.
//
// @param window  GtkWindow-Zeiger.
// @param isFullscreen 1 = Vollbild verlassen, 0 = Vollbild einschalten.
void toggleWindowFullscreen(void* window, int isFullscreen) {
    GtkWindow* win = GTK_WINDOW(window);
    if (isFullscreen) {
        gtk_window_unfullscreen(win);
    } else {
        gtk_window_fullscreen(win);
    }
}

// showFileDialog öffnet den nativen GTK-Datei-Öffnen-Dialog.
//
// WICHTIG: Muss auf dem GTK-Hauptthread aufgerufen werden!
//
// @param parentWindow GtkWindow-Zeiger (kann NULL sein).
// @return Ausgewählter Dateipfad (muss mit g_free freigegeben werden), oder NULL.
gchar* showFileDialog(void* parentWindow) {
    GtkWidget *dialog;
    gchar *filename = NULL;

    dialog = gtk_file_chooser_dialog_new(
        "Datei \u00f6ffnen \u2013 MD Reader",
        parentWindow ? GTK_WINDOW(parentWindow) : NULL,
        GTK_FILE_CHOOSER_ACTION_OPEN,
        "Abbrechen", GTK_RESPONSE_CANCEL,
        "\u00d6ffnen",   GTK_RESPONSE_ACCEPT,
        NULL
    );

    // Filter: Unterstützte Dateiformate
    GtkFileFilter *filter = gtk_file_filter_new();
    gtk_file_filter_set_name(filter, "Unterst\u00fctzte Dateien");
    const char *patterns[] = {"*.md","*.markdown","*.txt","*.epub","*.fb2","*.html","*.htm"};
    for (int i = 0; i < 7; i++) {
        gtk_file_filter_add_pattern(filter, patterns[i]);
    }
    gtk_file_chooser_add_filter(GTK_FILE_CHOOSER(dialog), filter);

    // Filter: Alle Dateien
    GtkFileFilter *allFilter = gtk_file_filter_new();
    gtk_file_filter_set_name(allFilter, "Alle Dateien");
    gtk_file_filter_add_pattern(allFilter, "*");
    gtk_file_chooser_add_filter(GTK_FILE_CHOOSER(dialog), allFilter);

    // Dialog öffnen und auf Auswahl warten (gtk_dialog_run hat eigene Event-Loop)
    if (gtk_dialog_run(GTK_DIALOG(dialog)) == GTK_RESPONSE_ACCEPT) {
        filename = gtk_file_chooser_get_filename(GTK_FILE_CHOOSER(dialog));
    }
    gtk_widget_destroy(dialog);

    // Verbleibende GTK-Events verarbeiten
    while (gtk_events_pending()) {
        gtk_main_iteration();
    }
    return filename;
}
*/
import "C"

import (
	"unsafe"

	webview "github.com/webview/webview_go"
)

// toggleNativeFullscreen wechselt den nativen GTK-Vollbild-Modus.
//
// @param w Die WebView-Instanz.
func toggleNativeFullscreen(w webview.WebView) {
	ptr := w.Window()
	if ptr == nil {
		return
	}
	var fullscreenInt C.int
	if app.isFullscreen {
		fullscreenInt = 1
	} else {
		fullscreenInt = 0
	}
	C.toggleWindowFullscreen(unsafe.Pointer(ptr), fullscreenInt)
	app.isFullscreen = !app.isFullscreen
}

// showOpenFileDialog öffnet den nativen GTK-Datei-Dialog.
//
// WICHTIG: Muss auf dem GTK-Hauptthread ausgeführt werden!
//
// @param parentWindow GtkWindow-Zeiger (kann nil sein).
// @return Gewählter Dateipfad oder leer bei Abbruch.
func showOpenFileDialog(parentWindow unsafe.Pointer) string {
	filename := C.showFileDialog(parentWindow)
	if filename == nil {
		return ""
	}
	// g_free gibt den von GTK allokierten Speicher frei
	defer C.g_free(C.gpointer(filename))
	return C.GoString(filename)
}

// openFilePickerBlocking öffnet den Datei-Dialog und blockiert bis zur Auswahl.
//
// Auf Linux muss showOpenFileDialog auf dem GTK-Hauptthread laufen.
// w.Dispatch() stellt das sicher; ein Channel überträgt das Ergebnis zurück.
//
// @param w Die WebView-Instanz.
// @return Gewählter Dateipfad oder leer bei Abbruch.
func openFilePickerBlocking(w webview.WebView) string {
	ch := make(chan string, 1)
	// GTK-Dialog auf dem Hauptthread ausführen
	w.Dispatch(func() {
		ch <- showOpenFileDialog(w.Window())
	})
	return <-ch
}
