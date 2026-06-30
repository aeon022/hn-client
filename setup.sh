#!/bin/bash
# hn-client Setup Utility

# Terminal-Farben definieren
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # Keine Farbe

echo -e "${BLUE}${BOLD}=========================================${NC}"
echo -e "${BLUE}${BOLD}        hn-client Setup & Installation   ${NC}"
echo -e "${BLUE}${BOLD}=========================================${NC}"
echo ""

# 1. Go Installation prüfen
echo -e "${BLUE}[1/3] Prüfe Go-Installation...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}Fehler: Go ist nicht installiert!${NC}"
    echo -e "Bitte installiere Go zuerst über Homebrew:"
    echo -e "  ${YELLOW}brew install go${NC}"
    exit 1
else
    GO_VERSION=$(go version)
    echo -e "${GREEN}✔ Go gefunden:${NC} $GO_VERSION"
fi
echo ""

# 2. Abhängigkeiten herunterladen
echo -e "${BLUE}[2/3] Lade Go-Abhängigkeiten herunter...${NC}"
if go mod download; then
    echo -e "${GREEN}✔ Abhängigkeiten erfolgreich geladen.${NC}"
else
    echo -e "${RED}Fehler beim Herunterladen der Abhängigkeiten!${NC}"
    exit 1
fi
echo ""

# 3. Binary kompilieren
echo -e "${BLUE}[3/3] Kompiliere hn-client...${NC}"
if go build -o hn-client .; then
    echo -e "${GREEN}✔ hn-client erfolgreich kompiliert.${NC}"
else
    echo -e "${RED}Fehler beim Kompilieren von hn-client!${NC}"
    exit 1
fi
echo ""

# 4. Optionaler globaler Installations-Schritt
echo -e "${BLUE}${BOLD}Globale Installation:${NC}"
echo -e "Möchtest du hn-client nach ${YELLOW}/usr/local/bin${NC} kopieren, damit du es von überall aus starten kannst?"
read -p "Installieren? (y/n): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Kopiere Binary nach /usr/local/bin... (sudo benötigt)${NC}"
    if sudo cp hn-client /usr/local/bin/hn-client; then
        echo -e "${GREEN}✔ Erfolgreich global installiert!${NC}"
        echo -e "Du kannst die App ab sofort überall mit dem Befehl ${GREEN}${BOLD}hn-client${NC} starten."
    else
        echo -e "${RED}Fehler beim Kopieren des Binaries. Du kannst es lokal ausführen:${NC}"
        echo -e "  ${YELLOW}./hn-client${NC}"
    fi
else
    echo -e "${YELLOW}Lokale Ausführung gewählt.${NC}"
    echo -e "Du kannst die TUI lokal über diesen Befehl starten:"
    echo -e "  ${GREEN}${BOLD}./hn-client${NC}"
fi

echo ""
echo -e "${GREEN}${BOLD}Setup erfolgreich abgeschlossen! 🎉${NC}"
