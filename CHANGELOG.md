# Changelog

Alle wichtigen Änderungen an diesem Projekt werden in dieser Datei dokumentiert.

---

## [v1.1.0] - 2026-06-30

### Hinzugefügt
- **Manueller Reload-Shortcut (`r`)**: Der aktuelle Story-Feed kann in der Listenansicht jetzt direkt per Tastendruck aktualisiert werden, ohne die Kategorie zu wechseln.
- **Robustes API-Timeout**: Die API-Anfragen an Hacker News haben nun ein Timeout von 10 Sekunden, um unendliche Freezes der TUI bei Netzwerkstörungen zu vermeiden.
- **Sicherheitsgrenzen für Layouts**: Mindestbreiten-Constraints (20 Zeichen) für Titel und Kommentarboxen verhindern Abstürze (Panics) bei extrem schmal skalierten Terminalfenstern.

### Geändert
- **Globale Lese-Historie (`~/.hn-history.json`)**: Die Historie besuchter Stories wird nun global im Home-Verzeichnis gespeichert statt im aktuellen Arbeitsordner.
  - *Migration*: Beim ersten Start wird eine evtl. vorhandene lokale `.hn-history.json` automatisch in den globalen Pfad überführt und am alten Ort gelöscht.
- **Verbesserte HTML-Bereinigung in Kommentaren**: HTML-Links (`<a>`) und Code-Tags (`<code>`/`<pre>`) werden jetzt über Regex-Parser sauber in lesbaren Text übersetzt (z. B. `Text (URL)` oder Inline-Code).

### Behoben
- **Listen-Scroll-Drift gelöst**: Lange Titel in der Story-Liste werden nun dynamisch basierend auf der Terminal-Breite abgeschnitten. Das verhindert unerwünschte Zeilenumbrüche, welche die feste Scroll-Positionierung (`m.cursor * 3`) asynchron verschoben haben.

---

## [v1.0.0] - 2026-06-19

### Hinzugefügt
- **Bubble Tea & Lip Gloss UI**: Modernes TUI-Layout für Hacker News mit Farbschemata (Orange/Cyan/Grau).
- **Kategorie-Tabs**: Feed-Wechsel per Tab/Shift-Tab oder Zifferntasten `1` bis `5` (Top, New, Best, Ask, Show).
- **Echtzeit-Suche**: Schnelles Filtern von Stories nach Begriffen im Titel oder Autor mittels `/` (Löschen via `x`).
- **Maus-Support**: Native Scroll-Unterstützung für das Scrollrad der Maus in der Story-Liste sowie in den Kommentaren.
- **Farbcodierte Threads**: Verschachtelte Kommentarketten werden mit farbigen Rändern je nach Einrückungstiefe dargestellt.
- **Lese-Markierung**: Gelesene Stories erhalten ein visuelles Haken-Symbol (`✓`) und neu hinzugefügte Kommentare in bereits besuchten Stories erhalten ein grünes `[NEU]`-Tag.
- **Browser-Integration**: Öffnen von Story-Links direkt mit der Taste `o` (oder Erstellen von Beiträgen via `w`).
