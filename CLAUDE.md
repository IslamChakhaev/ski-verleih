# Ski-Verleih

## Ziel

Implementiere einen einfachen, vollständig lauffähigen Prototyp für einen Ski-Verleih mit Dashboard. Bearbeite **Aufgabe 1, Teil a) Ski-Verleih (2 Punkte)** und **Teil b) Flexibilisierung der Preise (3 Punkte)** im selben Projekt.

Priorität: **funktional korrekt, benutzerfreundlich, schnell verständlich und leicht änderbar**. Vermeide Overengineering.

## Funktionsumfang

### Teil a) – Ski-Verleih

- Erstelle eine übersichtliche, möglichst benutzerfreundliche Weboberfläche mit Dashboard.
- Verwalte Ski-Artikel mit eindeutiger ID, Bezeichnung und Verfügbarkeit.
- Verwalte Personen/Kunden mit Name und Ausweisdaten.
- Erfasse einen Ausweis durch **Ausweistyp und Ausweisnummer**. Ein einfaches Eingabeformular genügt; ein Datei-Upload ist dafür nicht notwendig.
- Ermögliche das Anlegen einer Ausleihe mit Kunde, Ski-Artikel, **Verleihbeginn und Verleihdauer**.
- Ermögliche die Rückgabe und aktualisiere die Verfügbarkeit entsprechend.
- Ein Ski-Artikel darf nicht mehrfach gleichzeitig aktiv ausgeliehen sein.
- Das Dashboard zeigt die Anzahl der **verfügbaren Ski**, **ausgeliehenen Ski**, **gesamten Ski** und **registrierten Personen**. Die Werte müssen zum tatsächlichen Datenbestand passen.
- Stelle eine einfache Navigation zu Dashboard, Ski-Verwaltung, Personen und Ausleihen bereit.

### Teil b) – Flexibilisierung der Preise

- Ergänze denselben Ski-Verleih um **änderbare Preise/Tarife**.
- Unterstütze eine nachvollziehbare **stundenweise Preisberechnung** anhand von Tarif und Verleihdauer.
- Tarife sollen über eine einfache Oberfläche bearbeitet werden können, statt ausschließlich fest im Code zu stehen.
- Zeige den berechneten Preis bei der Ausleihe an.
- Verwende frei gewählte, nachvollziehbare Beispielpreise; konkrete Euro-Beträge sind nicht festgelegt.
- Validierung: keine negativen Preise, keine ungültigen Zeitangaben und keine unzulässigen Ausleihen.
- Der Preis einer bereits gespeicherten Ausleihe bleibt auch nach einer Tarifänderung nachvollziehbar.

**Umfang:** Implementiere jetzt ausschließlich Teil a) und b). Kein separates Fahrradverleih-Projekt und kein Plugin-Framework in diesem Projekt.

## Technologien

- **Backend:** Go mit Structs und Methoden; Interfaces nur, wenn sie tatsächlich helfen.
- **Frontend:** HTML, CSS und Vanilla JavaScript; kein Flutter, React oder Vue.
- **Datenhaltung:** thread-sicherer In-Memory-Store, z. B. `map` mit `sync.RWMutex`. Keine SQL-Datenbank und kein ORM.
- **Kommunikation:** HTTP/JSON. Das Frontend greift über Go-Endpoints auf Daten zu, nie direkt auf In-Memory-Strukturen.
- **Versionierung:** lokales Git; GitHub nur bei Bedarf.
- Bevorzuge die Go-Standardbibliothek. **Keine neuen Dependencies ohne Rückfrage.**

## Architektur

Halte die Struktur einfach:

```text
ski-verleih/
├── main.go                 # Serverstart und Routing
├── go.mod
├── internal/
│   ├── domain/             # Datenmodelle, Methoden, Geschäftsregeln
│   ├── store/              # Thread-sichere In-Memory-Datenhaltung
│   └── handler/            # HTTP-Handler und JSON
├── frontend/
│   ├── index.html
│   ├── style.css
│   └── app.js
├── CLAUDE.md
└── README.md
```

- Trenne Geschäftslogik, Datenhaltung, HTTP und Benutzeroberfläche (Separation of Concerns, Single Responsibility).
- Kapsle Zustandsänderungen und halte die Geschäftsregeln zentral und testbar.
- Hänge Methoden an Structs, wo es sinnvoll ist; verwende keine künstlichen Abstraktionen.
- Interfaces nur bei konkretem Nutzen; **kein Dependency-Injection-Framework**.
- Keine globalen Variablen außer einem Store-Singleton, falls wirklich erforderlich.
- Fehler nicht verschlucken; bei Bedarf mit `fmt.Errorf("%w", err)` wrappen.
- Schreibe lesbaren, wartbaren Code ohne unnötige Frameworks oder Design Patterns.

## Frontend

- Alle sichtbaren Beschriftungen und Fehlermeldungen auf **Deutsch**.
- Dashboard mit Kennzahlen, klarer Navigation, Tabellen, Formularen und verständlichen Rückmeldungen.
- Orientiere das Layout am Bild `references/dashboard-beispiel.jpg`, sofern es im Projekt vorhanden ist. Keine pixelgenaue Kopie notwendig.
- Funktionierende Schaltflächen für die benötigten Abläufe: Kunde/Ski erfassen, Ausleihe anlegen, Rückgabe durchführen, Tarife bearbeiten.
- Verwende möglichst wenig JavaScript; klare Trennung von UI und Backend-API.
- Responsives, schlichtes Design. Keine unnötigen Animationen.
- Liefere die statischen Dateien möglichst über den Go-Server aus, sodass der Prototyp mit einem Befehl startet.

## Validierung und Tests

- Prüfe unbekannte IDs, fehlende Pflichtangaben, doppelte Ausleihe, ungültige Dauer und ungültige Preise.
- Eine bereits zurückgegebene Ausleihe darf nicht erneut zurückgegeben werden.
- Teste die Geschäftsregeln und Preisberechnung mit `go test ./...`.
- Teste außerdem die wesentlichen HTTP-Abläufe und die Korrektheit der Dashboard-Zahlen nach Ausleihe und Rückgabe.
- Behebe Testfehler. Behaupte keine erfolgreichen Tests, die nicht tatsächlich ausgeführt wurden.

## Dokumentation und Abgabe

Erstelle eine kurze `README.md` mit:

- Funktionen des Projekts (Teil a und b).
- Voraussetzungen und genauer Startanleitung.
- Kurzer Anleitung zum Erfassen von Ski/Kunden, Ausleihen, Zurückgeben und Ändern der Preise.
- Testbefehl und Hinweisen auf Einschränkungen, insbesondere Datenverlust beim Neustart wegen In-Memory-Speicherung.

Das Projekt muss sich als **ein ZIP-Archiv für Teil a) und b)** abgeben lassen. Ins Archiv gehören Quellcode und notwendige Projektdateien; keine unnötigen Build-Artefakte oder echten personenbezogenen Daten.

## Befehle

```bash
go run .           # Anwendung starten
go test ./...      # Tests ausführen
go fmt ./...       # Go-Code formatieren
# golangci-lint run  # Nur falls eingerichtet
```

Für das HTML-Frontend ist kein eigener Build-Prozess nötig.

## Arbeitsregeln

- Setze die Anforderungen schrittweise und in kleinen, überprüfbaren Änderungen um.
- Stelle vor größeren Änderungen einen kurzen Plan vor.
- Keine generierten Dateien von Hand bearbeiten.
- Keine Secrets im Code. Für Beispiele nur fiktive Kunden- und Ausweisdaten verwenden.
- Commit-Messages kurz und im Imperativ formulieren.
- Bei wesentlichen Unklarheiten nachfragen; keine zusätzlichen Pflichtfunktionen erfinden.
- Bevorzuge einen vorführbaren, funktionierenden Prototyp gegenüber unnötiger technischer Komplexität.
