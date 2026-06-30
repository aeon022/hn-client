# Hacker News Terminal Client (hn-client)

Ein moderner, schneller und minimalistischer Hacker News Client direkt für dein Terminal. Entwickelt in Go mit den Bibliotheken `bubbletea` (TUI-Framework) und `lipgloss` (Styling).

---

## 🚀 Features

- **Übersichtliche Story-Liste**: Anzeige von Punkten, Autor, Veröffentlichungszeitpunkt und Kommentaranzahl.
- **Dynamische Titel-Kürzung**: Verhindert unschöne Zeilenumbrüche und Behebung des Scroll-Drifts (Offset-Verschiebung) in der Story-Liste.
- **Farbige Kategorien (Tabs)**: Einfacher Wechsel zwischen *Top*, *New*, *Best*, *Ask* und *Show* Stories.
- **Echtzeit-Filterung & Suche**: Filtern der aktuellen Stories nach Titel und Autor mit `/`.
- **Säuberung von Kommentaren**: Konvertiert HTML-Links (`<a>`) und Code-Formatierungen (`<code>`/`<pre>`) in sauberen Terminal-Text.
- **Vim-Style Navigation**: Navigieren ganz klassisch mit `j`/`k` oder den Pfeiltasten.
- **Maus-Support (Hijacked Scroll)**: Nahtloses Scrollen mit dem Mausrad direkt in der Liste und im Kommentarbereich.
- **Farbcodierte Kommentar-Threads**: Verschachtelte Kommentare haben farbige Ränder je nach Einrückungstiefe für bessere Lesbarkeit.
- **Browser-Integration**: Öffnen des Original-Links einer Story direkt im Standardbrowser mit `o`.
- **Globale Lese-Historie**: Ordnerunabhängige Persistenz gelesener Stories unter `~/.hn-history.json` mit automatischer Alt-Daten-Migration.
- **API-Timeout**: 10-Sekunden-Timeout verhindert unendliche TUI-Freezes bei Netzwerkproblemen.

---

## 🛠️ Installation & Start

### Voraussetzungen
Stelle sicher, dass **Go (Version 1.21 oder höher)** installiert ist.
Unter macOS kannst du Go via Homebrew installieren:
```bash
brew install go
```

### Setup & Ausführen

1. **Abhängigkeiten herunterladen**:
   ```bash
   go mod download
   ```

2. **App im Entwicklungsmodus starten**:
   ```bash
   go run main.go
   ```

3. **App kompilieren (optional)**:
   ```bash
   go build -o hn-client
   ./hn-client
   ```

---

## ⌨️ Tastenkombinationen (Keybindings)

### In der Story-Liste

| Taste(n) / Maus | Aktion |
| :--- | :--- |
| `j` / `↓` / **Mausrad ab** | Cursor nach unten bewegen (Liste scrollt mit) |
| `k` / `↑` / **Mausrad auf** | Cursor nach oben bewegen (Liste scrollt mit) |
| `Tab` | Nächste Kategorie wählen (Top -> New -> Best -> Ask -> Show) |
| `Shift + Tab` | Vorherige Kategorie wählen |
| `1` bis `5` | Direktwahl der Kategorie (`1`: Top, `2`: New, `3`: Best, `4`: Ask, `5`: Show) |
| `Enter` | Story-Details & Kommentare öffnen |
| `r` | Aktiven Feed neu laden (Aktualisieren) |
| `o` | Original-Link im Standard-Webbrowser öffnen |
| `/` | Echtzeit-Suche aktivieren (Titel/Autor filtern) |
| `x` | Aktiven Filter löschen (vollständige Liste anzeigen) |
| `w` | Neuen HN-Beitrag verfassen (öffnet die Submit-Seite im Browser) |
| `?` | Hilfe-Overlay öffnen/schließen |
| `q` / `Ctrl + C` | Anwendung beenden |

### In der Detailansicht (Kommentare)

| Taste(n) / Maus | Aktion |
| :--- | :--- |
| `j` / `↓` / **Mausrad ab** | Kommentare nach unten scrollen |
| `k` / `↑` / **Mausrad auf** | Kommentare nach oben scrollen |
| `Esc` / `q` / `Backspace` | Zurück zur Story-Liste |
| `o` | Original-Link im Standard-Webbrowser öffnen |
| `r` | Auf Story antworten (öffnet die Reply-Seite im Browser) |
| `?` | Hilfe-Overlay öffnen/schließen |

---

## 📂 Projektstruktur

- **[main.go](file:///Users/gweiher/Developing/Projects/hn-client/main.go)**: Der Einstiegspunkt der Anwendung. Enthält die komplette TUI-Zustandsverwaltung, Eingabebehandlung und das UI-Rendering.
- **[internal/hnapi/api.go](file:///Users/gweiher/Developing/Projects/hn-client/internal/hnapi/api.go)**: Die API-Integration. Holt Daten von der offiziellen Hacker News Firebase API im JSON-Format ab.
- **[docs/](file:///Users/gweiher/Developing/Projects/hn-client/docs)**: Weitere Konzeptentwürfe, Entwicklungsanleitungen und die Roadmap.
