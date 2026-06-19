# Entwickler-Anleitung

## Lokale Einrichtung

1. **Go Installieren:**
   Stelle sicher, dass Go 1.21 oder höher installiert ist (`brew install go`).

2. **Abhängigkeiten laden:**
   ```bash
   go mod download
   ```

3. **App ausführen:**
   ```bash
   go run main.go
   ```

## Projektstruktur

- `main.go`: Der Einstiegspunkt der App und die TUI-Logik.
- `docs/`: Dokumentation, Roadmap und Konzepte.
- `go.mod`: Abhängigkeiten und Moduldefinition.

## Tipps für Go-Einsteiger

- **Formatierung:** Nutze immer `go fmt ./...`, um den Code automatisch zu formatieren.
- **Typen:** Achte auf die strikte Typisierung. Wenn eine Funktion ein `int` erwartet, akzeptiert sie kein `float64`.
- **Fehlerbehandlung:** In Go prüfen wir Fehler explizit: `if err != nil { ... }`.
