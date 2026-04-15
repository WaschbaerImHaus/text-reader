// Plattformspezifische Implementierung für Linux (GTK + CGo).
//
// Enthält nativen Vollbild-Toggle, nativen Datei-Öffnen-Dialog,
// Fenster-Icon-Setzung und nativen GTK-Drag&Drop-Handler.
//
// Drag & Drop: WebKitGTK liefert im JS-Drop-Event weder e.dataTransfer.files
// noch getData('text/uri-list') – beide werden aus Sicherheitsgründen geleert.
// Die Lösung: GTK-Signal 'drag-data-received' direkt auf dem WebKitWebView-
// Widget abfangen, bevor WebKit es verarbeitet. Der C-Callback ruft
// goFileDropCallback() auf (definiert in platform_linux_drop.go via //export).
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-03-28

//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0

#include <gtk/gtk.h>
#include <string.h>

// Vorwärtsdeklaration der Go-Export-Funktion (implementiert in platform_linux_drop.go).
// CGo-Constraint: Dateien mit //export dürfen im Preamble nur Deklarationen haben,
// daher stehen alle C-Definitionen hier und //export ist in der separaten Datei.
extern void goFileDropCallback(const char* path);

// onDragMotion akzeptiert eingehende Datei-Drags und setzt den Kopier-Cursor.
//
// Muss TRUE zurückgeben damit GTK das Widget als gültiges Drop-Ziel behandelt.
// Ohne das bleibt der "verboten"-Cursor und drag-drop wird nie ausgelöst.
//
// @return TRUE = Drag wird akzeptiert.
static gboolean onDragMotion(
    GtkWidget *widget, GdkDragContext *context,
    gint x, gint y, guint time, gpointer userData
) {
    gdk_drag_status(context, GDK_ACTION_COPY, time);
    return TRUE;
}

// onDragDrop fordert die URI-Daten vom Drag-Quell-Prozess an.
//
// Wird nur beim tatsächlichen Loslassen der Maustaste ausgelöst (nicht bei Hover).
// gtk_drag_get_data löst danach onDragDataReceived aus.
//
// @return TRUE = Drop wird verarbeitet.
static gboolean onDragDrop(
    GtkWidget *widget, GdkDragContext *context,
    gint x, gint y, guint time, gpointer userData
) {
    GdkAtom target = gdk_atom_intern("text/uri-list", FALSE);
    gtk_drag_get_data(widget, context, target, time);
    return TRUE;
}

// onDragDataReceived verarbeitet die empfangenen URI-Daten und beendet den Drag.
//
// Wird ausgelöst nachdem onDragDrop gtk_drag_get_data aufgerufen hat.
// gtk_drag_finish MUSS hier aufgerufen werden – genau einmal – sonst bleibt
// der Mauszeiger im Drag-Modus hängen (Bug).
//
// @param selData  Selektionsdaten mit der URI-Liste (text/uri-list).
static void onDragDataReceived(
    GtkWidget *widget, GdkDragContext *context,
    gint x, gint y, GtkSelectionData *selData,
    guint info, guint time, gpointer userData
) {
    gboolean success = FALSE;
    gchar **uris = gtk_selection_data_get_uris(selData);
    if (uris) {
        int i;
        for (i = 0; uris[i] != NULL; i++) {
            GError *err = NULL;
            // file:///home/user/doc.md → /home/user/doc.md
            gchar *path = g_filename_from_uri(uris[i], NULL, &err);
            if (path) {
                goFileDropCallback(path);
                g_free(path);
                success = TRUE;
                break; // Nur erste Datei verarbeiten
            }
            if (err) g_error_free(err);
        }
        g_strfreev(uris);
    }
    // Drag korrekt beenden: success=ob Datei gefunden, del=FALSE (kein Move)
    // Ohne diesen Aufruf bleibt der Mauszeiger im Drag-Modus hängen!
    gtk_drag_finish(context, success, FALSE, time);
}

// setupNativeFileDrop richtet GTK-Drag&Drop direkt auf dem WebKitWebView ein.
//
// Widget-Hierarchie in webview_go (Linux):
//   w.Window() → GtkWindow
//                  └─ GtkScrolledWindow  (gtk_bin_get_child, Ebene 1)
//                       └─ WebKitWebView (gtk_bin_get_child, Ebene 2)
//
// WebKitGTK registriert text/uri-list bereits auf dem WebKitWebView.
// GTK routet Drags zur tiefsten passenden Widget → Handler auf GtkWindow
// oder GtkScrolledWindow wird nie aufgerufen, weil WebKit den Drop zuerst
// beansprucht. Daher muss unser Handler auf dem WebKitWebView sitzen.
//
// flags=0: vollständige manuelle Kontrolle über alle drei Drag-Phasen.
//
// WICHTIG: Muss auf dem GTK-Hauptthread aufgerufen werden!
//
// @param gtkWindow GtkWindow-Zeiger (wie von w.Window() zurückgegeben).
void setupNativeFileDrop(void *gtkWindow) {
    GtkWidget *win     = GTK_WIDGET(gtkWindow);
    GtkWidget *scroller = NULL;
    GtkWidget *webview  = NULL;

    // Ebene 1: GtkWindow → GtkScrolledWindow
    if (GTK_IS_BIN(win)) {
        scroller = gtk_bin_get_child(GTK_BIN(win));
    }
    // Ebene 2: GtkScrolledWindow → WebKitWebView
    if (scroller && GTK_IS_BIN(scroller)) {
        webview = gtk_bin_get_child(GTK_BIN(scroller));
    }

    // Fallback falls Widget-Hierarchie unerwartet ist
    GtkWidget *target = webview ? webview : (scroller ? scroller : win);

    // Unsere text/uri-list-Registrierung ersetzt WebKits eigene Drag-Targets
    // auf diesem Widget. flags=0: alle drei Phasen manuell behandeln.
    static GtkTargetEntry targets[] = {
        { "text/uri-list", 0, 0 }
    };
    gtk_drag_dest_set(target, (GtkDestDefaults)0, targets, 1, GDK_ACTION_COPY);

    g_signal_connect(target, "drag-motion",        G_CALLBACK(onDragMotion),       NULL);
    g_signal_connect(target, "drag-drop",          G_CALLBACK(onDragDrop),         NULL);
    g_signal_connect(target, "drag-data-received", G_CALLBACK(onDragDataReceived), NULL);
}

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
    const char *patterns[] = {"*.md","*.markdown","*.txt","*.epub","*.fb2","*.html","*.htm","*.tex","*.pdf","*.ps"};
    for (int i = 0; i < 10; i++) {
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
	patterns := "*.md *.markdown *.txt *.epub *.fb2 *.html *.htm *.tex *.pdf *.ps"

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

// setupNativeFileDrop aktiviert den nativen GTK-Drag&Drop-Handler.
//
// Muss nach dem Start der GTK-Hauptschleife aufgerufen werden, damit das
// WebKitWebView-Widget bereits existiert. Wird via w.Dispatch() auf dem
// GTK-Hauptthread ausgeführt.
//
// @param w Die WebView-Instanz.
func setupNativeFileDrop(w webview.WebView) {
	ptr := w.Window()
	if ptr == nil {
		return
	}
	w.Dispatch(func() {
		C.setupNativeFileDrop(unsafe.Pointer(ptr))
	})
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
