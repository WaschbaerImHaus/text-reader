// Plattformspezifische Implementierung für Linux (GTK + CGo).
//
// Enthält nativen Vollbild-Toggle, nativen Datei-Öffnen-Dialog und
// Fenster-Icon-Setzung via GTK-Funktionen. Der Datei-Dialog und die
// Icon-Setzung müssen auf dem GTK-Hauptthread ausgeführt werden
// (GTK ist nicht thread-sicher).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-28

//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0

#include <gtk/gtk.h>
#include <string.h>

// toggleWindowFullscreen wechselt den Vollbild-Modus des GTK-Fensters.
//
// @param window       GtkWindow-Zeiger.
// @param isFullscreen 1 = Vollbild verlassen, 0 = Vollbild einschalten.
void toggleWindowFullscreen(void* window, int isFullscreen) {
    GtkWindow* win = GTK_WINDOW(window);
    if (isFullscreen) {
        gtk_window_unfullscreen(win);
    } else {
        gtk_window_fullscreen(win);
    }
}

// setWindowIconFromPNG setzt das GTK-Fenster-Icon aus PNG-Rohdaten im Speicher.
//
// Lädt den PNG-Buffer als GdkPixbuf via InputStream und übergibt ihn an
// gtk_window_set_icon(). Schlägt die Konvertierung fehl, wird das Icon
// stillschweigend ignoriert (kein fataler Fehler).
//
// @param window  GtkWindow-Zeiger.
// @param data    PNG-Rohdaten.
// @param length  Länge der PNG-Rohdaten in Bytes.
void setWindowIconFromPNG(void* window, const guint8* data, gsize length) {
    GtkWindow* win = GTK_WINDOW(window);
    GInputStream* stream = g_memory_input_stream_new_from_data(data, (gssize)length, NULL);
    GError* err = NULL;
    GdkPixbuf* pixbuf = gdk_pixbuf_new_from_stream(stream, NULL, &err);
    g_object_unref(stream);
    if (pixbuf == NULL) {
        if (err) g_error_free(err);
        return;
    }
    gtk_window_set_icon(win, pixbuf);
    g_object_unref(pixbuf);
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
    const char *patterns[] = {"*.md","*.markdown","*.txt","*.epub","*.fb2","*.html","*.htm","*.tex"};
    for (int i = 0; i < 8; i++) {
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
	"os/exec"
	"strings"
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

// setAppIcon setzt das Fenster-Icon aus der eingebetteten PNG-Datei.
//
// Muss auf dem GTK-Hauptthread ausgeführt werden (via w.Dispatch).
//
// @param w Die WebView-Instanz.
func setAppIcon(w webview.WebView) {
	ptr := w.Window()
	if ptr == nil {
		return
	}
	data := appIconPNG
	if len(data) == 0 {
		return
	}
	w.Dispatch(func() {
		C.setWindowIconFromPNG(
			unsafe.Pointer(ptr),
			(*C.guint8)(unsafe.Pointer(&data[0])),
			C.gsize(len(data)),
		)
	})
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

// showOpenFileDialogExternal versucht zenity oder kdialog als externen Dateidialog.
//
// Externe Prozesse umgehen das GTK-Threading-Problem komplett, da sie in
// einem eigenen Prozess laufen und keine GTK-Event-Loop-Konflikte erzeugen.
// Rückgabe: (Pfad, true) wenn ein externes Tool gefunden wurde (auch bei Abbruch),
//           ("", false) wenn kein externes Tool verfügbar ist.
//
// @return Ausgewählter Dateipfad (leer bei Abbruch) und ob ein Tool gefunden wurde.
func showOpenFileDialogExternal() (string, bool) {
	patterns := "*.md *.markdown *.txt *.epub *.fb2 *.html *.htm *.tex"

	// Versuch 1: zenity (GNOME, Cinnamon, XFCE, Mint)
	if _, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command("zenity",
			"--file-selection",
			"--title=Datei öffnen – MD Reader",
			"--file-filter=Unterstützte Dateien|"+patterns,
			"--file-filter=Alle Dateien|*",
		)
		// Bei Abbruch liefert zenity Exit-Code 1; Output ist dann leer
		out, _ := cmd.Output()
		return strings.TrimSpace(string(out)), true
	}

	// Versuch 2: kdialog (KDE/Plasma)
	if _, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.Command("kdialog",
			"--getopenfilename", ".",
			patterns,
			"--title", "Datei öffnen – MD Reader",
		)
		out, _ := cmd.Output()
		return strings.TrimSpace(string(out)), true
	}

	// Kein externes Tool gefunden
	return "", false
}

// openFilePickerBlocking öffnet den Datei-Dialog und blockiert bis zur Auswahl.
//
// Bevorzugt externe Tools (zenity/kdialog), die keinen GTK-Hauptthread benötigen
// und deshalb keine Deadlocks erzeugen können. Nur wenn kein externes Tool
// verfügbar ist, wird der GTK-eigene Dialog auf dem Hauptthread ausgeführt.
//
// @param w Die WebView-Instanz.
// @return Gewählter Dateipfad oder leer bei Abbruch.
func openFilePickerBlocking(w webview.WebView) string {
	// Externe Tools bevorzugen – kein GTK-Threading-Problem
	if path, ok := showOpenFileDialogExternal(); ok {
		return path
	}

	// Fallback: GTK-Dialog auf dem GTK-Hauptthread (kann auf manchen Systemen hängen)
	ch := make(chan string, 1)
	w.Dispatch(func() {
		ch <- showOpenFileDialog(w.Window())
	})
	return <-ch
}
