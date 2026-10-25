// Nachfragen als Desktop-Benachrichtigung mit Aktionsknöpfen über D-Bus
// (org.freedesktop.Notifications, reines Go mit github.com/godbus/dbus/v5).
//
// Jede Frage öffnet eine eigene, private Verbindung zum Session-Bus und
// schließt sie danach wieder. Die Adresse wird selbst ermittelt (Umgebung
// bzw. $XDG_RUNTIME_DIR/bus) statt über dbus.SessionBusPrivate: Letzteres
// würde ohne Bus per dbus-launch einen neuen Bus starten und die Umgebung
// des Prozesses verändern.
//
// Antworten werden nur angenommen, wenn der Absender (eindeutiger Bus-Name)
// der aktuelle Eigentümer von org.freedesktop.Notifications ist – einmal pro
// Frage per GetNameOwner ermittelt – und die Nachrichten-ID passt. So kann ein
// anderes Programm am Bus keine Antwort per gezieltem Signal fälschen.
//
// Autor: Kurt Ingwer
// Letzte Änderung: 2026-10-25 (Korrekturrunde 2: Knopfprüfung, Rückfall nur bis Notify)
package linuxsetup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	// notifyName ist der Bus-Name des Benachrichtigungsdienstes.
	notifyName = "org.freedesktop.Notifications"
	// notifyPath ist der Objektpfad des Dienstes.
	notifyPath = dbus.ObjectPath("/org/freedesktop/Notifications")
	// notifyIface ist die Schnittstelle des Dienstes.
	notifyIface = "org.freedesktop.Notifications"
	// notifyAppName ist der angezeigte Programmname.
	notifyAppName = "MD Reader"
	// notifyAppIcon ist der Icon-Name aus dem Icon-Theme (wie die Desktop-ID).
	notifyAppIcon = AppID
)

// callTimeout begrenzt Verbindungsaufbau und jeden einzelnen D-Bus-Aufruf
// (Variable, damit Tests sie verkürzen können).
var callTimeout = 5 * time.Second

// sessionBusAddress ermittelt die Adresse des Session-Busses, ohne einen
// Bus zu starten.
//
// Reihenfolge: DBUS_SESSION_BUS_ADDRESS (außer "autolaunch:"), dann der
// Socket $XDG_RUNTIME_DIR/bus bzw. /run/user/<uid>/bus, falls vorhanden.
//
// @param env Funktion zum Lesen von Umgebungsvariablen.
// @return Bus-Adresse oder "" (kein Bus).
func sessionBusAddress(env func(string) string) string {
	if address := env("DBUS_SESSION_BUS_ADDRESS"); address != "" {
		if strings.HasPrefix(address, "autolaunch:") {
			// autolaunch würde einen neuen Bus starten – das wollen wir nie.
			return ""
		}
		return address
	}
	runtimeDir := env("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(runtimeDir) {
		runtimeDir = ""
	}
	if runtimeDir == "" {
		uid := os.Getuid()
		if uid < 0 {
			// Windows: keine Benutzer-ID, kein Session-Bus.
			return ""
		}
		runtimeDir = filepath.Join("/run/user", strconv.Itoa(uid))
	}
	socket := filepath.Join(runtimeDir, "bus")
	if _, err := os.Stat(socket); err != nil {
		return ""
	}
	return "unix:path=" + dbus.EscapeBusAddressValue(socket)
}

// connectBus öffnet eine eigene, private Verbindung (Auth und Hello inklusive).
//
// Achtung godbus: Der per dbus.WithContext übergebene Kontext bestimmt die
// Lebensdauer der ganzen Verbindung (bei Abbruch wird sie geschlossen). Deshalb
// bekommt die Verbindung einen eigenen Kontext, der nur während des Aufbaus
// durch callTimeout begrenzt ist – sonst endete eine Frage nach callTimeout.
//
// @param parent Übergeordneter Kontext (Abbruch schließt die Verbindung).
// @param address Bus-Adresse.
// @return conn Verbindung; release gibt Verbindung und Kontext frei (immer aufrufen).
func connectBus(parent context.Context, address string) (conn *dbus.Conn, release func(), err error) {
	if address == "" {
		return nil, nil, errors.New("kein Session-Bus")
	}
	connCtx, cancel := context.WithCancel(parent)
	// Hängt der Bus beim Anmelden, beendet der Zeitgeber den Aufbau.
	timer := time.AfterFunc(callTimeout, cancel)
	conn, err = dbus.Connect(address, dbus.WithContext(connCtx))
	if !timer.Stop() {
		// Zeitgeber ist schon abgelaufen: Aufbau gilt als gescheitert.
		if conn != nil {
			_ = conn.Close()
		}
		cancel()
		return nil, nil, errors.New("Zeitüberschreitung beim Verbinden mit dem Session-Bus")
	}
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return conn, func() {
		_ = conn.Close()
		cancel()
	}, nil
}

// probeNotifications fragt die Fähigkeiten des Benachrichtigungsdienstes ab.
//
// @param ctx Kontext mit Zeitlimit (Verbindung und Aufruf).
// @param address Bus-Adresse.
// @return caps Fähigkeiten; ok false, wenn kein Bus oder kein Dienst erreichbar ist.
func probeNotifications(ctx context.Context, address string) (caps []string, ok bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	conn, release, err := connectBus(ctx, address)
	if err != nil {
		return nil, false
	}
	defer release()
	call := conn.Object(notifyName, notifyPath).CallWithContext(ctx, notifyIface+".GetCapabilities", 0)
	if call.Err != nil {
		return nil, false
	}
	if err := call.Store(&caps); err != nil {
		return nil, false
	}
	return caps, true
}

// dbusAsker fragt per Desktop-Benachrichtigung mit Aktionsknöpfen.
type dbusAsker struct {
	// address ist die Adresse des Session-Busses.
	address string
	// bodyMarkup: Der Dienst wertet Auszeichnung im Text aus → maskieren.
	bodyMarkup bool
	// fallback übernimmt, wenn die Benachrichtigung nicht gezeigt werden kann.
	fallback Asker
}

// escapeMarkup maskiert die Zeichen, die ein Dienst mit "body-markup" als
// Auszeichnung lesen würde.
var escapeMarkup = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace

// notify zeigt eine Benachrichtigung über die bestehende Verbindung.
//
// @param expire Anzeigedauer in ms (0 = bleibt, -1 = Vorgabe des Dienstes).
// @return ID der Nachricht.
func (a *dbusAsker) notify(ctx context.Context, conn *dbus.Conn, title, body string, actions []string, expire int32) (uint32, error) {
	if a.bodyMarkup {
		body = escapeMarkup(body)
	}
	if actions == nil {
		actions = []string{}
	}
	var id uint32
	err := conn.Object(notifyName, notifyPath).CallWithContext(ctx, notifyIface+".Notify", 0,
		notifyAppName, uint32(0), notifyAppIcon, title, body, actions,
		map[string]dbus.Variant{}, expire).Store(&id)
	return id, err
}

// Ask zeigt die Frage als Benachrichtigung und wartet auf die Antwort.
//
// Ablauf: Signale ActionInvoked und NotificationClosed abonnieren, Notify mit
// den Knöpfen als Paaren (Schlüssel, Beschriftung) und ohne automatisches
// Ausblenden senden, dann auf ein Signal mit passender ID warten. Bei
// Zeitablauf wird die Nachricht per CloseNotification geschlossen und ""
// geliefert.
//
// Rückfall (kdialog/zenity/stderr) nur bis einschließlich Notify: Scheitern
// Verbindung, GetNameOwner, das Abonnieren oder Notify selbst, fragt der
// Rückfall. Ist die Nachricht einmal gezeigt, gibt es keinen Rückfall mehr –
// bricht der Bus danach ab, liefert Ask "" (nach der Knopf-Konvention gleich
// dem harmlosen letzten Knopf), damit der Nutzer nicht doppelt gefragt wird.
//
// @param title Titel (summary).
// @param body Fragetext.
// @param buttons 1 bis 3 Knöpfe; der Schlüssel ist der Aktionsname.
// @param timeout Höchste Wartezeit (0 = ohne Zeitlimit).
// @return Gewählter Schlüssel oder "" (Zeitablauf, geschlossen, unbekannte
// Aktion, Bus-Abbruch nach Notify); err errButtonCount bei 0 oder mehr als
// 3 Knöpfen (ohne Notify und ohne Rückfall).
func (a *dbusAsker) Ask(title, body string, buttons []Button, timeout time.Duration) (string, error) {
	if err := checkButtons(buttons); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	conn, release, err := connectBus(context.Background(), a.address)
	if err != nil {
		return a.fallbackAsk(title, body, buttons, timeout, err)
	}
	// Die private Verbindung wird nach der Frage immer geschlossen.
	defer release()

	// Eindeutigen Namen des Dienstes ermitteln; nur dessen Signale zählen.
	var owner string
	if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetNameOwner", 0,
		notifyName).Store(&owner); err != nil || owner == "" {
		if err == nil {
			err = errors.New("kein Eigentümer für " + notifyName)
		}
		return a.fallbackAsk(title, body, buttons, timeout, err)
	}

	// Erst abonnieren, dann senden – sonst könnte eine sofortige Antwort verloren gehen.
	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	for _, member := range []string{"ActionInvoked", "NotificationClosed"} {
		if err := conn.AddMatchSignalContext(ctx,
			dbus.WithMatchSender(notifyName),
			dbus.WithMatchObjectPath(notifyPath),
			dbus.WithMatchInterface(notifyIface),
			dbus.WithMatchMember(member)); err != nil {
			return a.fallbackAsk(title, body, buttons, timeout, err)
		}
	}

	actions := make([]string, 0, 2*len(buttons))
	for _, b := range buttons {
		actions = append(actions, b.Key, b.Label)
	}
	// expire 0: Die Nachricht blendet sich nicht selbst aus; das Zeitlimit regelt Ask.
	id, err := a.notify(ctx, conn, title, body, actions, 0)
	if err != nil {
		return a.fallbackAsk(title, body, buttons, timeout, err)
	}

	var timer <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timer = t.C
	}
	for {
		select {
		case sig, open := <-signals:
			if !open {
				// Verbindung nach Notify beendet: kein Rückfall, „keine Antwort“.
				return "", nil
			}
			if key, done := matchSignal(sig, owner, id, buttons); done {
				return key, nil
			}
		case <-timer:
			// Zeitablauf: Nachricht entfernen, „keine Antwort“.
			closeCtx, closeCancel := context.WithTimeout(context.Background(), callTimeout)
			_ = conn.Object(notifyName, notifyPath).CallWithContext(closeCtx,
				notifyIface+".CloseNotification", 0, id).Err
			closeCancel()
			return "", nil
		}
	}
}

// matchSignal wertet ein Signal aus.
//
// @param sig Empfangenes Signal.
// @param owner Eindeutiger Bus-Name des Benachrichtigungsdienstes.
// @param id ID der eigenen Nachricht.
// @param buttons Knöpfe der Frage.
// @return key Schlüssel bei ActionInvoked mit bekannter Aktion, sonst "";
// done true, wenn das Signal die eigene Nachricht betrifft (Antwort oder geschlossen).
func matchSignal(sig *dbus.Signal, owner string, id uint32, buttons []Button) (key string, done bool) {
	if sig == nil || sig.Path != notifyPath || len(sig.Body) < 2 {
		return "", false
	}
	if sig.Sender != owner {
		// Signal von einer anderen Verbindung (gefälscht oder fremd) → ignorieren.
		return "", false
	}
	sigID, ok := sig.Body[0].(uint32)
	if !ok || sigID != id {
		// Fremde Nachricht (anderes Programm oder ältere Frage) → weiter warten.
		return "", false
	}
	switch sig.Name {
	case notifyIface + ".ActionInvoked":
		action, _ := sig.Body[1].(string)
		for _, b := range buttons {
			if b.Key == action {
				return b.Key, true
			}
		}
		// Unbekannte Aktion (z. B. "default" beim Klick auf die Nachricht).
		return "", true
	case notifyIface + ".NotificationClosed":
		return "", true
	}
	return "", false
}

// fallbackAsk übergibt die Frage an den Rückfall, wenn D-Bus scheitert.
func (a *dbusAsker) fallbackAsk(title, body string, buttons []Button, timeout time.Duration, cause error) (string, error) {
	if a.fallback == nil {
		return "", cause
	}
	return a.fallback.Ask(title, body, buttons, timeout)
}

// Inform zeigt einen Hinweis als Benachrichtigung ohne Knöpfe und wartet
// nicht. Scheitert das, übernimmt der Rückfall.
//
// @param title Titel (summary).
// @param body Hinweistext.
func (a *dbusAsker) Inform(title, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	conn, release, err := connectBus(ctx, a.address)
	if err == nil {
		defer release()
		// expire -1: Anzeigedauer nach Vorgabe des Dienstes.
		if _, err = a.notify(ctx, conn, title, body, nil, -1); err == nil {
			return
		}
	}
	if a.fallback != nil {
		a.fallback.Inform(title, body)
	}
}
