// Plattformspezifische Implementierung für Linux (GTK + CGo).
//
// Enthält nativen Vollbild-Toggle, nativen Datei-Öffnen-Dialog,
// Fenster-Icon-Setzung, nativen GTK-Drag&Drop-Handler, die Programm-Identität
// (Programmname, Fensterklasse, Prozessname) und die Prüfung auf eine
// erreichbare grafische Oberfläche vor dem Anlegen des Fensters.
//
// Drag & Drop: WebKitGTK liefert im JS-Drop-Event weder e.dataTransfer.files
// noch getData('text/uri-list') – beide werden aus Sicherheitsgründen geleert.
// Die Lösung: GTK-Signal 'drag-data-received' direkt auf dem WebKitWebView-
// Widget abfangen, bevor WebKit es verarbeitet. Der C-Callback ruft
// goFileDropCallback() auf (definiert in platform_linux_drop.go via //export).
//
// Navigationsschutz (F1, Build 47): Ein decide-policy-Handler am
// WebKitWebView lässt nur die eigene Seite und die eigene Temp-Ansichtsdatei
// zu (ui.AllowNavigation). So kann auch das Kontextmenü („Verweis öffnen“)
// das App-Fenster nicht zu einer fremden Seite mit Go-Bindings navigieren.
// Zusätzlich unterdrückt das Signal context-menu das Kontextmenü ganz.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (setupNavigationGuard: decide-policy-Erlaubnisliste und kein Kontextmenü, F1)

//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.0

#include <gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <string.h>
#include <sys/prctl.h>

// setProgramName setzt Programmname und Prozessname fest auf "md-reader".
//
// Der Starter führt die GUI aus einem memfd als /proc/self/fd/N aus. Ohne
// diese Funktion leitet GTK den Namen aus argv[0] bzw. dem Dateinamen ab
// (z. B. "md-reader-test" oder "4") – dann passen Fensterklasse und
// Desktop-Datei (StartupWMClass=md-reader, Icon=md-reader) nicht zusammen,
// und die Taskleiste zeigt kein Icon bzw. gruppiert falsch.
//
// - g_set_prgname: Programmname → WM_CLASS-Name (X11), app_id (Wayland).
// - prctl(PR_SET_NAME): Prozessname (comm) für ps, top, killall, pgrep -x.
//   Gilt für den aufrufenden Thread; das ist hier der Hauptthread, weil
//   webview_go den main-Goroutine per init() an ihn bindet. Höchstens
//   15 Zeichen, "md-reader" passt.
//
// Muss vor gtk_init_check aufgerufen werden, weil GDK den Programmnamen
// beim Initialisieren übernimmt. Die WM_CLASS-Klasse setzt initDisplay.
static void setProgramName(void) {
	g_set_prgname("md-reader");
	prctl(PR_SET_NAME, "md-reader", 0, 0, 0);
}

// initDisplay initialisiert GTK, meldet, ob eine grafische Oberfläche
// (X11 oder Wayland) erreichbar ist, und setzt danach die WM_CLASS-Klasse.
//
// webview_go ruft intern ebenfalls gtk_init_check auf, liefert bei FALSE aber
// ein Go-Objekt mit NULL-Zeiger statt nil – der erste Aufruf (SetTitle)
// stürzt dann mit SIGSEGV ab. Deshalb wird hier vorher geprüft. Ein zweiter
// gtk_init_check-Aufruf nach Erfolg ist in GTK erlaubt und ändert nichts.
//
// gdk_set_program_class muss NACH gtk_init_check stehen: GDK überschreibt
// die Klasse beim Initialisieren mit dem Programmnamen und schreibt dabei
// den ersten Buchstaben groß ("Md-reader", gemessen unter Xvfb).
//
// @return TRUE, wenn GTK initialisiert werden konnte.
static gboolean initDisplay(void) {
	if (!gtk_init_check(NULL, NULL)) {
		return FALSE;
	}
	gdk_set_program_class("md-reader");
	return TRUE;
}

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

// Vorwärtsdeklaration der Go-Export-Funktion für den Navigationsschutz
// (implementiert in platform_linux_nav.go). Rückgabe 1 = erlauben.
extern int goDecideNavigation(const char* uri, int userAction);

// onDecidePolicy entscheidet über jede Navigation des WebKitWebView (F1).
//
// - NAVIGATION_ACTION: Go (ui.AllowNavigation) entscheidet; erlaubt sind nur
//   die eigene Seite (about:blank) und die eigene Temp-Ansichtsdatei.
//   Abgelehnte http/https/mailto-Ziele öffnet Go bei Nutzeraktion extern
//   (Geste oder Linkklick), nie im App-Fenster.
// - NEW_WINDOW_ACTION: immer ablehnen (kein zweites Fenster mit Bindings).
// - RESPONSE: WebKit-Standard (Antwort einer bereits erlaubten Navigation).
//
// @return TRUE = Entscheidung getroffen, FALSE = WebKit-Standard verwenden.
static gboolean onDecidePolicy(
    WebKitWebView *webView, WebKitPolicyDecision *decision,
    WebKitPolicyDecisionType type, gpointer userData
) {
    if (type == WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION) {
        webkit_policy_decision_ignore(decision);
        return TRUE;
    }
    if (type != WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION) {
        return FALSE;
    }
    WebKitNavigationAction *action = webkit_navigation_policy_decision_get_navigation_action(
        WEBKIT_NAVIGATION_POLICY_DECISION(decision));
    const char *uri = NULL;
    int userAction = 0;
    if (action) {
        WebKitURIRequest *request = webkit_navigation_action_get_request(action);
        if (request) uri = webkit_uri_request_get_uri(request);
        // Nutzeraktion: echte Geste oder Linkklick. „Verweis öffnen“ aus dem
        // Kontextmenü meldet WebKitGTK als LINK_CLICKED ohne Geste (gemessen).
        // Skripte könnten Links nicht anklicken: CSP mit Nonce, Bereiniger.
        userAction = (webkit_navigation_action_is_user_gesture(action) ||
            webkit_navigation_action_get_navigation_type(action) == WEBKIT_NAVIGATION_TYPE_LINK_CLICKED) ? 1 : 0;
    }
    // Ohne Adresse: ablehnen (fail closed)
    if (uri && goDecideNavigation(uri, userAction)) {
        webkit_policy_decision_use(decision);
    } else {
        webkit_policy_decision_ignore(decision);
    }
    return TRUE;
}

// onContextMenu unterdrückt das WebKit-Kontextmenü vollständig (F1,
// Nutzerentscheidung 2026-11-01): Es enthält „Verweis öffnen“, „In neuem
// Fenster öffnen“ und „Verweisziel herunterladen“. Wirkt auch, bevor das
// Seitenskript geladen ist. Tastenkürzel (Strg+C) sind nicht betroffen.
//
// @return TRUE = Menü nicht anzeigen.
static gboolean onContextMenu(
    WebKitWebView *webView, WebKitContextMenu *menu,
    GdkEvent *event, WebKitHitTestResult *hit, gpointer userData
) {
    return TRUE;
}

// setupNavigationGuard verbindet onDecidePolicy (Navigations-Erlaubnisliste)
// und onContextMenu (kein Kontextmenü) mit dem WebKitWebView.
//
// Die verwendete webview_go-Version legt das WebKitWebView direkt in das
// GtkWindow (gtk_container_add, webview.h). Ältere Versionen schoben ein
// GtkScrolledWindow dazwischen – deshalb wird bis zu zwei Ebenen tief
// gesucht (gtk_bin_get_child) und das Ziel per WEBKIT_IS_WEB_VIEW geprüft.
// Wird kein WebKitWebView gefunden, bleibt nur der JS-Schutz – Rückgabe 0,
// Go protokolliert das.
//
// WICHTIG: Muss auf dem GTK-Hauptthread aufgerufen werden!
//
// @param gtkWindow GtkWindow-Zeiger (wie von w.Window() zurückgegeben).
// @return 1 = Handler verbunden, 0 = WebKitWebView nicht gefunden.
int setupNavigationGuard(void *gtkWindow) {
    GtkWidget *child = GTK_WIDGET(gtkWindow);
    int depth;
    for (depth = 0; depth < 2 && child && GTK_IS_BIN(child) && !WEBKIT_IS_WEB_VIEW(child); depth++) {
        child = gtk_bin_get_child(GTK_BIN(child));
    }
    if (!child || !WEBKIT_IS_WEB_VIEW(child)) {
        return 0;
    }
    g_signal_connect(child, "decide-policy", G_CALLBACK(onDecidePolicy), NULL);
    g_signal_connect(child, "context-menu", G_CALLBACK(onContextMenu), NULL);
    return 1;
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
	"fmt"
	"log"
	"os"
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
// ("", false), wenn kein externes Tool verfügbar ist.
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

// setupNavigationGuard aktiviert den nativen Navigationsschutz (F1):
// Das App-Fenster darf nur die eigene Seite und die eigene Temp-Ansichtsdatei
// laden (ui.AllowNavigation), neue Fenster werden nie geöffnet.
//
// Wird direkt (ohne Dispatch) vor dem ersten SetHtml aufgerufen: main läuft
// auf dem GTK-Hauptthread (runtime.LockOSThread), und der Schutz muss stehen,
// bevor irgendein Inhalt geladen wird.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-11-01 (neu, F1)
//
// @param w Die WebView-Instanz.
func setupNavigationGuard(w webview.WebView) {
	ptr := w.Window()
	if ptr == nil || C.setupNavigationGuard(unsafe.Pointer(ptr)) == 0 {
		// Kein WebKitWebView gefunden: Es bleibt der JS-Schutz (scripts.js)
		log.Printf("Navigationsschutz nicht aktiv: WebKitWebView nicht gefunden")
	}
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

// setupPlatformMenu ist hier ein No-Op: Die App hat kein Menü, nur die Toolbar.
// Nur macOS braucht eine Menüleiste (Tastenkürzel ⌘C/⌘V/⌘Q, siehe platform_darwin.go).
//
// @param w Die WebView-Instanz (nicht verwendet).
func setupPlatformMenu(w webview.WebView) {}

// setupPlatformIdentity setzt Programmname und Prozessname fest auf
// "md-reader" (passend zu md-reader.desktop, StartupWMClass=md-reader).
//
// Muss vor ensureDisplayAvailable und webview.New aufgerufen werden, damit GTK
// beim Initialisieren schon den richtigen Namen kennt. Unabhängig davon, unter
// welchem Dateinamen die GUI gestartet wurde (umbenannt, memfd /proc/self/fd/N).
// Die WM_CLASS-Klasse setzt ensureDisplayAvailable (geht erst nach gtk_init).
func setupPlatformIdentity() {
	C.setProgramName()
}

// ensureDisplayAvailable beendet das Programm mit einer klaren Meldung und
// Exit-Code 1, wenn keine grafische Oberfläche erreichbar ist.
//
// Ohne diese Prüfung stürzt webview_go ohne Display mit SIGSEGV ab (Exit 2),
// weil webview.New dann ein Objekt mit NULL-Zeiger zurückgibt.
// Mit erreichbarer Oberfläche ist GTK danach initialisiert (webview.New
// übernimmt diesen Zustand) und die WM_CLASS-Klasse auf "md-reader" gesetzt.
func ensureDisplayAvailable() {
	if C.initDisplay() != 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "MD Reader: Keine grafische Oberfläche gefunden (DISPLAY/WAYLAND_DISPLAY nicht gesetzt oder nicht erreichbar).")
	os.Exit(1)
}
