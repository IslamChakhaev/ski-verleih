# Ski-Verleih

Prototyp eines Ski-Verleihs mit Web-Dashboard für Aufgabe 1 der Software-Engineering-Lehrveranstaltung. Das Projekt umfasst **Teil a) Ski-Verleih** und **Teil b) Flexibilisierung der Preise**.

## Funktionen

### Teil a) – Ski-Verleih

- **Dashboard:** verfügbare, ausgeliehene und gesamte Ski sowie registrierte Personen, jeweils aus dem aktuellen Datenbestand
- **Ski-Verwaltung:** Ski anlegen und bearbeiten, Verfügbarkeit einsehen
- **Personen:** Personen mit Name, Ausweistyp und Ausweisnummer anlegen und bearbeiten
- **Ausleihen:** Ausleihe mit Person, Ski, Verleihbeginn und Verleihdauer anlegen, Ski zurückgeben
- **Doppelausleihe verhindert:** Ein Ski kann nicht gleichzeitig mehrfach aktiv ausgeliehen sein.

### Teil b) – Flexible Preise

- **Tarife:** Tarife mit Stundenpreis über die Oberfläche anlegen und bearbeiten
- **Preisberechnung:** stundenweise Berechnung mit Anzeige des Preises bereits im Ausleihformular
- **Nachvollziehbarkeit:** Gespeicherte Ausleihen behalten ihren Preis auch nach einer Tarifänderung.

## Technologien

- **Backend:** Go, nur Standardbibliothek (`net/http`, `encoding/json`, `sync`, `embed`)
- **Frontend:** HTML, CSS und Vanilla JavaScript, ohne Build-Prozess
- **Datenhaltung:** thread-sicherer In-Memory-Store (`map` mit `sync.RWMutex`)
- **Kommunikation:** HTTP/JSON

## Projektstruktur

```text
ski-verleih/
├── main.go              # Serverstart, Demodaten, Auslieferung des Frontends
├── go.mod
├── internal/
│   ├── domain/          # Datenmodelle, Geschäftsregeln, Preisberechnung
│   ├── store/           # Thread-sichere Datenhaltung und Demodaten
│   └── handler/         # HTTP-Routen und JSON
├── frontend/            # index.html, style.css, app.js
├── references/          # Referenzbild für das Dashboard
├── CLAUDE.md
└── README.md
```

Die Geschäftsregeln liegen in `domain` und kennen weder HTTP noch die Datenhaltung. `store` führt zusammengehörige Änderungen unter einer Sperre aus, `handler` übersetzt nur zwischen HTTP und `store`. Das Frontend greift ausschließlich über die HTTP-API auf Daten zu.

## Starten

Voraussetzung ist **Go 1.22 oder neuer**. Weitere Abhängigkeiten gibt es nicht.

```bash
go run .
```

Der Befehl wird im Projektordner ausgeführt. Danach ist die Anwendung unter <http://127.0.0.1:8080> oder <http://localhost:8080> erreichbar. Beenden lässt sie sich mit `Strg+C`.

Beim Start werden fiktive Demodaten geladen: 20 Ski, 10 Personen, 3 Tarife und 3 aktive Ausleihen. Meldet der Start, dass die Adresse bereits verwendet wird, läuft der Server vermutlich schon in einem anderen Terminal.

## Tests

```bash
go test ./...
go test -race ./...   # Prüft zusätzlich auf Data Races (benötigt unter Windows einen C-Compiler)
```

Die Tests decken die Geschäftsregeln, die Preisberechnung, die Datenhaltung einschließlich gleichzeitiger Zugriffe sowie die HTTP-Abläufe und die Dashboard-Zahlen nach Ausleihe und Rückgabe ab.

## Bedienung

### Ski und Personen anlegen

1. **Ski-Verwaltung** oder **Personen** öffnen.
2. Oben rechts auf **Neuer Ski** oder **Neue Person** klicken, das Formular ausfüllen und bestätigen.
3. Zum Ändern in der Tabelle auf **Bearbeiten** klicken. Das Formular öffnet sich mit den vorhandenen Daten.

### Ausleihe anlegen

1. Im **Dashboard** oder unter **Ausleihen** auf **Neue Ausleihe** klicken.
2. Person, verfügbaren Ski und Tarif wählen.
3. Verleihbeginn und Verleihdauer in ganzen Stunden eingeben. Der Gesamtpreis wird direkt angezeigt.
4. **Ausleihe anlegen** klicken. Der Ski ist ab jetzt nicht mehr verfügbar, auch wenn der Verleihbeginn in der Zukunft liegt.

### Ski zurückgeben

1. Unter **Ausleihen** in der Ansicht **Aktuell** bei der passenden Ausleihe auf **Zurückgeben** klicken und bestätigen.
2. Der Ski ist sofort wieder verfügbar. Die Ausleihe erscheint in der Ansicht **Abgeschlossen**.

### Preise ändern

1. **Tarife** öffnen.
2. Bei einem Tarif auf **Bearbeiten** klicken oder mit **Neuer Tarif** einen Tarif anlegen.
3. Den Stundenpreis eingeben, zum Beispiel `6,50`, und speichern.

## Flexible Preise

Der Preis ergibt sich aus Verleihdauer und Stundenpreis des Tarifs:

```text
Gesamtpreis = Verleihdauer in Stunden × Stundenpreis
Beispiel:     3 Std. × 5,00 € = 15,00 €
```

- **Beispieltarife:** Kinder 3,00 €, Standard 5,00 € und Premium 8,00 € pro Stunde
- **Beträge in Cent:** Alle Beträge werden als ganze Cent gespeichert, um Rundungsfehler zu vermeiden.
- **Eine zentrale Berechnung:** Der Preis wird nur im Backend berechnet (`Tariff.Price`). Das Formular fragt ihn über `/api/price-quote` ab.
- **Preis bleibt erhalten:** Jede Ausleihe speichert beim Anlegen Tarifname, Stundenpreis und Gesamtpreis. Tarifänderungen gelten deshalb nur für neue Ausleihen.
- **Fester Preis:** Der Preis richtet sich nach der geplanten Dauer. Eine frühere oder spätere Rückgabe ändert ihn nicht.

## Einschränkungen

- **Datenverlust beim Neustart:** Alle Daten liegen nur im Arbeitsspeicher. Nach einem Neustart sind Änderungen verloren und die Demodaten werden neu geladen.
- **Kein Löschen:** Ski, Personen und Tarife können nicht gelöscht werden.
- **Keine Reservierungen:** Ein Ski ist vom Anlegen der Ausleihe bis zur Rückgabe gesperrt, unabhängig vom Verleihbeginn. Verleihbeginne in der Vergangenheit sind erlaubt, um Ausleihen nachtragen zu können.
- **Namensänderungen:** Geänderte Namen von Ski und Personen erscheinen auch bei früheren Ausleihen. Nur die Tarifdaten bleiben unverändert erhalten.
- **Keine Anmeldung:** Der Server ist nur lokal erreichbar, und die API weist Anfragen fremder Webseiten ab. Eine Benutzerverwaltung gibt es nicht. Der Prototyp ist nur für fiktive Daten gedacht und nicht für den Betrieb im Netzwerk.
- **Keine automatische Aktualisierung:** Änderungen aus einem anderen Browserfenster erscheinen erst beim nächsten Seitenwechsel oder nach der nächsten Aktion.
