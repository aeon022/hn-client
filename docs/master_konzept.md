# HN-Client Master Konzept

## Vision
Ein moderner, schneller und minimalistischer Hacker News Client für das Terminal. Fokus auf Lesbarkeit, Tastaturbedienung (Vim-Style) und später Interaktivität (Posten/Antworten).

## Technischer Stack
- **Sprache:** Go (1.21+)
- **UI-Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (The Elm Architecture)
- **Styling:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **API:** Offizielle Hacker News API (Firebase) & evtl. Scraper für Schreibzugriffe.

## UI/UX Design
- **Navigation:** Vim-Keybindings (`j`, `k`, `g`, `G`, `/` für Suche).
- **Layout:** 
  - Header (Titel & Status)
  - Content (Liste von Stories oder Kommentar-Baum)
  - Footer (Hilfe & Shortcuts)
- **Design:** Modernes Terminal-Styling mit dezenten Farben und klaren Abgrenzungen.

## Features (Geplant)
- [ ] Anzeige von Top, New, Ask, Show Stories.
- [ ] Ausklappbare Kommentar-Bäume.
- [ ] Öffnen von Links im Browser.
- [ ] Lokaler Cache für gelesene Inhalte.
- [ ] Login & Reply-Funktion.
- [ ] Benachrichtigungen bei Antworten.
