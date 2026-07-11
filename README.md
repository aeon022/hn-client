# Hacker News Terminal Client (hn-client)

Ein moderner, schneller und minimalistischer Hacker News Client direkt für dein Terminal. Entwickelt in Go mit den Bibliotheken `bubbletea` (TUI-Framework) und `lipgloss` (Styling).

---

## 🚀 Features

- **Übersichtliche Story-Liste**: Anzeige von Punkten, Autor, Veröffentlichungszeitpunkt und Kommentaranzahl.
- **Responsive Titel-Formatierung**: Verhindert unschöne Zeilenumbrüche durch intelligentes Lip Gloss Zeilen-Wrapping unter Berücksichtigung des Fenster-Layouts.
- **Dynamische Scroll-Berechnung**: Dynamische Höhenmessung der Beiträge verhindert jeglichen Scroll-Drift.
- **Farbige Kategorien (Tabs)**: Einfacher Wechsel zwischen *Top*, *New*, *Best*, *Ask*, *Show* und *Mine* (eigene Beiträge).
- **Feed-Pagination**: Bequemes Blättern durch die Beiträge (jeweils 30 Stories pro Seite) mittels `n` und `p`.
- **Echtzeit-Filterung & Suche**: Filtern der aktuellen Stories nach Titel und Autor mit `/`.
- **Säuberung von Kommentaren**: Konvertiert HTML-Links (`<a>`) und Code-Formatierungen in sauberen Terminal-Text.
- **Smart Comments Depth**: Erhöhte Rekursionstiefe von bis zu 8 Ebenen, um auch tiefere Unterhaltungen vollständig lesen zu können.
- **Multi-Theme Support**: Wechsle live das Farbschema (`t` drücken) zwischen *Classic Orange*, *Dracula* (Dark Purple), *Nord* (Frost Cyan) und *Monokai*. Die Auswahl wird in `~/.hn-config.json` gespeichert.
- **Interaktiver Login**: Hinterlege deinen HN-Benutzernamen und dein Passwort (`l` drücken), um personalisierte Feeds anzuzeigen und Beiträge zu löschen.
- **Inline Quick-Reply & Vim-Unterstützung**: 
  * Antworte blitzschnell inline über den TUI-Footer (`r` drücken).
  * Verwende deinen bevorzugten Terminal-Editor (`vim` oder `nano`), um längere Kommentare zu verfassen (`e` drücken).
- **Vim-Style Navigation**: Navigieren ganz klassisch mit `j`/`k` oder den Pfeiltasten.
- **Maus-Support (Hijacked Scroll)**: Nahtloses Scrollen mit dem Mausrad direkt in der Liste und im Kommentarbereich.
- **Farbcodierte Kommentar-Threads**: Verschachtelte Kommentare haben farbige Ränder je nach Einrückungstiefe für bessere Lesbarkeit.
- **Duales Browser-Verhalten (`b`)**:
  * **GUI-Modus**: Öffnet Links in deinem Standard-Desktop-Browser (z. B. Chrome/Safari) via `o`.
  * **Terminal-Modus**: Öffnet Links direkt im Terminal (benötigt einen installierten CLI-Browser wie `w3m`, `lynx` oder `links`). Falls keiner installiert ist, wird ein TUI-Warnhinweis angezeigt und auf den GUI-Browser zurückgegriffen.
- **Globale Lese-Historie**: Ordnerunabhängige Persistenz gelesener Stories unter `~/.hn-history.json` mit automatischer Alt-Daten-Migration.
- **API-Timeout**: 10-Sekunden-Timeout verhindert unendliche TUI-Freezes bei Netzwerkproblemen.

---

## 🛠️ Installation & Start

### Voraussetzungen
Stelle sicher, dass **Go (Version 1.21 oder höher)** installiert ist.

Für ein optimales TUI-Browser-Erlebnis empfehlen wir die Installation von `w3m` (Terminal-Browser):
```bash
# macOS
brew install w3m

# Debian / Ubuntu / Linux Mint
sudo apt install w3m
```

### Schnelle Installation (Empfohlen)
Wir bieten ein einfaches Installationsskript an, das alle Abhängigkeiten auflöst, das Tool kompiliert und auf Wunsch global in `/usr/local/bin` installiert:
```bash
chmod +x setup.sh
./setup.sh
```

### Manueller Start & Ausführen

1. **Abhängigkeiten herunterladen**:
   ```bash
   go mod download
   ```

2. **App im Entwicklungsmodus starten**:
   ```bash
   go run main.go
   ```

3. **App manuell kompilieren**:
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
| `Tab` | Nächste Kategorie wählen (Top -> New -> Best -> Ask -> Show -> Mine) |
| `Shift + Tab` | Vorherige Kategorie wählen |
| `p` | Vorherige Seite (Pagination) |
| `n` | Nächste Seite (Pagination) |
| `1` bis `6` | Direktwahl der Kategorie (`1`: Top, `2`: New, `3`: Best, `4`: Ask, `5`: Show, `6`: Mine) |
| `Enter` | Story-Details & Kommentare öffnen |
| `r` | Aktiven Feed neu laden (Aktualisieren) |
| `o` | Original-Link (oder HN-Diskussionsseite bei Text-Posts) öffnen |
| `/` | Echtzeit-Suche aktivieren (Titel/Autor filtern) |
| `x` | Aktiven Filter löschen (vollständige Liste anzeigen) |
| `w` | Neuen HN-Beitrag verfassen (TUI Formular) |
| `d` | Ausgewählten eigenen Beitrag löschen (nur unter Kategorie *Mine*) |
| `l` | HN-Username & Login-Daten verwalten |
| `t` | Farbschema/Theme umschalten (Orange / Dracula / Nord / Monokai) |
| `b` | Browser-Modus umschalten (GUI vs. Terminal) |
| `?` | Hilfe-Overlay öffnen/schließen |
| `q` / `Ctrl + C` | Anwendung beenden |

### In der Detailansicht (Kommentare)

| Taste(n) / Maus | Aktion |
| :--- | :--- |
| `j` / `↓` / **Mausrad ab** | Kommentare nach unten scrollen |
| `k` / `↑` / **Mausrad auf** | Kommentare nach oben scrollen |
| `Esc` / `q` / `Backspace` | Zurück zur Story-Liste |
| `o` | Original-Link (oder HN-Diskussionsseite bei Text-Posts) öffnen |
| `r` | Inline-Antwort im TUI-Footer verfassen |
| `e` | Antwort im externen Editor (Vim/Nano) verfassen |
| `b` | Browser-Modus umschalten (GUI vs. Terminal) |
| `?` | Hilfe-Overlay öffnen/schließen |

---

## 📂 Projektstruktur

- **[main.go](file:///Users/gweiher/Developing/Projects/hn-client/main.go)**: Der Einstiegspunkt der Anwendung. Enthält die komplette TUI-Zustandsverwaltung, Eingabebehandlung und das UI-Rendering.
- **[internal/hnapi/api.go](file:///Users/gweiher/Developing/Projects/hn-client/internal/hnapi/api.go)**: Die API-Integration. Holt Daten von der offiziellen Hacker News Firebase API im JSON-Format ab.
- **[setup.sh](file:///Users/gweiher/Developing/Projects/hn-client/setup.sh)**: Automatisches Build- und Installationsskript für macOS und Linux.
- **[docs/](file:///Users/gweiher/Developing/Projects/hn-client/docs)**: Weitere Konzeptentwürfe, Entwicklungsanleitungen und die Roadmap.

---

## 🤖 Verwendung mit KI-Assistenten (AI-ready)

Dieses Repository ist optimiert für die Arbeit mit KI-Coding-Assistenten (wie Google Antigravity, Claude, ChatGPT, etc.):

1. **Verständlicher Code für KI-Kopiloten**: Die Dateistruktur ist hochgradig modularisiert. `internal/hnapi/` kapselt die Netzwerkzugriffe ab, während `main.go` die TUI-Zustände verwaltet. Dies macht es KI-Agenten sehr einfach, Code-Fragmente zu analysieren, Fehler zu beheben oder neue Features ohne Seiteneffekte hinzuzufügen.
2. **Lern-Sandbox (`TUTORIAL.md`)**: Das Projekt enthält ein detailliertes [Go-Tutorial](file:///Users/gweiher/Developing/Projects/hn-client/TUTORIAL.md). Du kannst deine KI bitten, dir Aufgaben aus diesem Tutorial zu stellen, deinen Code zu korrigieren oder dir Konzepte wie Goroutines und Channels anhand dieser Codebasis zu erklären.
3. **Verarbeitung der Lesegeschichte (`~/.hn-history.json`)**: Da alle besuchten Stories mit Unix-Zeitstempeln in einer einfachen JSON-Struktur gespeichert werden, kann eine KI (oder ein externes Python-Script) deine Lesegewohnheiten analysieren, personalisierte Themen-Zusammenfassungen erstellen oder Lesestatistiken generieren.
4. **Sichere Layout-Constraints**: Durch die integrierten Mindestbreiten und dynamischen String-Kürzungen können KI-Modelle Layout-Vorschläge einbringen, ohne das Risiko einzugehen, TUI-Panics bei Breitenänderungen zu erzeugen.
